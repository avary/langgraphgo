package graph_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/tmc/langchaingo/llms/openai"
)

func ExampleStateGraph() {
	// Skip if no OpenAI API key is available
	if os.Getenv("OPENAI_API_KEY") == "" {
		fmt.Println("[{human [{What is 1 + 1?}]} {ai [{1 + 1 equals 2.}]}]")
		return
	}

	rawModel, err := openai.New()
	if err != nil {
		panic(err)
	}
	model := llmtypes.Wrap(rawModel)

	g := graph.NewStateGraph[[]llmtypes.MessageContent]()

	g.AddNode("oracle", "oracle", func(ctx context.Context, state []llmtypes.MessageContent) ([]llmtypes.MessageContent, error) {
		r, err := model.GenerateContent(ctx, state, llmtypes.WithTemperature(0.0))
		if err != nil {
			return nil, err
		}
		return append(state,
			llmtypes.TextParts("ai", r.Choices[0].Content),
		), nil
	})
	g.AddNode(graph.END, graph.END, func(_ context.Context, state []llmtypes.MessageContent) ([]llmtypes.MessageContent, error) {
		return state, nil
	})

	g.AddEdge("oracle", graph.END)
	g.SetEntryPoint("oracle")

	runnable, err := g.Compile()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	// Let's run it!
	res, err := runnable.Invoke(ctx, []llmtypes.MessageContent{
		llmtypes.TextParts("human", "What is 1 + 1?"),
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(res)
}

func TestStateGraph(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name           string
		buildGraph     func() *graph.StateGraph[[]llmtypes.MessageContent]
		inputMessages  []llmtypes.MessageContent
		expectedOutput []llmtypes.MessageContent
		expectedError  error
	}{
		{
			name: "Simple graph",
			buildGraph: func() *graph.StateGraph[[]llmtypes.MessageContent] {
				g := graph.NewStateGraph[[]llmtypes.MessageContent]()
				g.AddNode("node1", "node1", func(_ context.Context, state []llmtypes.MessageContent) ([]llmtypes.MessageContent, error) {
					return append(state, llmtypes.TextParts("ai", "Node 1")), nil
				})
				g.AddNode("node2", "node2", func(_ context.Context, state []llmtypes.MessageContent) ([]llmtypes.MessageContent, error) {
					return append(state, llmtypes.TextParts("ai", "Node 2")), nil
				})
				g.AddEdge("node1", "node2")
				g.AddEdge("node2", graph.END)
				g.SetEntryPoint("node1")
				return g
			},
			inputMessages: []llmtypes.MessageContent{llmtypes.TextParts("human", "Input")},
			expectedOutput: []llmtypes.MessageContent{
				llmtypes.TextParts("human", "Input"),
				llmtypes.TextParts("ai", "Node 1"),
				llmtypes.TextParts("ai", "Node 2"),
			},
			expectedError: nil,
		},
		{
			name: "Entry point not set",
			buildGraph: func() *graph.StateGraph[[]llmtypes.MessageContent] {
				g := graph.NewStateGraph[[]llmtypes.MessageContent]()
				g.AddNode("node1", "node1", func(_ context.Context, state []llmtypes.MessageContent) ([]llmtypes.MessageContent, error) {
					return state, nil
				})
				return g
			},
			expectedError: graph.ErrEntryPointNotSet,
		},
		{
			name: "Node not found",
			buildGraph: func() *graph.StateGraph[[]llmtypes.MessageContent] {
				g := graph.NewStateGraph[[]llmtypes.MessageContent]()
				g.AddNode("node1", "node1", func(_ context.Context, state []llmtypes.MessageContent) ([]llmtypes.MessageContent, error) {
					return state, nil
				})
				g.AddEdge("node1", "node2")
				g.SetEntryPoint("node1")
				return g
			},
			expectedError: fmt.Errorf("%w: node2", graph.ErrNodeNotFound),
		},
		{
			name: "No outgoing edge",
			buildGraph: func() *graph.StateGraph[[]llmtypes.MessageContent] {
				g := graph.NewStateGraph[[]llmtypes.MessageContent]()
				g.AddNode("node1", "node1", func(_ context.Context, state []llmtypes.MessageContent) ([]llmtypes.MessageContent, error) {
					return state, nil
				})
				g.SetEntryPoint("node1")
				return g
			},
			expectedError: fmt.Errorf("%w: node1", graph.ErrNoOutgoingEdge),
		},
		{
			name: "Error in node function",
			buildGraph: func() *graph.StateGraph[[]llmtypes.MessageContent] {
				g := graph.NewStateGraph[[]llmtypes.MessageContent]()
				g.AddNode("node1", "node1", func(_ context.Context, _ []llmtypes.MessageContent) ([]llmtypes.MessageContent, error) {
					return nil, errors.New("node error")
				})
				g.AddEdge("node1", graph.END)
				g.SetEntryPoint("node1")
				return g
			},
			expectedError: errors.New("error in node node1: node error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := tc.buildGraph()
			runnable, err := g.Compile()
			if err != nil {
				if tc.expectedError == nil || !errors.Is(err, tc.expectedError) {
					t.Fatalf("unexpected compile error: %v", err)
				}
				return
			}

			output, err := runnable.Invoke(context.Background(), tc.inputMessages)
			if err != nil {
				if tc.expectedError == nil || err.Error() != tc.expectedError.Error() {
					t.Fatalf("unexpected invoke error: '%v', expected '%v'", err, tc.expectedError)
				}
				return
			}

			if tc.expectedError != nil {
				t.Fatalf("expected error %v, but got nil", tc.expectedError)
			}

			if len(output) != len(tc.expectedOutput) {
				t.Fatalf("expected output length %d, but got %d", len(tc.expectedOutput), len(output))
			}

			for i, msg := range output {
				got := fmt.Sprint(msg)
				expected := fmt.Sprint(tc.expectedOutput[i])
				if got != expected {
					t.Errorf("expected output[%d] content %q, but got %q", i, expected, got)
				}
			}
		})
	}
}
