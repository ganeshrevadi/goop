package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/vamp/goop/agent"
	"github.com/vamp/goop/llm"
	"github.com/vamp/goop/tools"
)

func main() {
	modelFlag := flag.String("model", "", "Ollama model name (default: llama3.2)")
	timeoutFlag := flag.Int("timeout", 120000, "Max duration per LLM call in ms")
	flag.Parse()

	model := *modelFlag
	if model == "" {
		model = os.Getenv("GOOP_MODEL")
	}
	if model == "" {
		model = "llama3.2"
	}

	fmt.Printf("goop — agent loop (model: %s)\n", model)

	ollamaClient := llm.NewOllamaClient("", "")
	chatFunc := func(ctx context.Context, m string, msgs []agent.Message, defs []tools.Tool) (<-chan agent.StreamEvent, error) {
		return ollamaClient.ChatStream(ctx, m, msgs, defs)
	}

	registry := tools.New()
	registry.Register(tools.NewReadTool())
	registry.Register(tools.NewWriteTool())
	registry.Register(tools.NewBashTool())
	registry.Register(tools.NewSearchTool())

	a := agent.New(chatFunc, registry, model)

	scanner := bufio.NewScanner(os.Stdin)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	fmt.Println("Type exit or quit to quit.")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()

		if line == "exit" || line == "quit" {
			break
		}
		if line == "" {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutFlag)*time.Millisecond)

		go func() {
			select {
			case <-sigCh:
				cancel()
			case <-ctx.Done():
			}
		}()

		_, err := a.Run(ctx, line)
		cancel()

		if err != nil {
			fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		}
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading input: %v\n", err)
		os.Exit(1)
	}
}
