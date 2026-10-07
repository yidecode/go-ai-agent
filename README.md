# go-ai-agent

A minimal, **zero-dependency** ReAct-style AI agent in Go: the model thinks,
calls tools, observes results, and repeats until it has a final answer.

It speaks the OpenAI-compatible `/chat/completions` API, so it works with
OpenAI, DeepSeek, Ollama, vLLM, or any compatible backend — just point
`LLM_BASE_URL` at it.

## Features

- **Tool-calling loop** — system prompt, tool definitions, iterative
  observe → act loop with a configurable step budget
- **Built-in tools** — `calculator` (arithmetic expression parser),
  `utc_time`, `fetch_url` (safe fetcher with timeout + size cap)
- **Zero dependencies** — stdlib only; easy to audit and vendor
- **Traceable** — `-v` prints every tool call and result to stderr
- **Tested** — agent loop tested against a scripted fake LLM server

## Quickstart

```bash
# with OpenAI
export LLM_API_KEY=sk-...
go run ./cmd/agent -prompt "What is 15% of 240, plus the current UTC hour as a number?"

# with local Ollama (no key needed)
export LLM_BASE_URL=http://localhost:11434/v1
export LLM_MODEL=qwen2.5:7b
go run ./cmd/agent -v -prompt "Calculate (12.5*3 - 4) / 2 and tell me the UTC time"
```

The prompt can also come from arguments or stdin:

```bash
echo "What time is it in UTC?" | go run ./cmd/agent
```

## Configuration

| Variable       | Default                       | Description                        |
|----------------|-------------------------------|------------------------------------|
| `LLM_API_KEY`  | *(empty)*                     | API key (not needed for Ollama)    |
| `LLM_BASE_URL` | `https://api.openai.com/v1`   | OpenAI-compatible base URL         |
| `LLM_MODEL`    | `gpt-4o-mini`                 | Model name                         |

Flags: `-prompt`, `-max-steps` (default 8), `-v` (verbose trace).

## How it works

```
┌────────┐  prompt   ┌───────┐  chat/completions  ┌───────────┐
│  CLI   │ ────────► │ Agent │ ─────────────────► │ LLM backend│
└────────┘           └───┬───┘                    └───────────┘
                        │ tool_calls
                        ▼
                   ┌──────────┐
                   │  Tools   │  calculator · utc_time · fetch_url
                   └──────────┘
```

1. The agent sends the conversation (system + user) with tool definitions.
2. If the model requests tool calls, the agent executes them locally and
   appends the results as `tool` messages.
3. Repeat until the model answers directly or `max-steps` is reached.

## Project layout

```
cmd/agent            CLI: flags, env config, prompt from args/stdin
internal/llm         minimal OpenAI-compatible chat client
internal/agent       the ReAct loop + step tracing
internal/tools       tool registry
  tools.go           calculator / utc_time / fetch_url definitions
  calc.go            recursive-descent arithmetic parser
```

## Adding a tool

Append a `Definition` in `internal/tools/tools.go`:

```go
{
    Name:        "my_tool",
    Description: "What it does, when to use it.",
    Parameters:  map[string]any{ /* JSON schema */ },
    Execute: func(args map[string]any) string {
        return "result as text"
    },
},
```

No other changes needed — the agent picks it up automatically.

## Testing

```bash
go test ./...
go vet ./...
```

## License

MIT — see [LICENSE](LICENSE).
