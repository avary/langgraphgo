package prebuilt

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
	"github.com/smallnest/langgraphgo/tool"
	"github.com/stretchr/testify/assert"
)

// MockPlanningLLM is a mock LLM that returns a workflow plan
type MockPlanningLLM struct {
	planJSON      string
	responses     []llms.ContentResponse
	callCount     int
	capturedCalls [][]llms.MessageContent
}

func (m *MockPlanningLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	m.capturedCalls = append(m.capturedCalls, messages)

	// First call is the planning call
	if m.callCount == 0 {
		m.callCount++
		return &llms.ContentResponse{
			Choices: []*llms.ContentChoice{
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

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: "No more responses"},
		},
	}, nil
}

func (m *MockPlanningLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

func TestCreatePlanningAgentMap_SimpleWorkflow(t *testing.T) {
	// Define test nodes
	testNodes := []graph.TypedNode[map[string]any]{
		{
			Name:        "research",
			Description: "Research and gather information",
			Function: func(ctx context.Context, state map[string]any) (map[string]any, error) {
				messages := state["messages"].([]llms.MessageContent)
				researchMsg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Research completed")},
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
				messages := state["messages"].([]llms.MessageContent)
				analyzeMsg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Analysis completed")},
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
		responses: []llms.ContentResponse{},
	}

	// Create Planning Agent
	agent, err := CreatePlanningAgentMap(mockLLM, testNodes, []tool.Tool{})
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	// Initial State
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Please research and analyze"),
		},
	}

	// Run Agent
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)
	assert.NotNil(t, res)
}
