package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/smallnest/langgraphgo/ptc"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

// CalculatorTool performs arithmetic operations
type CalculatorTool struct{}

func (t CalculatorTool) Name() string {
	return "calculator"
}

func (t CalculatorTool) Description() string {
	return `Performs arithmetic calculations.

Input: A JSON string with the operation and numbers:
{"operation": "add|subtract|multiply|divide|power|sqrt", "a": number, "b": number (optional for sqrt)}

Examples:
- {"operation": "add", "a": 5, "b": 3} -> 8
- {"operation": "multiply", "a": 4, "b": 7} -> 28
- {"operation": "sqrt", "a": 16} -> 4`
}

func (t CalculatorTool) Call(ctx context.Context, input string) (string, error) {
	// Parse input
	var params struct {
		Operation string  `json:"operation"`
		A         float64 `json:"a"`
		B         float64 `json:"b"`
	}

	// Simple JSON parsing
	input = strings.TrimSpace(input)
	if strings.Contains(input, "operation") {
		// Extract operation
		if strings.Contains(input, `"add"`) {
			params.Operation = "add"
		} else if strings.Contains(input, `"subtract"`) {
			params.Operation = "subtract"
		} else if strings.Contains(input, `"multiply"`) {
			params.Operation = "multiply"
		} else if strings.Contains(input, `"divide"`) {
			params.Operation = "divide"
		} else if strings.Contains(input, `"power"`) {
			params.Operation = "power"
		} else if strings.Contains(input, `"sqrt"`) {
			params.Operation = "sqrt"
		}

		// Extract numbers
		parts := strings.Split(input, ",")
		for _, part := range parts {
			if strings.Contains(part, `"a"`) {
				numStr := strings.Split(part, ":")[1]
				numStr = strings.Trim(strings.TrimSpace(numStr), "}")
				if num, err := strconv.ParseFloat(numStr, 64); err == nil {
					params.A = num
				}
			}
			if strings.Contains(part, `"b"`) {
				numStr := strings.Split(part, ":")[1]
				numStr = strings.Trim(strings.TrimSpace(numStr), "}")
				if num, err := strconv.ParseFloat(numStr, 64); err == nil {
					params.B = num
				}
			}
		}
	}

	var result float64
	switch params.Operation {
	case "add":
		result = params.A + params.B
	case "subtract":
		result = params.A - params.B
	case "multiply":
		result = params.A * params.B
	case "divide":
		if params.B == 0 {
			return "", fmt.Errorf("division by zero")
		}
		result = params.A / params.B
	case "power":
		result = math.Pow(params.A, params.B)
	case "sqrt":
		result = math.Sqrt(params.A)
	default:
		return "", fmt.Errorf("unknown operation: %s", params.Operation)
	}

	return fmt.Sprintf("%.2f", result), nil
}

// WeatherTool gets weather information
type WeatherTool struct{}

func (t WeatherTool) Name() string {
	return "get_weather"
}

func (t WeatherTool) Description() string {
	return `Gets current weather for a location.

Input: A JSON string with the city name:
{"city": "city_name"}

Example: {"city": "San Francisco"}

Returns: Weather information as a JSON string with temperature and conditions.`
}

func (t WeatherTool) Call(ctx context.Context, input string) (string, error) {
	// Extract city name from input
	city := "Unknown"
	if strings.Contains(input, "city") {
		parts := strings.Split(input, ":")
		if len(parts) > 1 {
			city = strings.Trim(strings.TrimSpace(parts[1]), `"{}`)
		}
	} else {
		city = strings.Trim(input, `"{} `)
	}

	// Mock weather data
	temps := map[string]int{
		"San Francisco": 68,
		"New York":      55,
		"London":        52,
		"Tokyo":         70,
		"Paris":         58,
	}

	temp, ok := temps[city]
	if !ok {
		temp = 72
	}

	return fmt.Sprintf(`{"city": "%s", "temperature": %d, "conditions": "Sunny", "humidity": 65}`, city, temp), nil
}

// DataProcessorTool processes and filters data
type DataProcessorTool struct{}

func (t DataProcessorTool) Name() string {
	return "process_data"
}

func (t DataProcessorTool) Description() string {
	return `Processes and filters data arrays.

Input: A JSON string with operation and data:
{"operation": "sum|average|max|min|count", "data": [1, 2, 3, ...]}

Examples:
- {"operation": "sum", "data": [1, 2, 3, 4, 5]} -> 15
- {"operation": "average", "data": [10, 20, 30]} -> 20
- {"operation": "max", "data": [5, 2, 9, 1]} -> 9`
}

func (t DataProcessorTool) Call(ctx context.Context, input string) (string, error) {
	// For simplicity, return mock results
	if strings.Contains(input, "sum") {
		return "15", nil
	}
	if strings.Contains(input, "average") {
		return "20.5", nil
	}
	if strings.Contains(input, "max") {
		return "42", nil
	}
	return "10", nil
}

func main() {
	fmt.Println("=== Programmatic Tool Calling (PTC) Example ===\n")
	fmt.Println("This example demonstrates how PTC allows LLMs to generate")
	fmt.Println("code that calls tools programmatically, reducing API round-trips.\n")

	// Create model (supports any LLM that implements llms.Model)
	model, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Define tools
	toolList := []tools.Tool{
		CalculatorTool{},
		WeatherTool{},
		DataProcessorTool{},
	}

	// Create PTC agent
	agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
		Model:         model,
		Tools:         toolList,
		Language:      ptc.LanguagePython, // Python is recommended (better LLM support)
		MaxIterations: 5,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Run a query that benefits from PTC
	// This query requires multiple tool calls that can be done programmatically
	query := `What's the weather in San Francisco and New York?
	Calculate the average of their temperatures, then multiply by 2.`

	fmt.Printf("Query: %s\n\n", query)
	fmt.Println("Processing... (The LLM will generate code to call tools)")
	fmt.Println(strings.Repeat("-", 60))

	result, err := agent.Invoke(context.Background(), map[string]interface{}{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart(query)},
			},
		},
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Print result
	messages := result.(map[string]interface{})["messages"].([]llms.MessageContent)

	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("Execution Complete!")
	fmt.Println(strings.Repeat("-", 60))

	// Show all messages for transparency
	fmt.Println("\nAll Messages:")
	for i, msg := range messages {
		var role string
		switch msg.Role {
		case llms.ChatMessageTypeHuman:
			role = "User"
		case llms.ChatMessageTypeAI:
			role = "AI"
		case llms.ChatMessageTypeTool:
			role = "Tool"
		case llms.ChatMessageTypeSystem:
			role = "System"
		}

		fmt.Printf("\n[%d] %s:\n", i+1, role)
		for _, part := range msg.Parts {
			if textPart, ok := part.(llms.TextContent); ok {
				text := textPart.Text
				// Truncate long messages for readability
				if len(text) > 500 {
					fmt.Printf("%s...\n(truncated, total %d chars)\n", text[:500], len(text))
				} else {
					fmt.Println(text)
				}
			}
		}
	}

	// Extract and display final answer
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("FINAL ANSWER:")
	fmt.Println(strings.Repeat("=", 60))
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == llms.ChatMessageTypeAI {
			for _, part := range messages[i].Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					fmt.Println(textPart.Text)
				}
			}
			break
		}
	}
	fmt.Println(strings.Repeat("=", 60))
}
