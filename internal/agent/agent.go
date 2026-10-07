// Package agent implements a ReAct-style tool-calling loop: the model thinks,
// optionally calls tools, observes results, and repeats until it produces a
// final answer or the step budget is exhausted.
package agent

import (
	"context"
	"errors"

	"github.com/yidecode/go-ai-agent/internal/llm"
	"github.com/yidecode/go-ai-agent/internal/tools"
)

const systemPrompt = `You are a helpful assistant with access to tools. Think step by step.
Use tools when you need facts, calculations, or web content; do not guess what a tool can tell you.
When you have everything needed for the final answer, reply directly without calling more tools.`

// Step is one iteration of the loop, exposed for tracing/debugging.
type Step struct {
	Index      int
	Thought    string // assistant's textual content that step
	ToolName   string
	ToolArgs   string
	ToolResult string
}

// Agent runs the tool-calling loop against an LLM backend.
type Agent struct {
	client   *llm.Client
	tools    []tools.Definition
	maxSteps int
	onStep   func(Step)
}

// New builds an Agent.
func New(client *llm.Client, maxSteps int) *Agent {
	return &Agent{client: client, tools: tools.All(), maxSteps: maxSteps}
}

// OnStep registers a callback invoked after every tool execution.
func (a *Agent) OnStep(fn func(Step)) { a.onStep = fn }

// Run executes the loop for prompt and returns the final answer.
func (a *Agent) Run(ctx context.Context, prompt string) (string, error) {
	if a.maxSteps <= 0 {
		return "", errors.New("agent: maxSteps must be positive")
	}
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}
	defs := make([]llm.ToolDefinition, 0, len(a.tools))
	for _, t := range a.tools {
		defs = append(defs, llm.ToolDefinition{
			Type:     "function",
			Function: llm.FunctionDef{Name: t.Name, Description: t.Description, Parameters: t.Parameters},
		})
	}

	for i := 0; i < a.maxSteps; i++ {
		msg, err := a.client.Chat(ctx, messages, defs)
		if err != nil {
			return "", err
		}
		messages = append(messages, msg)
		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}
		for _, tc := range msg.ToolCalls {
			result := tools.ExecuteByName(a.tools, tc.Function.Name, tc.Function.Arguments)
			messages = append(messages, llm.Message{
				Role:       "tool",
				ToolCallID:  tc.ID,
				Content:    result,
			})
			if a.onStep != nil {
				a.onStep(Step{Index: i, Thought: msg.Content, ToolName: tc.Function.Name, ToolArgs: tc.Function.Arguments, ToolResult: result})
			}
		}
	}
	return "", errors.New("agent: max steps reached without a final answer")
}
