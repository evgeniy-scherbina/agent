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

// AddMessageWithDB adds a message to the conversation and saves it to the database
func (conv *Conversation) AddMessageWithDB(msg *Message, db *DB) error {
	conv.Messages = append(conv.Messages, msg)
	return db.SaveMessage(conv.ID, msg)
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
// This function validates that assistant messages with tool_calls are followed by tool responses
func (conv *Conversation) ToOpenAIMessages() []openai.ChatCompletionMessageParamUnion {
	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(conv.Messages))
	
	// Track pending tool calls that need responses
	pendingToolCalls := make(map[string]bool)
	
	for _, msg := range conv.Messages {
		// If this is an assistant message with tool calls, track them
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				pendingToolCalls[toolCall.ID] = true
			}
		}
		
		// If this is a tool message, mark the corresponding tool call as resolved
		if msg.Role == "tool" && msg.TollCallID != "" {
			delete(pendingToolCalls, msg.TollCallID)
		}
		
		// Before adding an assistant message with tool_calls, check if previous tool calls were resolved
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 && len(pendingToolCalls) > 0 {
			// There are still pending tool calls from a previous assistant message
			// This indicates a corrupted state - we should add error tool messages
			log.Printf("WARNING: Found assistant message with tool_calls while previous tool calls are still pending. This may indicate a corrupted conversation state.")
			for toolCallID := range pendingToolCalls {
				// Add an error tool message for the missing response
				errorToolMsg := openai.ToolMessage(
					fmt.Sprintf("Error: missing tool response for tool_call_id %s. Conversation state may be corrupted.", toolCallID),
					toolCallID,
				)
				openaiMessages = append(openaiMessages, errorToolMsg)
				delete(pendingToolCalls, toolCallID)
			}
		}
		
		openaiMessages = append(openaiMessages, ToOpenAIMessage(msg))
	}
	
	// If there are still pending tool calls at the end, add error responses
	if len(pendingToolCalls) > 0 {
		log.Printf("WARNING: Conversation has %d pending tool calls without responses. Adding error tool messages.", len(pendingToolCalls))
		for toolCallID := range pendingToolCalls {
			errorToolMsg := openai.ToolMessage(
				fmt.Sprintf("Error: missing tool response for tool_call_id %s. Conversation state may be corrupted.", toolCallID),
				toolCallID,
			)
			openaiMessages = append(openaiMessages, errorToolMsg)
		}
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
	client             *openai.Client
	conversations      map[string]*Conversation
	processManager     *ProcessManager
	db                 *DB
	conversationsMutex sync.RWMutex
}

func NewChatEngine(client *openai.Client) (*ChatEngine, error) {
	// Initialize database
	db, err := NewDB("agent.db")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	engine := &ChatEngine{
		client:             client,
		conversations:      make(map[string]*Conversation),
		processManager:     NewProcessManager(),
		db:                 db,
		conversationsMutex: sync.RWMutex{},
	}

	// Load all conversations from database
	if err := engine.loadAllConversations(); err != nil {
		log.Printf("Warning: failed to load conversations from database: %v", err)
	}

	return engine, nil
}

func (e *ChatEngine) loadAllConversations() error {
	conversationIDs, err := e.db.ListConversations()
	if err != nil {
		return err
	}

	for _, id := range conversationIDs {
		conv, err := e.db.LoadConversation(id)
		if err != nil {
			log.Printf("Failed to load conversation %s: %v", id, err)
			continue
		}
		if conv != nil {
			e.conversationsMutex.Lock()
			e.conversations[id] = conv
			e.conversationsMutex.Unlock()
		}
	}

	log.Printf("Loaded %d conversations from database", len(conversationIDs))
	return nil
}

func (e *ChatEngine) GetConversation(conversationID string) *Conversation {
	e.conversationsMutex.RLock()
	conv := e.conversations[conversationID]
	e.conversationsMutex.RUnlock()

	// If not in memory, try loading from database
	if conv == nil {
		dbConv, err := e.db.LoadConversation(conversationID)
		if err != nil {
			log.Printf("Failed to load conversation from database: %v", err)
			return nil
		}
		if dbConv != nil {
			e.conversationsMutex.Lock()
			e.conversations[conversationID] = dbConv
			e.conversationsMutex.Unlock()
			return dbConv
		}
	}

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
	// Try to get from memory first
	e.conversationsMutex.RLock()
	conv := e.conversations[conversationID]
	e.conversationsMutex.RUnlock()

	if conv != nil {
		return conv
	}

	// Try loading from database
	dbConv, err := e.db.LoadConversation(conversationID)
	if err != nil {
		log.Printf("Failed to load conversation from database: %v", err)
	}

	if dbConv != nil {
		e.conversationsMutex.Lock()
		e.conversations[conversationID] = dbConv
		e.conversationsMutex.Unlock()
		return dbConv
	}

	// Create new conversation
	conv = &Conversation{
		ID:       conversationID,
		Messages: make([]*Message, 0),
	}

	// Save to database
	if err := e.db.SaveConversation(conv); err != nil {
		log.Printf("Failed to save new conversation to database: %v", err)
	}

	e.conversationsMutex.Lock()
	e.conversations[conversationID] = conv
	e.conversationsMutex.Unlock()

	return conv
}

// GetProcesses returns all running background processes
func (e *ChatEngine) GetProcesses() []*ProcessInfo {
	return e.processManager.ListProcesses()
}

// KillProcess kills a background process by PID
func (e *ChatEngine) KillProcess(pid int) error {
	return e.processManager.KillProcess(pid)
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
	if err := conv.AddMessageWithDB(&userMessage, e.db); err != nil {
		log.Printf("Failed to save user message to database: %v", err)
	}
	if callback != nil {
		callback(&userMessage)
	}

	responseMessage, err := e.sendUserMessageToLLM(conv)
	if err != nil {
		return nil, err
	}
	if err := conv.AddMessageWithDB(responseMessage, e.db); err != nil {
		log.Printf("Failed to save assistant message to database: %v", err)
	}
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
		Model:    openai.ChatModelGPT5,
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

		// Track which tool calls we've processed to ensure all get responses
		processedToolCallIDs := make(map[string]bool)
		
		// Execute all tool calls in this round
		for _, toolCall := range toolCalls {
			var output string

			switch toolCall.Name {
			case "bash_command":
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Arguments), &args); err != nil {
					log.Printf("Error parsing tool call arguments: %v", err)
					output = fmt.Sprintf("Error: failed to parse tool call arguments: %v", err)
				} else {
					command, ok := args["command"].(string)
					if !ok {
						log.Printf("Tool call missing command argument")
						output = "Error: missing required 'command' argument"
					} else {
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
					output = fmt.Sprintf("Error: failed to parse tool call arguments: %v", err)
				} else {
					pidFloat, ok := args["pid"].(float64)
					if !ok {
						output = "Error: invalid PID"
					} else {
						pid := int(pidFloat)
						err = e.processManager.KillProcess(pid)
						if err != nil {
							output = fmt.Sprintf("Error killing process: %v", err)
						} else {
							output = fmt.Sprintf("Successfully killed process %d", pid)
						}
					}
				}

			default:
				log.Printf("Unknown tool call: %s", toolCall.Name)
				output = fmt.Sprintf("Error: unknown tool call '%s'", toolCall.Name)
			}

			// ALWAYS add tool response message, even for errors
			// This ensures every tool_call_id has a corresponding tool message
			toolMessage := Message{
				ID:         fmt.Sprintf("msg_%d", time.Now().UnixNano()),
				Role:       "tool",
				Content:    output,
				TollCallID: toolCall.ID,
			}
			if err := conv.AddMessageWithDB(&toolMessage, e.db); err != nil {
				log.Printf("Failed to save tool message to database: %v", err)
			}
			allNewMessages = append(allNewMessages, &toolMessage)
			processedToolCallIDs[toolCall.ID] = true
			if callback != nil {
				callback(&toolMessage)
			}
		}

		// Validate that all tool calls have responses
		for _, toolCall := range toolCalls {
			if !processedToolCallIDs[toolCall.ID] {
				log.Printf("WARNING: Tool call %s was not processed, adding error response", toolCall.ID)
				errorMessage := Message{
					ID:         fmt.Sprintf("msg_%d", time.Now().UnixNano()),
					Role:       "tool",
					Content:    fmt.Sprintf("Error: tool call %s was not processed", toolCall.ID),
					TollCallID: toolCall.ID,
				}
				if err := conv.AddMessageWithDB(&errorMessage, e.db); err != nil {
					log.Printf("Failed to save error tool message to database: %v", err)
				}
				allNewMessages = append(allNewMessages, &errorMessage)
				if callback != nil {
					callback(&errorMessage)
				}
			}
		}

		// Validate conversation state before sending to OpenAI
		openaiMessages := conv.ToOpenAIMessages()
		
		// Double-check that all assistant messages with tool_calls have corresponding tool responses
		pendingToolCalls := make(map[string]bool)
		for _, msg := range openaiMessages {
			if msg.OfAssistant != nil && len(msg.OfAssistant.ToolCalls) > 0 {
				for _, tc := range msg.OfAssistant.ToolCalls {
					if tc.OfFunction != nil {
						pendingToolCalls[tc.OfFunction.ID] = true
					}
				}
			}
			if msg.OfTool != nil {
				delete(pendingToolCalls, msg.OfTool.ToolCallID)
			}
		}
		
		if len(pendingToolCalls) > 0 {
			log.Printf("ERROR: Attempting to send messages with %d unresolved tool calls. This will fail. Adding error tool messages.", len(pendingToolCalls))
			for toolCallID := range pendingToolCalls {
				errorToolMsg := openai.ToolMessage(
					fmt.Sprintf("Error: missing tool response for tool_call_id %s", toolCallID),
					toolCallID,
				)
				openaiMessages = append(openaiMessages, errorToolMsg)
			}
		}
		
		// Get response from OpenAI after tool execution
		params := openai.ChatCompletionNewParams{
			Messages: openaiMessages,
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
		if err := conv.AddMessageWithDB(&assistantMessage, e.db); err != nil {
			log.Printf("Failed to save assistant message to database: %v", err)
		}
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
