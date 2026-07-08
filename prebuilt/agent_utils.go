package prebuilt

import (
	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/tooltypes"
)

// BuildToolDefinitions converts a slice of tooltypes.Tool to llmtypes.Tool definitions.
// This is a common pattern used across different agent implementations.
func BuildToolDefinitions(inputTools []tooltypes.Tool, getSchema func(tooltypes.Tool) map[string]any) []llmtypes.Tool {
	var toolDefs []llmtypes.Tool
	for _, t := range inputTools {
		toolDefs = append(toolDefs, llmtypes.Tool{
			Type: "function",
			Function: &llmtypes.FunctionDefinition{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  getSchema(t),
			},
		})
	}
	return toolDefs
}

// CreateStandardAgentSchema creates a standard map schema for agents with messages reducer.
// This is the common schema setup used by most agent implementations.
func CreateStandardAgentSchema() *graph.MapSchema {
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("messages", graph.AppendReducer)
	return agentSchema
}

// HasToolCallsInLastMessage checks if the last message in the messages slice contains tool calls.
// This is used for conditional edge routing in agent graphs.
// Returns true if any part in the last message is a ToolCall.
func HasToolCallsInLastMessage(messages []llmtypes.MessageContent) bool {
	if len(messages) == 0 {
		return false
	}
	lastMsg := messages[len(messages)-1]
	for _, part := range lastMsg.Parts {
		if _, ok := part.(llmtypes.ToolCall); ok {
			return true
		}
	}
	return false
}

// DefaultMaxIterations is the default maximum number of iterations for agent execution.
const DefaultMaxIterations = 20

// ApplyDefaultMaxIterations returns maxIterations if > 0, otherwise returns DefaultMaxIterations.
func ApplyDefaultMaxIterations(maxIterations int) int {
	if maxIterations == 0 {
		return DefaultMaxIterations
	}
	return maxIterations
}
