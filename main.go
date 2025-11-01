package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/evgeniy-scherbina/agent/chat_engine"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/openai/openai-go/v2"
)

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversationId,omitempty"`
}

// SendMessageResponse represents a response from the chat
type SendMessageResponse struct {
	Messages []*chat_engine.Message `json:"messages"`
	Error    string                 `json:"error,omitempty"`
}

// Server holds the application state
type Server struct {
	client     *openai.Client
	chatEngine *chat_engine.ChatEngine
}

func main() {
	// Initialize OpenAI client
	client := openai.NewClient(
		//option.WithAPIKey(""), // Will use OPENAI_API_KEY env var
	)

	server := &Server{
		client:     &client,
		chatEngine: chat_engine.NewChatEngine(&client),
	}

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Routes
	r.Post("/api/chat", server.handleSendMessage)
	r.Get("/api/conversations/{id}", server.handleGetConversation)
	r.Get("/api/conversations", server.handleListConversations)

	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", r)
}

// handleSendMessage processes chat messages
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use provided conversation ID or default
	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = "default"
	}

	newMessages, err := s.chatEngine.SendUserMessage(conversationID, req.Message)
	if err != nil {
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SendMessageResponse{
		Messages: newMessages,
	})
}

// handleGetConversation returns a specific conversation
func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := chi.URLParam(r, "id")

	conv := s.chatEngine.GetConversation(conversationID)

	// If conversation doesn't exist, create it (especially for "default")
	if conv == nil {
		conv = s.chatEngine.GetOrCreateConversation(conversationID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

// handleListConversations returns all conversations
func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	conversations := s.chatEngine.ListConversation()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversations)
}
