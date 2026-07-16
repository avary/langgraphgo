package adapter

import (
	"context"

	"github.com/smallnest/langgraphgo/llms"
)

// OpenAIAdapter adapts langchaingo's LLM to a simple interface
type OpenAIAdapter struct {
	llm llms.Model
}

// NewOpenAIAdapter creates a new adapter for OpenAI LLM
func NewOpenAIAdapter(llm llms.Model) *OpenAIAdapter {
	return &OpenAIAdapter{
		llm: llm,
	}
}

// Generate implements the simple generation interface
func (o *OpenAIAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, o.llm, prompt)
}

// GenerateWithConfig implements the simple generation interface with configuration
func (o *OpenAIAdapter) GenerateWithConfig(ctx context.Context, prompt string, config map[string]any) (string, error) {
	var options []llms.CallOption
	if temp, ok := config["temperature"].(float64); ok {
		options = append(options, llms.WithTemperature(temp))
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		options = append(options, llms.WithMaxTokens(maxTokens))
	}

	return llms.GenerateFromSinglePrompt(ctx, o.llm, prompt, options...)
}

// GenerateWithSystem implements the simple generation interface with system prompt
func (o *OpenAIAdapter) GenerateWithSystem(ctx context.Context, system, prompt string) (string, error) {
	// GenerateWithSystem involves multiple messages, so we use GenerateContent
	response, err := o.llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, system),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	})
	if err != nil {
		return "", err
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Content, nil
	}
	return "", nil
}

// StreamingLLM wraps an llms.Model to add streaming capability
type StreamingLLM struct {
	llms.Model
	streamCallback func(chunk string)
}

// GenerateContent implements the streaming generation with llms.Model interface
func (s *StreamingLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if s.streamCallback == nil {
		return s.Model.GenerateContent(ctx, messages, options...)
	}

	// Add streaming function to the options
	options = append(options, llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
		s.streamCallback(string(chunk))
		return nil
	}))

	// Call the original LLM with modified options
	return s.Model.GenerateContent(ctx, messages, options...)
}

func WrapLLMWithStreaming(llm llms.Model, streamCallback func(chunk string)) *StreamingLLM {
	return &StreamingLLM{
		Model:          llm,
		streamCallback: streamCallback,
	}
}
