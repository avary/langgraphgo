package prebuilt

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/tools"
)

// MockPlanningLLM is a mock LLM that returns a workflow plan
type MockPlanningLLM struct {
	planJSON      string
	responses     []llmtypes.ContentResponse
	callCount     int
	capturedCalls [][]llmtypes.MessageContent
}

func (m *MockPlanningLLM) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	m.capturedCalls = append(m.capturedCalls, messages)

	// First call is the planning call
	if m.callCount == 0 {
		m.callCount++
		return &llmtypes.ContentResponse{
			Choices: []*llmtypes.ContentChoice{
				{
					Content: m.planJSON,
				},
			},
		}, nil
	}

	// Subsequent calls use predefined responses
	if m.callCount-1 < len(m.responses) {
		resp := m.responses[m.callCount-1]
		m.callCount++
		return &resp, nil
	}

	return &llmtypes.ContentResponse{
		Choices: []*llmtypes.ContentChoice{
			{Content: "No more responses"},
		},
	}, nil
}

func (m *MockPlanningLLM) Call(ctx context.Context, prompt string, options ...llmtypes.CallOption) (string, error) {
	return "", nil
}

func TestCreatePlanningAgentMap_SimpleWorkflow(t *testing.T) {
	// Define test nodes
	testNodes := []graph.TypedNode[map[string]any]{
		{
			Name:        "research",
			Description: "Research and gather information",
			Function: func(ctx context.Context, state map[string]any) (map[string]any, error) {
				messages := state["messages"].([]llmtypes.MessageContent)
				researchMsg := llmtypes.MessageContent{
					Role:  llmtypes.ChatMessageTypeAI,
					Parts: []llmtypes.ContentPart{llmtypes.TextPart("Research completed")},
				}
				return map[string]any{
					"messages": append(messages, researchMsg),
				}, nil
			},
		},
		{
			Name:        "analyze",
			Description: "Analyze the gathered information",
			Function: func(ctx context.Context, state map[string]any) (map[string]any, error) {
				messages := state["messages"].([]llmtypes.MessageContent)
				analyzeMsg := llmtypes.MessageContent{
					Role:  llmtypes.ChatMessageTypeAI,
					Parts: []llmtypes.ContentPart{llmtypes.TextPart("Analysis completed")},
				}
				return map[string]any{
					"messages": append(messages, analyzeMsg),
				}, nil
			},
		},
	}

	// Create a workflow plan JSON
	planJSON := `{
		"nodes": [
			{"name": "research", "type": "process"},
			{"name": "analyze", "type": "process"}
		],
		"edges": [
			{"from": "START", "to": "research"},
			{"from": "research", "to": "analyze"},
			{"from": "analyze", "to": "END"}
		]
	}`

	// Setup Mock LLM
	mockLLM := &MockPlanningLLM{
		planJSON:  planJSON,
		responses: []llmtypes.ContentResponse{},
	}

	// Create Planning Agent
	agent, err := CreatePlanningAgentMap(mockLLM, testNodes, []tools.Tool{})
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	// Initial State
	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Please research and analyze"),
		},
	}

	// Run Agent
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)
	assert.NotNil(t, res)
}
