// Package ptc (Programmatic Tool Calling) provides advanced tool execution capabilities for LangGraph Go agents.
//
// This package implements a novel approach to tool calling where agents generate code to use tools
// programmatically, rather than using traditional function calling APIs. This enables more flexible,
// composable, and powerful tool usage patterns.
//
// # Core Concepts
//
// ## Programmatic Tool Calling (PTC)
// Instead of the agent making individual tool calls through a structured API, PTC agents generate
// code that imports and uses tools directly. This approach offers several advantages:
//
//   - More natural tool composition in code
//   - Ability to use control flow (loops, conditionals) with tools
//   - Easier debugging and inspection
//   - No need for complex tool schemas
//   - Better performance for multi-tool operations
//
// ## Supported Languages
//
// The package currently supports:
//
//   - Python (LanguagePython): Full Python runtime with standard library
//   - JavaScript (LanguageJavaScript): Node.js runtime execution
//   - Shell (LanguageShell): Bash shell command execution
//
// # Key Components
//
// ## PTCAgent
// The main agent implementation that generates and executes tool-calling code:
//
//	import (
//		"github.com/smallnest/langgraphgo/ptc"
//		"github.com/smallnest/langgraphgo/llms"
//		"github.com/smallnest/langgraphgo/tool"
//	)
//
//	// Create agent with Python execution
//	agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
//		Model:       llm,
//		Tools:       []tool.Tool{calculator, weatherTool},
//		Language:    ptc.LanguagePython,
//		MaxIterations: 10,
//	})
//
// ## Execution Modes
//
// Two execution modes are available:
//
//   - ModeDirect: Execute code in subprocess (default)
//   - ModeServer: Execute code via HTTP server (for sandboxing)
//
// ## PTCToolNode
// A graph node that handles the execution of generated code:
//
//	node := ptc.NewPTCToolNodeWithMode(
//		ptc.LanguagePython,
//		toolList,
//		ptc.ModeDirect,
//	)
//
// # Example Usage
//
// ## Basic Agent
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//
//		"github.com/smallnest/langgraphgo/ptc"
//		"github.com/tmc/langchaingo/llms/openai"
//		"github.com/smallnest/langgraphgo/tool"
//		"github.com/tmc/langchaingo/tools" // langchaingo tools satisfy tool.Tool
//	)
//
//	func main() {
//		// Initialize LLM
//		llm, _ := openai.New()
//
//		// Create a calculator tool
//		calculator := &tools.CalculatorTool{}
//
//		// Create PTC agent
//		agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
//			Model:    llm,
//			Tools:    []tool.Tool{calculator},
//			Language: ptc.LanguagePython,
//		})
//		if err != nil {
//			panic(err)
//		}
//
//		// Execute agent
//		ctx := context.Background()
//		result, err := agent.Invoke(ctx, map[string]any{
//			"messages": []llms.MessageContent{
//				{
//					Role: llms.ChatMessageTypeHuman,
//					Parts: []llms.ContentPart{
//						llms.TextPart("What is 123 * 456?"),
//					},
//				},
//			},
//		})
//
//		if err != nil {
//			panic(err)
//		}
//
//		fmt.Printf("Result: %v\n", result)
//	}
//
// ## Custom Tools
//
//	type WeatherTool struct{}
//
//	func (t *WeatherTool) Name() string { return "get_weather" }
//	func (t *WeatherTool) Description() string {
//		return "Get current weather for a city"
//	}
//
//	func (t *WeatherTool) Call(ctx context.Context, input string) (string, error) {
//		// Implementation
//		return "The weather in London is 15°C and sunny", nil
//	}
//
//	// Use with PTC agent
//	weather := &WeatherTool{}
//	agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
//		Model:    llm,
//		Tools:    []tool.Tool{weather},
//		Language: ptc.LanguageJavaScript,
//	})
//
// ## Server Mode Execution
//
//	// Start tool server for sandboxed execution
//	server := ptc.NewToolServer(8080)
//	go server.Start()
//	defer server.Stop()
//
//	agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
//		Model:         llm,
//		Tools:         toolList,
//		Language:      ptc.LanguagePython,
//		ExecutionMode: ptc.ModeServer,
//		ServerURL:     "http://localhost:8080",
//	})
//
// # Advanced Features
//
// ## Code Generation
// The agent generates code like this:
//
//	```python
//	import json
//
//	# Tool imports are automatically added
//	from tools import calculator, weather
//
//	# User query: "Calculate 15% tip on $100 bill"
//	bill_amount = 100
//	tip_rate = 0.15
//	tip = calculator.multiply(bill_amount, tip_rate)
//
//	result = {
//		"bill_amount": bill_amount,
//		"tip_rate": tip_rate,
//		"tip_amount": tip,
//		"total": bill_amount + tip
//	}
//
//	print(json.dumps(result))
//	```
//
// ## Error Handling
// The system includes comprehensive error handling:
//
//   - Syntax errors in generated code
//   - Runtime errors during execution
//   - Tool execution failures
//   - Timeout protection
//   - Resource usage limits
//
// # Security Considerations
//
//   - Use server mode for isolation
//   - Set appropriate timeouts
//   - Monitor resource usage
//   - Validate tool inputs/outputs
//   - Consider sandboxing for untrusted code
//
// # Performance
//
//   - Code execution is generally faster than multiple tool calls
//   - Consider caching for repeated operations
//   - Monitor execution time for long-running operations
//   - Use streaming for real-time feedback
//
// # Integration with LangGraph
//
// The PTC agent integrates seamlessly with LangGraph:
//
//	g := graph.NewStateGraph[map[string]any]()
//
//	// Add PTC node
//	ptcNode := ptc.NewPTCToolNode(ptc.LanguagePython, tools)
//	g.AddNode("tools", ptcNode.Invoke)
//
//	// Add LLM node for reasoning
//	g.AddNode("reason", llmNode)
//
//	// Define execution flow
//	g.SetEntry("reason")
//	g.AddEdge("reason", "tools")
//	g.AddConditionalEdge("tools", shouldContinue, "continue", "end")
//
//	// Compile and run
//	runnable := g.Compile()
//	result, _ := runnable.Invoke(ctx, initialState)
//
// # Best Practices
//
//  1. Choose appropriate execution language based on your tools
//  2. Use ModeServer for production environments
//  3. Set reasonable iteration limits
//  4. Provide clear tool descriptions
//  5. Handle errors gracefully in your tools
//  6. Test with various input patterns
//  7. Monitor execution for resource usage
//  8. Use timeouts for long-running operations
//
// # Limitations
//
//   - Requires runtime environment for chosen language
//   - Generated code might have bugs
//   - Debugging generated code can be challenging
//   - Security risks with unrestricted code execution
package ptc
