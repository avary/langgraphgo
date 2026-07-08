package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tooltypes"
)

// CalculatorTool is a simple tool for demonstration
type CalculatorTool struct{}

func (t CalculatorTool) Name() string {
	return "calculator"
}

func (t CalculatorTool) Description() string {
	return "Useful for performing basic arithmetic operations. Input should be a string like '2 + 2' or '5 * 10'."
}

func (t CalculatorTool) Call(ctx context.Context, input string) (string, error) {
	// Very simple parser for demo purposes
	parts := strings.Fields(input)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid input format, expected 'a op b'")
	}

	a, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return "", err
	}
	b, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return "", err
	}

	op := parts[1]
	var result float64

	switch op {
	case "+":
		result = a + b
	case "-":
		result = a - b
	case "*":
		result = a * b
	case "/":
		if b == 0 {
			return "", fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return "", fmt.Errorf("unknown operator: %s", op)
	}

	return fmt.Sprintf("%f", result), nil
}

func main() {
	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Define Tools
	inputTools := []tooltypes.Tool{
		CalculatorTool{},
	}

	// Create ReAct Agent using map state convenience function
	agent, err := prebuilt.CreateAgentMap(llm, inputTools, 20)
	if err != nil {
		log.Fatal(err)
	}

	// Execute
	query := "What is 25 * 4?"
	fmt.Printf("User: %s\n", query)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, query),
		},
	}

	res, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}

	// Print Result
	messages := res["messages"].([]llmtypes.MessageContent)
	lastMsg := messages[len(messages)-1]

	if len(lastMsg.Parts) > 0 {
		if textPart, ok := lastMsg.Parts[0].(llmtypes.TextContent); ok {
			fmt.Printf("Agent: %s\n", textPart.Text)
		}
	}
}
