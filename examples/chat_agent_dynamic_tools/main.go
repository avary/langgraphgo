package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tool"
)

// SimpleMockModel is a simple mock model for demonstration
type SimpleMockModel struct {
	turnCount int
}

func (m *SimpleMockModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	m.turnCount++

	// Extract tool information from options
	hasCalculator := false
	hasWeather := false

	// In a real scenario, we would inspect the options to see what tools are available
	_ = options // Suppress unused variable warning

	// Generate response based on turn
	var response string
	switch m.turnCount {
	case 1:
		response = "Hello! I can help you with various tasks. What would you like to do?"
	case 2:
		if hasCalculator {
			response = "I can now perform calculations! The result is 4."
		} else {
			response = "I now have access to a calculator tool. What calculation would you like me to perform?"
		}
	case 3:
		if hasWeather {
			response = "I now have access to weather tools! The weather in San Francisco is sunny, 72°F."
		} else {
			response = "I now have access to a weather tool. Which city's weather would you like to know?"
		}
	case 4:
		response = "I no longer have access to the calculator, but I still have the weather tool available."
	default:
		response = fmt.Sprintf("I'm ready to help! (Turn %d)", m.turnCount)
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: response},
		},
	}, nil
}

func (m *SimpleMockModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

// CalculatorTool is a simple calculator tool
type CalculatorTool struct{}

func (ct *CalculatorTool) Name() string {
	return "calculator"
}

func (ct *CalculatorTool) Description() string {
	return "Performs basic arithmetic calculations"
}

func (ct *CalculatorTool) Call(ctx context.Context, input string) (string, error) {
	return fmt.Sprintf("Calculated: %s = 4", input), nil
}

// WeatherTool is a simple weather tool
type WeatherTool struct{}

func (wt *WeatherTool) Name() string {
	return "weather"
}

func (wt *WeatherTool) Description() string {
	return "Gets current weather for a city"
}

func (wt *WeatherTool) Call(ctx context.Context, input string) (string, error) {
	return fmt.Sprintf("Weather in %s: Sunny, 72°F", input), nil
}

func main() {
	fmt.Println("=== ChatAgent Dynamic Tools Demo ===")
	fmt.Println("This example demonstrates how to add and remove tools dynamically during a conversation.")
	fmt.Println()

	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}
	// Create ChatAgent with no initial tools
	agent, err := prebuilt.NewChatAgent(llm, nil)
	if err != nil {
		log.Fatalf("Failed to create ChatAgent: %v", err)
	}

	ctx := context.Background()

	// Display session ID
	fmt.Printf("Session ID: %s\n\n", agent.ThreadID())

	// Turn 1: Initial chat with no tools
	fmt.Println("--- Turn 1: No tools available ---")
	fmt.Printf("Available tools: %d\n", len(agent.GetTools()))
	fmt.Println("User: Hello!")
	resp1, err := agent.Chat(ctx, "Hello!")
	if err != nil {
		log.Fatalf("Chat error: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", resp1)

	// Turn 2: Add calculator tool
	fmt.Println("--- Turn 2: Adding calculator tool ---")
	calcTool := &CalculatorTool{}
	agent.AddTool(calcTool)
	fmt.Printf("Available tools: %d (%s)\n", len(agent.GetTools()), agent.GetTools()[0].Name())
	fmt.Println("User: Calculate 2 + 2")
	resp2, err := agent.Chat(ctx, "Calculate 2 + 2")
	if err != nil {
		log.Fatalf("Chat error: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", resp2)

	// Turn 3: Add weather tool
	fmt.Println("--- Turn 3: Adding weather tool ---")
	weatherTool := &WeatherTool{}
	agent.AddTool(weatherTool)
	toolNames := []string{}
	for _, t := range agent.GetTools() {
		toolNames = append(toolNames, t.Name())
	}
	fmt.Printf("Available tools: %d (%v)\n", len(agent.GetTools()), toolNames)
	fmt.Println("User: What's the weather in San Francisco?")
	resp3, err := agent.Chat(ctx, "What's the weather in San Francisco?")
	if err != nil {
		log.Fatalf("Chat error: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", resp3)

	// Turn 4: Remove calculator tool
	fmt.Println("--- Turn 4: Removing calculator tool ---")
	removed := agent.RemoveTool("calculator")
	if removed {
		fmt.Println("Calculator tool removed successfully")
	}
	toolNames = []string{}
	for _, t := range agent.GetTools() {
		toolNames = append(toolNames, t.Name())
	}
	fmt.Printf("Available tools: %d (%v)\n", len(agent.GetTools()), toolNames)
	fmt.Println("User: What tools do you have now?")
	resp4, err := agent.Chat(ctx, "What tools do you have now?")
	if err != nil {
		log.Fatalf("Chat error: %v", err)
	}
	fmt.Printf("Agent: %s\n\n", resp4)

	// Demonstration of other tool management methods
	fmt.Println("--- Other Tool Management Features ---")

	// SetTools - replace all tools at once
	newTool1 := &CalculatorTool{}
	newTool2 := &WeatherTool{}
	agent.SetTools([]tool.Tool{newTool1, newTool2})
	fmt.Printf("After SetTools: %d tools\n", len(agent.GetTools()))

	// ClearTools - remove all tools
	agent.ClearTools()
	fmt.Printf("After ClearTools: %d tools\n", len(agent.GetTools()))

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("1. AddTool() - Add tools dynamically during conversation")
	fmt.Println("2. RemoveTool() - Remove specific tools by name")
	fmt.Println("3. GetTools() - Query currently available tools")
	fmt.Println("4. SetTools() - Replace all tools at once")
	fmt.Println("5. ClearTools() - Remove all dynamic tools")
	fmt.Println("\nThe agent can adapt its capabilities on-the-fly based on context!")
}
