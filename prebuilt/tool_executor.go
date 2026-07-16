package prebuilt

import (
	"context"
	"fmt"

	"github.com/smallnest/langgraphgo/tool"
)

// ToolWithSchema is an optional interface that tools can implement to provide their parameter schema
type ToolWithSchema interface {
	Schema() map[string]any
}

// ToolInvocation represents a request to execute a tool
type ToolInvocation struct {
	Tool      string `json:"tool"`
	ToolInput string `json:"tool_input"`
}

// ToolExecutor executes tools based on invocations
type ToolExecutor struct {
	Tools map[string]tool.Tool
}

// NewToolExecutor creates a new ToolExecutor with the given tools
func NewToolExecutor(inputTools []tool.Tool) *ToolExecutor {
	toolMap := make(map[string]tool.Tool)
	for _, t := range inputTools {
		toolMap[t.Name()] = t
	}
	return &ToolExecutor{
		Tools: toolMap,
	}
}

// Execute executes a single tool invocation
func (te *ToolExecutor) Execute(ctx context.Context, invocation ToolInvocation) (string, error) {
	tool, ok := te.Tools[invocation.Tool]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", invocation.Tool)
	}

	return tool.Call(ctx, invocation.ToolInput)
}

// getToolSchema returns the parameter schema for a tool.
// If the tool implements ToolWithSchema, it uses the tool's custom schema.
// Otherwise, it returns the default simple schema with an "input" string field.
func getToolSchema(tool tool.Tool) map[string]any {
	if st, ok := tool.(ToolWithSchema); ok {
		return st.Schema()
	}
	// Default schema for tools without custom schema
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{
				"type":        "string",
				"description": "The input query for the tool",
			},
		},
		"required":             []string{"input"},
		"additionalProperties": false,
	}
}

// ExecuteMany executes multiple tool invocations in parallel (if needed, but here sequential for simplicity)
// In a real graph, this might be a ParallelNode, but here we provide a helper.
func (te *ToolExecutor) ExecuteMany(ctx context.Context, invocations []ToolInvocation) ([]string, error) {
	results := make([]string, len(invocations))
	for i, inv := range invocations {
		res, err := te.Execute(ctx, inv)
		if err != nil {
			return nil, err // Or continue and return partial errors?
		}
		results[i] = res
	}
	return results, nil
}

// ToolNode is a graph node function that executes tools
// It expects the state to contain a list of ToolInvocation or a single ToolInvocation
// This is a simplified version. In a real agent, it would parse messages.
func (te *ToolExecutor) ToolNode(ctx context.Context, state any) (any, error) {
	// Try to parse state as ToolInvocation
	if inv, ok := state.(ToolInvocation); ok {
		return te.Execute(ctx, inv)
	}

	// Try to parse as []ToolInvocation
	if invs, ok := state.([]ToolInvocation); ok {
		return te.ExecuteMany(ctx, invs)
	}

	// Try to parse from map
	if m, ok := state.(map[string]any); ok {
		// Check for "tool" and "tool_input" keys
		if t, ok := m["tool"].(string); ok {
			input := ""
			if i, ok := m["tool_input"].(string); ok {
				input = i
			}
			return te.Execute(ctx, ToolInvocation{Tool: t, ToolInput: input})
		}
	}

	return nil, fmt.Errorf("invalid state for ToolNode: %T", state)
}
