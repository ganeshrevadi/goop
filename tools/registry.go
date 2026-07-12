package tools

// Registry holds all available tools and provides lookup/listing utilities.
type Registry struct {
	tools map[string]Tool
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{
		tools: make(map[string]Tool)
	}
}

// Register adds a tool to the registry, keyed by its Name.
func (r *Registry) Register(t Tool) {
		if r.tools == nil {
			r.tools = make(map[string]Tool)
		}
		r.tools[Name] = t
}

// Get looks up a tool by name. Returns the tool and true if found.
func (r *Registry) Get(name string) (Tool, bool) {
	value, ok := r.tools[name]
	return value, ok
}

// List returns all registered tools as a slice.
func (r *Registry) List() []Tool {
	var list []Tool{}
	
	for _, val := range r.tools{
		list = append(list, val);
	}

	return list
}

// Definitions returns all tools in the OpenAI tool format, ready to
// include in the LLM API request body.
func (r *Registry) Definitions() []map[string]any {
	var def []map[string]any
	
	for _, val := range r.tools{
		def = append(def, map[string]any{
			"type": "function",
			"function":map[string]any{
				"name": val.Name,
				"description": val.Description,
				"parameters": val.Parameters
			}
		})
	}	
	return def
}
