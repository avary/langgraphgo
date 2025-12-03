package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// CreateAgentOptions contains options for creating an agent
type CreateAgentOptions struct {
	skillDir      string
	SystemMessage string
	StateModifier func(messages []llms.MessageContent) []llms.MessageContent
	Checkpointer  graph.CheckpointStore
}

// CreateAgentOption is a function that configures CreateAgentOptions
type CreateAgentOption func(*CreateAgentOptions)

// WithSystemMessage sets the system message for the agent
func WithSystemMessage(message string) CreateAgentOption {
	return func(o *CreateAgentOptions) {
		o.SystemMessage = message
	}
}

// WithStateModifier sets a function to modify messages before they are sent to the model
func WithStateModifier(modifier func(messages []llms.MessageContent) []llms.MessageContent) CreateAgentOption {
	return func(o *CreateAgentOptions) {
		o.StateModifier = modifier
	}
}

// WithCheckpointer sets the checkpointer for the agent
// Note: Currently this is a placeholder and may not be fully integrated into the graph execution yet
func WithCheckpointer(checkpointer graph.CheckpointStore) CreateAgentOption {
	return func(o *CreateAgentOptions) {
		o.Checkpointer = checkpointer
	}
}

// WithSkillDir sets the skill directory for the agent
func WithSkillDir(skillDir string) CreateAgentOption {
	return func(o *CreateAgentOptions) {
		o.skillDir = skillDir
	}
}

// CreateAgent creates a new agent graph with options
func CreateAgent(model llms.Model, inputTools []tools.Tool, opts ...CreateAgentOption) (*graph.StateRunnable, error) {
	options := &CreateAgentOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Define the tool executor
	toolExecutor := NewToolExecutor(inputTools)

	// Define the graph
	workflow := graph.NewStateGraph()

	// Define the state schema
	// We use a MapSchema with AppendReducer for messages
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("messages", graph.AppendReducer)
	workflow.SetSchema(agentSchema)

	// Define the agent node
	workflow.AddNode("agent", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState, ok := state.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid state type: %T", state)
		}

		messages, ok := mState["messages"].([]llms.MessageContent)
		if !ok {
			return nil, fmt.Errorf("messages key not found or invalid type")
		}

		// Convert tools to ToolInfo for the model
		var toolDefs []llms.Tool
		for _, t := range inputTools {
			toolDefs = append(toolDefs, llms.Tool{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"input": map[string]interface{}{
								"type":        "string",
								"description": "The input query for the tool",
							},
						},
						"required":             []string{"input"},
						"additionalProperties": false,
					},
				},
			})
		}

		// We need to pass tools to the model
		callOpts := []llms.CallOption{
			llms.WithTools(toolDefs),
		}

		// Apply StateModifier if provided
		msgsToSend := messages

		// Prepend system message if provided (and not handled by StateModifier)
		// If StateModifier is provided, it's responsible for the whole message list structure,
		// but usually SystemMessage is separate.
		// LangChain logic: SystemMessage is prepended. StateModifier can modify everything.
		// Let's prepend SystemMessage first, then apply StateModifier?
		// Or apply StateModifier to the raw history, then prepend SystemMessage?
		// LangChain docs say: "This is useful for doing things like... removing the system message"
		// So StateModifier should probably run AFTER SystemMessage is added?
		// But if SystemMessage is just a string, we construct it here.
		// Let's construct SystemMessage first.

		if options.SystemMessage != "" {
			sysMsg := llms.TextParts(llms.ChatMessageTypeSystem, options.SystemMessage)
			// Check if the first message is already a system message?
			// For simplicity, just prepend.
			msgsToSend = append([]llms.MessageContent{sysMsg}, msgsToSend...)
		}

		// Now apply StateModifier if it exists
		// Wait, if StateModifier is used to REMOVE system message, it must run AFTER.
		// But if it's used to filter history, it might run BEFORE.
		// LangChain `create_react_agent` source:
		// 1. `_modify_state` runs on input state.
		// 2. `system_message` is added.
		// Actually, `create_agent` in LangChain 0.2+ might be different.
		// Let's stick to: SystemMessage is added to the front. StateModifier sees the result.
		if options.StateModifier != nil {
			msgsToSend = options.StateModifier(msgsToSend)
		}

		resp, err := model.GenerateContent(ctx, msgsToSend, callOpts...)
		if err != nil {
			return nil, err
		}

		choice := resp.Choices[0]

		// Create AIMessage
		aiMsg := llms.MessageContent{
			Role: llms.ChatMessageTypeAI,
		}

		if choice.Content != "" {
			aiMsg.Parts = append(aiMsg.Parts, llms.TextPart(choice.Content))
		}

		// Handle tool calls
		if len(choice.ToolCalls) > 0 {
			for _, tc := range choice.ToolCalls {
				// ToolCall implements ContentPart
				aiMsg.Parts = append(aiMsg.Parts, tc)
			}
		}

		return map[string]interface{}{
			"messages": []llms.MessageContent{aiMsg},
		}, nil
	})

	// Define the tools node
	workflow.AddNode("tools", func(ctx context.Context, state interface{}) (interface{}, error) {
		mState, ok := state.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid state")
		}

		messages := mState["messages"].([]llms.MessageContent)
		lastMsg := messages[len(messages)-1]

		if lastMsg.Role != llms.ChatMessageTypeAI {
			return nil, fmt.Errorf("last message is not an AI message")
		}

		var toolMessages []llms.MessageContent

		for _, part := range lastMsg.Parts {
			if tc, ok := part.(llms.ToolCall); ok {
				// Parse arguments to get input
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
					// If unmarshal fails, try to use the raw string if it's not JSON object
				}

				inputVal := ""
				if val, ok := args["input"].(string); ok {
					inputVal = val
				} else {
					inputVal = tc.FunctionCall.Arguments
				}

				// Execute tool
				res, err := toolExecutor.Execute(ctx, ToolInvocation{
					Tool:      tc.FunctionCall.Name,
					ToolInput: inputVal,
				})
				if err != nil {
					res = fmt.Sprintf("Error: %v", err)
				}

				// Create ToolMessage
				toolMsg := llms.MessageContent{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{
						llms.ToolCallResponse{
							ToolCallID: tc.ID,
							Name:       tc.FunctionCall.Name,
							Content:    res,
						},
					},
				}
				toolMessages = append(toolMessages, toolMsg)
			}
		}

		return map[string]interface{}{
			"messages": toolMessages,
		}, nil
	})

	// Define edges
	workflow.SetEntryPoint("agent")

	workflow.AddConditionalEdge("agent", func(ctx context.Context, state interface{}) string {
		mState := state.(map[string]interface{})
		messages := mState["messages"].([]llms.MessageContent)
		lastMsg := messages[len(messages)-1]

		hasToolCalls := false
		for _, part := range lastMsg.Parts {
			if _, ok := part.(llms.ToolCall); ok {
				hasToolCalls = true
				break
			}
		}

		if hasToolCalls {
			return "tools"
		}
		return graph.END
	})

	workflow.AddEdge("tools", "agent")

	return workflow.Compile()
}
