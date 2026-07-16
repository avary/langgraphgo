package nativeopenai

import (
	goopenai "github.com/sashabaranov/go-openai"
	"github.com/smallnest/langgraphgo/llms"
)

func (o *LLM) buildRequest(messages []llms.MessageContent, opts *llms.CallOptions) goopenai.ChatCompletionRequest {
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

func toOpenAIMessages(messages []llms.MessageContent) []goopenai.ChatCompletionMessage {
	out := make([]goopenai.ChatCompletionMessage, 0, len(messages))
	for _, m := range messages {
		msg := goopenai.ChatCompletionMessage{Role: toOpenAIRole(m.Role)}
		hasImage := false
		for _, part := range m.Parts {
			if _, ok := part.(llms.ImageURLContent); ok {
				hasImage = true
				break
			}
		}
		var multi []goopenai.ChatMessagePart
		for _, part := range m.Parts {
			switch p := part.(type) {
			case llms.TextContent:
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
			case llms.ImageURLContent:
				multi = append(multi, goopenai.ChatMessagePart{
					Type:     goopenai.ChatMessagePartTypeImageURL,
					ImageURL: &goopenai.ChatMessageImageURL{URL: p.URL, Detail: goopenai.ImageURLDetail(p.Detail)},
				})
			case llms.ToolCall:
				msg.ToolCalls = append(msg.ToolCalls, toOpenAIToolCall(p))
			case llms.ToolCallResponse:
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

func toOpenAIRole(role llms.ChatMessageType) string {
	switch role {
	case llms.ChatMessageTypeSystem:
		return goopenai.ChatMessageRoleSystem
	case llms.ChatMessageTypeAI:
		return goopenai.ChatMessageRoleAssistant
	case llms.ChatMessageTypeTool:
		return goopenai.ChatMessageRoleTool
	case llms.ChatMessageTypeFunction:
		return goopenai.ChatMessageRoleFunction
	default:
		return goopenai.ChatMessageRoleUser
	}
}

func toOpenAITools(tools []llms.Tool) []goopenai.Tool {
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
	tc, ok := choice.(llms.ToolChoice)
	if !ok {
		return choice
	}
	out := goopenai.ToolChoice{Type: goopenai.ToolType(tc.Type)}
	if tc.Function != nil {
		out.Function = goopenai.ToolFunction{Name: tc.Function.Name}
	}
	return out
}

func toOpenAIToolCall(tc llms.ToolCall) goopenai.ToolCall {
	call := goopenai.ToolCall{ID: tc.ID, Type: goopenai.ToolType(tc.Type)}
	if tc.FunctionCall != nil {
		call.Function = goopenai.FunctionCall{
			Name:      tc.FunctionCall.Name,
			Arguments: tc.FunctionCall.Arguments,
		}
	}
	return call
}

func fromOpenAIToolCalls(calls []goopenai.ToolCall) []llms.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]llms.ToolCall, len(calls))
	for i, c := range calls {
		out[i] = llms.ToolCall{
			ID:   c.ID,
			Type: string(c.Type),
			FunctionCall: &llms.FunctionCall{
				Name:      c.Function.Name,
				Arguments: c.Function.Arguments,
			},
		}
	}
	return out
}
