//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
)

// Example 1: Using OpenAI with LangChain
func OpenAIExample() {
	fmt.Println("\n🤖 OpenAI Example with LangChain")
	fmt.Println("==================================")

	// Create OpenAI LLM client using LangChain
	model, err := openai.New()
	if err != nil {
		log.Printf("OpenAI initialization failed: %v", err)
		return
	}

	// Create a graph that uses the LLM
	g := graph.NewStateGraph[[]llms.MessageContent]()

	g.AddNode("chat", "chat", func(ctx context.Context, messages []llms.MessageContent) ([]llms.MessageContent, error) {
		// Use LangChain's GenerateContent method
		response, err := model.GenerateContent(ctx, messages,
			llms.WithTemperature(0.7),
			llms.WithMaxTokens(150),
		)
		if err != nil {
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}

		// Append the response to messages
		return append(messages,
			llms.TextParts("ai", response.Choices[0].Content),
		), nil
	})

	g.AddEdge("chat", graph.END)
	g.SetEntryPoint("chat")

	runnable, err := g.Compile()
	if err != nil {
		log.Fatalf("Failed to compile graph: %v", err)
	}

	// Execute with initial message
	ctx := context.Background()
	result, err := runnable.Invoke(ctx, []llms.MessageContent{
		llms.TextParts("human", "What are the benefits of using LangChain with Go?"),
	})
	if err != nil {
		log.Printf("Execution failed: %v", err)
		return
	}

	// Print the conversation
	messages := result
	for _, msg := range messages {
		fmt.Printf("%s: %s\n", msg.Role, msg.Parts[0])
	}
}

// Example 3: Multi-step reasoning with LangChain
func MultiStepReasoningExample() {
	fmt.Println("\n🧠 Multi-Step Reasoning with LangChain")
	fmt.Println("======================================")

	// Use whichever LLM is available
	var model llms.Model
	var err error

	ctx := context.Background()

	if os.Getenv("OPENAI_API_KEY") != "" {
		model, err = openai.New()
		fmt.Println("Using OpenAI...")
	} else {
		fmt.Println("No API keys found. Set OPENAI_API_KEY or GOOGLE_API_KEY")
		return
	}

	if err != nil {
		log.Fatalf("Failed to initialize LLM: %v", err)
	}

	// Create a multi-step reasoning graph
	g := graph.NewCheckpointableStateGraph[map[string]any]()

	// Step 1: Analyze the problem
	g.AddNode("analyze", "analyze", func(ctx context.Context, data map[string]any) (map[string]any, error) {
		messages := []llms.MessageContent{
			llms.TextParts("system", "You are a helpful assistant that breaks down problems step by step."),
			llms.TextParts("human", data["problem"].(string)),
		}

		response, err := model.GenerateContent(ctx, messages,
			llms.WithTemperature(0.3), // Lower temperature for analysis
		)
		if err != nil {
			return nil, err
		}

		data["analysis"] = response.Choices[0].Content
		return data, nil
	})

	// Step 2: Generate solution
	g.AddNode("solve", "solve", func(ctx context.Context, data map[string]any) (map[string]any, error) {
		messages := []llms.MessageContent{
			llms.TextParts("system", "Based on the analysis, provide a clear solution."),
			llms.TextParts("human", fmt.Sprintf(
				"Problem: %s\nAnalysis: %s\n\nProvide a solution:",
				data["problem"], data["analysis"],
			)),
		}

		response, err := model.GenerateContent(ctx, messages,
			llms.WithTemperature(0.5),
		)
		if err != nil {
			return nil, err
		}

		data["solution"] = response.Choices[0].Content
		return data, nil
	})

	// Step 3: Verify solution
	g.AddNode("verify", "verify", func(ctx context.Context, data map[string]any) (map[string]any, error) {
		messages := []llms.MessageContent{
			llms.TextParts("system", "Verify if the solution is correct and complete."),
			llms.TextParts("human", fmt.Sprintf(
				"Problem: %s\nSolution: %s\n\nVerify this solution:",
				data["problem"], data["solution"],
			)),
		}

		response, err := model.GenerateContent(ctx, messages,
			llms.WithTemperature(0.2), // Very low temperature for verification
		)
		if err != nil {
			return nil, err
		}

		data["verification"] = response.Choices[0].Content
		return data, nil
	})

	// Connect the nodes
	g.AddEdge("analyze", "solve")
	g.AddEdge("solve", "verify")
	g.AddEdge("verify", graph.END)
	g.SetEntryPoint("analyze")

	// Enable checkpointing
	g.SetCheckpointConfig(graph.CheckpointConfig{
		Store:    graph.NewMemoryCheckpointStore(),
		AutoSave: true,
	})

	runnable, err := g.CompileCheckpointable()
	if err != nil {
		log.Fatalf("Failed to compile graph: %v", err)
	}

	// Execute with a problem
	problem := map[string]any{
		"problem": "How can I optimize a Go web server that's handling 10,000 concurrent connections?",
	}

	result, err := runnable.Invoke(ctx, problem)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}

	// Display results
	data := result
	fmt.Printf("\n📊 Analysis:\n%s\n", data["analysis"])
	fmt.Printf("\n💡 Solution:\n%s\n", data["solution"])
	fmt.Printf("\n✅ Verification:\n%s\n", data["verification"])

	// Show checkpoints
	checkpoints, _ := runnable.ListCheckpoints(ctx)
	fmt.Printf("\n📍 Created %d checkpoints during reasoning\n", len(checkpoints))
}

func main() {
	fmt.Println("🦜🔗 LangChain Integration Examples for LangGraphGo")
	fmt.Println("===================================================")

	// Run examples based on available API keys
	if os.Getenv("OPENAI_API_KEY") != "" {
		OpenAIExample()
	} else {
		fmt.Println("\n⚠️  OpenAI example skipped (OPENAI_API_KEY not set)")
	}

	// Multi-step example works with either API
	if os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("GOOGLE_API_KEY") != "" {
		MultiStepReasoningExample()
	} else {
		fmt.Println("\n⚠️  Multi-step reasoning example skipped (no API keys set)")
		fmt.Println("\nTo run these examples, set one of the following environment variables:")
		fmt.Println("  - OPENAI_API_KEY")
		fmt.Println("  - GOOGLE_API_KEY")
	}
}
