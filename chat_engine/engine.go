package chat_engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/openai/openai-go/v2"
)

type Conversation struct {
	ID       string     `json:"id"`
	Messages []*Message `json:"messages"`
}

func (conv *Conversation) AddMessage(msg *Message) {
	conv.Messages = append(conv.Messages, msg)
}

// ToOpenAIMessages return messages in a format which can be used in OpenAI API
func (conv *Conversation) ToOpenAIMessages() []openai.ChatCompletionMessageParamUnion {
	// Convert messages to OpenAI format
	var openaiMessages []openai.ChatCompletionMessageParamUnion
	for _, msg := range conv.Messages {
		if msg.Role == "user" {
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		} else if msg.Role == "assistant" {
			openaiMessages = append(openaiMessages, openai.AssistantMessage(msg.Content))
		} else if msg.Role == "tool" {
			openaiMessages = append(openaiMessages, openai.ToolMessage(msg.Content, msg.TollCallID))
		}
	}

	return openaiMessages
}

type Message struct {
	ID      string `json:"ID"`
	Role    string `json:"role"` // "user", "assistant", "tool"
	Content string `json:"content"`
	//ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// If non-empty - means it's a response to LLM tool call request
	TollCallID string
}

//type ToolCall struct {
//	ID        string `json:"id"`
//	Type      string `json:"type"`
//	Name      string `json:"name"`
//	Arguments string `json:"arguments"`
//}

type ChatEngine struct {
	client             *openai.Client
	conversations      map[string]*Conversation
	conversationsMutex sync.RWMutex
}

func NewChatEngine(client *openai.Client) *ChatEngine {
	return &ChatEngine{
		client:             client,
		conversations:      make(map[string]*Conversation),
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

func (e *ChatEngine) SendUserMessage(conversationID, content string) ([]*Message, error) {
	conv := e.GetOrCreateConversation(conversationID)

	userMessage := Message{
		ID:      fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:    "user",
		Content: content,
	}
	conv.AddMessage(&userMessage)

	responseMessage, err := e.sendUserMessageToLLM(conv)
	if err != nil {
		return nil, err
	}
	conv.AddMessage(responseMessage)

	//newMessages := make([]*Message, 0)
	//if len(responseMessage.ToolCalls) > 0 {
	//	newMessages, err = e.executeLLMRequestedToolCalls(conv, responseMessage.ToolCalls)
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	allNewMessages := make([]*Message, 0)
	allNewMessages = append(allNewMessages, &userMessage) // Include user message
	allNewMessages = append(allNewMessages, responseMessage)
	//allNewMessages = append(allNewMessages, newMessages...)

	return allNewMessages, nil
}

func (e *ChatEngine) sendUserMessageToLLM(conv *Conversation) (*Message, error) {
	ctx := context.Background()

	params := openai.ChatCompletionNewParams{
		Messages: conv.ToOpenAIMessages(),
		//Tools:    allTools,
		Model: openai.ChatModelGPT4o,
	}

	completion, err := e.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	//toolCalls := make([]ToolCall, len(completion.Choices[0].Message.ToolCalls))
	//for i, toolCall := range completion.Choices[0].Message.ToolCalls {
	//	toolCalls[i] = ToolCall{
	//		ID:        toolCall.ID,
	//		Type:      string(toolCall.Type),
	//		Name:      toolCall.Function.Name,
	//		Arguments: toolCall.Function.Arguments,
	//	}
	//}

	responseMessage := Message{
		ID:      fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:    "assistant",
		Content: completion.Choices[0].Message.Content,
		//ToolCalls: toolCalls,
	}

	return &responseMessage, nil
}
