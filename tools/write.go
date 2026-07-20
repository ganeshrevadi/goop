package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// NewWriteTool creates a tool that writes content to a file.
func NewWriteTool() Tool {
	return Tool{
		Name:        "write",
		Description: "Create or overwrite a file with content",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File path to write to",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the file",
				},
			},
			"required": []any{"path", "content"},
		},
		Execute: func(args map[string]any) (string, error) {
			path, _ := args["path"].(string)
			if path == "" {
				return "", fmt.Errorf("path is required")
			}

			content, _ := args["content"].(string)

			dir := filepath.Dir(path)
			if dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return "", fmt.Errorf("create directories: %w", err)
				}
			}

			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return "", fmt.Errorf("write file: %w", err)
			}

			return fmt.Sprintf("Written %d bytes to %s", len(content), path), nil
		},
	}
}
