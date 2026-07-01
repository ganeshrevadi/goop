# goop — Agent Loop in Go

goop is an interactive coding agent that runs in your terminal. It connects to
Ollama (local LLMs) via an OpenAI-compatible API and follows a ReAct loop:
receive input → call LLM → execute tools → repeat until done.

## Features

- Zero external dependencies — pure Go stdlib
- Streaming responses via SSE
- Tool use: read, write, and execute shell commands
- Configurable model and timeout

## Prerequisites

- Go 1.26+
- [Ollama](https://ollama.ai) running locally with a model (e.g. `llama3.2`)

## Installation

```bash
git clone https://github.com/vamp/goop
cd goop
go build -o goop .
```

## Usage

```bash
./goop
```

Then type prompts at the `>` prompt. The agent can read/write files and run
shell commands. Type `exit` or `quit` to quit.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-model` | `llama3.2` | Ollama model name |
| `-timeout` | `120000` | Max duration per LLM call (ms) |

The model can also be set via the `GOOP_MODEL` environment variable.

## Project Structure

```
goop/
├── main.go             # CLI entrypoint
├── agent/              # Agent state and loop
├── llm/                # Ollama SSE client
└── tools/              # Tool registry and implementations
```

## License

MIT
