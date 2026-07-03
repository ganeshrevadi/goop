package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vamp/goop/agent"
	toolspkg "github.com/vamp/goop/tools"
)

// OllamaClient implements the Client interface for Ollama.
// It sends requests to Ollama's OpenAI-compatible endpoint and
// parses the SSE stream into StreamEvent values.
type OllamaClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewOllamaClient creates a new client for the given Ollama base URL.
func NewOllamaClient(baseURL, apiKey string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
	}
}

// ChatStream sends the conversation to Ollama and returns a channel of
// streaming events. The caller must drain the channel; it will be closed
// when the stream ends or ctx is cancelled.
func (c *OllamaClient) ChatStream(ctx context.Context, model string, messages []agent.Message, toolDefs []toolspkg.Tool) (<-chan agent.StreamEvent, error) {
	events := make(chan agent.StreamEvent)

	bodyMap := map[string]any{}
	bodyMap["model"] = model
	bodyMap["stream"] = true
	bodyMap["messages"] = buildMessages(messages)
	bodyMap["tools"] = buildTools(toolDefs)

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	go func() {
		defer resp.Body.Close()
		defer close(events)
		readSSEStream(resp.Body, events)
	}()

	return events, nil
}

// buildMessages converts agent.Message values into the OpenAI API format.
func buildMessages(messages []agent.Message) []map[string]any {
	llmMessage := []map[string]any{
		{"role": "system", "content": "You are goop, a helpful coding agent."},
	}

	for _, msg := range messages {
		res := map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		}

		switch msg.Role {
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				res["tool_calls"] = msg.ToolCalls
			}
		case "tool":
			if msg.ToolCallID != "" {
				res["tool_call_id"] = msg.ToolCallID
			}
		}

		llmMessage = append(llmMessage, res)
	}

	return llmMessage
}

// buildTools converts agent.Tool values into the OpenAI tool definition format.
func buildTools(toolDefs []toolspkg.Tool) []map[string]any {
	toolMap := []map[string]any{}

	for _, tool := range toolDefs {
		res := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		}

		toolMap = append(toolMap, res)
	}
	return toolMap
}

// ollamaStreamChunk represents a single SSE data line from Ollama.
type ollamaStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// readSSEStream reads an SSE stream from reader and sends StreamEvents
// on the provided channel. It handles content deltas, tool call deltas,
// and finish reason signals.
func readSSEStream(reader io.Reader, events chan<- agent.StreamEvent) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return
		}

		var ch ollamaStreamChunk
		if err := json.Unmarshal([]byte(data), &ch); err != nil {
			events <- agent.StreamEvent{Type: "error", Error: err}
			continue
		}

		for _, choice := range ch.Choices {
			if choice.Delta.Content != "" {
				events <- agent.StreamEvent{
					Type:    "content",
					Content: choice.Delta.Content,
				}
			}

			for _, tc := range choice.Delta.ToolCalls {
				toolCall := &agent.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
				}
				toolCall.Function.Name = tc.Function.Name
				toolCall.Function.Arguments = tc.Function.Arguments
				events <- agent.StreamEvent{
					Type:     "tool_call",
					ToolCall: toolCall,
				}
			}

			if choice.FinishReason != nil && *choice.FinishReason != "" {
				events <- agent.StreamEvent{
					Type:         "done",
					FinishReason: *choice.FinishReason,
				}
				return
			}
		}
	}
	if err := scanner.Err(); err != nil {
		events <- agent.StreamEvent{Type: "error", Error: err}
	}
}
