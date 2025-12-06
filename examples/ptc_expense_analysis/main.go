package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smallnest/langgraphgo/ptc"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

func main() {
	fmt.Println("=== PTC (Programmatic Tool Calling) Example ===")
	fmt.Println("This example demonstrates how PTC reduces latency and token usage")
	fmt.Println("by allowing the LLM to write code that calls tools programmatically.\n")

	// Initialize OpenAI model
	model, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create OpenAI client: %v", err)
	}

	// Create expense tools
	tools := []tools.Tool{
		GetTeamMembersTool{},
		GetExpensesTool{},
		GetCustomBudgetTool{},
	}

	// Create PTC agent
	fmt.Println("Creating PTC Agent...")
	agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
		Model:    model,
		Tools:    tools,
		Language: ptc.LanguagePython,
		SystemPrompt: `You are a helpful financial analysis assistant.
You can write Python code to analyze expense data efficiently.`,
		MaxIterations: 10,
	})
	if err != nil {
		log.Fatalf("Failed to create PTC agent: %v", err)
	}

	// Run example queries
	queries := []string{
		`Which engineering team members exceeded their Q3 travel budget?
The standard quarterly travel budget is $5,000.
However, some employees have custom budget limits.
For anyone who exceeded the $5,000 standard budget, check if they have a custom budget exception.
If they do, use that custom limit instead to determine if they truly exceeded their budget.
Only count approved expenses.`,
	}

	for i, query := range queries {
		fmt.Printf("\n=== Query %d ===\n", i+1)
		fmt.Printf("Question: %s\n\n", query)

		startTime := time.Now()

		// Create initial state
		initialState := map[string]interface{}{
			"messages": []llms.MessageContent{
				{
					Role: llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{
						llms.TextPart(query),
					},
				},
			},
		}

		// Invoke the agent
		result, err := agent.Invoke(context.Background(), initialState)
		if err != nil {
			log.Printf("Error running agent: %v", err)
			continue
		}

		elapsed := time.Since(startTime)

		// Extract final answer
		resultState := result.(map[string]interface{})
		messages := resultState["messages"].([]llms.MessageContent)

		fmt.Println("\n--- Conversation Flow ---")
		for idx, msg := range messages {
			role := "Unknown"
			switch msg.Role {
			case llms.ChatMessageTypeHuman:
				role = "Human"
			case llms.ChatMessageTypeAI:
				role = "AI"
			case llms.ChatMessageTypeTool:
				role = "Tool Result"
			case llms.ChatMessageTypeSystem:
				role = "System"
			}

			fmt.Printf("\n[%d] %s:\n", idx+1, role)
			for _, part := range msg.Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					text := textPart.Text
					if len(text) > 500 {
						fmt.Printf("%s... (truncated)\n", text[:500])
					} else {
						fmt.Println(text)
					}
				}
			}
		}

		fmt.Printf("\n--- Execution Stats ---\n")
		fmt.Printf("Total time: %v\n", elapsed)
		fmt.Printf("Messages exchanged: %d\n", len(messages))

		// Get last AI message as final answer
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == llms.ChatMessageTypeAI {
				fmt.Printf("\n--- Final Answer ---\n")
				for _, part := range messages[i].Parts {
					if textPart, ok := part.(llms.TextContent); ok {
						fmt.Println(textPart.Text)
					}
				}
				break
			}
		}
	}

	fmt.Println("\n=== Comparison: PTC vs Traditional Tool Calling ===")
	fmt.Println("PTC Advantages:")
	fmt.Println("1. Reduced Latency: Eliminates multiple API round-trips for sequential tool calls")
	fmt.Println("2. Token Efficiency: Code can filter large datasets before sending results back")
	fmt.Println("3. Programmatic Control: Write code to process data with loops, conditionals, etc.")
	fmt.Println("\nTraditional Tool Calling Issues:")
	fmt.Println("1. Each tool call requires a complete API round-trip")
	fmt.Println("2. Large tool results consume significant tokens")
	fmt.Println("3. Sequential dependencies require multiple API calls")
}
