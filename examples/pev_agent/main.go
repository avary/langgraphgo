package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tooltypes"
)

// CalculatorTool for PEV demo
type CalculatorTool struct{}

func (t CalculatorTool) Name() string { return "calculator" }
func (t CalculatorTool) Description() string {
	return "Useful for basic math. Input: 'a op b' (e.g. '2 + 2')"
}
func (t CalculatorTool) Call(ctx context.Context, input string) (string, error) {
	// Simple implementation
	parts := strings.Fields(input)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid format")
	}
	return fmt.Sprintf("Result of %s is simulated as 42", input), nil
}

func main() {
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	model, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	config := prebuilt.PEVAgentConfig{
		Model:      model,
		Tools:      []tooltypes.Tool{CalculatorTool{}},
		MaxRetries: 3,
		Verbose:    true,
	}

	// Use map state convenience function
	agent, err := prebuilt.CreatePEVAgentMap(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	query := "Calculate 15 * 3 and verify if it's correct"
	fmt.Printf("User: %s\n\n", query)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, query),
		},
	}

	res, err := agent.Invoke(ctx, initialState)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nFinal Answer: %v\n", res["final_answer"])
}
