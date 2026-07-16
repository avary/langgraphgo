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

// WeatherTool is a simple tool to get weather
type WeatherTool struct{}

func (t *WeatherTool) Name() string {
	return "get_weather"
}

func (t *WeatherTool) Description() string {
	return "Get the weather for a city"
}

func (t *WeatherTool) Call(ctx context.Context, input string) (string, error) {
	return fmt.Sprintf("The weather in %s is sunny and 25°C", input), nil
}

func main() {
	// Check if OPENAI_API_KEY is set
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Println("OPENAI_API_KEY not set, skipping example execution")
		return
	}

	ctx := context.Background()

	// Initialize LLM
	model, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Define tools
	inputTools := []tool.Tool{&WeatherTool{}}

	// Create agent with options using CreateAgentMap
	agent, err := prebuilt.CreateAgentMap(model, inputTools, 0,
		prebuilt.WithSystemMessage("You are a helpful weather assistant. Always be polite."),
		prebuilt.WithStateModifier(func(msgs []llms.MessageContent) []llms.MessageContent {
			// Example modifier: Log the number of messages
			log.Printf("Current message count: %d", len(msgs))
			return msgs
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Initial input
	inputs := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "What is the weather in San Francisco?"),
		},
	}

	// Run the agent
	log.Println("Starting agent...")
	result, err := agent.Invoke(ctx, inputs)
	if err != nil {
		log.Fatalf("Agent execution failed: %v", err)
	}

	// Print result
	messages := result["messages"].([]llms.MessageContent)
	lastMsg := messages[len(messages)-1]

	for _, part := range lastMsg.Parts {
		switch p := part.(type) {
		case llms.TextContent:
			fmt.Printf("Agent Response: %s\n", p.Text)
		case llms.ToolCall:
			fmt.Printf("Tool Call: %s\n", p.FunctionCall.Name)
		}
	}
}
