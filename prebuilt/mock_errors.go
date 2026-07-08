package prebuilt

import (
	"context"
	"fmt"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/tmc/langchaingo/tools"
)

// Enforce that MockToolError implements tools.Tool
var _ tools.Tool = (*MockToolError)(nil)

// MockLLMError for testing GenerateContent error
type MockLLMError struct{}

func (m *MockLLMError) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	return nil, fmt.Errorf("mock LLM GenerateContent error")
}

func (m *MockLLMError) Call(ctx context.Context, prompt string, options ...llmtypes.CallOption) (string, error) {
	return "", fmt.Errorf("mock LLM Call error")
}

// MockLLMEmptyContent for testing empty content response
type MockLLMEmptyContent struct{}

func (m *MockLLMEmptyContent) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	return &llmtypes.ContentResponse{
		Choices: []*llmtypes.ContentChoice{
			{
				Content: "", // Empty content
			},
		},
	}, nil
}

func (m *MockLLMEmptyContent) Call(ctx context.Context, prompt string, options ...llmtypes.CallOption) (string, error) {
	return "", nil // Not used for this test scenario
}

// MockToolError for testing tool execution error
type MockToolError struct {
	name string
}

func (t *MockToolError) Name() string        { return t.name }
func (t *MockToolError) Description() string { return "A mock tool that returns an error" }
func (t *MockToolError) Call(ctx context.Context, input string) (string, error) {
	return "", fmt.Errorf("mock tool execution error")
}
