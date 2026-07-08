package nativeopenai

import (
	"context"
	"errors"
	"io"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/smallnest/langgraphgo/llmtypes"
)

// Call generates a completion from a single prompt.
func (o *LLM) Call(ctx context.Context, prompt string, options ...llmtypes.CallOption) (string, error) {
	return llmtypes.GenerateFromSinglePrompt(ctx, o, prompt, options...)
}

// GenerateContent generates a completion from a sequence of messages.
func (o *LLM) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	opts := llmtypes.CallOptions{Model: o.model}
	for _, apply := range options {
		apply(&opts)
	}

	req := o.buildRequest(messages, &opts)

	if opts.StreamingFunc != nil {
		return o.generateStreaming(ctx, req, opts.StreamingFunc)
	}
	return o.generate(ctx, req)
}

func (o *LLM) generate(ctx context.Context, req goopenai.ChatCompletionRequest) (*llmtypes.ContentResponse, error) {
	req.Stream = false
	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, llmtypes.ErrEmptyResponse
	}

	out := &llmtypes.ContentResponse{Choices: make([]*llmtypes.ContentChoice, len(resp.Choices))}
	for i, c := range resp.Choices {
		out.Choices[i] = &llmtypes.ContentChoice{
			Content:          c.Message.Content,
			StopReason:       string(c.FinishReason),
			ReasoningContent: c.Message.ReasoningContent,
			ToolCalls:        fromOpenAIToolCalls(c.Message.ToolCalls),
			GenerationInfo: map[string]any{
				"prompt_tokens":     resp.Usage.PromptTokens,
				"completion_tokens": resp.Usage.CompletionTokens,
				"total_tokens":      resp.Usage.TotalTokens,
			},
		}
	}
	return out, nil
}

func (o *LLM) generateStreaming(ctx context.Context, req goopenai.ChatCompletionRequest, streamingFunc func(context.Context, []byte) error) (*llmtypes.ContentResponse, error) {
	req.Stream = true
	stream, err := o.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var (
		content    string
		reasoning  string
		stopReason string
	)
	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return nil, recvErr
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		choice := chunk.Choices[0]
		reasoning += choice.Delta.ReasoningContent
		if delta := choice.Delta.Content; delta != "" {
			content += delta
			if streamErr := streamingFunc(ctx, []byte(delta)); streamErr != nil {
				return nil, streamErr
			}
		}
		if choice.FinishReason != "" {
			stopReason = string(choice.FinishReason)
		}
	}

	return &llmtypes.ContentResponse{
		Choices: []*llmtypes.ContentChoice{{
			Content:          content,
			StopReason:       stopReason,
			ReasoningContent: reasoning,
		}},
	}, nil
}
