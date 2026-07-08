package prebuilt

import (
	"context"
	"fmt"
	"testing"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/tooltypes"
	"github.com/stretchr/testify/assert"
)

// WeatherTool implements tooltypes.Tool for testing
type WeatherTool struct {
	currentTemp int
}

func NewWeatherTool(temp int) *WeatherTool {
	return &WeatherTool{currentTemp: temp}
}

func (t *WeatherTool) Name() string        { return "get_weather" }
func (t *WeatherTool) Description() string { return "Get weather" }
func (t *WeatherTool) Call(ctx context.Context, input string) (string, error) {
	return fmt.Sprintf("Weather: %d°C", t.currentTemp), nil
}

// ReactMockLLM implements llmtypes.Model for testing
type ReactMockLLM struct {
	responses []llmtypes.ContentResponse
	callCount int
}

func (m *ReactMockLLM) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	if m.callCount >= len(m.responses) {
		return &llmtypes.ContentResponse{
			Choices: []*llmtypes.ContentChoice{
				{Content: "No more responses"},
			},
		}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return &resp, nil
}

func (m *ReactMockLLM) Call(ctx context.Context, prompt string, options ...llmtypes.CallOption) (string, error) {
	return "", nil
}

func TestReactAgentWithWeatherTool(t *testing.T) {
	weatherTool := NewWeatherTool(25)
	mockLLM := &ReactMockLLM{
		responses: []llmtypes.ContentResponse{
			{Choices: []*llmtypes.ContentChoice{{ToolCalls: []llmtypes.ToolCall{{ID: "call-1", Type: "function", FunctionCall: &llmtypes.FunctionCall{Name: "get_weather", Arguments: `{"input": "beijing"}`}}}}}},
			{Choices: []*llmtypes.ContentChoice{{Content: "Beijing is 25°C."}}},
		},
	}
	agent, err := CreateReactAgentMap(mockLLM, []tooltypes.Tool{weatherTool}, 5)
	assert.NoError(t, err)
	res, err := agent.Invoke(context.Background(), map[string]any{"messages": []llmtypes.MessageContent{llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "Weather in Beijing?")}})
	assert.NoError(t, err)
	messages := res["messages"].([]llmtypes.MessageContent)
	assert.True(t, len(messages) >= 2)
}
