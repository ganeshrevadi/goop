package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vamp/goop/tools"
)

func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	a.AddUserMessage(input)

	toolDefs := a.registry.List()

	for iter := 0; iter < a.MaxIterations; iter++ {
		events, err := a.chat(ctx, a.model, a.Messages, toolDefs)
		if err != nil {
			return "", fmt.Errorf("chat: %w", err)
		}

		var content string
		var toolCalls []ToolCall
		finishReason := ""

	drain:
		for evt := range events {
			switch evt.Type {
			case "content":
				content += evt.Content
				fmt.Print(evt.Content)
			case "tool_call":
				toolCalls = accumulateToolCalls(toolCalls, evt.ToolCall)
			case "done":
				finishReason = evt.FinishReason
				break drain
			case "error":
				return "", evt.Error
			}
		}

		a.AddAssistantMessage(content, toolCalls)

		switch finishReason {
		case "stop":
			return content, nil
		case "tool_calls":
			if len(toolCalls) == 0 {
				return content, nil
			}
			fmt.Println()
			for _, tc := range toolCalls {
				result := a.executeToolCall(tc)
				fmt.Printf("  → %s (%s): %s\n", tc.Function.Name, tc.ID, result)
				a.AddToolResult(tc.ID, tc.Function.Name, result)
			}
		default:
			return content, nil
		}
	}

	return "", fmt.Errorf("agent loop: reached max iterations (%d)", a.MaxIterations)
}

// accumulateToolCalls merges streaming tool call deltas into a single
// ToolCall slice.
func accumulateToolCalls(existing []ToolCall, delta *ToolCall) []ToolCall {
	if delta == nil {
		return existing
	}
	if delta.ID != "" {
		existing = append(existing, *delta)
		return existing
	}
	if len(existing) > 0 {
		last := &existing[len(existing)-1]
		last.Function.Arguments += delta.Function.Arguments
	}
	return existing
}

// executeToolCall runs a single tool call and returns the result string.
func (a *Agent) executeToolCall(toolCall ToolCall) string {
	var args map[string]any
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return fmt.Sprintf("Error: failed to parse arguments: %v", err)
	}

	tool, ok := a.registry.Get(toolCall.Function.Name)
	if !ok {
		return fmt.Sprintf("Error: tool not found: %s", toolCall.Function.Name)
	}

	result, err := tool.Execute(args)
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, result)
	}
	return result
}

// ensure registry implements tools.Registry interface
var _ *tools.Registry
