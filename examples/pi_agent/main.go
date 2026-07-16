// PiAgent example demonstrates the usage of PiAgent
// Inspired by pi-mono/packages/agent
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tool"
)

func main() {
	ctx := context.Background()

	// Create OpenAI LLM
	model, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Create a simple tool
	calculator := &CalculatorTool{}

	// Create PiAgent
	agent, err := prebuilt.NewPiAgent(
		model,
		[]tool.Tool{calculator},
		prebuilt.WithPiSystemPrompt("You are a helpful assistant that can perform calculations."),
		prebuilt.WithPiMaxIterations(10),
	)
	if err != nil {
		fmt.Printf("Failed to create agent: %v\n", err)
		os.Exit(1)
	}

	// Subscribe to agent events
	unsubscribe := agent.Subscribe(func(event prebuilt.PiAgentEvent) {
		switch event.Type {
		case prebuilt.EventAgentStart:
			fmt.Println("=== Agent Started ===")
		case prebuilt.EventAgentEnd:
			fmt.Println("=== Agent Ended ===")
		case prebuilt.EventTurnStart:
			fmt.Println("--- Turn Started ---")
		case prebuilt.EventTurnEnd:
			fmt.Println("--- Turn Ended ---")
		case prebuilt.EventToolExecutionStart:
			fmt.Printf("Tool Started: %s\n", event.ToolName)
		case prebuilt.EventToolExecutionEnd:
			if event.ToolError {
				fmt.Printf("Tool Error: %s - %v\n", event.ToolName, event.ToolResult)
			} else {
				fmt.Printf("Tool Completed: %s\n", event.ToolName)
			}
		}
	})
	defer unsubscribe()

	// Example: Simple calculation
	fmt.Println("\n=== Example: Simple Calculation ===")
	msg := llms.TextParts(llms.ChatMessageTypeHuman, "What is 25 + 17?")

	err = agent.Prompt(ctx, msg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Example: Multiple calculations
	fmt.Println("\n=== Example: Multiple Calculations ===")
	msg2 := llms.TextParts(llms.ChatMessageTypeHuman, "What is 100 * 42 and what is 144 divided by 12?")

	err = agent.Prompt(ctx, msg2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Check final state
	state := agent.GetState()
	fmt.Printf("\n=== Final State ===\n")
	fmt.Printf("Total messages: %d\n", len(state.Messages))
	fmt.Printf("Tools: %d\n", len(state.Tools))
	fmt.Printf("Thinking Level: %s\n", state.ThinkingLevel)

	fmt.Println("\n=== Done ===")
}

// CalculatorTool is a simple example tool that performs basic calculations
type CalculatorTool struct{}

func (t *CalculatorTool) Name() string {
	return "calculator"
}

func (t *CalculatorTool) Description() string {
	return "Performs basic arithmetic calculations. Use this for addition, subtraction, multiplication, and division."
}

func (t *CalculatorTool) Call(ctx context.Context, input string) (string, error) {
	// Simplified implementation - in production, parse and evaluate the expression
	result := "Calculation result: " + input
	return result, nil
}

// Schema returns the parameter schema for the calculator tool
func (t *CalculatorTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "The mathematical expression to evaluate (e.g., '25 + 17', '100 * 42')",
			},
		},
		"required":             []string{"expression"},
		"additionalProperties": false,
	}
}
