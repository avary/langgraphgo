package prebuilt

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/llms"
	"github.com/smallnest/langgraphgo/tool"
	"github.com/stretchr/testify/assert"
)

func TestToolNodeMap(t *testing.T) {
	mockTool := &MockTool{name: "test-tool"}
	executor := NewToolExecutor([]tool.Tool{mockTool})
	node := ToolNodeMap(executor)

	// Construct state with AIMessage containing ToolCall
	toolCall := llms.ToolCall{
		ID:   "call_1",
		Type: "function",
		FunctionCall: &llms.FunctionCall{
			Name:      "test-tool",
			Arguments: `{"input": "test-input"}`,
		},
	}

	aiMsg := llms.MessageContent{
		Role: llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{
			toolCall,
		},
	}

	state := map[string]any{
		"messages": []llms.MessageContent{aiMsg},
	}

	// Invoke ToolNode
	res, err := node(context.Background(), state)
	assert.NoError(t, err)

	msgs, ok := res["messages"].([]llms.MessageContent)
	assert.True(t, ok)
	assert.Len(t, msgs, 1)

	toolMsg := msgs[0]
	assert.Equal(t, llms.ChatMessageTypeTool, toolMsg.Role)
	assert.Len(t, toolMsg.Parts, 1)

	toolResp, ok := toolMsg.Parts[0].(llms.ToolCallResponse)
	assert.True(t, ok)
	assert.Equal(t, "call_1", toolResp.ToolCallID)
	assert.Equal(t, "test-tool", toolResp.Name)
	assert.Equal(t, "Executed test-tool with test-input", toolResp.Content)
}
