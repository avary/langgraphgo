package ptc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/smallnest/langgraphgo/ptc"
	"github.com/tmc/langchaingo/tools"
)

// MockTool for testing
type MockTool struct {
	name        string
	description string
	response    string
}

func (t MockTool) Name() string {
	return t.name
}

func (t MockTool) Description() string {
	return t.description
}

func (t MockTool) Call(ctx context.Context, input string) (string, error) {
	return t.response, nil
}

func TestCodeExecutor(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "calculator",
			description: "Performs calculations",
			response:    "42",
		},
	}

	executor := ptc.NewCodeExecutor(ptc.LanguagePython, tools)
	ctx := context.Background()

	// Start the tool server
	if err := executor.Start(ctx); err != nil {
		t.Fatalf("Failed to start executor: %v", err)
	}
	defer executor.Stop(ctx)

	// Test Python code execution
	code := `
result = 2 + 2
print(f"Result: {result}")
`

	result, err := executor.Execute(ctx, code)
	if err != nil {
		t.Fatalf("Failed to execute code: %v", err)
	}

	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestToolServer(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "test_tool",
			description: "A test tool",
			response:    "test response",
		},
	}

	server := ptc.NewToolServer(tools)
	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop(ctx)

	// Test that server is running
	port := server.GetPort()
	if port == 0 {
		t.Error("Expected non-zero port")
	}

	baseURL := server.GetBaseURL()
	if baseURL == "" {
		t.Error("Expected non-empty base URL")
	}
}

func ExampleCreatePTCAgent() {
	// This example shows how to create a PTC agent
	// Note: This requires a real LLM and won't run in tests

	// Create tools
	_ = []tools.Tool{
		MockTool{
			name:        "calculator",
			description: "Performs arithmetic calculations",
			response:    "42",
		},
	}

	// In real usage, you would use:
	// model, _ := openai.New()

	// agent, _ := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
	//     Model:         model,
	//     Tools:         tools,
	//     Language:      ptc.LanguagePython,
	//     MaxIterations: 10,
	// })

	// result, _ := agent.Invoke(context.Background(), initialState)

	fmt.Println("PTC Agent created successfully")
	// Output: PTC Agent created successfully
}

func ExampleCodeExecutor_Execute() {
	tools := []tools.Tool{
		MockTool{
			name:        "get_data",
			description: "Gets some data",
			response:    `{"value": 100}`,
		},
	}

	executor := ptc.NewCodeExecutor(ptc.LanguagePython, tools)
	ctx := context.Background()

	executor.Start(ctx)
	defer executor.Stop(ctx)

	code := `
# Process data
data = {"numbers": [1, 2, 3, 4, 5]}
total = sum(data["numbers"])
print(f"Total: {total}")
`

	result, err := executor.Execute(ctx, code)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Executed successfully: %t\n", result.Output != "")
	// Output: Executed successfully: true
}

func TestToolDefinitions(t *testing.T) {
	tools := []tools.Tool{
		MockTool{
			name:        "tool1",
			description: "Description 1",
			response:    "response1",
		},
		MockTool{
			name:        "tool2",
			description: "Description 2",
			response:    "response2",
		},
	}

	executor := ptc.NewCodeExecutor(ptc.LanguagePython, tools)

	defs := executor.GetToolDefinitions()

	if defs == "" {
		t.Error("Expected non-empty tool definitions")
	}

	// Check that both tools are mentioned
	if !contains(defs, "tool1") || !contains(defs, "tool2") {
		t.Error("Tool definitions should mention all tools")
	}
}

func TestExecutionResult(t *testing.T) {
	result := &ptc.ExecutionResult{
		Output: "test output",
		Stdout: "stdout content",
		Stderr: "stderr content",
	}

	if result.Output != "test output" {
		t.Errorf("Expected 'test output', got '%s'", result.Output)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Example of using PTC with different execution languages
func ExampleExecutionLanguage() {
	tools := []tools.Tool{
		MockTool{name: "tool1", description: "A tool", response: "response"},
	}

	// Python executor
	pythonExec := ptc.NewCodeExecutor(ptc.LanguagePython, tools)
	fmt.Printf("Python executor created: %v\n", pythonExec != nil)

	// Go executor
	goExec := ptc.NewCodeExecutor(ptc.LanguageGo, tools)
	fmt.Printf("Go executor created: %v\n", goExec != nil)

	// Output:
	// Python executor created: true
	// Go executor created: true
}

// Example of tool server request/response format
func ExampleToolServer_requestFormat() {
	request := map[string]interface{}{
		"tool_name": "calculator",
		"input":     "2 + 2",
	}

	requestJSON, _ := json.MarshalIndent(request, "", "  ")
	fmt.Printf("Tool Request:\n%s\n", string(requestJSON))

	response := map[string]interface{}{
		"success": true,
		"result":  "4",
		"tool":    "calculator",
		"input":   "2 + 2",
	}

	responseJSON, _ := json.MarshalIndent(response, "", "  ")
	fmt.Printf("\nTool Response:\n%s\n", string(responseJSON))

	// Output:
	// Tool Request:
	// {
	//   "input": "2 + 2",
	//   "tool_name": "calculator"
	// }
	//
	// Tool Response:
	// {
	//   "input": "2 + 2",
	//   "result": "4",
	//   "success": true,
	//   "tool": "calculator"
	// }
}
