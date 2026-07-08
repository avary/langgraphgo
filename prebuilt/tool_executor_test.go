package prebuilt

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/tooltypes"
	"github.com/stretchr/testify/assert"
)

// MockTool implements tooltypes.Tool for testing
type MockTool struct {
	name string
}

func (t *MockTool) Name() string {
	return t.name
}

func (t *MockTool) Description() string {
	return "A mock tool"
}

func (t *MockTool) Call(ctx context.Context, input string) (string, error) {
	return "Executed " + t.name + " with " + input, nil
}

func TestToolExecutor(t *testing.T) {
	mockTool := &MockTool{name: "test-tool"}
	executor := NewToolExecutor([]tooltypes.Tool{mockTool})

	// Test single invocation
	inv := ToolInvocation{
		Tool:      "test-tool",
		ToolInput: "input",
	}
	res, err := executor.Execute(context.Background(), inv)
	assert.NoError(t, err)
	assert.Equal(t, "Executed test-tool with input", res)

	// Test ToolNode with struct
	resNode, err := executor.ToolNode(context.Background(), inv)
	assert.NoError(t, err)
	assert.Equal(t, "Executed test-tool with input", resNode)

	// Test ToolNode with map
	mapState := map[string]any{
		"tool":       "test-tool",
		"tool_input": "map-input",
	}
	resMap, err := executor.ToolNode(context.Background(), mapState)
	assert.NoError(t, err)
	assert.Equal(t, "Executed test-tool with map-input", resMap)
}
