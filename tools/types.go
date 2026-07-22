package tools

// Tool is a capability registered with the agent.
//
// Each tool has a name, description, and parameter schema so the
// LLM knows when and how to call it. The Execute function runs
// the tool and returns a string result (or an error).
//
// Parameters uses JSON Schema format (as a map) so it serializes
// directly into the Ollama/OpenAI tools array.
//
// Example schema for a read tool:
//
//	map[string]any{
//	  "type": "object",
//	  "properties": map[string]any{
//	    "path": map[string]any{
//	      "type":        "string",
//	      "description": "File path to read",
//	    },
//	  },
//	  "required": []any{"path"},
//	}
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Execute     func(args map[string]any) (string, error)
}
