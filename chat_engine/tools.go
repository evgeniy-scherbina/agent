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
			Description: openai.String("Execute a bash command and return the output"),
			Parameters: openai.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]string{
						"type":        "string",
						"description": "The bash command to execute",
					},
				},
				"required": []string{"command"},
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
