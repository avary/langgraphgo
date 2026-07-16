package graph_test

import (
	"context"
	"strings"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
)

//nolint:gocognit,dupl,cyclop // This is a comprehensive test that needs to check multiple scenarios with similar setup
func TestConditionalEdges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		buildGraph     func() *graph.StateGraph[map[string]any]
		initialState   any
		expectedResult any
		expectError    bool
	}{
		{
			name: "Simple conditional routing based on content",
			buildGraph: func() *graph.StateGraph[map[string]any] {
				g := graph.NewStateGraph[map[string]any]()

				// Add nodes
				g.AddNode("start", "start", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					return state, nil
				})

				g.AddNode("calculator", "calculator", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					messages := state["messages"].([]llms.MessageContent)
					state["messages"] = append(messages, llms.TextParts("ai", "Calculating: 2+2=4"))
					return state, nil
				})

				g.AddNode("general", "general", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					messages := state["messages"].([]llms.MessageContent)
					state["messages"] = append(messages, llms.TextParts("ai", "General response"))
					return state, nil
				})

				// Add conditional edge from start
				g.AddConditionalEdge("start", func(ctx context.Context, state map[string]any) string {
					messages := state["messages"].([]llms.MessageContent)
					if len(messages) > 0 {
						lastMessage := messages[len(messages)-1]
						if content, ok := lastMessage.Parts[0].(llms.TextContent); ok {
							if strings.Contains(content.Text, "calculate") || strings.Contains(content.Text, "math") {
								return "calculator"
							}
						}
					}
					return "general"
				})

				// Add regular edges to END
				g.AddEdge("calculator", graph.END)
				g.AddEdge("general", graph.END)

				g.SetEntryPoint("start")
				return g
			},
			initialState: map[string]any{"messages": []llms.MessageContent{
				llms.TextParts("human", "I need to calculate something"),
			}},
			expectedResult: map[string]any{"messages": []llms.MessageContent{
				llms.TextParts("human", "I need to calculate something"),
				llms.TextParts("ai", "Calculating: 2+2=4"),
			}},
			expectError: false,
		},
		{
			name: "Conditional routing to general path",
			buildGraph: func() *graph.StateGraph[map[string]any] {
				g := graph.NewStateGraph[map[string]any]()

				g.AddNode("start", "start", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					return state, nil
				})

				g.AddNode("calculator", "calculator", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					messages := state["messages"].([]llms.MessageContent)
					state["messages"] = append(messages, llms.TextParts("ai", "Calculating: 2+2=4"))
					return state, nil
				})

				g.AddNode("general", "general", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					messages := state["messages"].([]llms.MessageContent)
					state["messages"] = append(messages, llms.TextParts("ai", "General response"))
					return state, nil
				})

				g.AddConditionalEdge("start", func(ctx context.Context, state map[string]any) string {
					messages := state["messages"].([]llms.MessageContent)
					if len(messages) > 0 {
						lastMessage := messages[len(messages)-1]
						if content, ok := lastMessage.Parts[0].(llms.TextContent); ok {
							if strings.Contains(content.Text, "calculate") || strings.Contains(content.Text, "math") {
								return "calculator"
							}
						}
					}
					return "general"
				})

				g.AddEdge("calculator", graph.END)
				g.AddEdge("general", graph.END)

				g.SetEntryPoint("start")
				return g
			},
			initialState: map[string]any{"messages": []llms.MessageContent{
				llms.TextParts("human", "Tell me a story"),
			}},
			expectedResult: map[string]any{"messages": []llms.MessageContent{
				llms.TextParts("human", "Tell me a story"),
				llms.TextParts("ai", "General response"),
			}},
			expectError: false,
		},
		{
			name: "Multi-level conditional routing",
			buildGraph: func() *graph.StateGraph[map[string]any] {
				g := graph.NewStateGraph[map[string]any]()

				g.AddNode("router", "router", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					return state, nil
				})

				g.AddNode("urgent", "urgent", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					s := state["message"].(string)
					state["message"] = s + " -> handled urgently"
					return state, nil
				})

				g.AddNode("normal", "normal", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					s := state["message"].(string)
					state["message"] = s + " -> handled normally"
					return state, nil
				})

				g.AddNode("low", "low", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					s := state["message"].(string)
					state["message"] = s + " -> handled with low priority"
					return state, nil
				})

				// Conditional routing based on priority keywords
				g.AddConditionalEdge("router", func(ctx context.Context, state map[string]any) string {
					s := state["message"].(string)
					if strings.Contains(s, "URGENT") || strings.Contains(s, "ASAP") {
						return "urgent"
					}
					if strings.Contains(s, "NORMAL") || strings.Contains(s, "REGULAR") {
						return "normal"
					}
					return "low"
				})

				g.AddEdge("urgent", graph.END)
				g.AddEdge("normal", graph.END)
				g.AddEdge("low", graph.END)

				g.SetEntryPoint("router")
				return g
			},
			initialState:   map[string]any{"message": "URGENT: Fix the bug"},
			expectedResult: map[string]any{"message": "URGENT: Fix the bug -> handled urgently"},
			expectError:    false,
		},
		{
			name: "Conditional edge to END",
			buildGraph: func() *graph.StateGraph[map[string]any] {
				g := graph.NewStateGraph[map[string]any]()

				g.AddNode("check", "check", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					return state, nil
				})

				g.AddNode("process", "process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					n := state["value"].(int)
					state["value"] = n * 2
					return state, nil
				})

				// Conditional edge that can go directly to END
				g.AddConditionalEdge("check", func(ctx context.Context, state map[string]any) string {
					n := state["value"].(int)
					if n < 0 {
						return graph.END
					}
					return "process"
				})

				g.AddEdge("process", graph.END)

				g.SetEntryPoint("check")
				return g
			},
			initialState:   map[string]any{"value": -5},
			expectedResult: map[string]any{"value": -5}, // Should go directly to END without processing
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := tt.buildGraph()
			runnable, err := g.Compile()
			if err != nil {
				t.Fatalf("Failed to compile graph: %v", err)
			}

			ctx := context.Background()
			result, err := runnable.Invoke(ctx, tt.initialState.(map[string]any))

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				// Check if the result has "messages" field for message-based tests
				if _, hasMessages := result["messages"]; hasMessages {
					resultMessages := result["messages"].([]llms.MessageContent)
					expectedMessages := tt.expectedResult.(map[string]any)["messages"].([]llms.MessageContent)
					if len(resultMessages) != len(expectedMessages) {
						t.Errorf("Expected %d messages, got %d", len(expectedMessages), len(resultMessages))
					} else {
						for i := range resultMessages {
							if resultMessages[i].Role != expectedMessages[i].Role {
								t.Errorf("Message %d: expected role %s, got %s", i, expectedMessages[i].Role, resultMessages[i].Role)
							}
							expectedText := expectedMessages[i].Parts[0].(llms.TextContent).Text
							actualText := resultMessages[i].Parts[0].(llms.TextContent).Text
							if actualText != expectedText {
								t.Errorf("Message %d: expected text %q, got %q", i, expectedText, actualText)
							}
						}
					}
				} else {
					// For non-message based tests, just compare the entire result
					expected := tt.expectedResult.(map[string]any)
					for k, expectedVal := range expected {
						if result[k] != expectedVal {
							t.Errorf("Expected %v for key %s, got %v", expectedVal, k, result[k])
						}
					}
				}
			}
		})
	}
}

func TestConditionalEdges_ChainedConditions(t *testing.T) {
	t.Parallel()

	g := graph.NewStateGraph[map[string]any]()

	// Create a chain of conditional decisions
	g.AddNode("start", "start", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return state, nil
	})

	g.AddNode("step1", "step1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		n := state["value"].(int)
		state["value"] = n + 10
		return state, nil
	})

	g.AddNode("step2", "step2", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		n := state["value"].(int)
		state["value"] = n * 2
		return state, nil
	})

	g.AddNode("step3", "step3", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		n := state["value"].(int)
		state["value"] = n - 5
		return state, nil
	})

	// First conditional
	g.AddConditionalEdge("start", func(ctx context.Context, state map[string]any) string {
		n := state["value"].(int)
		if n > 0 {
			return "step1"
		}
		return "step2"
	})

	// Second conditional
	g.AddConditionalEdge("step1", func(ctx context.Context, state map[string]any) string {
		n := state["value"].(int)
		if n > 15 {
			return "step3"
		}
		return graph.END
	})

	g.AddEdge("step2", graph.END)
	g.AddEdge("step3", graph.END)
	g.SetEntryPoint("start")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile graph: %v", err)
	}

	// Test with positive number (should go: start -> step1 -> step3 -> END)
	ctx := context.Background()
	result, err := runnable.Invoke(ctx, map[string]any{"value": 10})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 10 + 10 = 20 (step1), then 20 > 15 so go to step3, 20 - 5 = 15
	if result["value"] != 15 {
		t.Errorf("Expected result 15, got %v", result)
	}

	// Test with negative number (should go: start -> step2 -> END)
	result, err = runnable.Invoke(ctx, map[string]any{"value": -5})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// -5 * 2 = -10 (step2)
	if result["value"] != -10 {
		t.Errorf("Expected result -10, got %v", result)
	}
}
