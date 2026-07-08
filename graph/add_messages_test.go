package graph

import (
	"testing"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/stretchr/testify/assert"
)

// Custom message struct with ID
type TestMessage struct {
	ID      string
	Content string
}

func TestAddMessages(t *testing.T) {
	// Case 1: Standard llmtypes.MessageContent (No ID, simple append)
	t.Run("StandardMessages", func(t *testing.T) {
		current := []llmtypes.MessageContent{
			{Role: "user", Parts: []llmtypes.ContentPart{llmtypes.TextPart("Hello")}},
		}
		newMsg := llmtypes.MessageContent{Role: "ai", Parts: []llmtypes.ContentPart{llmtypes.TextPart("Hi")}}

		res, err := AddMessages(current, newMsg)
		assert.NoError(t, err)

		slice, ok := res.([]llmtypes.MessageContent)
		assert.True(t, ok)
		assert.Len(t, slice, 2)
		assert.Equal(t, llmtypes.ChatMessageType("user"), slice[0].Role)
		assert.Equal(t, llmtypes.ChatMessageType("ai"), slice[1].Role)
	})

	// Case 2: Structs with ID (Upsert logic)
	t.Run("MessagesWithID", func(t *testing.T) {
		current := []TestMessage{
			{ID: "1", Content: "First"},
			{ID: "2", Content: "Second"},
		}

		// Update message 1, add message 3
		newMessages := []TestMessage{
			{ID: "1", Content: "First Updated"},
			{ID: "3", Content: "Third"},
		}

		res, err := AddMessages(current, newMessages)
		assert.NoError(t, err)

		slice, ok := res.([]TestMessage)
		assert.True(t, ok)
		assert.Len(t, slice, 3)

		// Check order and content
		assert.Equal(t, "1", slice[0].ID)
		assert.Equal(t, "First Updated", slice[0].Content) // Updated
		assert.Equal(t, "2", slice[1].ID)
		assert.Equal(t, "Second", slice[1].Content) // Unchanged
		assert.Equal(t, "3", slice[2].ID)
		assert.Equal(t, "Third", slice[2].Content) // Appended
	})

	// Case 3: Mixed append (some have ID, some don't)
	// Note: In Go, slices must be of same type. So we can't easily mix types unless using []any
	// But we can test structs where ID is optional (empty string)
	t.Run("OptionalIDs", func(t *testing.T) {
		current := []TestMessage{
			{ID: "1", Content: "Msg1"},
			{ID: "", Content: "Msg2"}, // No ID
		}

		newMessages := []TestMessage{
			{ID: "1", Content: "Msg1-Updated"},
			{ID: "", Content: "Msg3"},
		}

		res, err := AddMessages(current, newMessages)
		assert.NoError(t, err)

		slice, ok := res.([]TestMessage)
		assert.True(t, ok)
		assert.Len(t, slice, 3) // 1 updated, Msg2 kept, Msg3 appended

		assert.Equal(t, "Msg1-Updated", slice[0].Content)
		assert.Equal(t, "Msg2", slice[1].Content)
		assert.Equal(t, "Msg3", slice[2].Content) // Appended
		// Note: Msg3 is at index 2 or 3 depending on implementation details of non-ID append?
		// Actually, our implementation:
		// 1. Copy current: [Msg1(id=1), Msg2(no-id)]
		// 2. Process new:
		//    - Msg1-Updated (id=1) -> Updates index 0
		//    - Msg3 (no-id) -> Appends
		// Result: [Msg1-Updated, Msg2, Msg3] -> Length 3?
		// Wait, let's re-read the code.
		// "No ID, just append" -> Yes.

		// Let's re-verify the length assertion.
		// Current: [Msg1, Msg2]
		// New: [Msg1-Upd, Msg3]
		// Result should be: [Msg1-Upd, Msg2, Msg3] -> Length 3.
		assert.Len(t, slice, 3)
	})
}
