// PiAgent OpenAI example demonstrates PiAgent with a real LLM
package main

import (
	"context"
	"fmt"
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
		fmt.Printf("Failed to create LLM: %v\n", err)
		os.Exit(1)
	}

	// Create a simple search tool (mock implementation)
	searchTool := &SearchTool{}

	// Create PiAgent
	agent, err := prebuilt.NewPiAgent(
		model,
		[]tool.Tool{searchTool},
		prebuilt.WithPiSystemPrompt("You are a helpful research assistant. Use the search tool when you need to find current information."),
		prebuilt.WithPiMaxIterations(5),
	)
	if err != nil {
		fmt.Printf("Failed to create agent: %v\n", err)
		os.Exit(1)
	}

	// Subscribe to events for real-time feedback
	unsubscribe := agent.Subscribe(func(event prebuilt.PiAgentEvent) {
		switch event.Type {
		case prebuilt.EventAgentStart:
			fmt.Println("\n🤖 Agent started")
		case prebuilt.EventAgentEnd:
			fmt.Println("\n✅ Agent completed")
		case prebuilt.EventTurnStart:
			fmt.Println("\n🔄 Thinking...")
		case prebuilt.EventTurnEnd:
			fmt.Println("✓ Turn complete")
		case prebuilt.EventToolExecutionStart:
			fmt.Printf("  🔧 Using tool: %s\n", event.ToolName)
		case prebuilt.EventToolExecutionEnd:
			if event.ToolError {
				fmt.Printf("  ❌ Tool error: %v\n", event.ToolResult)
			} else {
				fmt.Printf("  ✓ Tool done\n")
			}
		}
	})
	defer unsubscribe()

	// Example: Ask a question that might need the search tool
	fmt.Println("\n=== Example: Research Query ===")
	msg := llms.TextParts(llms.ChatMessageTypeHuman, "What's the latest news about AI agents?")

	err = agent.Prompt(ctx, msg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Print final messages
	state := agent.GetState()
	fmt.Println("\n=== Conversation ===")
	for i, m := range state.Messages {
		fmt.Printf("[%d] %s: ", i+1, m.Role)
		for _, part := range m.Parts {
			if text, ok := part.(llms.TextContent); ok {
				fmt.Println(text.Text)
			}
		}
	}

	fmt.Println("\n=== Done ===")
}

// SearchTool is a mock search tool for demonstration
type SearchTool struct{}

func (t *SearchTool) Name() string {
	return "search"
}

func (t *SearchTool) Description() string {
	return "Search for current information on the web. Use this when you need up-to-date facts."
}

func (t *SearchTool) Call(ctx context.Context, input string) (string, error) {
	// Mock implementation - in real use, integrate with a real search API
	return fmt.Sprintf("Search results for '%s': Found 5 relevant articles with recent updates.", input), nil
}

func (t *SearchTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query",
			},
		},
		"required":             []string{"query"},
		"additionalProperties": false,
	}
}
