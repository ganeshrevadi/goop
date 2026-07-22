// Challenge 2: LLM Client
//
// Goal: Define an abstraction over LLM providers and implement it for Ollama.
//
// What you're learning:
//   - Interface design for swappable backends
//   - The <-chan pattern for streaming data
//   - Context propagation for cancellation

package llm

import (
	"context"

	"github.com/vamp/goop/agent"
	"github.com/vamp/goop/tools"
)

// ChatStream is the interface any LLM provider must implement.
//
// It takes the full conversation history plus available tool definitions
// and returns a channel of stream events. The caller iterates over the
// channel until it closes (or ctx is cancelled).
//
// The model string is provider-specific, e.g. "llama3.2" for Ollama.
type Client interface {
	ChatStream(ctx context.Context, model string, messages []agent.Message, toolDefs []tools.Tool) (<-chan agent.StreamEvent, error)
}
