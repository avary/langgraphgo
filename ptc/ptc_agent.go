package ptc

import (
	"context"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/tooltypes"
)

// PTCAgentConfig configures a PTC agent
type PTCAgentConfig struct {
	// Model is the LLM to use
	Model llmtypes.Model

	// Tools are the available tools
	Tools []tooltypes.Tool

	// Language is the execution language for code
	Language ExecutionLanguage

	// ExecutionMode determines how tools are executed (default: ModeDirect)
	// - ModeDirect: Tools are executed directly via subprocess (default)
	// - ModeServer: Tools are executed via HTTP server (alternative)
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
		config.ExecutionMode = ModeDirect // Default to direct mode
	}

	if config.MaxIterations == 0 {
		config.MaxIterations = 20
	}

	// Create PTC tool node with execution mode
	ptcNode := NewPTCToolNodeWithMode(config.Language, config.Tools, config.ExecutionMode)

	// Start the tool server
	if err := ptcNode.Executor.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to start tool server: %w", err)
	}

	// Build system prompt with tool definitions
	systemPrompt := BuildSystemPrompt(config.SystemPrompt, config.Language, ptcNode.Executor)

	// Create the graph
	workflow := graph.NewStateGraph[map[string]any]()

	// Define the state schema
	schema := graph.NewMapSchema()
	schema.RegisterReducer("messages", graph.AddMessages)
	workflow.SetSchema(schema)

	// Add agent node
	workflow.AddNode("agent", "LLM agent that generates code for tool calling", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return agentNode(ctx, state, config.Model, systemPrompt, config.MaxIterations)
	})

	// Add PTC execution node
	workflow.AddNode("execute_code", "Executes generated code with tool access", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		result, err := ptcNode.Invoke(ctx, state)
		if err != nil {
			return nil, err
		}
		if resultMap, ok := result.(map[string]any); ok {
			return resultMap, nil
		}
		return state, nil
	})

	// Set entry point
	workflow.SetEntryPoint("agent")

	// Add conditional routing
	workflow.AddConditionalEdge("agent", func(ctx context.Context, state map[string]any) string {
		messages := state["messages"].([]llmtypes.MessageContent)

		if len(messages) == 0 {
			return graph.END
		}

		lastMsg := messages[len(messages)-1]

		// Check if the message contains code to execute
		if lastMsg.Role == llmtypes.ChatMessageTypeAI && ContainsCode(lastMsg) {
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
func agentNode(ctx context.Context, state map[string]any, model llmtypes.Model, systemPrompt string, maxIterations int) (map[string]any, error) {
	messages := state["messages"].([]llmtypes.MessageContent)

	// Check iteration count
	iterationCount := 0
	if count, ok := state["iteration_count"].(int); ok {
		iterationCount = count
	}

	if iterationCount >= maxIterations {
		// Max iterations reached, return final message
		finalMsg := llmtypes.MessageContent{
			Role: llmtypes.ChatMessageTypeAI,
			Parts: []llmtypes.ContentPart{
				llmtypes.TextPart("Maximum iterations reached. Please try a simpler query."),
			},
		}
		state["messages"] = []llmtypes.MessageContent{finalMsg}
		return state, nil
	}

	// Increment iteration count
	state["iteration_count"] = iterationCount + 1

	// Prepend system message if not already present
	if len(messages) == 0 || messages[0].Role != llmtypes.ChatMessageTypeSystem {
		messages = append([]llmtypes.MessageContent{
			{
				Role: llmtypes.ChatMessageTypeSystem,
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart(systemPrompt),
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
	var responseContent []llmtypes.ContentPart
	for _, choice := range resp.Choices {
		if choice.Content != "" {
			responseContent = append(responseContent, llmtypes.TextPart(choice.Content))
		}
	}

	if len(responseContent) == 0 {
		return nil, fmt.Errorf("empty response from model")
	}

	// Add AI response to messages
	aiMsg := llmtypes.MessageContent{
		Role:  llmtypes.ChatMessageTypeAI,
		Parts: responseContent,
	}
	state["messages"] = append(state["messages"].([]llmtypes.MessageContent), aiMsg)

	return state, nil
}

// buildSystemPrompt builds the system prompt with tool definitions
// BuildSystemPrompt builds the system prompt with tool definitions
func BuildSystemPrompt(userPrompt string, language ExecutionLanguage, executor *CodeExecutor) string {
	toolDefs := executor.GetToolDefinitions()

	langName := "Python"
	if language == LanguageGo {
		langName = "Go"
	}
	langNameLower := strings.ToLower(langName)

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
`+"```"+langNameLower+`
# Your code here
`+"```", langName, langName, toolDefs, langName)

	if userPrompt != "" {
		return userPrompt + "\n\n" + basePrompt
	}

	return basePrompt
}

// containsCode checks if a message contains code to execute
// ContainsCode checks if a message contains code to execute
func ContainsCode(msg llmtypes.MessageContent) bool {
	for _, part := range msg.Parts {
		if textPart, ok := part.(llmtypes.TextContent); ok {
			text := textPart.Text
			textLower := strings.ToLower(text)
			if (strings.Contains(textLower, "```python") || strings.Contains(textLower, "```go")) && strings.Count(textLower, "```") >= 2 {
				return true
			}
		}
	}
	return false
}
