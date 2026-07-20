package tools

import (
	"fmt"
	"os"
	"strings"
)

func NewReadTool() Tool {
	return Tool{
		Name:        "read",
		Description: "Read file contents, optionally with line range",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "File path to read",
				},
				"offset": map[string]any{
					"type":        "integer",
					"description": "Starting line number (1-indexed, optional)",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum lines to return (optional)",
				},
			},
			"required": []any{"path"},
		},
		Execute: func(args map[string]any) (string, error) {
			path, _ := args["path"].(string)
			if path == "" {
				return "", fmt.Errorf("path is required")
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("read file: %w", err)
			}

			content := string(data)

			offset, hasOffset := args["offset"].(float64)
			limit, hasLimit := args["limit"].(float64)

			if hasOffset || hasLimit {
				lines := strings.Split(content, "\n")
				start := 0
				if hasOffset {
					start = int(offset) - 1
					if start < 0 {
						start = 0
					}
				}
				end := len(lines)
				if hasLimit {
					end = start + int(limit)
					if end > len(lines) {
						end = len(lines)
					}
				}
				if start >= len(lines) {
					return "", nil
				}
				content = strings.Join(lines[start:end], "\n")
			}

			return content, nil
		},
	}
}
