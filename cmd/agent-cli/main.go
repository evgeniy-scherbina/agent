package main

import (
	"fmt"
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

var sendMessageCmd = &cobra.Command{
	Use:   "send_message",
	Short: "Send a message to agent",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	rootCmd.AddCommand(helloCmd)
	rootCmd.AddCommand(sendMessageCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
