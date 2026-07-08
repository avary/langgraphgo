package graph_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
)

// TestEmptyGraph tests behavior with empty graphs
func TestEmptyGraph(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		buildGraph  func() *graph.StateGraph[map[string]any]
		expectError bool
		errorMsg    string
	}{
		{
			name: "Graph with no nodes",
			buildGraph: func() *graph.StateGraph[map[string]any] {
				g := graph.NewStateGraph[map[string]any]()
				return g
			},
			expectError: true,
			errorMsg:    "entry point not set",
		},
		{
			name: "Graph with nodes but no entry point",
			buildGraph: func() *graph.StateGraph[map[string]any] {
				g := graph.NewStateGraph[map[string]any]()
				g.AddNode("node1", "node1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					return state, nil
				})
				return g
			},
			expectError: true,
			errorMsg:    "entry point not set",
		},
		{
			name: "Graph with self-referencing node",
			buildGraph: func() *graph.StateGraph[map[string]any] {
				g := graph.NewStateGraph[map[string]any]()
				g.AddNode("node1", "node1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
					return state, nil
				})
				g.AddEdge("node1", "node1") // Self-loop
				g.SetEntryPoint("node1")
				return g
			},
			expectError: false, // Will create infinite loop, but that's valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := tt.buildGraph()
			_, err := g.Compile()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestLargeGraph tests performance with large graphs
func TestLargeGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large graph test in short mode")
	}

	g := graph.NewStateGraph[int]()

	// Create a chain of 1000 nodes
	nodeCount := 1000
	for i := range nodeCount {
		nodeName := fmt.Sprintf("node_%d", i)
		g.AddNode(nodeName, nodeName, func(ctx context.Context, state int) (int, error) {
			return state + 1, nil
		})

		if i > 0 {
			prevNode := fmt.Sprintf("node_%d", i-1)
			g.AddEdge(prevNode, nodeName)
		}
	}

	lastNode := fmt.Sprintf("node_%d", nodeCount-1)
	g.AddEdge(lastNode, graph.END)
	g.SetEntryPoint("node_0")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile large graph: %v", err)
	}

	start := time.Now()
	result, err := runnable.Invoke(context.Background(), 0)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to execute large graph: %v", err)
	}

	if result != nodeCount {
		t.Errorf("Expected result %d, got %v", nodeCount, result)
	}

	t.Logf("Large graph with %d nodes executed in %v", nodeCount, duration)
}

// TestConcurrentExecution tests thread safety
func TestConcurrentExecution(t *testing.T) {
	t.Parallel()

	g := graph.NewStateGraph[map[string]any]()

	var counter int32
	g.AddNode("increment", "increment", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		atomic.AddInt32(&counter, 1)
		return state, nil
	})

	g.AddEdge("increment", graph.END)
	g.SetEntryPoint("increment")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Run multiple executions concurrently
	concurrency := 100
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := range concurrency {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			_, err := runnable.Invoke(ctx, map[string]any{"id": id})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent execution error: %v", err)
	}

	// Verify all executions completed
	finalCount := atomic.LoadInt32(&counter)
	if finalCount != int32(concurrency) {
		t.Errorf("Expected %d executions, got %d", concurrency, finalCount)
	}
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	t.Parallel()

	g := graph.NewStateGraph[map[string]any]()

	// Add a slow node
	g.AddNode("slow_node", "slow_node", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		select {
		case <-time.After(5 * time.Second):
			return state, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	g.AddEdge("slow_node", graph.END)
	g.SetEntryPoint("slow_node")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = runnable.Invoke(ctx, map[string]any{"test": "value"})
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected context cancellation error")
	}

	if duration > 200*time.Millisecond {
		t.Errorf("Cancellation took too long: %v", duration)
	}
}

// TestPanicRecovery tests panic handling in node functions
func TestPanicRecovery(t *testing.T) {
	t.Parallel()

	g := graph.NewStateGraph[map[string]any]()

	g.AddNode("panic_node", "panic_node", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		panic("intentional panic")
	})

	g.AddEdge("panic_node", graph.END)
	g.SetEntryPoint("panic_node")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// This should handle the panic gracefully and convert it to an error
	ctx := context.Background()
	_, err = runnable.Invoke(ctx, map[string]any{"test": "value"})
	if err == nil {
		t.Error("Expected error from panic recovery")
	}
	if err != nil && !strings.Contains(err.Error(), "panic in node panic_node") {
		t.Errorf("Expected panic error, got: %v", err)
	}
}

// TestComplexConditionalRouting tests complex conditional edge scenarios
func TestComplexConditionalRouting(t *testing.T) {
	t.Parallel()

	g := graph.NewStateGraph[map[string]any]()

	// Create a decision tree with multiple levels
	g.AddNode("root", "root", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return state, nil
	})

	g.AddNode("branch_a", "branch_a", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		path := append(state["path"].([]string), "A")
		state["path"] = path
		return state, nil
	})

	g.AddNode("branch_b", "branch_b", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		path := append(state["path"].([]string), "B")
		state["path"] = path
		return state, nil
	})

	g.AddNode("leaf_a1", "leaf_a1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		path := append(state["path"].([]string), "A1")
		state["path"] = path
		return state, nil
	})

	g.AddNode("leaf_a2", "leaf_a2", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		path := append(state["path"].([]string), "A2")
		state["path"] = path
		return state, nil
	})

	g.AddNode("leaf_b1", "leaf_b1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		path := append(state["path"].([]string), "B1")
		state["path"] = path
		return state, nil
	})

	// Root conditional
	g.AddConditionalEdge("root", func(ctx context.Context, state map[string]any) string {
		if state["choice"].(string) == "A" {
			return "branch_a"
		}
		return "branch_b"
	})

	// Branch A conditional
	g.AddConditionalEdge("branch_a", func(ctx context.Context, state map[string]any) string {
		if state["subchoice"].(int) == 1 {
			return "leaf_a1"
		}
		return "leaf_a2"
	})

	// Branch B always goes to B1
	g.AddEdge("branch_b", "leaf_b1")

	// All leaves go to END
	g.AddEdge("leaf_a1", graph.END)
	g.AddEdge("leaf_a2", graph.END)
	g.AddEdge("leaf_b1", graph.END)

	g.SetEntryPoint("root")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	// Test different paths
	tests := []struct {
		input        map[string]any
		expectedPath []string
	}{
		{
			input: map[string]any{
				"choice":    "A",
				"subchoice": 1,
				"path":      []string{},
			},
			expectedPath: []string{"A", "A1"},
		},
		{
			input: map[string]any{
				"choice":    "A",
				"subchoice": 2,
				"path":      []string{},
			},
			expectedPath: []string{"A", "A2"},
		},
		{
			input: map[string]any{
				"choice":    "B",
				"subchoice": 1,
				"path":      []string{},
			},
			expectedPath: []string{"B", "B1"},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("Path_%d", i), func(t *testing.T) {
			result, err := runnable.Invoke(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			path := result["path"].([]string)

			if len(path) != len(tt.expectedPath) {
				t.Errorf("Expected path %v, got %v", tt.expectedPath, path)
			} else {
				for j, p := range path {
					if p != tt.expectedPath[j] {
						t.Errorf("Path mismatch at %d: expected %s, got %s", j, tt.expectedPath[j], p)
					}
				}
			}
		})
	}
}

// TestStateModification tests various state modification scenarios
//
//nolint:gocognit,cyclop // Complex state modification scenarios require extensive testing
func TestStateModification(t *testing.T) {
	t.Parallel()

	t.Run("Accumulator pattern", func(t *testing.T) {
		t.Parallel()

		g := graph.NewStateGraph[[]int]()
		g.AddNode("accumulate", "accumulate", func(ctx context.Context, state []int) ([]int, error) {
			return append(state, len(state)+1), nil
		})
		g.AddConditionalEdge("accumulate", func(ctx context.Context, state []int) string {
			if len(state) >= 5 {
				return graph.END
			}
			return "accumulate"
		})
		g.SetEntryPoint("accumulate")

		runnable, err := g.Compile()
		if err != nil {
			t.Fatalf("Failed to compile: %v", err)
		}

		result, err := runnable.Invoke(context.Background(), []int{})
		if err != nil {
			t.Fatalf("Execution failed: %v", err)
		}

		expected := []int{1, 2, 3, 4, 5}
		if len(result) != len(expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		} else {
			for i := range expected {
				if result[i] != expected[i] {
					t.Errorf("Mismatch at index %d: expected %d, got %d", i, expected[i], result[i])
				}
			}
		}
	})

	t.Run("Map transformation", func(t *testing.T) {
		t.Parallel()

		g := graph.NewStateGraph[map[string]any]()
		g.AddNode("transform", "transform", func(ctx context.Context, state map[string]any) (map[string]any, error) {
			// Transform each value
			for k, v := range state {
				if num, ok := v.(int); ok {
					state[k] = num * 2
				}
			}
			return state, nil
		})
		g.AddEdge("transform", graph.END)
		g.SetEntryPoint("transform")

		runnable, err := g.Compile()
		if err != nil {
			t.Fatalf("Failed to compile: %v", err)
		}

		result, err := runnable.Invoke(context.Background(), map[string]any{
			"a": 1,
			"b": 2,
			"c": 3,
		})
		if err != nil {
			t.Fatalf("Execution failed: %v", err)
		}

		expected := map[string]any{
			"a": 2,
			"b": 4,
			"c": 6,
		}
		for k, v := range expected {
			if result[k] != v {
				t.Errorf("Map mismatch for key %s: expected %v, got %v", k, v, result[k])
			}
		}
	})
}

// TestErrorPropagation tests error handling and propagation
func TestErrorPropagation(t *testing.T) {
	t.Parallel()

	g := graph.NewStateGraph[string]()

	g.AddNode("node1", "node1", func(ctx context.Context, state string) (string, error) {
		return "step1", nil
	})

	g.AddNode("node2", "node2", func(ctx context.Context, state string) (string, error) {
		return "", errors.New("deliberate error in node2")
	})

	g.AddNode("node3", "node3", func(ctx context.Context, state string) (string, error) {
		t.Error("node3 should not be executed after error in node2")
		return state, nil
	})

	g.AddEdge("node1", "node2")
	g.AddEdge("node2", "node3")
	g.AddEdge("node3", graph.END)
	g.SetEntryPoint("node1")

	runnable, err := g.Compile()
	if err != nil {
		t.Fatalf("Failed to compile: %v", err)
	}

	_, err = runnable.Invoke(context.Background(), "start")
	if err == nil {
		t.Error("Expected error from node2")
	} else if !errors.Is(err, errors.New("deliberate error in node2")) {
		// Check error message contains expected text
		if err.Error() != "error in node node2: deliberate error in node2" {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

// TestMessageContentEdgeCases tests edge cases with message content
func TestMessageContentEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		initialState []llmtypes.MessageContent
		transform    func([]llmtypes.MessageContent) []llmtypes.MessageContent
		validate     func(*testing.T, []llmtypes.MessageContent)
	}{
		{
			name:         "Empty message list",
			initialState: []llmtypes.MessageContent{},
			transform: func(msgs []llmtypes.MessageContent) []llmtypes.MessageContent {
				return append(msgs, llmtypes.TextParts("ai", "Hello"))
			},
			validate: func(t *testing.T, msgs []llmtypes.MessageContent) {
				if len(msgs) != 1 {
					t.Errorf("Expected 1 message, got %d", len(msgs))
				}
			},
		},
		{
			name: "Multiple parts in message",
			initialState: []llmtypes.MessageContent{
				{
					Role: "human",
					Parts: []llmtypes.ContentPart{
						llmtypes.TextContent{Text: "Part 1"},
						llmtypes.TextContent{Text: "Part 2"},
					},
				},
			},
			transform: func(msgs []llmtypes.MessageContent) []llmtypes.MessageContent {
				// Count total parts
				totalParts := 0
				for _, msg := range msgs {
					totalParts += len(msg.Parts)
				}
				return append(msgs, llmtypes.TextParts("ai", fmt.Sprintf("You have %d parts", totalParts)))
			},
			validate: func(t *testing.T, msgs []llmtypes.MessageContent) {
				if len(msgs) != 2 {
					t.Errorf("Expected 2 messages, got %d", len(msgs))
				}
				lastMsg := msgs[len(msgs)-1]
				if lastMsg.Parts[0].(llmtypes.TextContent).Text != "You have 2 parts" {
					t.Errorf("Unexpected response: %v", lastMsg.Parts[0])
				}
			},
		},
		{
			name: "Very long message",
			initialState: []llmtypes.MessageContent{
				llmtypes.TextParts("human", string(make([]byte, 10000))), // 10KB message
			},
			transform: func(msgs []llmtypes.MessageContent) []llmtypes.MessageContent {
				// Should handle large messages
				return append(msgs, llmtypes.TextParts("ai", "Handled large message"))
			},
			validate: func(t *testing.T, msgs []llmtypes.MessageContent) {
				if len(msgs) != 2 {
					t.Errorf("Expected 2 messages, got %d", len(msgs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := graph.NewStateGraph[[]llmtypes.MessageContent]()
			g.AddNode("process", "process", func(ctx context.Context, state []llmtypes.MessageContent) ([]llmtypes.MessageContent, error) {
				return tt.transform(state), nil
			})
			g.AddEdge("process", graph.END)
			g.SetEntryPoint("process")

			runnable, err := g.Compile()
			if err != nil {
				t.Fatalf("Failed to compile: %v", err)
			}

			result, err := runnable.Invoke(context.Background(), tt.initialState)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			tt.validate(t, result)
		})
	}
}

// BenchmarkConditionalEdges benchmarks conditional edge performance
func BenchmarkConditionalEdges(b *testing.B) {
	g := graph.NewStateGraph[int]()

	// Create a graph with conditional routing
	g.AddNode("router", "router", func(ctx context.Context, state int) (int, error) {
		return state, nil
	})

	for i := range 10 {
		nodeName := fmt.Sprintf("node_%d", i)
		g.AddNode(nodeName, nodeName, func(ctx context.Context, state int) (int, error) {
			return state + 1, nil
		})
		g.AddEdge(nodeName, graph.END)
	}

	g.AddConditionalEdge("router", func(ctx context.Context, state int) string {
		return fmt.Sprintf("node_%d", state%10)
	})

	g.SetEntryPoint("router")

	runnable, err := g.Compile()
	if err != nil {
		b.Fatalf("Failed to compile: %v", err)
	}

	ctx := context.Background()

	for i := 0; b.Loop(); i++ {
		_, err := runnable.Invoke(ctx, i)
		if err != nil {
			b.Fatalf("Execution failed: %v", err)
		}
	}
}

// BenchmarkLargeStateTransfer benchmarks performance with large state objects
func BenchmarkLargeStateTransfer(b *testing.B) {
	g := graph.NewStateGraph[[]byte]()

	// Create nodes that pass large state
	g.AddNode("node1", "node1", func(ctx context.Context, state []byte) ([]byte, error) {
		return state, nil
	})
	g.AddNode("node2", "node2", func(ctx context.Context, state []byte) ([]byte, error) {
		return state, nil
	})
	g.AddNode("node3", "node3", func(ctx context.Context, state []byte) ([]byte, error) {
		return state, nil
	})

	g.AddEdge("node1", "node2")
	g.AddEdge("node2", "node3")
	g.AddEdge("node3", graph.END)
	g.SetEntryPoint("node1")

	runnable, err := g.Compile()
	if err != nil {
		b.Fatalf("Failed to compile: %v", err)
	}

	// Create large state object (1MB)
	largeState := make([]byte, 1024*1024)
	ctx := context.Background()

	b.SetBytes(int64(len(largeState)))

	for b.Loop() {
		_, err := runnable.Invoke(ctx, largeState)
		if err != nil {
			b.Fatalf("Execution failed: %v", err)
		}
	}
}
