//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
)

// StreamingState holds the streaming callback function
type StreamingState struct {
	Messages       []llms.MessageContent
	StreamCallback func(chunk string)
}

// Example 1: Basic streaming with LangChainGo
func basicStreamingExample() {
	fmt.Println("\n=== Example 1: Basic Streaming ===")

	var llm llms.Model
	var err error

	llm, err = openai.New()
	if err != nil {
		log.Printf("Failed to initialize LLM: %v", err)
		return
	}

	// Create a graph with streaming support
	g := graph.NewStateGraph[StreamingState]()

	g.AddNode("stream_chat", "stream_chat", func(ctx context.Context, state StreamingState) (StreamingState, error) {
		// Add streaming function to the LLM call
		_, err := llm.GenerateContent(ctx, state.Messages,
			llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
				// Call the streaming callback for each chunk
				state.StreamCallback(string(chunk))
				return nil
			}),
			llms.WithTemperature(0.7),
		)
		if err != nil {
			return state, fmt.Errorf("LLM generation failed: %w", err)
		}

		return state, nil
	})

	g.AddEdge("stream_chat", graph.END)
	g.SetEntryPoint("stream_chat")

	runnable, err := g.Compile()
	if err != nil {
		log.Fatalf("Failed to compile graph: %v", err)
	}

	// Prepare state with streaming callback
	fullResponse := strings.Builder{}
	state := StreamingState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Explain Go's concurrency model in simple terms."),
		},
		StreamCallback: func(chunk string) {
			// Print each chunk as it arrives
			fmt.Print(chunk)
			fullResponse.WriteString(chunk)
		},
	}

	ctx := context.Background()
	fmt.Println("\nStreaming response:")
	fmt.Println("-------------------")
	_, err = runnable.Invoke(ctx, state)
	if err != nil {
		log.Printf("Execution failed: %v", err)
		return
	}

	fmt.Println("\n-------------------")
	fmt.Printf("\nTotal characters received: %d\n", fullResponse.Len())
}

// Example 2: Streaming with ListenableGraph for event notifications
func streamingWithEventsExample() {
	fmt.Println("\n=== Example 2: Streaming with Events ===")

	llm, err := openai.New()
	if err != nil {
		log.Printf("Failed to initialize OpenAI: %v", err)
		return
	}

	ctx := context.Background()

	// Create a listenable graph for event tracking
	g := graph.NewListenableStateGraph[StreamingState]()

	// Custom listener that can be triggered from streaming callback
	type ProgressListener struct {
		graph.NodeListenerFunc[StreamingState]
		chunkCount int
		chunks     [][]byte   // Store all chunks in order
		mu         sync.Mutex // Thread-safe chunk access
	}

	progressListener := &ProgressListener{}

	progressListener.NodeListenerFunc = graph.NodeListenerFunc[StreamingState](func(ctx context.Context, event graph.NodeEvent, nodeName string, state StreamingState, err error) {
		switch event {
		case graph.NodeEventStart:
			fmt.Printf("\n[EVENT] Node '%s' started\n", nodeName)
		case graph.NodeEventComplete:
			// Calculate total bytes from stored chunks
			totalBytes := 0
			for _, chunk := range progressListener.chunks {
				totalBytes += len(chunk)
			}
			fmt.Printf("[EVENT] Node '%s' completed (chunks: %d, bytes: %d)\n", nodeName, progressListener.chunkCount, totalBytes)

			// Verify chunks are in order by concatenating them
			reconstructed := string(bytes.Join(progressListener.chunks, nil))
			fmt.Printf("[EVENT] Reconstructed response length: %d chars\n", len(reconstructed))
		case graph.NodeEventProgress:
			// Called for each streaming chunk
			progressListener.chunkCount++
			if progressListener.chunkCount%10 == 1 {
				// Print every first chunk to avoid too much output
				fmt.Printf("\n[EVENT] Node '%s' progress: chunk #%d received", nodeName, progressListener.chunkCount)
			}
		case graph.NodeEventError:
			fmt.Printf("[EVENT] Node '%s' error: %v\n", nodeName, err)
		}
	})

	node := g.AddNode("stream_with_events", "stream_with_events", func(ctx context.Context, state StreamingState) (StreamingState, error) {
		// Stream the response with progress events
		response, err := llm.GenerateContent(ctx, state.Messages,
			llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
				// Save chunk to listener (thread-safe)
				progressListener.mu.Lock()
				// Make a copy of chunk since the underlying array may be reused
				chunkCopy := make([]byte, len(chunk))
				copy(chunkCopy, chunk)
				progressListener.chunks = append(progressListener.chunks, chunkCopy)
				progressListener.mu.Unlock()

				// Emit progress event for each chunk
				progressListener.OnNodeEvent(ctx, graph.NodeEventProgress, "stream_with_events", state, nil)

				// Also stream to output
				state.StreamCallback(string(chunk))
				return nil
			}),
			llms.WithTemperature(0.8),
		)
		if err != nil {
			return state, fmt.Errorf("LLM generation failed: %w", err)
		}

		// Add the full response to messages
		state.Messages = append(state.Messages,
			llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content),
		)

		return state, nil
	})

	node.AddListener(progressListener)

	g.AddEdge("stream_with_events", graph.END)
	g.SetEntryPoint("stream_with_events")

	runnable, err := g.CompileListenable()
	if err != nil {
		log.Fatalf("Failed to compile graph: %v", err)
	}

	// Prepare state
	state := StreamingState{
		Messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Write a short haiku about programming."),
		},
		StreamCallback: func(chunk string) {
			fmt.Print(chunk)
		},
	}

	fmt.Println("\nStreaming response with progress events:")
	fmt.Println("-----------------------------------------")
	_, err = runnable.Invoke(ctx, state)
	if err != nil {
		log.Printf("Execution failed: %v", err)
		return
	}
	fmt.Println("\n-----------------------------------------")
}

// Example 3: Multi-step streaming with checkpointing
func multiStepStreamingExample() {
	fmt.Println("\n=== Example 3: Multi-Step Streaming ===")

	llm, err := openai.New()
	if err != nil {
		log.Printf("Failed to initialize OpenAI: %v", err)
		return
	}

	ctx := context.Background()

	// Create a checkpointable graph for multi-step processing
	g := graph.NewCheckpointableStateGraph[map[string]any]()

	// Step 1: Analyze with streaming
	g.AddNode("analyze", "analyze", func(ctx context.Context, data map[string]any) (map[string]any, error) {
		messages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, "You are a helpful assistant. Analyze the given topic briefly."),
			llms.TextParts(llms.ChatMessageTypeHuman, data["topic"].(string)),
		}

		fmt.Println("\n[Step 1] Analysis:")
		fmt.Print("  ")

		// Accumulate streaming output
		var analysisBuilder strings.Builder
		_, err := llm.GenerateContent(ctx, messages,
			llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
				fmt.Print(string(chunk))
				analysisBuilder.Write(chunk)
				return nil
			}),
			llms.WithMaxTokens(100),
		)
		if err != nil {
			return nil, err
		}

		// Save the analysis result to state for next node
		data["analysis"] = analysisBuilder.String()
		data["step1_completed"] = true
		return data, nil
	})

	// Step 2: Expand with streaming
	g.AddNode("expand", "expand", func(ctx context.Context, data map[string]any) (map[string]any, error) {
		messages := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, "Provide more details about the topic based on the analysis."),
			llms.TextParts(llms.ChatMessageTypeHuman, fmt.Sprintf("Topic: %s\nAnalysis: %s", data["topic"], data["analysis"])),
		}

		fmt.Println("\n[Step 2] Expansion:")
		fmt.Print("  ")

		// Accumulate streaming output
		var expansionBuilder strings.Builder
		_, err := llm.GenerateContent(ctx, messages,
			llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
				fmt.Print(string(chunk))
				expansionBuilder.Write(chunk)
				return nil
			}),
			llms.WithMaxTokens(150),
		)
		if err != nil {
			return nil, err
		}

		// Save the expansion result to state
		data["expansion"] = expansionBuilder.String()
		data["step2_completed"] = true
		return data, nil
	})

	// Connect nodes
	g.AddEdge("analyze", "expand")
	g.AddEdge("expand", graph.END)
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

	// Execute multi-step flow
	input := map[string]any{
		"topic": "Go programming language",
	}

	fmt.Println("\nMulti-step streaming response:")
	fmt.Println("-------------------------------")
	result, err := runnable.Invoke(ctx, input)
	if err != nil {
		log.Printf("Execution failed: %v", err)
		return
	}

	fmt.Println("\n-------------------------------")
	fmt.Printf("\nSteps completed: step1=%v, step2=%v\n",
		result["step1_completed"], result["step2_completed"])
	fmt.Printf("Analysis length: %d chars\n", len(result["analysis"].(string)))
	fmt.Printf("Expansion length: %d chars\n", len(result["expansion"].(string)))
}

func main() {
	fmt.Println("🦜🔗 LangChainGo Streaming Examples for LangGraphGo")
	fmt.Println("====================================================")

	// Check for OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		fmt.Println("\n⚠️  OPENAI_API_KEY not found!")
		fmt.Println("Set the environment variable:")
		fmt.Println("  export OPENAI_API_KEY=your-key-here")
		return
	}

	// Run all examples
	basicStreamingExample()
	streamingWithEventsExample()
	multiStepStreamingExample()

	fmt.Println("\n✅ All examples completed!")
}
