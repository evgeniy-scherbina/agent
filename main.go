package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/openai/openai-go/v2"
)

type Server struct {
	client *openai.Client
}

func main() {
	// Initialize OpenAI client
	client := openai.NewClient(
	//option.WithAPIKey(""), // Will use OPENAI_API_KEY env var
	)

	server := &Server{
		client: &client,
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
	_ = server
	//r.Post("/api/chat", server.handleSendMessage)
	//r.Get("/api/conversations/{id}", server.handleGetConversation)
	//r.Get("/api/conversations", server.handleListConversations)

	fmt.Println("Server starting on :8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal(err)
	}
}
