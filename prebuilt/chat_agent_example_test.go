package prebuilt_test

import (
	"context"
	"fmt"
	"os"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/llms/openai"
)

// Example demonstrating multi-turn conversation with ChatAgent
func ExampleChatAgent() {
	// Check if API key is available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY not set, skipping example")
		return
	}

	// Create OpenAI model
	model, err := openai.New()
	if err != nil {
		fmt.Printf("Error creating model: %v\n", err)
		return
	}

	// Create ChatAgent with no tools
	agent, err := prebuilt.NewChatAgent(llmtypes.Wrap(model), nil)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		return
	}

	ctx := context.Background()

	// First turn
	fmt.Println("User: Hello! My name is Alice.")
	resp1, err := agent.Chat(ctx, "Hello! My name is Alice.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Agent: %s\n\n", resp1)

	// Second turn - agent should remember the name
	fmt.Println("User: What's my name?")
	resp2, err := agent.Chat(ctx, "What's my name?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Agent: %s\n\n", resp2)

	// Third turn - continue the conversation
	fmt.Println("User: Tell me a short joke about programmers.")
	resp3, err := agent.Chat(ctx, "Tell me a short joke about programmers.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Agent: %s\n", resp3)

	// Display the session ID
	fmt.Printf("\nSession ID: %s\n", agent.ThreadID())
}
