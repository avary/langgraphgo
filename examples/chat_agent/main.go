package main

import (
	"context"
	"fmt"
	"log"

	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/prebuilt"
)

func main() {
	fmt.Println("=== ChatAgent Multi-Turn Conversation Demo ===")
	fmt.Println()

	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Create ChatAgent with no tools
	agent, err := prebuilt.NewChatAgent(llm, nil)
	if err != nil {
		log.Fatalf("Failed to create ChatAgent: %v", err)
	}

	ctx := context.Background()

	// Display session ID
	fmt.Printf("Session ID: %s\n\n", agent.ThreadID())

	// Turn 1: Greeting
	fmt.Println("User: Hello!")
	resp1, err := agent.Chat(ctx, "Hello!")
	if err != nil {
		log.Fatalf("Chat error: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", resp1)

	// Turn 2: Introduce name
	fmt.Println("User: My name is Alice")
	resp2, err := agent.Chat(ctx, "My name is Alice")
	if err != nil {
		log.Fatalf("Chat error: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", resp2)

	// Turn 3: Ask agent to recall name (testing memory)
	fmt.Println("User: What's my name?")
	resp3, err := agent.Chat(ctx, "What's my name?")
	if err != nil {
		log.Fatalf("Chat error: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", resp3)

	// Turn 4: Another question
	fmt.Println("User: How many messages have we exchanged?")
	resp4, err := agent.Chat(ctx, "How many messages have we exchanged?")
	if err != nil {
		log.Fatalf("Chat error: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", resp4)

	fmt.Println("=== Conversation Complete ===")
	fmt.Printf("\nThis demo shows that ChatAgent maintains conversation history across multiple turns.\n")
	fmt.Printf("The agent can reference previous messages (like your name) in later responses.\n")
}
