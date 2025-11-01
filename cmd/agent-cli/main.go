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
)

var sendMessageCmd = &cobra.Command{
	Use:   "send_message",
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

func init() {
	rootCmd.AddCommand(helloCmd)
	rootCmd.AddCommand(sendMessageCmd)

	// Flags for send_message command
	sendMessageCmd.Flags().StringVarP(&message, "message", "m", "", "Message to send to the agent (required)")
	sendMessageCmd.Flags().StringVarP(&conversationID, "conversation-id", "c", "", "Conversation ID (optional)")
	sendMessageCmd.Flags().StringVarP(&serverURL, "server", "s", "http://localhost:8080", "Server URL")

	sendMessageCmd.MarkFlagRequired("message")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
