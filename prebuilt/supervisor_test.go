package prebuilt

import (
	"context"
	"errors"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SupervisorMockLLM for supervisor testing
type SupervisorMockLLM struct {
	responses   []llmtypes.ContentResponse
	currentIdx  int
	returnError error
}

func (m *SupervisorMockLLM) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	if m.returnError != nil {
		return nil, m.returnError
	}
	if m.currentIdx >= len(m.responses) {
		return &llmtypes.ContentResponse{
			Choices: []*llmtypes.ContentChoice{
				{Content: "No more responses"},
			},
		}, nil
	}
	resp := m.responses[m.currentIdx]
	m.currentIdx++
	return &resp, nil
}

func (m *SupervisorMockLLM) Call(ctx context.Context, prompt string, options ...llmtypes.CallOption) (string, error) {
	return "", nil
}

// MockAgent for testing supervisor with various behaviors
type MockAgent struct {
	name        string
	response    string
	shouldError bool
	errorMsg    string
}

func NewMockAgent(name, response string) *MockAgent {
	return &MockAgent{
		name:     name,
		response: response,
	}
}

func NewMockErrorAgent(name, errorMsg string) *MockAgent {
	return &MockAgent{
		name:        name,
		shouldError: true,
		errorMsg:    errorMsg,
	}
}

func (a *MockAgent) Invoke(ctx context.Context, state any) (any, error) {
	if a.shouldError {
		return nil, errors.New(a.errorMsg)
	}

	// Extract existing messages
	mState, ok := state.(map[string]any)
	if !ok {
		return nil, errors.New("invalid state type")
	}

	messages, ok := mState["messages"].([]llmtypes.MessageContent)
	if !ok {
		return nil, errors.New("messages key not found or invalid type")
	}

	// Append agent response
	newMessages := append(messages, llmtypes.TextParts(llmtypes.ChatMessageTypeAI, a.response))

	return map[string]any{
		"messages": newMessages,
	}, nil
}

func (a *MockAgent) Compile() (*graph.StateRunnable[map[string]any], error) {
	workflow := graph.NewStateGraph[map[string]any]()

	// Define state schema
	schema := graph.NewMapSchema()
	schema.RegisterReducer("messages", graph.AppendReducer)
	workflow.SetSchema(schema)

	workflow.AddNode("run", "Agent run node", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		result, err := a.Invoke(ctx, state)
		if err != nil {
			return nil, err
		}
		if m, ok := result.(map[string]any); ok {
			return m, nil
		}
		return state, nil
	})
	workflow.SetEntryPoint("run")
	workflow.AddEdge("run", graph.END)

	return workflow.Compile()
}

func TestCreateSupervisor_DirectFinish(t *testing.T) {
	// Test supervisor that directly routes to FINISH
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "FINISH"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent", "Should not be called")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Complete immediately"),
		},
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	messages := res["messages"].([]llmtypes.MessageContent)
	// Should only have initial message, no agent responses
	assert.Equal(t, 1, len(messages))
	assert.Equal(t, "Complete immediately", messages[0].Parts[0].(llmtypes.TextContent).Text)
	assert.Equal(t, "FINISH", res["next"])
}

func TestCreateSupervisor_AgentError(t *testing.T) {
	// Test handling of agent errors
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "ErrorAgent"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	errorAgent := NewMockErrorAgent("ErrorAgent", "Agent failed to process")
	errorAgentRunnable, err := errorAgent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"ErrorAgent": errorAgentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Trigger error"),
		},
	}

	// Should return error from agent
	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Agent failed to process")
}

func TestCreateSupervisor_NoToolCall(t *testing.T) {
	// Test when LLM doesn't make a tool call
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						Content: "I don't know what to do",
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Test"),
		},
	}

	// Should return error about not selecting next step
	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supervisor did not select a next step")
}

func TestCreateSupervisor_InvalidRouteArguments(t *testing.T) {
	// Test when route tool has invalid JSON
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "route",
									Arguments: `{invalid json`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse route arguments")
}

func TestCreateSupervisor_InvalidStateType(t *testing.T) {
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Agent1"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent1", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"Agent1": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	// Pass state without messages key
	_, err = supervisor.Invoke(context.Background(), map[string]any{"foo": "bar"})
	assert.Error(t, err)
}

func TestCreateSupervisor_MissingMessages(t *testing.T) {
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Agent1"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent1", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"Agent1": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	// Pass state without messages
	invalidState := map[string]any{
		"other": "value",
	}
	_, err = supervisor.Invoke(context.Background(), invalidState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "messages key not found or invalid type")
}

func TestCreateSupervisor_LLMError(t *testing.T) {
	// Test when LLM returns an error
	mockLLM := &SupervisorMockLLM{
		responses:   []llmtypes.ContentResponse{},
		returnError: errors.New("LLM connection failed"),
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM connection failed")
}

func TestCreateSupervisor_EmptyMembers(t *testing.T) {
	// Test with no members
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "FINISH"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	// Empty members map
	members := map[string]*graph.StateRunnable[map[string]any]{}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Test"),
		},
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	messages := res["messages"].([]llmtypes.MessageContent)
	assert.Equal(t, 1, len(messages)) // Only initial message
	assert.Equal(t, "FINISH", res["next"])
}

func TestCreateSupervisor_UnknownAgent(t *testing.T) {
	// Test when LLM routes to an unknown agent
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "UnknownAgent"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("KnownAgent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"KnownAgent": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Test unknown agent"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	// The error message will depend on graph implementation
	assert.Error(t, err)
}

func TestCreateSupervisor_RouteWithoutFunctionCall(t *testing.T) {
	// Test when tool call has no function call
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								// No FunctionCall field
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
}

func TestCreateSupervisor_NoChoices(t *testing.T) {
	// Test when LLM returns no choices
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{}, // Empty choices
			},
		},
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
}

func TestCreateSupervisor_EmptyRouteName(t *testing.T) {
	// Test when route tool call has empty name
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "",
									Arguments: `{"next": "Agent"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Agent", "Response")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"Agent": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Test"),
		},
	}

	_, err = supervisor.Invoke(context.Background(), initialState)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "supervisor did not select a next step")
}

func TestCreateSupervisor_SingleAgent(t *testing.T) {
	// Test with single agent
	mockLLM := &SupervisorMockLLM{
		responses: []llmtypes.ContentResponse{
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "Worker"}`,
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llmtypes.ContentChoice{
					{
						ToolCalls: []llmtypes.ToolCall{
							{
								FunctionCall: &llmtypes.FunctionCall{
									Name:      "route",
									Arguments: `{"next": "FINISH"}`,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := NewMockAgent("Worker", "Task completed")
	agentRunnable, err := agent.Compile()
	require.NoError(t, err)

	members := map[string]*graph.StateRunnable[map[string]any]{
		"Worker": agentRunnable,
	}
	supervisor, err := CreateSupervisorMap(mockLLM, members)
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Single task"),
		},
	}

	res, err := supervisor.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	messages := res["messages"].([]llmtypes.MessageContent)
	// Should have initial + worker message + potential routing messages
	assert.True(t, len(messages) >= 2)
	assert.Equal(t, "Single task", messages[0].Parts[0].(llmtypes.TextContent).Text)
	// Find the worker response
	found := false
	for _, msg := range messages[1:] {
		if msg.Role == llmtypes.ChatMessageTypeAI {
			if txt, ok := msg.Parts[0].(llmtypes.TextContent); ok && txt.Text == "Task completed" {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Worker response should be in messages")
}
