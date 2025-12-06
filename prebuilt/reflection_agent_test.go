package prebuilt

import (
	"context"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// MockLLM is a mock LLM for testing
type MockReflectionLLM struct {
	responses []string
	callCount int
}

func (m *MockReflectionLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	response := m.responses[m.callCount%len(m.responses)]
	m.callCount++

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: response,
			},
		},
	}, nil
}

func (m *MockReflectionLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

func TestCreateReflectionAgent(t *testing.T) {
	mockLLM := &MockReflectionLLM{
		responses: []string{
			"This is my initial response to the question.",
			"**Strengths:** The response is clear.\n**Weaknesses:** Could be more detailed.\n**Suggestions for improvement:** Add more examples.",
			"This is my improved response with more details and examples.",
			"**Strengths:** Excellent improvement. Comprehensive and detailed.\n**Weaknesses:** No major issues.\n**Suggestions for improvement:** No improvements needed.",
		},
	}

	config := ReflectionAgentConfig{
		Model:         mockLLM,
		MaxIterations: 3,
		Verbose:       false,
	}

	agent, err := CreateReflectionAgent(config)
	if err != nil {
		t.Fatalf("Failed to create reflection agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent is nil")
	}

	// Test invocation
	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart("Explain quantum computing")},
			},
		},
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		t.Fatalf("Failed to invoke agent: %v", err)
	}

	finalState := result.(map[string]interface{})

	// Verify final state has messages
	messages, ok := finalState["messages"].([]llms.MessageContent)
	if !ok {
		t.Fatal("Final state does not contain messages")
	}

	if len(messages) == 0 {
		t.Fatal("No messages in final state")
	}

	// Verify we have a draft
	_, ok = finalState["draft"].(string)
	if !ok {
		t.Fatal("Final state does not contain draft")
	}

	// Verify iterations occurred
	iteration, ok := finalState["iteration"].(int)
	if !ok {
		t.Fatal("Final state does not contain iteration count")
	}

	if iteration < 1 {
		t.Fatal("Expected at least one iteration")
	}
}

func TestReflectionAgentWithoutModel(t *testing.T) {
	config := ReflectionAgentConfig{
		Model: nil,
	}

	_, err := CreateReflectionAgent(config)
	if err == nil {
		t.Fatal("Expected error when creating agent without model")
	}
}

func TestReflectionAgentMaxIterations(t *testing.T) {
	mockLLM := &MockReflectionLLM{
		responses: []string{
			"Initial response.",
			"**Weaknesses:** Needs improvement.",
			"Improved response.",
			"**Weaknesses:** Still needs more work.",
			"Final improved response.",
		},
	}

	config := ReflectionAgentConfig{
		Model:         mockLLM,
		MaxIterations: 2,
		Verbose:       false,
	}

	agent, err := CreateReflectionAgent(config)
	if err != nil {
		t.Fatalf("Failed to create reflection agent: %v", err)
	}

	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart("Test question")},
			},
		},
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		t.Fatalf("Failed to invoke agent: %v", err)
	}

	finalState := result.(map[string]interface{})
	iteration := finalState["iteration"].(int)

	// Should stop at max iterations
	if iteration > 2 {
		t.Fatalf("Expected max 2 iterations, got %d", iteration)
	}
}

func TestIsResponseSatisfactory(t *testing.T) {
	tests := []struct {
		name       string
		reflection string
		want       bool
	}{
		{
			name:       "Excellent response",
			reflection: "**Strengths:** Excellent response. **Weaknesses:** No major issues. **Suggestions:** No improvements needed.",
			want:       true,
		},
		{
			name:       "Has issues",
			reflection: "**Strengths:** Good start. **Weaknesses:** Missing key details. **Suggestions:** Should include more examples.",
			want:       false,
		},
		{
			name:       "Satisfactory",
			reflection: "The response is satisfactory and meets all requirements.",
			want:       true,
		},
		{
			name:       "Needs improvement",
			reflection: "The response is incomplete and lacks important information.",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isResponseSatisfactory(tt.reflection)
			if got != tt.want {
				t.Errorf("isResponseSatisfactory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOriginalRequest(t *testing.T) {
	messages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextPart("What is AI?")},
		},
		{
			Role:  llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{llms.TextPart("AI is...")},
		},
	}

	request := getOriginalRequest(messages)
	if request != "What is AI?" {
		t.Errorf("Expected 'What is AI?', got '%s'", request)
	}
}

func TestDefaultReflectionPrompt(t *testing.T) {
	prompt := buildDefaultReflectionPrompt()

	if prompt == "" {
		t.Error("Default reflection prompt should not be empty")
	}

	// Check for key components
	requiredPhrases := []string{
		"Strengths",
		"Weaknesses",
		"Suggestions",
		"Accuracy",
		"Completeness",
	}

	for _, phrase := range requiredPhrases {
		if !contains(prompt, phrase) {
			t.Errorf("Default prompt missing required phrase: %s", phrase)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
