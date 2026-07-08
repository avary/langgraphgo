package prebuilt

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/tools"
)

func TestToolNodeMap(t *testing.T) {
	mockTool := &MockTool{name: "test-tool"}
	executor := NewToolExecutor([]tools.Tool{mockTool})
	node := ToolNodeMap(executor)

	// Construct state with AIMessage containing ToolCall
	toolCall := llmtypes.ToolCall{
		ID:   "call_1",
		Type: "function",
		FunctionCall: &llmtypes.FunctionCall{
			Name:      "test-tool",
			Arguments: `{"input": "test-input"}`,
		},
	}

	aiMsg := llmtypes.MessageContent{
		Role: llmtypes.ChatMessageTypeAI,
		Parts: []llmtypes.ContentPart{
			toolCall,
		},
	}

	state := map[string]any{
		"messages": []llmtypes.MessageContent{aiMsg},
	}

	// Invoke ToolNode
	res, err := node(context.Background(), state)
	assert.NoError(t, err)

	msgs, ok := res["messages"].([]llmtypes.MessageContent)
	assert.True(t, ok)
	assert.Len(t, msgs, 1)

	toolMsg := msgs[0]
	assert.Equal(t, llmtypes.ChatMessageTypeTool, toolMsg.Role)
	assert.Len(t, toolMsg.Parts, 1)

	toolResp, ok := toolMsg.Parts[0].(llmtypes.ToolCallResponse)
	assert.True(t, ok)
	assert.Equal(t, "call_1", toolResp.ToolCallID)
	assert.Equal(t, "test-tool", toolResp.Name)
	assert.Equal(t, "Executed test-tool with test-input", toolResp.Content)
}
