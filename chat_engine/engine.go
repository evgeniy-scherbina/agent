package chat_engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/shared/constant"
)

type Conversation struct {
	ID       string     `json:"id"`
	Messages []*Message `json:"messages"`
}

func (conv *Conversation) AddMessage(msg *Message) {
	conv.Messages = append(conv.Messages, msg)
}

// ToOpenAIMessage converts a single Message to OpenAI format
func ToOpenAIMessage(msg *Message) openai.ChatCompletionMessageParamUnion {
	switch msg.Role {
	case "user":
		return openai.UserMessage(msg.Content)
	case "assistant":
		return ToOpenAIMessageWithTools(msg)
	case "tool":
		return openai.ToolMessage(msg.Content, msg.TollCallID)
	default:
		// Fallback for unknown roles
		return openai.UserMessage(msg.Content)
	}
}

// ToOpenAIMessageWithTools converts an assistant message to OpenAI format, including tool_calls if present
func ToOpenAIMessageWithTools(msg *Message) openai.ChatCompletionMessageParamUnion {
	if len(msg.ToolCalls) == 0 {
		return openai.AssistantMessage(msg.Content)
	}

	assistant := openai.ChatCompletionAssistantMessageParam{
		Content: openai.ChatCompletionAssistantMessageParamContentUnion{
			OfString: param.NewOpt(msg.Content),
		},
		ToolCalls: make([]openai.ChatCompletionMessageToolCallUnionParam, len(msg.ToolCalls)),
	}

	// Convert tool calls to OpenAI format
	for i, toolCall := range msg.ToolCalls {
		assistant.ToolCalls[i] = openai.ChatCompletionMessageToolCallUnionParam{
			OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
				ID:   toolCall.ID,
				Type: constant.Function("function"),
				Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
					Name:      toolCall.Name,
					Arguments: toolCall.Arguments,
				},
			},
		}
	}

	return openai.ChatCompletionMessageParamUnion{
		OfAssistant: &assistant,
	}
}

// ToOpenAIMessages return messages in a format which can be used in OpenAI API
func (conv *Conversation) ToOpenAIMessages() []openai.ChatCompletionMessageParamUnion {
	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(conv.Messages))
	for _, msg := range conv.Messages {
		openaiMessages = append(openaiMessages, ToOpenAIMessage(msg))
	}

	return openaiMessages
}

type Message struct {
	ID        string     `json:"ID"`
	Role      string     `json:"role"` // "user", "assistant", "tool"
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// If non-empty - means it's a response to LLM tool call request
	TollCallID string
}

type ToolCall struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatEngine struct {
	client         *openai.Client
	conversations  map[string]*Conversation
	processManager *ProcessManager
	conversationsMutex sync.RWMutex
}

func NewChatEngine(client *openai.Client) *ChatEngine {
	return &ChatEngine{
		client:         client,
		conversations:  make(map[string]*Conversation),
		processManager: NewProcessManager(),
		conversationsMutex: sync.RWMutex{},
	}
}

func (e *ChatEngine) GetConversation(conversationID string) *Conversation {
	e.conversationsMutex.RLock()
	conv := e.conversations[conversationID]
	e.conversationsMutex.RUnlock()

	return conv
}

func (e *ChatEngine) ListConversation() []*Conversation {
	conversations := make([]*Conversation, 0)
	for _, conv := range e.conversations {
		conversations = append(conversations, conv)
	}

	return conversations
}

func (e *ChatEngine) GetOrCreateConversation(conversationID string) *Conversation {
	// Get or create conversation
	e.conversationsMutex.Lock()
	conv, exists := e.conversations[conversationID]
	if !exists {
		conv = &Conversation{
			ID:       conversationID,
			Messages: make([]*Message, 0),
		}
		e.conversations[conversationID] = conv
	}
	e.conversationsMutex.Unlock()

	return conv
}

// MessageUpdateCallback is called whenever a new message is added during processing
type MessageUpdateCallback func(*Message)

func (e *ChatEngine) SendUserMessage(conversationID, content string) ([]*Message, error) {
	return e.SendUserMessageWithCallback(conversationID, content, nil)
}

func (e *ChatEngine) SendUserMessageWithCallback(conversationID, content string, callback MessageUpdateCallback) ([]*Message, error) {
	conv := e.GetOrCreateConversation(conversationID)

	userMessage := Message{
		ID:      fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:    "user",
		Content: content,
	}
	conv.AddMessage(&userMessage)
	if callback != nil {
		callback(&userMessage)
	}

	responseMessage, err := e.sendUserMessageToLLM(conv)
	if err != nil {
		return nil, err
	}
	conv.AddMessage(responseMessage)
	if callback != nil {
		callback(responseMessage)
	}

	log.Printf("going to execute %v tool calls", len(responseMessage.ToolCalls))
	toolMessages := make([]*Message, 0)
	if len(responseMessage.ToolCalls) > 0 {
		toolMessages, err = e.executeLLMRequestedToolCalls(conv, responseMessage.ToolCalls, callback)
		if err != nil {
			log.Printf("can't executeLLMRequestedToolCalls: %v", err)
			return nil, err
		}
	}

	allNewMessages := make([]*Message, 0)
	allNewMessages = append(allNewMessages, &userMessage) // Include user message
	allNewMessages = append(allNewMessages, responseMessage)
	allNewMessages = append(allNewMessages, toolMessages...)

	return allNewMessages, nil
}

func (e *ChatEngine) sendUserMessageToLLM(conv *Conversation) (*Message, error) {
	ctx := context.Background()

	params := openai.ChatCompletionNewParams{
		Messages: conv.ToOpenAIMessages(),
		Tools:    allTools,
		Model:    openai.ChatModelGPT4o,
	}

	completion, err := e.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	toolCalls := make([]ToolCall, len(completion.Choices[0].Message.ToolCalls))
	for i, toolCall := range completion.Choices[0].Message.ToolCalls {
		toolCalls[i] = ToolCall{
			ID:        toolCall.ID,
			Type:      string(toolCall.Type),
			Name:      toolCall.Function.Name,
			Arguments: toolCall.Function.Arguments,
		}
	}

	responseMessage := Message{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      "assistant",
		Content:   completion.Choices[0].Message.Content,
		ToolCalls: toolCalls,
	}

	return &responseMessage, nil
}

func (e *ChatEngine) executeLLMRequestedToolCalls(
	conv *Conversation,
	toolCalls []ToolCall,
	callback MessageUpdateCallback,
) ([]*Message, error) {
	allNewMessages := make([]*Message, 0)
	maxIterations := 10 // Prevent infinite loops
	iteration := 0

	for len(toolCalls) > 0 && iteration < maxIterations {
		iteration++
		log.Printf("Tool call iteration %d: executing %d tool calls", iteration, len(toolCalls))

		// Execute all tool calls in this round
	for _, toolCall := range toolCalls {
			var output string
			var err error

			switch toolCall.Name {
			case "bash_command":
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Arguments), &args); err != nil {
					log.Printf("Error parsing tool call arguments: %v", err)
				continue
			}
			command, ok := args["command"].(string)
			if !ok {
					log.Printf("Tool call missing command argument")
				continue
			}

				// Check if command should run in background
				background, _ := args["background"].(bool)
				if background {
					output, err = executeBashCommandBackground(command, e.processManager, conv.ID)
				} else {
					output, err = executeBashCommand(command)
			if err != nil {
				fmt.Printf("Error executing bash command: %v, output: %s\n", err, output)
					}
				}

			case "list_processes":
				processes := e.processManager.ListProcesses()
				if len(processes) == 0 {
					output = "No background processes running."
				} else {
					var lines []string
					for _, proc := range processes {
						duration := time.Since(proc.StartTime).Round(time.Second)
						lines = append(lines, fmt.Sprintf("PID: %d | Command: %s | Running for: %s", proc.PID, proc.Command, duration))
					}
					output = fmt.Sprintf("Running background processes (%d):\n%s", len(processes), strings.Join(lines, "\n"))
				}

			case "kill_process":
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Arguments), &args); err != nil {
					log.Printf("Error parsing tool call arguments: %v", err)
					continue
				}
				pidFloat, ok := args["pid"].(float64)
				if !ok {
					output = "Error: invalid PID"
					break
				}
				pid := int(pidFloat)
				err = e.processManager.KillProcess(pid)
				if err != nil {
					output = fmt.Sprintf("Error killing process: %v", err)
				} else {
					output = fmt.Sprintf("Successfully killed process %d", pid)
				}

			default:
				log.Printf("Unknown tool call: %s", toolCall.Name)
				continue
			}

			// Add tool response message
			toolMessage := Message{
				ID:         fmt.Sprintf("msg_%d", time.Now().UnixNano()),
				Role:       "tool",
				Content:    output,
				TollCallID: toolCall.ID,
			}
			conv.AddMessage(&toolMessage)
			allNewMessages = append(allNewMessages, &toolMessage)
			if callback != nil {
				callback(&toolMessage)
		}
	}

		// Get response from OpenAI after tool execution
	params := openai.ChatCompletionNewParams{
		Messages: conv.ToOpenAIMessages(),
		Tools:    allTools,
		Model:    openai.ChatModelGPT4o,
	}
		completion, err := e.client.Chat.Completions.New(context.Background(), params)
	if err != nil {
			return nil, fmt.Errorf("can't send message with tool responses: %v", err)
	}

		// Extract tool calls from the response
		toolCalls = make([]ToolCall, len(completion.Choices[0].Message.ToolCalls))
		for i, toolCall := range completion.Choices[0].Message.ToolCalls {
			toolCalls[i] = ToolCall{
				ID:        toolCall.ID,
				Type:      string(toolCall.Type),
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			}
		}

		// Create assistant message
		assistantMessage := Message{
			ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Role:      "assistant",
			Content:   completion.Choices[0].Message.Content,
			ToolCalls: toolCalls,
	}
		conv.AddMessage(&assistantMessage)
		allNewMessages = append(allNewMessages, &assistantMessage)
		if callback != nil {
			callback(&assistantMessage)
		}

		// If there are no more tool calls, we're done
		if len(toolCalls) == 0 {
			log.Printf("No more tool calls, conversation complete")
			break
		}
	}

	if iteration >= maxIterations {
		log.Printf("Warning: reached max iterations (%d) for tool calls", maxIterations)
	}

	return allNewMessages, nil
}
