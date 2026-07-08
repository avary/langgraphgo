package main

import (
	"context"
	"fmt"
	"log"
	"time"

	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/prebuilt"
)

func main() {
	fmt.Println("=== ChatAgent AsyncChat Demo ===")
	fmt.Println("This example demonstrates streaming responses from the agent.")
	fmt.Println()

	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Create ChatAgent
	agent, err := prebuilt.NewChatAgent(llm, nil)
	if err != nil {
		log.Fatalf("Failed to create ChatAgent: %v", err)
	}

	ctx := context.Background()

	// Demo 1: Character-by-character streaming with AsyncChat
	fmt.Println("--- Demo 1: Character-by-Character Streaming ---")
	fmt.Print("User: Hello!\n")
	fmt.Print("Agent: ")

	respChan1, err := agent.AsyncChat(ctx, "Hello!")
	if err != nil {
		log.Fatalf("AsyncChat failed: %v", err)
	}

	for char := range respChan1 {
		fmt.Print(char)
		time.Sleep(20 * time.Millisecond) // Simulate typing effect
	}
	fmt.Println("")

	// Demo 2: Word-by-word streaming with AsyncChatWithChunks
	fmt.Println("--- Demo 2: Word-by-Word Streaming ---")
	fmt.Print("User: Can you explain async chat?\n")
	fmt.Print("Agent: ")

	respChan2, err := agent.AsyncChatWithChunks(ctx, "Can you explain async chat?")
	if err != nil {
		log.Fatalf("AsyncChatWithChunks failed: %v", err)
	}

	start := time.Now()
	for word := range respChan2 {
		fmt.Print(word)
		time.Sleep(100 * time.Millisecond) // Simulate thinking/typing
		if time.Since(start) > 5*time.Second {
			break
		}
	}
	fmt.Println("")

	// Demo 3: Collecting full response
	fmt.Println("--- Demo 3: Collecting Full Response ---")
	fmt.Println("User: What's the benefit of streaming?")
	fmt.Print("Agent: ")

	respChan3, err := agent.AsyncChatWithChunks(ctx, "What's the benefit of streaming?")
	if err != nil {
		log.Fatalf("AsyncChatWithChunks failed: %v", err)
	}

	start = time.Now()
	var fullResponse string
	chunkCount := 0
	for chunk := range respChan3 {
		fullResponse += chunk
		chunkCount++
		fmt.Print(chunk)
		if time.Since(start) > 5*time.Second {
			break
		}
		time.Sleep(80 * time.Millisecond)
	}
	fmt.Printf("\n\n[Received %d chunks, total length: %d characters]\n\n", chunkCount, len(fullResponse))

	// Demo 4: Using context cancellation
	fmt.Println("--- Demo 4: Context Cancellation ---")
	fmt.Println("User: Tell me a very long story...")
	fmt.Print("Agent: ")

	// Create a context with timeout
	ctx4, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	respChan4, err := agent.AsyncChat(ctx4, "Tell me a very long story")
	if err != nil {
		log.Fatalf("AsyncChat failed: %v", err)
	}

	start = time.Now()
	receivedChunks := 0
	for char := range respChan4 {
		fmt.Print(char)
		receivedChunks++
		time.Sleep(30 * time.Millisecond)
		if time.Since(start) > 5*time.Second {
			break
		}
	}

	fmt.Printf("\n\n[Stream was interrupted after receiving %d characters due to context timeout]\n\n", receivedChunks)

	// Demo 5: Comparison with regular Chat
	fmt.Println("--- Demo 5: Comparison with Regular Chat ---")
	fmt.Println("User: One more question please")
	fmt.Print("Agent (regular Chat): ")

	start = time.Now()
	regularResp, err := agent.Chat(context.Background(), "One more question please")
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	elapsed := time.Since(start)

	fmt.Println(regularResp)
	fmt.Printf("[Regular chat returned in %v]\n\n", elapsed)

	fmt.Println("=== Demo Complete ===")
	fmt.Println("\nKey Takeaways:")
	fmt.Println("1. AsyncChat streams character-by-character for real-time typing effect")
	fmt.Println("2. AsyncChatWithChunks streams word-by-word for better readability")
	fmt.Println("3. Use context for timeouts and cancellation")
	fmt.Println("4. Channel is automatically closed when response is complete")
	fmt.Println("5. Regular Chat still available for non-streaming use cases")
}
