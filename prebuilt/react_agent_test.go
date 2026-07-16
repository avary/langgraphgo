package prebuilt

import (
	"context"
	"fmt"
	"testing"

	"github.com/smallnest/langgraphgo/llms"
	"github.com/smallnest/langgraphgo/tool"
	"github.com/stretchr/testify/assert"
)

// WeatherTool implements tool.Tool for testing
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

// ReactMockLLM implements llms.Model for testing
type ReactMockLLM struct {
	responses []llms.ContentResponse
	callCount int
}

func (m *ReactMockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.callCount >= len(m.responses) {
		return &llms.ContentResponse{
			Choices: []*llms.ContentChoice{
				{Content: "No more responses"},
			},
		}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return &resp, nil
}

func (m *ReactMockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

func TestReactAgentWithWeatherTool(t *testing.T) {
	weatherTool := NewWeatherTool(25)
	mockLLM := &ReactMockLLM{
		responses: []llms.ContentResponse{
			{Choices: []*llms.ContentChoice{{ToolCalls: []llms.ToolCall{{ID: "call-1", Type: "function", FunctionCall: &llms.FunctionCall{Name: "get_weather", Arguments: `{"input": "beijing"}`}}}}}},
			{Choices: []*llms.ContentChoice{{Content: "Beijing is 25°C."}}},
		},
	}
	agent, err := CreateReactAgentMap(mockLLM, []tool.Tool{weatherTool}, 5)
	assert.NoError(t, err)
	res, err := agent.Invoke(context.Background(), map[string]any{"messages": []llms.MessageContent{llms.TextParts(llms.ChatMessageTypeHuman, "Weather in Beijing?")}})
	assert.NoError(t, err)
	messages := res["messages"].([]llms.MessageContent)
	assert.True(t, len(messages) >= 2)
}
