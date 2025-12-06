package ptc

import (
	"context"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// PTCAgentConfig configures a PTC agent
type PTCAgentConfig struct {
	// Model is the LLM to use
	Model llms.Model

	// Tools are the available tools
	Tools []tools.Tool

	// Language is the execution language for code
	Language ExecutionLanguage

	// ExecutionMode determines how tools are executed (default: ModeServer)
	// - ModeServer: Tools are executed via HTTP server (recommended, fully implemented)
	// - ModeDirect: Tools are executed directly via subprocess (experimental, placeholder implementation)
	ExecutionMode ExecutionMode

	// SystemPrompt is the system prompt for the agent
	SystemPrompt string

	// MaxIterations is the maximum number of iterations (default: 10)
	MaxIterations int
}

// CreatePTCAgent creates a new agent that uses programmatic tool calling
// This agent generates code to call tools programmatically rather than
// using traditional tool calling with round-trips
func CreatePTCAgent(config PTCAgentConfig) (*graph.Runnable, error) {
	if config.Model == nil {
		return nil, fmt.Errorf("model is required")
	}

	if len(config.Tools) == 0 {
		return nil, fmt.Errorf("at least one tool is required")
	}

	if config.Language == "" {
		config.Language = LanguagePython // Default to Python
	}

	if config.ExecutionMode == "" {
		config.ExecutionMode = ModeServer // Default to server mode (more reliable)
	}

	if config.MaxIterations == 0 {
		config.MaxIterations = 10
	}

	// Create PTC tool node with execution mode
	ptcNode := NewPTCToolNodeWithMode(config.Language, config.Tools, config.ExecutionMode)

	// Start the tool server
	if err := ptcNode.Executor.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to start tool server: %w", err)
	}

	// Build system prompt with tool definitions
	systemPrompt := buildSystemPrompt(config.SystemPrompt, config.Language, ptcNode.Executor)

	// Create the graph
	workflow := graph.NewMessageGraph()

	// Add agent node
	workflow.AddNode("agent", "LLM agent that generates code for tool calling", func(ctx context.Context, state interface{}) (interface{}, error) {
		return agentNode(ctx, state, config.Model, systemPrompt, config.MaxIterations)
	})

	// Add PTC execution node
	workflow.AddNode("execute_code", "Executes generated code with tool access", func(ctx context.Context, state interface{}) (interface{}, error) {
		return ptcNode.Invoke(ctx, state)
	})

	// Set entry point
	workflow.SetEntryPoint("agent")

	// Add conditional routing
	workflow.AddConditionalEdge("agent", func(ctx context.Context, state interface{}) string {
		mState := state.(map[string]interface{})
		messages := mState["messages"].([]llms.MessageContent)

		if len(messages) == 0 {
			return graph.END
		}

		lastMsg := messages[len(messages)-1]

		// Check if the message contains code to execute
		if lastMsg.Role == llms.ChatMessageTypeAI && containsCode(lastMsg) {
			return "execute_code"
		}

		// Otherwise, we're done
		return graph.END
	})

	// Add edge from execute_code back to agent
	workflow.AddEdge("execute_code", "agent")

	// Compile the graph
	app, err := workflow.Compile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile graph: %w", err)
	}

	return app, nil
}

// agentNode is the main agent logic node
func agentNode(ctx context.Context, state interface{}, model llms.Model, systemPrompt string, maxIterations int) (interface{}, error) {
	mState := state.(map[string]interface{})
	messages := mState["messages"].([]llms.MessageContent)

	// Check iteration count
	iterationCount := 0
	if count, ok := mState["iteration_count"].(int); ok {
		iterationCount = count
	}

	if iterationCount >= maxIterations {
		// Max iterations reached, return final message
		finalMsg := llms.MessageContent{
			Role: llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{
				llms.TextPart("Maximum iterations reached. Please try a simpler query."),
			},
		}
		mState["messages"] = append(messages, finalMsg)
		return mState, nil
	}

	// Increment iteration count
	mState["iteration_count"] = iterationCount + 1

	// Prepend system message if not already present
	if len(messages) == 0 || messages[0].Role != llms.ChatMessageTypeSystem {
		messages = append([]llms.MessageContent{
			{
				Role: llms.ChatMessageTypeSystem,
				Parts: []llms.ContentPart{
					llms.TextPart(systemPrompt),
				},
			},
		}, messages...)
	}

	// Call the model
	resp, err := model.GenerateContent(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// Extract response
	var responseContent []llms.ContentPart
	for _, choice := range resp.Choices {
		if choice.Content != "" {
			responseContent = append(responseContent, llms.TextPart(choice.Content))
		}
	}

	if len(responseContent) == 0 {
		return nil, fmt.Errorf("empty response from model")
	}

	// Add AI response to messages
	aiMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: responseContent,
	}
	mState["messages"] = append(mState["messages"].([]llms.MessageContent), aiMsg)

	return mState, nil
}

// buildSystemPrompt builds the system prompt with tool definitions
func buildSystemPrompt(userPrompt string, language ExecutionLanguage, executor *CodeExecutor) string {
	toolDefs := executor.GetToolDefinitions()

	langName := "Python"
	if language == LanguageGo {
		langName = "Go"
	}

	basePrompt := fmt.Sprintf(`You are an AI assistant that can write %s code to solve problems using available tools.

When you need to use tools to answer a question, write %s code that calls the tools programmatically.
The code you write will be executed in a secure environment with access to all the tools.

%s

IMPORTANT GUIDELINES:
1. Write complete, executable %s code
2. Use the tool functions provided above to call tools
3. Process and filter data programmatically to extract only relevant information
4. Print the final result to stdout
5. Handle errors gracefully
6. When you have the final answer, respond with just the answer (no code)

Format your code in markdown code blocks:
`+"```"+langName+`
# Your code here
`+"```", langName, langName, toolDefs, langName)

	if userPrompt != "" {
		return userPrompt + "\n\n" + basePrompt
	}

	return basePrompt
}

// containsCode checks if a message contains code to execute
func containsCode(msg llms.MessageContent) bool {
	for _, part := range msg.Parts {
		if textPart, ok := part.(llms.TextContent); ok {
			text := textPart.Text
			// Check for code blocks (case-insensitive)
			textLower := strings.ToLower(text)
			if len(text) > 6 && (contains(textLower, "```python") || contains(textLower, "```go")) {
				return true
			}
		}
	}
	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
