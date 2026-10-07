// Package llm is a minimal client for OpenAI-compatible chat completion APIs
// (OpenAI, DeepSeek, Ollama, vLLM, ...). It speaks the /chat/completions
// endpoint with function-calling style tools.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message is a single chat message. ToolCalls is populated on assistant
// messages that request tool execution; ToolCallID links a tool result back.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall is one function invocation requested by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall carries the function name and its JSON arguments.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition describes a function the model may call.
type ToolDefinition struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef is the JSON-schema description of one function.
type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Client talks to an OpenAI-compatible API.
type Client struct {
	http    *http.Client
	baseURL string
	apiKey  string
	model   string
}

// NewClient builds a client. baseURL is like "https://api.openai.com/v1".
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		http:    &http.Client{Timeout: 120 * time.Second},
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
	}
}

// Chat sends the conversation and returns the assistant's next message.
func (c *Client) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (Message, error) {
	body, err := json.Marshal(chatRequest{
		Model:       c.model,
		Messages:    messages,
		Tools:       tools,
		Temperature: 0.2,
	})
	if err != nil {
		return Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return Message{}, err
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Message{}, fmt.Errorf("decode response: %w", err)
	}
	if out.Error != nil {
		return Message{}, fmt.Errorf("api error: %s", out.Error.Message)
	}
	if resp.StatusCode >= 400 {
		return Message{}, fmt.Errorf("api status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	if len(out.Choices) == 0 {
		return Message{}, fmt.Errorf("api returned no choices")
	}
	return out.Choices[0].Message, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
