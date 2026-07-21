package agent

import (
	"context"

	"github.com/vamp/goop/tools"
)

// ChatStreamFunc is the function signature for a streaming LLM call.
type ChatStreamFunc func(ctx context.Context, model string, messages []Message, toolDefs []tools.Tool) (<-chan StreamEvent, error)

// Agent is the core orchestrator. It manages the conversation loop:
// receive user input → call LLM → handle tool calls → repeat.
//
// Fields:
//   - chat: the streaming LLM function
//   - Messages: the full conversation history (exported so main can inspect/save it)
//   - registry: registered tools the LLM can call
//   - model: the model name to use (e.g. "llama3.2")
//   - MaxIterations: safety limit on tool-call rounds (default 25)
type Agent struct {
	chat          ChatStreamFunc
	Messages      []Message      `json:"messages"`
	registry      *tools.Registry
	model         string
	MaxIterations int            `json:"max_iterations"`
}

// New creates an Agent with the given chat function, tool registry, and model.
func New(chat ChatStreamFunc, registry *tools.Registry, model string) *Agent {
	return &Agent{
		chat:          chat,
		Messages:      []Message{},
		registry:      registry,
		model:         model,
		MaxIterations: 25,
	}
}

// AddUserMessage appends a user message to the conversation history.
func (a *Agent) AddUserMessage(content string) {
	a.Messages = append(a.Messages, Message{Role: "user", Content: content})
}

// AddAssistantMessage appends an assistant message with optional tool calls.
func (a *Agent) AddAssistantMessage(content string, toolCalls []ToolCall) {
	a.Messages = append(a.Messages, Message{Role: "assistant", Content: content, ToolCalls: toolCalls})
}

// AddToolResult appends the result of executing a tool.
func (a *Agent) AddToolResult(toolCallID, toolName, content string) {
	a.Messages = append(a.Messages, Message{Role: "tool", Content: content, ToolCallID: toolCallID, ToolName: toolName})
}

// ClearHistory resets the conversation history.
func (a *Agent) ClearHistory() {
	a.Messages = []Message{}
}
