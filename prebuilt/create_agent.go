package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/smallnest/goskills"
	adapter "github.com/smallnest/langgraphgo/adapter/goskills"
	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/tooltypes"
)

// CreateAgentOptions contains options for creating an agent
type CreateAgentOptions struct {
	skillDir               string
	Verbose                bool
	SystemMessage          string
	StateModifier          func(messages []llmtypes.MessageContent) []llmtypes.MessageContent
	MaxIterations          int
	DisableModelInvocation bool
}

type CreateAgentOption func(*CreateAgentOptions)

func WithSystemMessage(message string) CreateAgentOption {
	return func(o *CreateAgentOptions) { o.SystemMessage = message }
}

func WithStateModifier(modifier func(messages []llmtypes.MessageContent) []llmtypes.MessageContent) CreateAgentOption {
	return func(o *CreateAgentOptions) { o.StateModifier = modifier }
}

func WithSkillDir(skillDir string) CreateAgentOption {
	return func(o *CreateAgentOptions) { o.skillDir = skillDir }
}

func WithVerbose(verbose bool) CreateAgentOption {
	return func(o *CreateAgentOptions) { o.Verbose = verbose }
}

func WithMaxIterations(maxIterations int) CreateAgentOption {
	return func(o *CreateAgentOptions) { o.MaxIterations = maxIterations }
}

func WithDisableModelInvocation(disable bool) CreateAgentOption {
	return func(o *CreateAgentOptions) { o.DisableModelInvocation = disable }
}

// CreateAgentMap creates a new agent graph with map[string]any state
func CreateAgentMap(model llmtypes.Model, inputTools []tooltypes.Tool, maxIterations int, opts ...CreateAgentOption) (*graph.StateRunnable[map[string]any], error) {
	options := &CreateAgentOptions{}
	for _, opt := range opts {
		opt(options)
	}
	if maxIterations == 0 {
		maxIterations = 20
	}
	if options.MaxIterations > 0 {
		maxIterations = options.MaxIterations
	}

	workflow := graph.NewStateGraph[map[string]any]()
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("messages", graph.AppendReducer)
	agentSchema.RegisterReducer("extra_tools", graph.AppendReducer)
	agentSchema.RegisterReducer("iteration_count", graph.OverwriteReducer)
	workflow.SetSchema(agentSchema)

	if options.skillDir != "" {
		workflow.AddNode("skill", "Skill discovery node", func(ctx context.Context, state map[string]any) (map[string]any, error) {
			messages, _ := state["messages"].([]llmtypes.MessageContent)
			if len(messages) == 0 {
				return nil, nil
			}
			userPrompt := ""
			for i := len(messages) - 1; i >= 0; i-- {
				if messages[i].Role == llmtypes.ChatMessageTypeHuman {
					userPrompt = messages[i].Parts[0].(llmtypes.TextContent).Text
					break
				}
			}
			if userPrompt == "" {
				return nil, nil
			}

			availableSkills, err := discoverSkills(options.skillDir)
			if err != nil {
				return nil, err
			}
			selectedSkillName, err := selectSkill(ctx, model, userPrompt, availableSkills)
			if err != nil || selectedSkillName == "" {
				return nil, err
			}
			selectedSkill, ok := availableSkills[selectedSkillName]
			if !ok {
				return nil, nil
			}
			skillTools, err := adapter.SkillsToTools(selectedSkill)
			if err != nil {
				return nil, err
			}
			return map[string]any{"extra_tools": skillTools}, nil
		})
	}

	workflow.AddNode("agent", "Agent decision node", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		messages, _ := state["messages"].([]llmtypes.MessageContent)
		var allTools []tooltypes.Tool
		allTools = append(allTools, inputTools...)
		if extra, ok := state["extra_tools"].([]tooltypes.Tool); ok {
			allTools = append(allTools, extra...)
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

		var toolDefs []llmtypes.Tool
		for _, t := range allTools {
			toolSchema := getToolSchema(t)
			toolDefs = append(toolDefs, llmtypes.Tool{
				Type: "function",
				Function: &llmtypes.FunctionDefinition{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  toolSchema,
				},
			})
			// Debug logging
			if options.Verbose {
				fmt.Printf("[DEBUG] Tool: %s, Schema: %+v\n", t.Name(), toolSchema)
			}
		}

		if options.Verbose {
			fmt.Printf("[DEBUG] Total tools passed to LLM: %d\n", len(toolDefs))
		}

		msgsToSend := messages
		if options.SystemMessage != "" {
			msgsToSend = append([]llmtypes.MessageContent{llmtypes.TextParts(llmtypes.ChatMessageTypeSystem, options.SystemMessage)}, msgsToSend...)
		}
		if options.StateModifier != nil {
			msgsToSend = options.StateModifier(msgsToSend)
		}

		var resp *llmtypes.ContentResponse
		var err error

		if options.DisableModelInvocation {
			// Skip model invocation, create a dummy response
			resp = &llmtypes.ContentResponse{
				Choices: []*llmtypes.ContentChoice{
					{
						Content:   "",
						ToolCalls: []llmtypes.ToolCall{},
					},
				},
			}
		} else {
			resp, err = model.GenerateContent(ctx, msgsToSend, llmtypes.WithTools(toolDefs), llmtypes.WithToolChoice("auto"))
			if err != nil {
				return nil, err
			}
		}

		// Debug: check for tool calls in response
		if options.Verbose {
			fmt.Printf("[DEBUG] LLM Response - Content: %s, ToolCalls: %d\n", resp.Choices[0].Content, len(resp.Choices[0].ToolCalls))
			for i, tc := range resp.Choices[0].ToolCalls {
				fmt.Printf("[DEBUG]   ToolCall %d: %s (%s)\n", i, tc.FunctionCall.Name, tc.FunctionCall.Arguments)
			}
		}

		choice := resp.Choices[0]
		aiMsg := llmtypes.MessageContent{Role: llmtypes.ChatMessageTypeAI}
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

	workflow.AddNode("tools", "Tool execution node", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		messages := state["messages"].([]llmtypes.MessageContent)
		lastMsg := messages[len(messages)-1]
		var allTools []tooltypes.Tool
		allTools = append(allTools, inputTools...)
		if extra, ok := state["extra_tools"].([]tooltypes.Tool); ok {
			allTools = append(allTools, extra...)
		}
		toolExecutor := NewToolExecutor(allTools)

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

				res, err := toolExecutor.Execute(ctx, ToolInvocation{Tool: tc.FunctionCall.Name, ToolInput: inputVal})
				if err != nil {
					res = fmt.Sprintf("Error: %v", err)
				}
				toolMessages = append(toolMessages, llmtypes.MessageContent{
					Role:  llmtypes.ChatMessageTypeTool,
					Parts: []llmtypes.ContentPart{llmtypes.ToolCallResponse{ToolCallID: tc.ID, Name: tc.FunctionCall.Name, Content: res}},
				})
			}
		}
		return map[string]any{"messages": toolMessages}, nil
	})

	if options.skillDir != "" {
		workflow.SetEntryPoint("skill")
		workflow.AddEdge("skill", "agent")
	} else {
		workflow.SetEntryPoint("agent")
	}

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

// CreateAgent creates a generic agent graph
func CreateAgent[S any](
	model llmtypes.Model,
	inputTools []tooltypes.Tool,
	getMessages func(S) []llmtypes.MessageContent,
	setMessages func(S, []llmtypes.MessageContent) S,
	getExtraTools func(S) []tooltypes.Tool,
	setExtraTools func(S, []tooltypes.Tool) S,
	opts ...CreateAgentOption,
) (*graph.StateRunnable[S], error) {
	options := &CreateAgentOptions{}
	for _, opt := range opts {
		opt(options)
	}

	workflow := graph.NewStateGraph[S]()

	workflow.AddNode("agent", "Agent decision node", func(ctx context.Context, state S) (S, error) {
		messages := getMessages(state)
		allTools := append(inputTools, getExtraTools(state)...)

		var toolDefs []llmtypes.Tool
		for _, t := range allTools {
			toolDefs = append(toolDefs, llmtypes.Tool{
				Type: "function",
				Function: &llmtypes.FunctionDefinition{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  getToolSchema(t),
				},
			})
		}

		msgsToSend := messages
		if options.SystemMessage != "" {
			msgsToSend = append([]llmtypes.MessageContent{llmtypes.TextParts(llmtypes.ChatMessageTypeSystem, options.SystemMessage)}, msgsToSend...)
		}
		if options.StateModifier != nil {
			msgsToSend = options.StateModifier(msgsToSend)
		}

		var resp *llmtypes.ContentResponse
		var err error

		if options.DisableModelInvocation {
			// Skip model invocation, create a dummy response
			resp = &llmtypes.ContentResponse{
				Choices: []*llmtypes.ContentChoice{
					{
						Content:   "",
						ToolCalls: []llmtypes.ToolCall{},
					},
				},
			}
		} else {
			resp, err = model.GenerateContent(ctx, msgsToSend, llmtypes.WithTools(toolDefs))
			if err != nil {
				return state, err
			}
		}

		choice := resp.Choices[0]
		aiMsg := llmtypes.MessageContent{Role: llmtypes.ChatMessageTypeAI}
		if choice.Content != "" {
			aiMsg.Parts = append(aiMsg.Parts, llmtypes.TextPart(choice.Content))
		}
		for _, tc := range choice.ToolCalls {
			aiMsg.Parts = append(aiMsg.Parts, tc)
		}

		return setMessages(state, append(messages, aiMsg)), nil
	})

	workflow.AddNode("tools", "Tool execution node", func(ctx context.Context, state S) (S, error) {
		messages := getMessages(state)
		lastMsg := messages[len(messages)-1]
		toolExecutor := NewToolExecutor(append(inputTools, getExtraTools(state)...))

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

				res, err := toolExecutor.Execute(ctx, ToolInvocation{Tool: tc.FunctionCall.Name, ToolInput: inputVal})
				if err != nil {
					res = fmt.Sprintf("Error: %v", err)
				}
				toolMessages = append(toolMessages, llmtypes.MessageContent{
					Role:  llmtypes.ChatMessageTypeTool,
					Parts: []llmtypes.ContentPart{llmtypes.ToolCallResponse{ToolCallID: tc.ID, Name: tc.FunctionCall.Name, Content: res}},
				})
			}
		}
		return setMessages(state, append(messages, toolMessages...)), nil
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

func discoverSkills(skillDir string) (map[string]*goskills.SkillPackage, error) {
	packages, err := goskills.ParseSkillPackages(skillDir)
	if err != nil {
		return nil, err
	}
	skills := make(map[string]*goskills.SkillPackage)
	for _, pkg := range packages {
		skills[pkg.Meta.Name] = pkg
	}
	return skills, nil
}

func selectSkill(ctx context.Context, model llmtypes.Model, userPrompt string, availableSkills map[string]*goskills.SkillPackage) (string, error) {
	var skillDescriptions strings.Builder
	for name, pkg := range availableSkills {
		skillDescriptions.WriteString(fmt.Sprintf("- %s: %s\n", name, pkg.Meta.Description))
	}
	prompt := fmt.Sprintf("Select the most appropriate skill for: \"%s\"\n\nSkills:\n%s\nReturn only the skill name or 'None'.", userPrompt, skillDescriptions.String())
	resp, err := model.GenerateContent(ctx, []llmtypes.MessageContent{llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, prompt)})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Choices[0].Content), nil
}
