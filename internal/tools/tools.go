// Package tools implements the agent's tool registry: a calculator, a clock,
// and a safe URL fetcher. New tools are added by appending a Definition.
package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Definition describes one callable tool.
type Definition struct {
	Name        string
	Description string
	Parameters  map[string]any
	Execute     func(args map[string]any) string
}

// All returns every tool available to the agent.
func All() []Definition {
	return []Definition{
		{
			Name:        "calculator",
			Description: "Evaluate an arithmetic expression, e.g. (12.5*3 - 4) / 2. Supports + - * / % ^ and parentheses.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"expression": map[string]any{"type": "string"}},
				"required":   []string{"expression"},
			},
			Execute: func(args map[string]any) string {
				expr, _ := args["expression"].(string)
				v, err := Eval(expr)
				if err != nil {
					return "error: " + err.Error()
				}
				return fmt.Sprintf("%g", v)
			},
		},
		{
			Name:        "utc_time",
			Description: "Return the current UTC date and time in RFC3339 format.",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
			Execute: func(args map[string]any) string {
				return time.Now().UTC().Format(time.RFC3339)
			},
		},
		{
			Name:        "fetch_url",
			Description: "Fetch a URL over HTTP(S) and return its status code plus the first ~2000 characters of the body. Useful for reading small pages or APIs.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"url": map[string]any{"type": "string"}},
				"required":   []string{"url"},
			},
			Execute: func(args map[string]any) string {
				raw, _ := args["url"].(string)
				if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
					return "error: url must start with http:// or https://"
				}
				client := &http.Client{Timeout: 10 * time.Second}
				resp, err := client.Get(raw)
				if err != nil {
					return "error: " + err.Error()
				}
				defer resp.Body.Close()
				body, err := io.ReadAll(io.LimitReader(resp.Body, 200_000))
				if err != nil {
					return "error: " + err.Error()
				}
				text := string(body)
				if len(text) > 2000 {
					text = text[:2000] + "...[truncated]"
				}
				return fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, text)
			},
		},
	}
}

// ExecuteByName runs the named tool with JSON arguments, returning a string result.
func ExecuteByName(defs []Definition, name, jsonArgs string) string {
	for _, d := range defs {
		if d.Name == name {
			args := map[string]any{}
			_ = json.Unmarshal([]byte(jsonArgs), &args)
			return d.Execute(args)
		}
	}
	return "error: unknown tool " + name
}
