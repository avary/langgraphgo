package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
)

func main() {
	// Initialize LLM
	model, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Create state graph
	g := graph.NewStateGraph[map[string]any]()

	// Add arith node that calls LLM to calculate expression
	g.AddNode("arith", "arith", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		expression, ok := state["expression"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid expression")
		}

		// Call LLM to calculate
		prompt := fmt.Sprintf("Calculate: %s. Only return the number.", expression)
		result, err := model.Call(ctx, prompt)
		if err != nil {
			return nil, err
		}

		// Update state with result
		state["result"] = result
		return state, nil
	})

	g.AddEdge("arith", graph.END)
	g.SetEntryPoint("arith")

	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Invoke with expression in state
	res, err := runnable.Invoke(context.Background(), map[string]any{
		"expression": "123 + 456",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("123 + 456 = %s\n", res["result"])
}
