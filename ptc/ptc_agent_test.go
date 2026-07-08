package ptc_test

import (
	"context"
	"strings"
	"testing"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/ptc"
	"github.com/tmc/langchaingo/tools"
)

// MockLLM for testing
type MockLLM struct {
	response  string
	callCount int
}

func (m *MockLLM) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	m.callCount++
	return &llmtypes.ContentResponse{
		Choices: []*llmtypes.ContentChoice{
			{
				Content: m.response,
			},
		},
	}, nil
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llmtypes.CallOption) (string, error) {
	m.callCount++
	return m.response, nil
}

// TestPTCToolNode tests PTCToolNode functionality
func TestPTCToolNode(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "calculator",
			description: "Performs calculations",
			response:    "42",
		},
	}

	node := ptc.NewPTCToolNode(ptc.LanguagePython, tools)
	ctx := context.Background()

	// Start the tool server
	if err := node.Executor.Start(ctx); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	defer node.Close(ctx)

	// Create state with AI message containing code
	state := map[string]any{
		"messages": []llmtypes.MessageContent{
			{
				Role: llmtypes.ChatMessageTypeAI,
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("```python\nresult = calculator('2+2')\nprint(result)\n```"),
				},
			},
		},
	}

	// Invoke the node
	newState, err := node.Invoke(ctx, state)
	if err != nil {
		t.Fatalf("Failed to invoke node: %v", err)
	}

	// Check that a new message was added
	messages := newState.(map[string]any)["messages"].([]llmtypes.MessageContent)
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	// Check that the last message contains execution result
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != llmtypes.ChatMessageTypeHuman {
		t.Errorf("Expected last message to be Human, got %s", lastMsg.Role)
	}
}

// TestPTCToolNodeWithGoCode tests PTCToolNode with Go code
func TestPTCToolNodeWithGoCode(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "greet",
			description: "Greets someone",
			response:    "Hello!",
		},
	}

	node := ptc.NewPTCToolNode(ptc.LanguageGo, tools)
	ctx := context.Background()

	if err := node.Executor.Start(ctx); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	defer node.Close(ctx)

	state := map[string]any{
		"messages": []llmtypes.MessageContent{
			{
				Role: llmtypes.ChatMessageTypeAI,
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("```go\nresult, _ := greet(ctx, \"World\")\nfmt.Println(result)\n```"),
				},
			},
		},
	}

	newState, err := node.Invoke(ctx, state)
	if err != nil {
		t.Fatalf("Failed to invoke node: %v", err)
	}

	messages := newState.(map[string]any)["messages"].([]llmtypes.MessageContent)
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

// TestPTCToolNodeErrorHandling tests error handling in PTCToolNode
func TestPTCToolNodeErrorHandling(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "test",
			description: "Test tool",
			response:    "ok",
		},
	}

	node := ptc.NewPTCToolNode(ptc.LanguagePython, tools)
	ctx := context.Background()

	if err := node.Executor.Start(ctx); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	defer node.Close(ctx)

	// State with code that has syntax error
	state := map[string]any{
		"messages": []llmtypes.MessageContent{
			{
				Role: llmtypes.ChatMessageTypeAI,
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("```python\nprint('unclosed string\n```"),
				},
			},
		},
	}

	newState, err := node.Invoke(ctx, state)
	// Should not return error, but should add error message
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	messages := newState.(map[string]any)["messages"].([]llmtypes.MessageContent)
	lastMsg := messages[len(messages)-1]
	lastText := lastMsg.Parts[0].(llmtypes.TextContent).Text

	if !strings.Contains(lastText, "Error") && !strings.Contains(lastText, "error") {
		t.Error("Expected error message in output")
	}
}

// TestPTCToolNodeWithoutCode tests PTCToolNode with plain text (no code blocks)
func TestPTCToolNodeWithoutCode(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "test",
			description: "Test",
			response:    "ok",
		},
	}

	node := ptc.NewPTCToolNode(ptc.LanguagePython, tools)
	ctx := context.Background()

	if err := node.Executor.Start(ctx); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	defer node.Close(ctx)

	// State with plain text (will be treated as code and may execute or error)
	state := map[string]any{
		"messages": []llmtypes.MessageContent{
			{
				Role: llmtypes.ChatMessageTypeAI,
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("Just some text without code blocks"),
				},
			},
		},
	}

	// The node should process this (may succeed or fail depending on execution)
	// We just verify it doesn't panic
	_, err := node.Invoke(ctx, state)
	// Error is acceptable here as plain text may not be valid Python
	_ = err
}

// TestPTCAgentConfig tests PTCAgentConfig validation
func TestPTCAgentConfig(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "test",
			description: "Test",
			response:    "ok",
		},
	}

	// Test without model
	_, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
		Tools: tools,
	})
	if err == nil {
		t.Error("Expected error when model is not provided")
	}

	// Test without tools
	_, err = ptc.CreatePTCAgent(ptc.PTCAgentConfig{
		Model: &MockLLM{response: "test"},
	})
	if err == nil {
		t.Error("Expected error when tools are not provided")
	}
}

// TestPTCAgentDefaultConfig tests default configuration
func TestPTCAgentDefaultConfig(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "calculator",
			description: "Calculates",
			response:    "42",
		},
	}

	agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
		Model: &MockLLM{response: "```python\nprint('test')\n```"},
		Tools: tools,
	})

	if err != nil {
		t.Fatalf("Failed to create agent with defaults: %v", err)
	}

	// Agent should be created successfully
	if agent == nil {
		t.Error("Expected non-nil agent")
	}
}

// TestPTCAgentWithCustomConfig tests custom configuration
func TestPTCAgentWithCustomConfig(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "test",
			description: "Test",
			response:    "ok",
		},
	}

	agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
		Model:         &MockLLM{response: "```go\nfmt.Println(\"test\")\n```"},
		Tools:         tools,
		Language:      ptc.LanguageGo,
		ExecutionMode: ptc.ModeServer,
		SystemPrompt:  "You are a helpful assistant",
		MaxIterations: 5,
	})

	if err != nil {
		t.Fatalf("Failed to create agent with custom config: %v", err)
	}

	if agent == nil {
		t.Error("Expected non-nil agent")
	}
}

// TestSanitizeFunctionName tests function name sanitization
// This is an indirect test through tool definitions
func TestSanitizeFunctionName(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "tool-with-dashes",
			description: "Test tool with dashes",
			response:    "ok",
		},
		MockTool{
			name:        "tool.with.dots",
			description: "Test tool with dots",
			response:    "ok",
		},
		MockTool{
			name:        "tool with spaces",
			description: "Test tool with spaces",
			response:    "ok",
		},
	}

	executor := ptc.NewCodeExecutor(ptc.LanguagePython, tools)
	ctx := context.Background()

	if err := executor.Start(ctx); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	defer executor.Stop(ctx)

	defs := executor.GetToolDefinitions()

	// Check that sanitized names are present
	if !strings.Contains(defs, "tool_with_dashes") {
		t.Error("Expected sanitized function name 'tool_with_dashes'")
	}

	if !strings.Contains(defs, "tool_with_dots") {
		t.Error("Expected sanitized function name 'tool_with_dots'")
	}

	if !strings.Contains(defs, "tool_with_spaces") {
		t.Error("Expected sanitized function name 'tool_with_spaces'")
	}
}
