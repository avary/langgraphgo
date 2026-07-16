package main

import (
	"context"
	"fmt"

	_ "github.com/smallnest/langgraphgo/llms"
	_ "github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tool"
)

// SimpleCounterTool is a basic tool that counts
type SimpleCounterTool struct {
	count int
}

func (t *SimpleCounterTool) Name() string {
	return "counter"
}

func (t *SimpleCounterTool) Description() string {
	return "A simple counter tool. Input should be the number to add to the counter."
}

func (t *SimpleCounterTool) Call(ctx context.Context, input string) (string, error) {
	// For simplicity, just increment
	t.count++
	return fmt.Sprintf("Counter is now: %d", t.count), nil
}

// EchoTool echoes back the input
type EchoTool struct{}

func (t *EchoTool) Name() string {
	return "echo"
}

func (t *EchoTool) Description() string {
	return "Echoes back the input text"
}

func (t *EchoTool) Call(ctx context.Context, input string) (string, error) {
	return fmt.Sprintf("Echo: %s", input), nil
}

func main() {
	fmt.Println("🤖 Typed ReAct Agent Example")
	fmt.Println("==========================")

	// Create tools
	tools := []tool.Tool{
		&SimpleCounterTool{},
		&EchoTool{},
	}

	// Note: You would typically use a real LLM here
	// For this example, we'll just show the structure
	fmt.Println("\nAvailable tools:")
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name(), tool.Description())
	}

	// Example of creating a typed ReAct agent
	fmt.Println("\n💡 Creating Typed ReAct Agent...")
	fmt.Println("```go")
	fmt.Println("// Define your state type")
	fmt.Println("type AgentState struct {")
	fmt.Println("    Messages []llms.MessageContent")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// Create the agent")
	fmt.Println("agent, err := prebuilt.CreateReactAgentTyped[model, []tool.Tool]()")
	fmt.Println("```")

	// Example with custom state
	fmt.Println("\n💡 Creating ReAct Agent with Custom State...")
	fmt.Println("```go")
	fmt.Println("// Define custom state")
	fmt.Println("type CustomState struct {")
	fmt.Println("    Messages []llms.MessageContent")
	fmt.Println("    StepCount int")
	fmt.Println("    ToolUseCount map[string]int")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// Create agent with custom state")
	fmt.Println("agent, err := prebuilt.CreateReactAgentWithCustomStateTyped[CustomState] (")
	fmt.Println("    model,")
	fmt.Println("    tools,")
	fmt.Println("    func(s CustomState) []llms.MessageContent { return s.Messages },")
	fmt.Println("    func(s CustomState, msgs []llms.MessageContent) CustomState {")
	fmt.Println("        s.Messages = msgs")
	fmt.Println("        s.StepCount++")
	fmt.Println("        return s")
	fmt.Println("    },")
	fmt.Println("    func(msgs []llms.MessageContent) bool { /* check for tool calls */ },")
	fmt.Println(")")
	fmt.Println("```")

	// Demonstrate type safety benefits
	fmt.Println("\n✨ Benefits of Typed ReAct Agent:")
	fmt.Println("1. Compile-time type safety - no runtime type assertions needed!")
	fmt.Println("2. Better IDE support with full autocomplete")
	fmt.Println("3. Self-documenting code with explicit state types")
	fmt.Println("4. Custom state with additional fields (counters, metadata, etc.)")
	fmt.Println("5. Type-safe tool integration")

	// Example of how you would use it
	fmt.Println("\n📝 Example Usage Pattern:")
	fmt.Println("```go")
	fmt.Println("// Initial state")
	fmt.Println("state := ReactAgentState{")
	fmt.Println("    Messages: []llms.MessageContent{")
	fmt.Println("        llms.TextParts(llms.ChatMessageTypeHuman, \"What's 2+2?\"),")
	fmt.Println("    },")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// Execute agent")
	fmt.Println("result, err := agent.Invoke(ctx, state)")
	fmt.Println("")
	fmt.Println("// Result is fully typed!")
	fmt.Println("fmt.Printf(\"Final messages: %v\\n\", result.Messages)")
	fmt.Println("```")

	fmt.Println("\n✅ Example completed successfully!")
	fmt.Println("\nNote: To run with a real LLM, replace the model parameter")
	fmt.Println("      with an actual LLM instance (e.g., OpenAI, Anthropic, etc.)")
}
