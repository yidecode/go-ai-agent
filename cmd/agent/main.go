// Command agent runs a single ReAct-style tool-calling task from the CLI.
//
// Configuration comes from the environment:
//
//	LLM_API_KEY   API key for the backend (not needed for local Ollama)
//	LLM_BASE_URL  OpenAI-compatible base URL (default https://api.openai.com/v1)
//	LLM_MODEL     Model name (default gpt-4o-mini)
//
// The prompt is taken from -prompt, the remaining arguments, or stdin.
//
// Example:
//
//	LLM_API_KEY=sk-... go run ./cmd/agent -prompt "What is 15% of 240 plus the current UTC hour?"
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/yidecode/go-ai-agent/internal/agent"
	"github.com/yidecode/go-ai-agent/internal/llm"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	promptFlag := flag.String("prompt", "", "task prompt for the agent")
	maxSteps := flag.Int("max-steps", 8, "maximum tool-calling iterations")
	verbose := flag.Bool("v", false, "print each tool call and result")
	flag.Parse()

	prompt := *promptFlag
	if prompt == "" {
		prompt = strings.Join(flag.Args(), " ")
	}
	if prompt == "" {
		sc := bufio.NewScanner(os.Stdin)
		var b strings.Builder
		for sc.Scan() {
			b.WriteString(sc.Text() + "\n")
		}
		prompt = strings.TrimSpace(b.String())
	}
	if prompt == "" {
		fmt.Fprintln(os.Stderr, "no prompt given (use -prompt, arguments, or stdin)")
		os.Exit(2)
	}

	client := llm.NewClient(
		getenv("LLM_BASE_URL", "https://api.openai.com/v1"),
		os.Getenv("LLM_API_KEY"),
		getenv("LLM_MODEL", "gpt-4o-mini"),
	)
	ag := agent.New(client, *maxSteps)
	if *verbose {
		ag.OnStep(func(s agent.Step) {
			fmt.Fprintf(os.Stderr, "[step %d] tool=%s args=%s\n  -> %s\n", s.Index, s.ToolName, s.ToolArgs, s.ToolResult)
		})
	}

	answer, err := ag.Run(context.Background(), prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(answer)
}
