package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/smallnest/langgraphgo/adapter"
	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
)

// State defines the structure for the graph state
type State struct {
	Messages []llms.MessageContent
	Output   strings.Builder
}

func main() {
	// Check for API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY is not set")
	}

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Wrap LLM with streaming capability
	var fullResponse strings.Builder
	streamingLLM := adapter.WrapLLMWithStreaming(llm, func(chunk string) {
		// Print each chunk as it arrives
		fmt.Print(chunk)
		fullResponse.WriteString(chunk)
	})

	// Create a state graph
	g := graph.NewStateGraph[State]()

	// Add a node that uses streaming LLM
	g.AddNode("chat", "chat", func(ctx context.Context, state State) (State, error) {
		response, err := streamingLLM.GenerateContent(ctx, state.Messages)
		if err != nil {
			return state, err
		}

		if len(response.Choices) > 0 {
			state.Output.WriteString(response.Choices[0].Content)
		}
		return state, nil
	})

	g.AddEdge("chat", graph.END)
	g.SetEntryPoint("chat")

	// Compile the graph
	runnable, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Prepare input
	userInput := "Explain Go's concurrency model in simple terms."
	if len(os.Args) > 1 {
		userInput = strings.Join(os.Args[1:], " ")
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are a helpful assistant that explains technical concepts clearly."),
		llms.TextParts(llms.ChatMessageTypeHuman, userInput),
	}

	// Execute the graph
	fmt.Print("Response: ")
	state := State{
		Messages: messages,
	}

	ctx := context.Background()
	_, err = runnable.Invoke(ctx, state)
	if err != nil {
		log.Fatal(err)
	}

	// Print final full response
	fmt.Printf("\n\nFull response captured: %d characters\n", fullResponse.Len())
}
