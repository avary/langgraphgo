package tool

import "context"

// Tool is an executable capability an LLM agent can invoke.
//
// The interface is intentionally identical in shape to
// github.com/tmc/langchaingo/tools.Tool, so any value implementing one also
// implements the other. This lets the framework own its tool abstraction while
// remaining a drop-in for langchaingo tools without conversion.
type Tool interface {
	// Name returns the tool's unique identifier.
	Name() string
	// Description explains, in natural language, what the tool does and how to
	// call it. Models use this to decide when to invoke the tool.
	Description() string
	// Call runs the tool with the given input and returns its textual result.
	Call(ctx context.Context, input string) (string, error)
}
