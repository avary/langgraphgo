// Package langchainadapter bridges langchaingo's memory implementations to the
// framework's own message types. It is the sole importer of
// github.com/tmc/langchaingo in the memory tree, keeping that dependency out of
// the core memory package.
package langchainadapter

import (
	"context"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/tmc/langchaingo/llms"
	langchainmemory "github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
)

// ChatMemory is the interface for conversation memory management in langgraphgo
type ChatMemory interface {
	// SaveContext saves the context from this conversation to buffer
	SaveContext(ctx context.Context, inputValues map[string]any, outputValues map[string]any) error
	// LoadMemoryVariables loads memory variables
	LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error)
	// Clear clears memory contents
	Clear(ctx context.Context) error
	// GetMessages returns all messages in memory
	GetMessages(ctx context.Context) ([]llmtypes.ChatMessage, error)
}

// LangChainMemory adapts langchaingo's memory implementations to our ChatMemory interface
type LangChainMemory struct {
	buffer schema.Memory
}

// NewConversationBufferMemory creates a new conversation buffer memory with default settings
func NewConversationBufferMemory(options ...langchainmemory.ConversationBufferOption) *LangChainMemory {
	return &LangChainMemory{
		buffer: langchainmemory.NewConversationBuffer(options...),
	}
}

// NewConversationWindowBufferMemory creates a new conversation window buffer memory
// that keeps only the last N conversation turns
func NewConversationWindowBufferMemory(windowSize int, options ...langchainmemory.ConversationBufferOption) *LangChainMemory {
	return &LangChainMemory{
		buffer: langchainmemory.NewConversationWindowBuffer(windowSize, options...),
	}
}

// SaveContext saves the context from this conversation to buffer
func (m *LangChainMemory) SaveContext(ctx context.Context, inputValues map[string]any, outputValues map[string]any) error {
	return m.buffer.SaveContext(ctx, inputValues, outputValues)
}

// LoadMemoryVariables loads memory variables
func (m *LangChainMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return m.buffer.LoadMemoryVariables(ctx, inputs)
}

// Clear clears memory contents
func (m *LangChainMemory) Clear(ctx context.Context) error {
	return m.buffer.Clear(ctx)
}

// GetMessages returns all messages in memory
// This is a convenience method that extracts messages from the memory buffer
func (m *LangChainMemory) GetMessages(ctx context.Context) ([]llmtypes.ChatMessage, error) {
	// Load memory variables to get the conversation history
	memVars, err := m.buffer.LoadMemoryVariables(ctx, map[string]any{})
	if err != nil {
		return nil, err
	}

	// Try to get messages from any memory key
	// The default memory key is "history" for ConversationBuffer
	// but it can be customized with WithMemoryKey option
	for _, value := range memVars {
		// If return_messages is true, langchaingo returns []llms.ChatMessage,
		// which we convert into the framework's own message type.
		if messages, ok := value.([]llms.ChatMessage); ok {
			return llmtypes.FromLangchainMessages(messages), nil
		}
	}

	// If return_messages is false, history will be a string
	// In this case, we can't easily convert back to messages
	// So we return an empty slice
	return []llmtypes.ChatMessage{}, nil
}

// ChatMessageHistory provides direct access to chat message history
type ChatMessageHistory struct {
	history *langchainmemory.ChatMessageHistory
}

// NewChatMessageHistory creates a new chat message history
func NewChatMessageHistory(options ...langchainmemory.ChatMessageHistoryOption) *ChatMessageHistory {
	return &ChatMessageHistory{
		history: langchainmemory.NewChatMessageHistory(options...),
	}
}

// AddMessage adds a message to the history
func (h *ChatMessageHistory) AddMessage(ctx context.Context, message llmtypes.ChatMessage) error {
	return h.history.AddMessage(ctx, llmtypes.ToLangchainMessage(message))
}

// AddUserMessage adds a user message to the history
func (h *ChatMessageHistory) AddUserMessage(ctx context.Context, message string) error {
	return h.history.AddUserMessage(ctx, message)
}

// AddAIMessage adds an AI message to the history
func (h *ChatMessageHistory) AddAIMessage(ctx context.Context, message string) error {
	return h.history.AddAIMessage(ctx, message)
}

// Messages returns all messages in the history
func (h *ChatMessageHistory) Messages(ctx context.Context) ([]llmtypes.ChatMessage, error) {
	msgs, err := h.history.Messages(ctx)
	if err != nil {
		return nil, err
	}
	return llmtypes.FromLangchainMessages(msgs), nil
}

// Clear clears all messages from the history
func (h *ChatMessageHistory) Clear(ctx context.Context) error {
	return h.history.Clear(ctx)
}

// SetMessages sets the messages in the history
func (h *ChatMessageHistory) SetMessages(ctx context.Context, messages []llmtypes.ChatMessage) error {
	return h.history.SetMessages(ctx, llmtypes.ToLangchainMessages(messages))
}

// GetHistory returns the underlying langchaingo ChatMessageHistory
func (h *ChatMessageHistory) GetHistory() schema.ChatMessageHistory {
	return h.history
}
