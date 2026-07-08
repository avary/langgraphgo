// Package prebuilt provides prebuilt agent implementations.
// This file contains tests for PiAgent
package prebuilt

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/tooltypes"
)

// MockLLM for testing PiAgent
type mockPiAgentLLM struct {
	responses []*llmtypes.ContentResponse
	callCount int
}

func (m *mockPiAgentLLM) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, opts ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	idx := m.callCount
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	}
	m.callCount++
	return m.responses[idx], nil
}

func (m *mockPiAgentLLM) Call(ctx context.Context, prompt string, opts ...llmtypes.CallOption) (string, error) {
	return "", nil
}

// MockTool for testing
type mockCalculatorTool struct{}

func (t *mockCalculatorTool) Name() string {
	return "calculator"
}

func (t *mockCalculatorTool) Description() string {
	return "Performs basic arithmetic calculations"
}

func (t *mockCalculatorTool) Call(ctx context.Context, input string) (string, error) {
	return "Result: 42", nil
}

func (t *mockCalculatorTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "The mathematical expression to evaluate",
			},
		},
		"required": []string{"expression"},
	}
}

// TestPiAgentMessageDuplication tests that messages are not duplicated
// This was the bug where nodes returned full state instead of just new messages
func TestPiAgentMessageDuplication(t *testing.T) {
	// Create mock LLM that will return one response with a tool call
	mockLLM := &mockPiAgentLLM{
		responses: []*llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						Content: "",
						ToolCalls: []llmtypes.ToolCall{
							{
								ID: "call_123",
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "calculator",
									Arguments: `{"expression": "25+17"}`,
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llmtypes.ContentChoice{
					{
						Content: "The answer is 42",
					},
				},
			},
		},
	}

	// Create agent with mock tool
	agent := &PiAgent{
		state:         NewPiAgentState(),
		model:         mockLLM,
		streamMode:    graph.StreamModeValues,
		maxIterations: 20, // Set max iterations to avoid early termination
	}
	agent.state.SystemPrompt = "You are a helpful assistant."

	// Build the graph
	inputTools := []tooltypes.Tool{&mockCalculatorTool{}}
	runnable, err := buildPiAgentGraph(agent, mockLLM, inputTools)
	if err != nil {
		t.Fatalf("Failed to build agent graph: %v", err)
	}
	agent.runnable = runnable

	// Start with one user message
	agent.state.Messages = []llmtypes.MessageContent{
		llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "What is 25 + 17?"),
	}

	t.Logf("Starting state: %d messages", len(agent.state.Messages))

	ctx := context.Background()
	finalState, err := runnable.Invoke(ctx, agent.state)
	if err != nil {
		t.Fatalf("Agent execution failed: %v", err)
	}

	t.Logf("Final state: %d messages", len(finalState.Messages))

	// Expected message flow:
	// 1. user: What is 25 + 17?
	// 2. ai: (tool call)
	// 3. tool: (result)
	// 4. ai: The answer is 42
	// Total: 4 messages

	expectedCount := 4
	actualCount := len(finalState.Messages)

	if actualCount != expectedCount {
		t.Errorf("Message count mismatch: expected %d, got %d", expectedCount, actualCount)
		t.Log("Actual messages:")
		for i, msg := range finalState.Messages {
			t.Logf("  [%d] Role: %s, Parts: %d", i, msg.Role, len(msg.Parts))
		}
	}

	// Verify message order
	if len(finalState.Messages) >= 4 {
		// Message 0 should be user
		if finalState.Messages[0].Role != llmtypes.ChatMessageTypeHuman {
			t.Errorf("Message 0 role mismatch: expected %s, got %s", llmtypes.ChatMessageTypeHuman, finalState.Messages[0].Role)
		}
		// Message 1 should be AI with tool call
		if finalState.Messages[1].Role != llmtypes.ChatMessageTypeAI {
			t.Errorf("Message 1 role mismatch: expected %s, got %s", llmtypes.ChatMessageTypeAI, finalState.Messages[1].Role)
		}
		// Message 2 should be tool
		if finalState.Messages[2].Role != llmtypes.ChatMessageTypeTool {
			t.Errorf("Message 2 role mismatch: expected %s, got %s", llmtypes.ChatMessageTypeTool, finalState.Messages[2].Role)
		}
		// Message 3 should be AI with final answer
		if finalState.Messages[3].Role != llmtypes.ChatMessageTypeAI {
			t.Errorf("Message 3 role mismatch: expected %s, got %s", llmtypes.ChatMessageTypeAI, finalState.Messages[3].Role)
		}
	}
}
