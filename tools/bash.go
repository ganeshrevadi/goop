package tools

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// NewBashTool creates a tool that executes shell commands.
func NewBashTool() Tool {
	return Tool{
		Name:        "bash",
		Description: "Execute a shell command",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Shell command to execute",
				},
				"timeout": map[string]any{
					"type":        "integer",
					"description": "Timeout in milliseconds (optional, default 30000)",
				},
			},
			"required": []any{"command"},
		},
		Execute: func(args map[string]any) (string, error) {
			cmd, _ := args["command"].(string)
			if cmd == "" {
				return "", fmt.Errorf("command can't be empty")
			}

			timeoutMs := 30000
			if t, ok := args["timeout"].(float64); ok {
				timeoutMs = int(t)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
			defer cancel()

			out, err := exec.CommandContext(ctx, "sh", "-c", cmd).CombinedOutput()
			return string(out), err
		},
	}
}
