package chat_engine

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/openai/openai-go/v2"
)

var (
	allTools = []openai.ChatCompletionToolUnionParam{
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "bash_command",
			Description: openai.String("Execute a bash command and return the output. Use background=true for long-running commands like servers."),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]string{
						"type":        "string",
						"description": "The bash command to execute",
					},
					"background": map[string]any{
						"type":        "boolean",
						"description": "If true, run the command in the background. Use for long-running commands like servers. Returns process ID instead of output.",
					},
				},
				"required": []string{"command"},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "list_processes",
			Description: openai.String("List all currently running background processes started by bash_command"),
			Parameters: openai.FunctionParameters{
				"type":       "object",
				"properties": map[string]any{},
			},
		}),
		openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        "kill_process",
			Description: openai.String("Kill a background process by its process ID (PID)"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"pid": map[string]any{
						"type":        "integer",
						"description": "The process ID (PID) to kill",
					},
				},
				"required": []string{"pid"},
			},
		}),
	}
)

// executeBashCommand executes a bash command and returns the output
func executeBashCommand(command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("empty command")
	}

	// Use bash to execute the command to handle quotes and special characters properly
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing bash command: %v, output: %s, command: %s\n", err, output, command)
		return string(output), err
	}

	return string(output), nil
}

// executeBashCommandBackground executes a bash command in the background and returns the process info
func executeBashCommandBackground(command string, pm *ProcessManager, conversationID string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("empty command")
	}

	info, err := pm.StartProcess(command, conversationID)
	if err != nil {
		return "", fmt.Errorf("failed to start background process: %w", err)
	}

	return fmt.Sprintf("Started background process (PID: %d)\nCommand: %s", info.PID, info.Command), nil
}
