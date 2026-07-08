package prebuilt

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/tooltypes"
	"github.com/stretchr/testify/assert"
)

func TestCreateAgentMap(t *testing.T) {
	mockLLM := &MockLLM{}
	inputTools := []tooltypes.Tool{}
	systemMessage := "You are a helpful assistant."

	t.Run("Basic Agent Creation", func(t *testing.T) {
		agent, err := CreateAgentMap(mockLLM, inputTools, 0, WithSystemMessage(systemMessage))
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Agent with State Modifier", func(t *testing.T) {
		mockLLM := &MockLLMWithInputCapture{}
		modifier := func(messages []llmtypes.MessageContent) []llmtypes.MessageContent {
			return append(messages, llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Modified"))
		}

		agent, err := CreateAgentMap(mockLLM, inputTools, 0, WithStateModifier(modifier))
		assert.NoError(t, err)

		_, err = agent.Invoke(context.Background(), map[string]any{"messages": []llmtypes.MessageContent{}})
		assert.NoError(t, err)

		// Verify modifier was called (last message should be "Modified")
		assert.True(t, len(mockLLM.lastMessages) > 0)
		lastMsg := mockLLM.lastMessages[len(mockLLM.lastMessages)-1]
		assert.Equal(t, "Modified", lastMsg.Parts[0].(llmtypes.TextContent).Text)
	})

	t.Run("Agent with System Message", func(t *testing.T) {
		mockLLM := &MockLLMWithInputCapture{}
		systemMsg := "You are a specialized bot."

		agent, err := CreateAgentMap(mockLLM, inputTools, 0, WithSystemMessage(systemMsg))
		assert.NoError(t, err)

		_, err = agent.Invoke(context.Background(), map[string]any{"messages": []llmtypes.MessageContent{}})
		assert.NoError(t, err)

		// Verify system message was prepended
		assert.True(t, len(mockLLM.lastMessages) > 0)
		firstMsg := mockLLM.lastMessages[0]
		assert.Equal(t, llmtypes.ChatMessageTypeSystem, firstMsg.Role)
		assert.Equal(t, systemMsg, firstMsg.Parts[0].(llmtypes.TextContent).Text)
	})

	t.Run("Agent with Verbose option", func(t *testing.T) {
		// Test that WithVerbose option is properly set
		agent, err := CreateAgentMap(mockLLM, inputTools, 0, WithVerbose(true))
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Agent with tools", func(t *testing.T) {
		mockTool := &MockToolWithResponse{name: "test_tool", description: "A test tool", response: "Tool response"}
		agent, err := CreateAgentMap(mockLLM, []tooltypes.Tool{mockTool}, 0)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Agent Invoke with messages", func(t *testing.T) {
		mockLLM := &MockLLMWithInputCapture{}
		agent, err := CreateAgentMap(mockLLM, inputTools, 0)
		assert.NoError(t, err)

		messages := []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Hello"),
		}
		result, err := agent.Invoke(context.Background(), map[string]any{"messages": messages})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestCreateAgentGeneric(t *testing.T) {
	mockLLM := &MockLLM{}

	t.Run("Create generic AgentState agent", func(t *testing.T) {
		inputTools := []tooltypes.Tool{}

		agent, err := CreateAgent[AgentState](
			mockLLM,
			inputTools,
			func(s AgentState) []llmtypes.MessageContent { return s.Messages },
			func(s AgentState, msgs []llmtypes.MessageContent) AgentState {
				s.Messages = msgs
				return s
			},
			func(s AgentState) []tooltypes.Tool { return s.ExtraTools },
			func(s AgentState, tools []tooltypes.Tool) AgentState {
				s.ExtraTools = tools
				return s
			},
		)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Create generic agent with system message", func(t *testing.T) {
		inputTools := []tooltypes.Tool{}
		systemMsg := "You are a helpful assistant."

		agent, err := CreateAgent[AgentState](
			mockLLM,
			inputTools,
			func(s AgentState) []llmtypes.MessageContent { return s.Messages },
			func(s AgentState, msgs []llmtypes.MessageContent) AgentState {
				s.Messages = msgs
				return s
			},
			func(s AgentState) []tooltypes.Tool { return s.ExtraTools },
			func(s AgentState, tools []tooltypes.Tool) AgentState {
				s.ExtraTools = tools
				return s
			},
			WithSystemMessage(systemMsg),
		)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Create generic agent with state modifier", func(t *testing.T) {
		inputTools := []tooltypes.Tool{}
		modifier := func(messages []llmtypes.MessageContent) []llmtypes.MessageContent {
			return append(messages, llmtypes.TextParts(llmtypes.ChatMessageTypeSystem, "Modified"))
		}

		agent, err := CreateAgent[AgentState](
			mockLLM,
			inputTools,
			func(s AgentState) []llmtypes.MessageContent { return s.Messages },
			func(s AgentState, msgs []llmtypes.MessageContent) AgentState {
				s.Messages = msgs
				return s
			},
			func(s AgentState) []tooltypes.Tool { return s.ExtraTools },
			func(s AgentState, tools []tooltypes.Tool) AgentState {
				s.ExtraTools = tools
				return s
			},
			WithStateModifier(modifier),
		)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Generic agent invoke", func(t *testing.T) {
		inputTools := []tooltypes.Tool{}

		agent, err := CreateAgent[AgentState](
			mockLLM,
			inputTools,
			func(s AgentState) []llmtypes.MessageContent { return s.Messages },
			func(s AgentState, msgs []llmtypes.MessageContent) AgentState {
				s.Messages = msgs
				return s
			},
			func(s AgentState) []tooltypes.Tool { return s.ExtraTools },
			func(s AgentState, tools []tooltypes.Tool) AgentState {
				s.ExtraTools = tools
				return s
			},
		)
		assert.NoError(t, err)

		state := AgentState{
			Messages: []llmtypes.MessageContent{
				llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Hello"),
			},
		}
		result, err := agent.Invoke(context.Background(), state)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Generic agent with extra tools", func(t *testing.T) {
		inputTools := []tooltypes.Tool{
			&MockToolWithResponse{name: "base_tool", description: "Base tool", response: "base response"},
		}

		agent, err := CreateAgent[AgentState](
			mockLLM,
			inputTools,
			func(s AgentState) []llmtypes.MessageContent { return s.Messages },
			func(s AgentState, msgs []llmtypes.MessageContent) AgentState {
				s.Messages = msgs
				return s
			},
			func(s AgentState) []tooltypes.Tool { return s.ExtraTools },
			func(s AgentState, tools []tooltypes.Tool) AgentState {
				s.ExtraTools = tools
				return s
			},
		)
		assert.NoError(t, err)

		state := AgentState{
			Messages: []llmtypes.MessageContent{
				llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Hello"),
			},
			ExtraTools: []tooltypes.Tool{
				&MockToolWithResponse{name: "extra_tool", description: "Extra tool", response: "extra response"},
			},
		}
		result, err := agent.Invoke(context.Background(), state)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestCreateAgentWithToolCalls(t *testing.T) {
	t.Run("AgentMap with tool call response", func(t *testing.T) {
		mockLLM := &MockLLMWithToolCalls{}

		mockTool := &MockToolWithResponse{
			name:        "test_tool",
			description: "A test tool",
			response:    "Tool executed successfully",
		}

		agent, err := CreateAgentMap(mockLLM, []tooltypes.Tool{mockTool}, 0)
		assert.NoError(t, err)

		messages := []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Use the test tool"),
		}
		result, err := agent.Invoke(context.Background(), map[string]any{"messages": messages})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Generic agent with tool calls", func(t *testing.T) {
		mockLLM := &MockLLMWithToolCalls{}

		mockTool := &MockToolWithResponse{
			name:        "test_tool",
			description: "A test tool",
			response:    "Tool executed successfully",
		}

		agent, err := CreateAgent[AgentState](
			mockLLM,
			[]tooltypes.Tool{mockTool},
			func(s AgentState) []llmtypes.MessageContent { return s.Messages },
			func(s AgentState, msgs []llmtypes.MessageContent) AgentState {
				s.Messages = msgs
				return s
			},
			func(s AgentState) []tooltypes.Tool { return s.ExtraTools },
			func(s AgentState, tools []tooltypes.Tool) AgentState {
				s.ExtraTools = tools
				return s
			},
		)
		assert.NoError(t, err)

		state := AgentState{
			Messages: []llmtypes.MessageContent{
				llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Use the test tool"),
			},
		}
		result, err := agent.Invoke(context.Background(), state)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// Mock structures for testing
type MockLLM struct {
	llmtypes.Model
}

func (m *MockLLM) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	return &llmtypes.ContentResponse{
		Choices: []*llmtypes.ContentChoice{
			{
				Content: "Hello! I'm a mock AI.",
			},
		},
	}, nil
}

type MockLLMWithInputCapture struct {
	llmtypes.Model
	lastMessages []llmtypes.MessageContent
}

func (m *MockLLMWithInputCapture) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	m.lastMessages = messages
	return &llmtypes.ContentResponse{
		Choices: []*llmtypes.ContentChoice{
			{
				Content: "Response",
			},
		},
	}, nil
}

type MockLLMWithToolCalls struct {
	llmtypes.Model
	callCount int
}

func (m *MockLLMWithToolCalls) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	m.callCount++

	if m.callCount == 1 {
		// First call returns a tool call
		return &llmtypes.ContentResponse{
			Choices: []*llmtypes.ContentChoice{
				{
					Content: "I'll use the tool for you.",
					ToolCalls: []llmtypes.ToolCall{
						{
							ID:   "call_123",
							Type: "function",
							FunctionCall: &llmtypes.FunctionCall{
								Name:      "test_tool",
								Arguments: `{"input":"test input"}`,
							},
						},
					},
					StopReason: "tool_calls",
				},
			},
		}, nil
	}

	// Second call returns final response
	return &llmtypes.ContentResponse{
		Choices: []*llmtypes.ContentChoice{
			{
				Content:    "Tool execution complete. Result: Tool executed successfully",
				StopReason: "stop",
			},
		},
	}, nil
}

type MockToolWithResponse struct {
	name        string
	description string
	response    string
}

func (t *MockToolWithResponse) Name() string {
	return t.name
}

func (t *MockToolWithResponse) Description() string {
	return t.description
}

func (t *MockToolWithResponse) Call(ctx context.Context, input string) (string, error) {
	if t.response != "" {
		return t.response, nil
	}
	return "Mock tool response", nil
}
