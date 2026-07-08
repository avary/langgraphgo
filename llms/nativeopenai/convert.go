package nativeopenai

import (
	goopenai "github.com/sashabaranov/go-openai"
	"github.com/smallnest/langgraphgo/llmtypes"
)

func (o *LLM) buildRequest(messages []llmtypes.MessageContent, opts *llmtypes.CallOptions) goopenai.ChatCompletionRequest {
	req := goopenai.ChatCompletionRequest{
		Model:            opts.Model,
		Messages:         toOpenAIMessages(messages),
		MaxTokens:        opts.MaxTokens,
		Temperature:      float32(opts.Temperature),
		TopP:             float32(opts.TopP),
		N:                opts.N,
		Stop:             opts.StopWords,
		FrequencyPenalty: float32(opts.FrequencyPenalty),
		PresencePenalty:  float32(opts.PresencePenalty),
	}
	if opts.Seed != 0 {
		seed := opts.Seed
		req.Seed = &seed
	}
	if opts.JSONMode {
		req.ResponseFormat = &goopenai.ChatCompletionResponseFormat{
			Type: goopenai.ChatCompletionResponseFormatTypeJSONObject,
		}
	}
	if len(opts.Tools) > 0 {
		req.Tools = toOpenAITools(opts.Tools)
	}
	if opts.ToolChoice != nil {
		req.ToolChoice = toOpenAIToolChoice(opts.ToolChoice)
	}
	return req
}

func toOpenAIMessages(messages []llmtypes.MessageContent) []goopenai.ChatCompletionMessage {
	out := make([]goopenai.ChatCompletionMessage, 0, len(messages))
	for _, m := range messages {
		msg := goopenai.ChatCompletionMessage{Role: toOpenAIRole(m.Role)}
		hasImage := false
		for _, part := range m.Parts {
			if _, ok := part.(llmtypes.ImageURLContent); ok {
				hasImage = true
				break
			}
		}
		var multi []goopenai.ChatMessagePart
		for _, part := range m.Parts {
			switch p := part.(type) {
			case llmtypes.TextContent:
				// Only role "user" accepts array content; for text-only
				// messages (and any assistant/tool message) keep a string
				// body so tool-call messages stay API-valid.
				if hasImage {
					multi = append(multi, goopenai.ChatMessagePart{
						Type: goopenai.ChatMessagePartTypeText,
						Text: p.Text,
					})
				} else {
					msg.Content += p.Text
				}
			case llmtypes.ImageURLContent:
				multi = append(multi, goopenai.ChatMessagePart{
					Type:     goopenai.ChatMessagePartTypeImageURL,
					ImageURL: &goopenai.ChatMessageImageURL{URL: p.URL, Detail: goopenai.ImageURLDetail(p.Detail)},
				})
			case llmtypes.ToolCall:
				msg.ToolCalls = append(msg.ToolCalls, toOpenAIToolCall(p))
			case llmtypes.ToolCallResponse:
				msg.ToolCallID = p.ToolCallID
				msg.Name = p.Name
				msg.Content = p.Content
			}
		}
		if len(multi) > 0 {
			msg.MultiContent = multi
		}
		out = append(out, msg)
	}
	return out
}

func toOpenAIRole(role llmtypes.ChatMessageType) string {
	switch role {
	case llmtypes.ChatMessageTypeSystem:
		return goopenai.ChatMessageRoleSystem
	case llmtypes.ChatMessageTypeAI:
		return goopenai.ChatMessageRoleAssistant
	case llmtypes.ChatMessageTypeTool:
		return goopenai.ChatMessageRoleTool
	case llmtypes.ChatMessageTypeFunction:
		return goopenai.ChatMessageRoleFunction
	default:
		return goopenai.ChatMessageRoleUser
	}
}

func toOpenAITools(tools []llmtypes.Tool) []goopenai.Tool {
	out := make([]goopenai.Tool, 0, len(tools))
	for _, t := range tools {
		ot := goopenai.Tool{Type: goopenai.ToolType(t.Type)}
		if t.Function != nil {
			ot.Function = &goopenai.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Strict:      t.Function.Strict,
				Parameters:  t.Function.Parameters,
			}
		}
		out = append(out, ot)
	}
	return out
}

func toOpenAIToolChoice(choice any) any {
	tc, ok := choice.(llmtypes.ToolChoice)
	if !ok {
		return choice
	}
	out := goopenai.ToolChoice{Type: goopenai.ToolType(tc.Type)}
	if tc.Function != nil {
		out.Function = goopenai.ToolFunction{Name: tc.Function.Name}
	}
	return out
}

func toOpenAIToolCall(tc llmtypes.ToolCall) goopenai.ToolCall {
	call := goopenai.ToolCall{ID: tc.ID, Type: goopenai.ToolType(tc.Type)}
	if tc.FunctionCall != nil {
		call.Function = goopenai.FunctionCall{
			Name:      tc.FunctionCall.Name,
			Arguments: tc.FunctionCall.Arguments,
		}
	}
	return call
}

func fromOpenAIToolCalls(calls []goopenai.ToolCall) []llmtypes.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]llmtypes.ToolCall, len(calls))
	for i, c := range calls {
		out[i] = llmtypes.ToolCall{
			ID:   c.ID,
			Type: string(c.Type),
			FunctionCall: &llmtypes.FunctionCall{
				Name:      c.Function.Name,
				Arguments: c.Function.Arguments,
			},
		}
	}
	return out
}
