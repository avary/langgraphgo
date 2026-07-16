package adapter

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/smallnest/langgraphgo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// mockLLM is a mock implementation of llms.Model for testing
type mockLLM struct {
	generateResponse      string
	generateError         error
	generateContentResult *llms.ContentResponse
	generateContentError  error
	calls                 []mockCall
}

type mockCall struct {
	method string
	prompt string
}

func (m *mockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.calls = append(m.calls, mockCall{method: "GenerateContent", prompt: messages[0].Parts[0].(llms.TextContent).Text})

	if m.generateContentError != nil {
		return nil, m.generateContentError
	}

	if m.generateContentResult != nil {
		return m.generateContentResult, nil
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: m.generateResponse,
			},
		},
	}, nil
}

func (m *mockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	m.calls = append(m.calls, mockCall{method: "Call", prompt: prompt})

	if m.generateError != nil {
		return "", m.generateError
	}
	return m.generateResponse, nil
}

func (m *mockLLM) GetNumTokens(text string) int {
	return len(text) // Simplified token count
}

func TestNewOpenAIAdapter(t *testing.T) {
	llm := &mockLLM{generateResponse: "test"}
	adapter := NewOpenAIAdapter(llm)

	if adapter == nil {
		t.Fatal("NewOpenAIAdapter returned nil")
	}

	if adapter.llm != llm {
		t.Error("adapter.llm is not the same as the provided llm")
	}
}

func TestOpenAIAdapter_Generate(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		response       string
		expectedResult string
	}{
		{
			name:           "successful generation",
			prompt:         "Hello, world!",
			response:       "Hello! How can I help you?",
			expectedResult: "Hello! How can I help you?",
		},
		{
			name:           "empty prompt",
			prompt:         "",
			response:       "Empty response",
			expectedResult: "Empty response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &mockLLM{generateResponse: tt.response}
			adapter := NewOpenAIAdapter(llm)

			ctx := context.Background()
			result, err := adapter.Generate(ctx, tt.prompt)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

func TestOpenAIAdapter_GenerateWithConfig(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		config         map[string]any
		response       string
		expectedResult string
	}{
		{
			name:           "no config",
			prompt:         "Test prompt",
			config:         nil,
			response:       "Response",
			expectedResult: "Response",
		},
		{
			name:           "with temperature",
			prompt:         "Test prompt",
			config:         map[string]any{"temperature": 0.7},
			response:       "Response with temp",
			expectedResult: "Response with temp",
		},
		{
			name:           "with max_tokens",
			prompt:         "Test prompt",
			config:         map[string]any{"max_tokens": 100},
			response:       "Response with max tokens",
			expectedResult: "Response with max tokens",
		},
		{
			name:           "with temperature and max_tokens",
			prompt:         "Test prompt",
			config:         map[string]any{"temperature": 0.5, "max_tokens": 200},
			response:       "Response with both",
			expectedResult: "Response with both",
		},
		{
			name:           "with invalid temperature type (ignored)",
			prompt:         "Test prompt",
			config:         map[string]any{"temperature": "invalid"},
			response:       "Response",
			expectedResult: "Response",
		},
		{
			name:           "with invalid max_tokens type (ignored)",
			prompt:         "Test prompt",
			config:         map[string]any{"max_tokens": "invalid"},
			response:       "Response",
			expectedResult: "Response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &mockLLM{generateResponse: tt.response}
			adapter := NewOpenAIAdapter(llm)

			ctx := context.Background()
			result, err := adapter.GenerateWithConfig(ctx, tt.prompt, tt.config)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

func TestOpenAIAdapter_GenerateWithSystem(t *testing.T) {
	tests := []struct {
		name           string
		system         string
		prompt         string
		response       *llms.ContentResponse
		err            error
		expectedResult string
		expectedErr    bool
	}{
		{
			name:   "successful generation with system prompt",
			system: "You are a helpful assistant.",
			prompt: "Hello!",
			response: &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{Content: "Hello! How can I assist you today?"},
				},
			},
			expectedResult: "Hello! How can I assist you today?",
			expectedErr:    false,
		},
		{
			name:   "empty system and prompt",
			system: "",
			prompt: "",
			response: &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{Content: "OK"},
				},
			},
			expectedResult: "OK",
			expectedErr:    false,
		},
		{
			name:           "LLM error",
			system:         "You are helpful.",
			prompt:         "Test",
			err:            errors.New("generation error"),
			expectedResult: "",
			expectedErr:    true,
		},
		{
			name:   "empty choices returns empty string",
			system: "You are helpful.",
			prompt: "Test",
			response: &llms.ContentResponse{
				Choices: []*llms.ContentChoice{},
			},
			expectedResult: "",
			expectedErr:    false,
		},
		{
			name:   "nil choices returns empty string",
			system: "You are helpful.",
			prompt: "Test",
			response: &llms.ContentResponse{
				Choices: nil,
			},
			expectedResult: "",
			expectedErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &mockLLM{generateContentResult: tt.response, generateContentError: tt.err}
			adapter := NewOpenAIAdapter(llm)

			ctx := context.Background()
			result, err := adapter.GenerateWithSystem(ctx, tt.system, tt.prompt)

			if tt.expectedErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectedErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

func TestOpenAIAdapter_ContextCancellation(t *testing.T) {
	llm := &mockLLM{generateResponse: "response"}
	adapter := NewOpenAIAdapter(llm)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := adapter.Generate(ctx, "test")
	if err == nil {
		t.Error("expected error due to context cancellation")
	}
}

func TestStreamingLLM_WithStreamCallback(t *testing.T) {
	llm, err := openai.New()
	if err != nil {
		t.Skip("skipping test: LLM creation failed (likely no API key configured)")
	}

	i := 1
	var receivedChunks []string
	streamCallback := func(chunk string) {
		t.Logf("chunk #%d: %s", i, chunk)
		receivedChunks = append(receivedChunks, chunk)
		i++
	}

	streamingLLM := WrapLLMWithStreaming(llms.Wrap(llm), streamCallback)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "Say 'hello' in one word"),
	}

	ctx := context.Background()
	result, err := streamingLLM.GenerateContent(ctx, messages)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(receivedChunks) == 0 {
		t.Error("expected at least one chunk through callback")
	}
	if len(result.Choices) == 0 {
		t.Error("expected at least one choice in result")
	}

	t.Logf("chunks: %s, result: %s", strings.Join(receivedChunks, ""), result.Choices[0].Content)
}

func TestStreamingLLM_WithNilStreamCallback(t *testing.T) {
	llm, err := openai.New()
	if err != nil {
		t.Skip("skipping test: LLM creation failed (likely no API key configured)")
	}

	streamingLLM := WrapLLMWithStreaming(llms.Wrap(llm), nil)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "Say 'hello' in one word"),
	}

	ctx := context.Background()
	result, err := streamingLLM.GenerateContent(ctx, messages)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Choices) == 0 {
		t.Error("expected at least one choice in result")
	}
}
