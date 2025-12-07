package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

type Server struct {
	client     *openai.Client
	chatEngine *chat_engine.ChatEngine
}

func main() {
	// Initialize OpenAI client
	client := openai.NewClient(
	//option.WithAPIKey(""), // Will use OPENAI_API_KEY env var
	)

	chatEngine, err := chat_engine.NewChatEngine(&client)
	if err != nil {
		log.Fatalf("Failed to initialize chat engine: %v", err)
	}

	server := &Server{
		client:     &client,
		chatEngine: chatEngine,
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

	// API Routes
	r.Route("/api", func(r chi.Router) {
		r.Post("/chat", server.handleSendMessage)
		r.Post("/chat/stream", server.handleSendMessageStream)
		r.Get("/conversations/{id}", server.handleGetConversation)
		r.Get("/conversations", server.handleListConversations)
		r.Get("/processes", server.handleListProcesses)
		r.Post("/processes/{pid}/kill", server.handleKillProcess)
	})

	// Serve static files from ui/dist
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "ui", "dist")
	
	// Serve static assets directory
	assetsDir := filepath.Join(filesDir, "assets")
	r.Handle("/assets/*", http.StripPrefix("/assets", http.FileServer(http.Dir(assetsDir))))
	
	// Catch-all handler for SPA: serve files if they exist, otherwise serve index.html
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Don't serve index.html for API routes
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}
		
		// Check if the requested file exists
		requestedPath := filepath.Join(filesDir, r.URL.Path)
		if info, err := os.Stat(requestedPath); err == nil && !info.IsDir() {
			// File exists, serve it
			http.ServeFile(w, r, requestedPath)
			return
		}
		
		// File doesn't exist, serve index.html for SPA routing
		indexPath := filepath.Join(filesDir, "index.html")
		http.ServeFile(w, r, indexPath)
	})

	fmt.Println("Server starting on :8080")
	fmt.Println("Serving frontend from: ui/dist")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
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

// handleSendMessageStream processes chat messages with Server-Sent Events streaming
func (s *Server) handleSendMessageStream(w http.ResponseWriter, r *http.Request) {
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

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a flusher to send data immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection message
	fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected"}`)
	flusher.Flush()

	// Callback to send messages as they're created
	callback := func(msg *chat_engine.Message) {
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshaling message for stream: %v", err)
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", string(msgJSON))
		flusher.Flush()
	}

	// Process message with streaming updates in a goroutine
	done := make(chan bool)
	go func() {
		defer func() {
			done <- true
		}()

		_, err := s.chatEngine.SendUserMessageWithCallback(conversationID, req.Message, callback)
		if err != nil {
			errorMsg := fmt.Sprintf(`{"type":"error","error":"%s"}`, err.Error())
			fmt.Fprintf(w, "data: %s\n\n", errorMsg)
			flusher.Flush()
		} else {
			// Send completion message
			fmt.Fprintf(w, "data: %s\n\n", `{"type":"done"}`)
			flusher.Flush()
		}
	}()

	// Keep connection alive and wait for completion
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-done:
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// handleListProcesses returns all running background processes
func (s *Server) handleListProcesses(w http.ResponseWriter, r *http.Request) {
	processes := s.chatEngine.GetProcesses()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processes)
}

// handleKillProcess kills a background process by PID
func (s *Server) handleKillProcess(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "pid")
	var pid int
	if _, err := fmt.Sscanf(pidStr, "%d", &pid); err != nil {
		http.Error(w, "Invalid PID", http.StatusBadRequest)
		return
	}

	err := s.chatEngine.KillProcess(pid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Process %d killed", pid),
	})
}
