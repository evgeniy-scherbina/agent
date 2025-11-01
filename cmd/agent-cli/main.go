package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agent-cli",
	Short: "Agent CLI is a command-line tool for agent operations",
	Long: `Agent CLI is a command-line tool that provides various commands
for managing and interacting with the agent system.`,
}

var helloCmd = &cobra.Command{
	Use:   "hello",
	Short: "Print a hello message",
	Long:  `Print a friendly hello message from agent-cli.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello from agent-cli!")
	},
}

var (
	message        string
	conversationID string
	serverURL      string
	getConvID      string
	listConvURL    string
)

var sendMessageCmd = &cobra.Command{
	Use:   "send-message",
	Short: "Send a message to agent",
	Long:  `Send a message to the agent API server and receive a response.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if message == "" {
			return fmt.Errorf("message is required")
		}

		// Default server URL if not provided
		if serverURL == "" {
			serverURL = "http://localhost:8080"
		}

		// Prepare request
		reqBody := map[string]interface{}{
			"message": message,
		}
		if conversationID != "" {
			reqBody["conversationId"] = conversationID
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		// Make HTTP request
		url := serverURL + "/api/chat"
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check status code
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		// Parse and display response
		var apiResponse struct {
			Messages []struct {
				ID      string `json:"ID"`
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Error string `json:"error,omitempty"`
		}

		if err := json.Unmarshal(body, &apiResponse); err != nil {
			// If JSON parsing fails, just print the raw response
			fmt.Println(string(body))
			return nil
		}

		// Display messages
		if apiResponse.Error != "" {
			fmt.Printf("Error: %s\n", apiResponse.Error)
		}

		for _, msg := range apiResponse.Messages {
			fmt.Printf("[%s]: %s\n", msg.Role, msg.Content)
		}

		return nil
	},
}

var getConvCmd = &cobra.Command{
	Use:   "get-conv",
	Short: "Get a specific conversation by ID",
	Long:  `Retrieve a conversation by its ID from the agent API server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if getConvID == "" {
			return fmt.Errorf("conversation ID is required")
		}

		// Default server URL if not provided
		url := serverURL
		if url == "" {
			url = "http://localhost:8080"
		}

		// Make HTTP GET request
		apiURL := url + "/api/conversations/" + getConvID
		resp, err := http.Get(apiURL)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check status code
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		// Parse and display response
		var conversation struct {
			ID       string `json:"id"`
			Messages []struct {
				ID      string `json:"ID"`
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}

		if err := json.Unmarshal(body, &conversation); err != nil {
			// If JSON parsing fails, just print the raw response
			fmt.Println(string(body))
			return nil
		}

		// Display conversation
		fmt.Printf("Conversation ID: %s\n", conversation.ID)
		fmt.Printf("Messages (%d):\n\n", len(conversation.Messages))
		for _, msg := range conversation.Messages {
			fmt.Printf("[%s] %s: %s\n", msg.ID, msg.Role, msg.Content)
		}

		return nil
	},
}

var listConvCmd = &cobra.Command{
	Use:   "list-conv",
	Short: "List all conversations",
	Long:  `Retrieve a list of all conversations from the agent API server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default server URL if not provided
		url := listConvURL
		if url == "" {
			url = "http://localhost:8080"
		}

		// Make HTTP GET request
		apiURL := url + "/api/conversations"
		resp, err := http.Get(apiURL)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check status code
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}

		// Parse and display response
		var conversations []struct {
			ID       string `json:"id"`
			Messages []struct {
				ID      string `json:"ID"`
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}

		if err := json.Unmarshal(body, &conversations); err != nil {
			// If JSON parsing fails, just print the raw response
			fmt.Println(string(body))
			return nil
		}

		// Display conversations
		if len(conversations) == 0 {
			fmt.Println("No conversations found.")
			return nil
		}

		fmt.Printf("Found %d conversation(s):\n\n", len(conversations))
		for i, conv := range conversations {
			fmt.Printf("%d. Conversation ID: %s (%d messages)\n", i+1, conv.ID, len(conv.Messages))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(helloCmd)
	rootCmd.AddCommand(sendMessageCmd)
	rootCmd.AddCommand(getConvCmd)
	rootCmd.AddCommand(listConvCmd)

	// Flags for send_message command
	sendMessageCmd.Flags().StringVarP(&message, "message", "m", "", "Message to send to the agent (required)")
	sendMessageCmd.Flags().StringVarP(&conversationID, "conversation-id", "c", "", "Conversation ID (optional)")
	sendMessageCmd.Flags().StringVarP(&serverURL, "server", "s", "http://localhost:8080", "Server URL")

	sendMessageCmd.MarkFlagRequired("message")

	// Flags for get-conv command
	getConvCmd.Flags().StringVarP(&getConvID, "id", "i", "", "Conversation ID (required)")
	getConvCmd.Flags().StringVarP(&serverURL, "server", "s", "http://localhost:8080", "Server URL")
	getConvCmd.MarkFlagRequired("id")

	// Flags for list-conv command
	listConvCmd.Flags().StringVarP(&listConvURL, "server", "s", "http://localhost:8080", "Server URL")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
