package main

import (
	"context"
	"fmt"
	"log"

	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/ptc"
	"github.com/smallnest/langgraphgo/tooltypes"
)

// CalculatorTool performs arithmetic operations
type CalculatorTool struct{}

func (t CalculatorTool) Name() string {
	return "calculator"
}

func (t CalculatorTool) Description() string {
	return "Performs arithmetic calculations. Input should be a mathematical expression as a string (e.g., '2 + 2', '10 * 5')."
}

func (t CalculatorTool) Call(ctx context.Context, input string) (string, error) {
	// Simple calculator implementation
	// In a real implementation, use a proper expression parser
	return fmt.Sprintf("Result of '%s' would be calculated here", input), nil
}

// WeatherTool gets weather information
type WeatherTool struct{}

func (t WeatherTool) Name() string {
	return "get_weather"
}

func (t WeatherTool) Description() string {
	return "Gets current weather for a location. Input should be the city name."
}

func (t WeatherTool) Call(ctx context.Context, input string) (string, error) {
	// Mock weather data
	return fmt.Sprintf("Weather in %s: Sunny, 72°F", input), nil
}

func main() {
	fmt.Println("=== Simple PTC Example ===\n")

	// Create model (supports any LLM that implements llmtypes.Model)
	model, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Define tools
	toolList := []tooltypes.Tool{
		CalculatorTool{},
		WeatherTool{},
	}

	// Create PTC agent
	agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
		Model:         model,
		Tools:         toolList,
		Language:      ptc.LanguagePython, // or ptc.LanguageGo
		MaxIterations: 5,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Run a query
	query := "What's the weather in San Francisco and New York? Also calculate 125 * 8."

	fmt.Printf("Query: %s\n\n", query)

	result, err := agent.Invoke(context.Background(), map[string]any{
		"messages": []llmtypes.MessageContent{
			{
				Role:  llmtypes.ChatMessageTypeHuman,
				Parts: []llmtypes.ContentPart{llmtypes.TextPart(query)},
			},
		},
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Print result
	messages := result["messages"].([]llmtypes.MessageContent)
	lastMsg := messages[len(messages)-1]

	fmt.Println("Answer:")
	for _, part := range lastMsg.Parts {
		if textPart, ok := part.(llmtypes.TextContent); ok {
			fmt.Println(textPart.Text)
		}
	}
}
