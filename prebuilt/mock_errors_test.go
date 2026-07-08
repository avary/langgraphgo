package prebuilt

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/stretchr/testify/assert"
)

func TestMockLLMError(t *testing.T) {
	mock := &MockLLMError{}

	t.Run("GenerateContent returns error", func(t *testing.T) {
		ctx := context.Background()
		messages := []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "test"),
		}

		resp, err := mock.GenerateContent(ctx, messages)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "mock LLM GenerateContent error")
	})

	t.Run("Call returns error", func(t *testing.T) {
		ctx := context.Background()
		resp, err := mock.Call(ctx, "test prompt")
		assert.Error(t, err)
		assert.Empty(t, resp)
		assert.Contains(t, err.Error(), "mock LLM Call error")
	})
}

func TestMockLLMEmptyContent(t *testing.T) {
	mock := &MockLLMEmptyContent{}

	t.Run("GenerateContent returns empty content", func(t *testing.T) {
		ctx := context.Background()
		messages := []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, "test"),
		}

		resp, err := mock.GenerateContent(ctx, messages)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Choices, 1)
		assert.Equal(t, "", resp.Choices[0].Content)
	})

	t.Run("Call returns empty string", func(t *testing.T) {
		ctx := context.Background()
		resp, err := mock.Call(ctx, "test prompt")
		assert.NoError(t, err)
		assert.Equal(t, "", resp)
	})
}

func TestMockToolError(t *testing.T) {
	mock := &MockToolError{name: "error_tool"}

	t.Run("Name returns tool name", func(t *testing.T) {
		assert.Equal(t, "error_tool", mock.Name())
	})

	t.Run("Description returns description", func(t *testing.T) {
		assert.Equal(t, "A mock tool that returns an error", mock.Description())
	})

	t.Run("Call returns error", func(t *testing.T) {
		ctx := context.Background()
		resp, err := mock.Call(ctx, "test input")
		assert.Error(t, err)
		assert.Empty(t, resp)
		assert.Contains(t, err.Error(), "mock tool execution error")
	})

	t.Run("Call with empty input", func(t *testing.T) {
		ctx := context.Background()
		resp, err := mock.Call(ctx, "")
		assert.Error(t, err)
		assert.Empty(t, resp)
	})
}

func TestMockErrorsImplementInterfaces(t *testing.T) {
	t.Run("MockLLMError implements Model interface", func(t *testing.T) {
		var _ llmtypes.Model = (*MockLLMError)(nil)
		mock := &MockLLMError{}
		assert.NotNil(t, mock)
	})

	t.Run("MockLLMEmptyContent implements Model interface", func(t *testing.T) {
		var _ llmtypes.Model = (*MockLLMEmptyContent)(nil)
		mock := &MockLLMEmptyContent{}
		assert.NotNil(t, mock)
	})

	t.Run("MockToolError implements Tool interface", func(t *testing.T) {
		var _ interface {
			Name() string
			Description() string
			Call(ctx context.Context, input string) (string, error)
		} = (*MockToolError)(nil)
		mock := &MockToolError{}
		assert.NotNil(t, mock)
	})
}
