package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yidecode/go-ai-agent/internal/llm"
)

// fakeLLM plays a scripted two-turn conversation: first it requests the
// calculator tool, then it returns a final answer incorporating the result.
func fakeLLM(t *testing.T) *httptest.Server {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		w.Header().Set("Content-Type", "application/json")
		var msg llm.Message
		if call == 1 {
			msg = llm.Message{
				Role:    "assistant",
				Content: "I'll calculate that.",
				ToolCalls: []llm.ToolCall{{
					ID:   "call_1",
					Type: "function",
					Function: llm.FunctionCall{
						Name:      "calculator",
						Arguments: `{"expression":"(12.5*3 - 4) / 2"}`,
					},
				}},
			}
		} else {
			msg = llm.Message{Role: "assistant", Content: "The result is 16.75."}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": msg}},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAgentRunsToolLoop(t *testing.T) {
	srv := fakeLLM(t)
	client := llm.NewClient(srv.URL, "", "test-model")
	ag := New(client, 5)

	var steps []Step
	ag.OnStep(func(s Step) { steps = append(steps, s) })

	answer, err := ag.Run(context.Background(), "What is (12.5*3 - 4) / 2?")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(answer, "16.75") {
		t.Fatalf("answer %q does not contain 16.75", answer)
	}
	if len(steps) != 1 {
		t.Fatalf("got %d steps, want 1", len(steps))
	}
	if steps[0].ToolName != "calculator" {
		t.Fatalf("tool %q, want calculator", steps[0].ToolName)
	}
	if !strings.Contains(steps[0].ToolResult, "16.75") {
		t.Fatalf("tool result %q does not contain 16.75", steps[0].ToolResult)
	}
}

func TestAgentMaxSteps(t *testing.T) {
	// Server always requests a tool -> agent must give up after maxSteps.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": llm.Message{
				Role: "assistant",
				ToolCalls: []llm.ToolCall{{
					ID:       "c1",
					Type:     "function",
					Function: llm.FunctionCall{Name: "utc_time", Arguments: "{}"},
				}},
			}}},
		})
	}))
	t.Cleanup(srv.Close)

	client := llm.NewClient(srv.URL, "", "test-model")
	ag := New(client, 2)
	if _, err := ag.Run(context.Background(), "hi"); err == nil {
		t.Fatal("expected max-steps error, got nil")
	}
}
