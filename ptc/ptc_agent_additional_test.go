package ptc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/tooltypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTool for testing
type MockTool struct {
	name        string
	description string
	// If returnError is true, the tool will return an error
	returnError bool
}

func (t *MockTool) Name() string {
	return t.name
}

func (t *MockTool) Description() string {
	return t.description
}

func (t *MockTool) Call(ctx context.Context, input string) (string, error) {
	if t.returnError {
		return "", fmt.Errorf("tool execution failed")
	}
	return fmt.Sprintf("Result for %s", input), nil
}

// MockLLM for testing
type MockLLM struct {
	response string
}

func (m *MockLLM) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	return &llmtypes.ContentResponse{
		Choices: []*llmtypes.ContentChoice{
			{
				Content: m.response,
			},
		},
	}, nil
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llmtypes.CallOption) (string, error) {
	return m.response, nil
}

func TestContainsCode(t *testing.T) {
	tests := []struct {
		name     string
		message  llmtypes.MessageContent
		expected bool
	}{
		{
			name: "Python code block",
			message: llmtypes.MessageContent{
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("Here is some python code:\n```python\nprint('Hello, world!')\n```"),
				},
			},
			expected: true,
		},
		{
			name: "Go code block",
			message: llmtypes.MessageContent{
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("Here is some go code:\n```go\nfmt.Println(\"Hello, world!\")\n```"),
				},
			},
			expected: true,
		},
		{
			name: "No code block",
			message: llmtypes.MessageContent{
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("This is a regular message."),
				},
			},
			expected: false,
		},
		{
			name: "Partial code block",
			message: llmtypes.MessageContent{
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("This is a partial code block: ```py"),
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ContainsCode(tt.message))
		})
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	executor := &CodeExecutor{
		Tools: []tooltypes.Tool{
			&MockTool{name: "test_tool", description: "A test tool"},
		},
	}

	tests := []struct {
		name       string
		userPrompt string
		language   ExecutionLanguage
		expected   string
	}{
		{
			name:       "Python with user prompt",
			userPrompt: "You are a helpful assistant.",
			language:   LanguagePython,
			expected:   "```python",
		},
		{
			name:       "Go with no user prompt",
			userPrompt: "",
			language:   LanguageGo,
			expected:   "```go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := BuildSystemPrompt(tt.userPrompt, tt.language, executor)
			if !strings.Contains(prompt, tt.expected) {
				t.Errorf("Expected %s code block marker for language %s", tt.expected, tt.language)
			}
		})
	}
}

func TestAgentNodeMaxIterations(t *testing.T) {
	mockLLM := &MockLLM{response: "This is a response."}
	maxIterations := 3

	initialState := map[string]any{
		"messages":        []llmtypes.MessageContent{},
		"iteration_count": maxIterations,
	}

	_, err := agentNode(context.Background(), initialState, mockLLM, "system prompt", maxIterations)
	require.NoError(t, err)

	finalState := initialState
	messages := finalState["messages"].([]llmtypes.MessageContent)
	lastMessage := messages[len(messages)-1]
	lastPart := lastMessage.Parts[0].(llmtypes.TextContent)
	assert.Contains(t, lastPart.Text, "Maximum iterations reached")
}

func TestCreatePTCAgent(t *testing.T) {
	mockLLM := &MockLLM{response: "This is a response."}
	tool := &MockTool{name: "test_tool"}

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "mock server response"}`))
	}))
	defer server.Close()

	config := PTCAgentConfig{
		Model:         mockLLM,
		Tools:         []tooltypes.Tool{tool},
		Language:      LanguagePython,
		ExecutionMode: ModeServer, // Use server mode for testing
	}

	// The executor in CreatePTCAgent starts a server, which we can't easily do in a test.
	// So we can't fully test CreatePTCAgent here, but we can check the config validation.

	// Test without model
	config.Model = nil
	_, err := CreatePTCAgent(config)
	assert.Error(t, err)
	config.Model = mockLLM

	// Test without tools
	config.Tools = []tooltypes.Tool{}
	_, err = CreatePTCAgent(config)
	assert.Error(t, err)
	config.Tools = []tooltypes.Tool{tool}
}
