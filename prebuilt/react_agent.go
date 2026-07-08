package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/tmc/langchaingo/tools"
)

// CreateReactAgentMap creates a new ReAct agent graph with map[string]any state
//
// Deprecated: Use CreateAgentMap instead, which now includes the same iteration limiting functionality.
// This function is kept for backward compatibility and will be removed in a future version.
func CreateReactAgentMap(model llmtypes.Model, inputTools []tools.Tool, maxIterations int) (*graph.StateRunnable[map[string]any], error) {
	if maxIterations == 0 {
		maxIterations = 20
	}
	// Define the tool executor
	toolExecutor := NewToolExecutor(inputTools)

	// Define the graph
	workflow := graph.NewStateGraph[map[string]any]()

	// Define the state schema
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("messages", graph.AppendReducer)
	workflow.SetSchema(agentSchema)

	// Define the agent node
	workflow.AddNode("agent", "ReAct agent decision maker", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		messages, ok := state["messages"].([]llmtypes.MessageContent)
		if !ok {
			return nil, fmt.Errorf("messages key not found or invalid type")
		}

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
			return map[string]any{
				"messages": []llmtypes.MessageContent{finalMsg},
			}, nil
		}

		// Convert tools to ToolInfo for the model
		var toolDefs []llmtypes.Tool
		for _, t := range inputTools {
			toolDefs = append(toolDefs, llmtypes.Tool{
				Type: "function",
				Function: &llmtypes.FunctionDefinition{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  getToolSchema(t),
				},
			})
		}

		// Call model with tools
		resp, err := model.GenerateContent(ctx, messages, llmtypes.WithTools(toolDefs))
		if err != nil {
			return nil, err
		}

		choice := resp.Choices[0]
		aiMsg := llmtypes.MessageContent{
			Role: llmtypes.ChatMessageTypeAI,
		}
		if choice.Content != "" {
			aiMsg.Parts = append(aiMsg.Parts, llmtypes.TextPart(choice.Content))
		}
		for _, tc := range choice.ToolCalls {
			aiMsg.Parts = append(aiMsg.Parts, tc)
		}

		return map[string]any{
			"messages":        []llmtypes.MessageContent{aiMsg},
			"iteration_count": iterationCount + 1,
		}, nil
	})

	// Define the tools node
	workflow.AddNode("tools", "Tool execution node", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		messages := state["messages"].([]llmtypes.MessageContent)
		lastMsg := messages[len(messages)-1]

		if lastMsg.Role != llmtypes.ChatMessageTypeAI {
			return nil, fmt.Errorf("last message is not an AI message")
		}

		var toolMessages []llmtypes.MessageContent
		for _, part := range lastMsg.Parts {
			if tc, ok := part.(llmtypes.ToolCall); ok {
				// Get the tool to check if it has a custom schema
				tool, hasTool := toolExecutor.Tools[tc.FunctionCall.Name]

				var inputVal string
				if hasTool {
					// Check if tool has custom schema
					if _, hasCustomSchema := tool.(ToolWithSchema); hasCustomSchema {
						// Tool has custom schema, pass JSON arguments directly
						inputVal = tc.FunctionCall.Arguments
					} else {
						// Tool uses default schema, try to extract "input" field
						var args map[string]any
						_ = json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)
						if val, ok := args["input"].(string); ok {
							inputVal = val
						} else {
							inputVal = tc.FunctionCall.Arguments
						}
					}
				} else {
					// Tool not found, use arguments as-is
					inputVal = tc.FunctionCall.Arguments
				}

				res, err := toolExecutor.Execute(ctx, ToolInvocation{
					Tool:      tc.FunctionCall.Name,
					ToolInput: inputVal,
				})
				if err != nil {
					res = fmt.Sprintf("Error: %v", err)
				}

				toolMsg := llmtypes.MessageContent{
					Role: llmtypes.ChatMessageTypeTool,
					Parts: []llmtypes.ContentPart{
						llmtypes.ToolCallResponse{
							ToolCallID: tc.ID,
							Name:       tc.FunctionCall.Name,
							Content:    res,
						},
					},
				}
				toolMessages = append(toolMessages, toolMsg)
			}
		}

		return map[string]any{
			"messages": toolMessages,
		}, nil
	})

	workflow.SetEntryPoint("agent")
	workflow.AddConditionalEdge("agent", func(ctx context.Context, state map[string]any) string {
		messages := state["messages"].([]llmtypes.MessageContent)
		lastMsg := messages[len(messages)-1]
		for _, part := range lastMsg.Parts {
			if _, ok := part.(llmtypes.ToolCall); ok {
				return "tools"
			}
		}
		return graph.END
	})
	workflow.AddEdge("tools", "agent")

	return workflow.Compile()
}

// CreateReactAgent creates a new typed ReAct agent graph
func CreateReactAgent[S any](
	model llmtypes.Model,
	inputTools []tools.Tool,
	getMessages func(S) []llmtypes.MessageContent,
	setMessages func(S, []llmtypes.MessageContent) S,
	getIterationCount func(S) int,
	setIterationCount func(S, int) S,
	maxIterations int,
) (*graph.StateRunnable[S], error) {
	if maxIterations == 0 {
		maxIterations = 20
	}
	toolExecutor := NewToolExecutor(inputTools)
	workflow := graph.NewStateGraph[S]()

	workflow.AddNode("agent", "ReAct agent decision maker", func(ctx context.Context, state S) (S, error) {
		iterationCount := getIterationCount(state)
		if iterationCount >= maxIterations {
			finalMsg := llmtypes.MessageContent{
				Role: llmtypes.ChatMessageTypeAI,
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("Maximum iterations reached. Please try a simpler query."),
				},
			}
			return setMessages(state, append(getMessages(state), finalMsg)), nil
		}

		var toolDefs []llmtypes.Tool
		for _, t := range inputTools {
			toolDefs = append(toolDefs, llmtypes.Tool{
				Type: "function",
				Function: &llmtypes.FunctionDefinition{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  getToolSchema(t),
				},
			})
		}

		messages := getMessages(state)
		resp, err := model.GenerateContent(ctx, messages, llmtypes.WithTools(toolDefs))
		if err != nil {
			return state, err
		}

		choice := resp.Choices[0]
		aiMsg := llmtypes.MessageContent{
			Role: llmtypes.ChatMessageTypeAI,
		}
		if choice.Content != "" {
			aiMsg.Parts = append(aiMsg.Parts, llmtypes.TextPart(choice.Content))
		}
		for _, tc := range choice.ToolCalls {
			aiMsg.Parts = append(aiMsg.Parts, tc)
		}

		state = setMessages(state, append(messages, aiMsg))
		state = setIterationCount(state, iterationCount+1)
		return state, nil
	})

	workflow.AddNode("tools", "Tool execution node", func(ctx context.Context, state S) (S, error) {
		messages := getMessages(state)
		lastMsg := messages[len(messages)-1]

		var toolMessages []llmtypes.MessageContent
		for _, part := range lastMsg.Parts {
			if tc, ok := part.(llmtypes.ToolCall); ok {
				// Get the tool to check if it has a custom schema
				tool, hasTool := toolExecutor.Tools[tc.FunctionCall.Name]

				var inputVal string
				if hasTool {
					// Check if tool has custom schema
					if _, hasCustomSchema := tool.(ToolWithSchema); hasCustomSchema {
						// Tool has custom schema, pass JSON arguments directly
						inputVal = tc.FunctionCall.Arguments
					} else {
						// Tool uses default schema, try to extract "input" field
						var args map[string]any
						_ = json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)
						if val, ok := args["input"].(string); ok {
							inputVal = val
						} else {
							inputVal = tc.FunctionCall.Arguments
						}
					}
				} else {
					// Tool not found, use arguments as-is
					inputVal = tc.FunctionCall.Arguments
				}

				res, err := toolExecutor.Execute(ctx, ToolInvocation{
					Tool:      tc.FunctionCall.Name,
					ToolInput: inputVal,
				})
				if err != nil {
					res = fmt.Sprintf("Error: %v", err)
				}

				toolMsg := llmtypes.MessageContent{
					Role: llmtypes.ChatMessageTypeTool,
					Parts: []llmtypes.ContentPart{
						llmtypes.ToolCallResponse{
							ToolCallID: tc.ID,
							Name:       tc.FunctionCall.Name,
							Content:    res,
						},
					},
				}
				toolMessages = append(toolMessages, toolMsg)
			}
		}

		return setMessages(state, append(getMessages(state), toolMessages...)), nil
	})

	workflow.SetEntryPoint("agent")
	workflow.AddConditionalEdge("agent", func(ctx context.Context, state S) string {
		messages := getMessages(state)
		lastMsg := messages[len(messages)-1]
		for _, part := range lastMsg.Parts {
			if _, ok := part.(llmtypes.ToolCall); ok {
				return "tools"
			}
		}
		return graph.END
	})
	workflow.AddEdge("tools", "agent")

	return workflow.Compile()
}
