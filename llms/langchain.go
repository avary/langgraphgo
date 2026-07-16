package llms

import (
	"context"

	lcllms "github.com/tmc/langchaingo/llms"
)

// Wrap adapts a langchaingo lcllms.Model into the framework's Model interface.
func Wrap(m lcllms.Model) Model {
	if m == nil {
		return nil
	}
	return &langchainModel{inner: m}
}

// ToLangchain converts a framework Model back into a langchaingo lcllms.Model.
// If m was produced by Wrap, the original provider is returned unchanged.
func ToLangchain(m Model) lcllms.Model {
	if m == nil {
		return nil
	}
	if lm, ok := m.(*langchainModel); ok {
		return lm.inner
	}
	return &reverseModel{inner: m}
}

// langchainModel adapts an lcllms.Model to the framework Model interface.
type langchainModel struct {
	inner lcllms.Model
}

func (m *langchainModel) Unwrap() lcllms.Model { return m.inner }

func (m *langchainModel) GenerateContent(ctx context.Context, messages []MessageContent, options ...CallOption) (*ContentResponse, error) {
	resp, err := m.inner.GenerateContent(ctx, toLLMSMessages(messages), toLLMSOptions(options)...)
	if err != nil {
		return nil, err
	}
	return fromLLMSResponse(resp), nil
}

func (m *langchainModel) Call(ctx context.Context, prompt string, options ...CallOption) (string, error) {
	return GenerateFromSinglePrompt(ctx, m, prompt, options...)
}

// reverseModel adapts a framework Model to the lcllms.Model interface, for code
// paths that still hand a model to langchaingo.
type reverseModel struct {
	inner Model
}

func (m *reverseModel) GenerateContent(ctx context.Context, messages []lcllms.MessageContent, options ...lcllms.CallOption) (*lcllms.ContentResponse, error) {
	resp, err := m.inner.GenerateContent(ctx, fromLLMSMessages(messages), fromLLMSOptions(options)...)
	if err != nil {
		return nil, err
	}
	return toLLMSResponse(resp), nil
}

func (m *reverseModel) Call(ctx context.Context, prompt string, options ...lcllms.CallOption) (string, error) {
	return m.inner.Call(ctx, prompt, fromLLMSOptions(options)...)
}

// --- message conversion ---

func toLLMSMessages(msgs []MessageContent) []lcllms.MessageContent {
	out := make([]lcllms.MessageContent, len(msgs))
	for i, m := range msgs {
		out[i] = lcllms.MessageContent{
			Role:  lcllms.ChatMessageType(m.Role),
			Parts: toLLMSParts(m.Parts),
		}
	}
	return out
}

func fromLLMSMessages(msgs []lcllms.MessageContent) []MessageContent {
	out := make([]MessageContent, len(msgs))
	for i, m := range msgs {
		out[i] = MessageContent{
			Role:  ChatMessageType(m.Role),
			Parts: fromLLMSParts(m.Parts),
		}
	}
	return out
}

func toLLMSParts(parts []ContentPart) []lcllms.ContentPart {
	out := make([]lcllms.ContentPart, len(parts))
	for i, p := range parts {
		switch v := p.(type) {
		case TextContent:
			out[i] = lcllms.TextContent{Text: v.Text}
		case ImageURLContent:
			out[i] = lcllms.ImageURLContent{URL: v.URL, Detail: v.Detail}
		case BinaryContent:
			out[i] = lcllms.BinaryContent{MIMEType: v.MIMEType, Data: v.Data}
		case ToolCall:
			out[i] = toLLMSToolCall(v)
		case ToolCallResponse:
			out[i] = lcllms.ToolCallResponse{ToolCallID: v.ToolCallID, Name: v.Name, Content: v.Content}
		default:
			out[i] = lcllms.TextContent{Text: ""}
		}
	}
	return out
}

func fromLLMSParts(parts []lcllms.ContentPart) []ContentPart {
	out := make([]ContentPart, len(parts))
	for i, p := range parts {
		switch v := p.(type) {
		case lcllms.TextContent:
			out[i] = TextContent{Text: v.Text}
		case lcllms.ImageURLContent:
			out[i] = ImageURLContent{URL: v.URL, Detail: v.Detail}
		case lcllms.BinaryContent:
			out[i] = BinaryContent{MIMEType: v.MIMEType, Data: v.Data}
		case lcllms.ToolCall:
			out[i] = fromLLMSToolCall(v)
		case lcllms.ToolCallResponse:
			out[i] = ToolCallResponse{ToolCallID: v.ToolCallID, Name: v.Name, Content: v.Content}
		default:
			out[i] = TextContent{Text: ""}
		}
	}
	return out
}

// --- tool call / function conversion ---

func toLLMSToolCall(tc ToolCall) lcllms.ToolCall {
	return lcllms.ToolCall{ID: tc.ID, Type: tc.Type, FunctionCall: toLLMSFuncCall(tc.FunctionCall)}
}

func fromLLMSToolCall(tc lcllms.ToolCall) ToolCall {
	return ToolCall{ID: tc.ID, Type: tc.Type, FunctionCall: fromLLMSFuncCall(tc.FunctionCall)}
}

func toLLMSFuncCall(fc *FunctionCall) *lcllms.FunctionCall {
	if fc == nil {
		return nil
	}
	return &lcllms.FunctionCall{Name: fc.Name, Arguments: fc.Arguments}
}

func fromLLMSFuncCall(fc *lcllms.FunctionCall) *FunctionCall {
	if fc == nil {
		return nil
	}
	return &FunctionCall{Name: fc.Name, Arguments: fc.Arguments}
}

func toLLMSTools(tools []Tool) []lcllms.Tool {
	if tools == nil {
		return nil
	}
	out := make([]lcllms.Tool, len(tools))
	for i, t := range tools {
		out[i] = lcllms.Tool{Type: t.Type}
		if t.Function != nil {
			out[i].Function = &lcllms.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
				Strict:      t.Function.Strict,
			}
		}
	}
	return out
}

func toLLMSToolChoice(choice any) any {
	tc, ok := choice.(ToolChoice)
	if !ok {
		return choice
	}
	out := lcllms.ToolChoice{Type: tc.Type}
	if tc.Function != nil {
		out.Function = &lcllms.FunctionReference{Name: tc.Function.Name}
	}
	return out
}

// --- response conversion ---

func fromLLMSResponse(r *lcllms.ContentResponse) *ContentResponse {
	if r == nil {
		return nil
	}
	out := &ContentResponse{Choices: make([]*ContentChoice, len(r.Choices))}
	for i, c := range r.Choices {
		out.Choices[i] = fromLLMSChoice(c)
	}
	return out
}

func toLLMSResponse(r *ContentResponse) *lcllms.ContentResponse {
	if r == nil {
		return nil
	}
	out := &lcllms.ContentResponse{Choices: make([]*lcllms.ContentChoice, len(r.Choices))}
	for i, c := range r.Choices {
		out.Choices[i] = toLLMSChoice(c)
	}
	return out
}

func fromLLMSChoice(c *lcllms.ContentChoice) *ContentChoice {
	if c == nil {
		return nil
	}
	out := &ContentChoice{
		Content:          c.Content,
		StopReason:       c.StopReason,
		GenerationInfo:   c.GenerationInfo,
		FuncCall:         fromLLMSFuncCall(c.FuncCall),
		ReasoningContent: c.ReasoningContent,
	}
	if c.ToolCalls != nil {
		out.ToolCalls = make([]ToolCall, len(c.ToolCalls))
		for i, tc := range c.ToolCalls {
			out.ToolCalls[i] = fromLLMSToolCall(tc)
		}
	}
	return out
}

func toLLMSChoice(c *ContentChoice) *lcllms.ContentChoice {
	if c == nil {
		return nil
	}
	out := &lcllms.ContentChoice{
		Content:          c.Content,
		StopReason:       c.StopReason,
		GenerationInfo:   c.GenerationInfo,
		FuncCall:         toLLMSFuncCall(c.FuncCall),
		ReasoningContent: c.ReasoningContent,
	}
	if c.ToolCalls != nil {
		out.ToolCalls = make([]lcllms.ToolCall, len(c.ToolCalls))
		for i, tc := range c.ToolCalls {
			out.ToolCalls[i] = toLLMSToolCall(tc)
		}
	}
	return out
}

// --- option conversion ---

func toLLMSOptions(opts []CallOption) []lcllms.CallOption {
	var co CallOptions
	for _, o := range opts {
		o(&co)
	}
	return []lcllms.CallOption{func(lo *lcllms.CallOptions) {
		lo.Model = co.Model
		lo.CandidateCount = co.CandidateCount
		lo.MaxTokens = co.MaxTokens
		lo.Temperature = co.Temperature
		lo.StopWords = co.StopWords
		lo.StreamingFunc = co.StreamingFunc
		lo.StreamingReasoningFunc = co.StreamingReasoningFunc
		lo.TopK = co.TopK
		lo.TopP = co.TopP
		lo.Seed = co.Seed
		lo.MinLength = co.MinLength
		lo.MaxLength = co.MaxLength
		lo.N = co.N
		lo.RepetitionPenalty = co.RepetitionPenalty
		lo.FrequencyPenalty = co.FrequencyPenalty
		lo.PresencePenalty = co.PresencePenalty
		lo.JSONMode = co.JSONMode
		lo.Tools = toLLMSTools(co.Tools)
		lo.ToolChoice = toLLMSToolChoice(co.ToolChoice)
		lo.Metadata = co.Metadata
		lo.ResponseMIMEType = co.ResponseMIMEType
	}}
}

func fromLLMSOptions(opts []lcllms.CallOption) []CallOption {
	var lo lcllms.CallOptions
	for _, o := range opts {
		o(&lo)
	}
	return []CallOption{func(co *CallOptions) {
		co.Model = lo.Model
		co.CandidateCount = lo.CandidateCount
		co.MaxTokens = lo.MaxTokens
		co.Temperature = lo.Temperature
		co.StopWords = lo.StopWords
		co.StreamingFunc = lo.StreamingFunc
		co.StreamingReasoningFunc = lo.StreamingReasoningFunc
		co.TopK = lo.TopK
		co.TopP = lo.TopP
		co.Seed = lo.Seed
		co.MinLength = lo.MinLength
		co.MaxLength = lo.MaxLength
		co.N = lo.N
		co.RepetitionPenalty = lo.RepetitionPenalty
		co.FrequencyPenalty = lo.FrequencyPenalty
		co.PresencePenalty = lo.PresencePenalty
		co.JSONMode = lo.JSONMode
		co.Tools = fromLLMSTools(lo.Tools)
		co.ToolChoice = lo.ToolChoice
		co.Metadata = lo.Metadata
		co.ResponseMIMEType = lo.ResponseMIMEType
	}}
}

func fromLLMSTools(tools []lcllms.Tool) []Tool {
	if tools == nil {
		return nil
	}
	out := make([]Tool, len(tools))
	for i, t := range tools {
		out[i] = Tool{Type: t.Type}
		if t.Function != nil {
			out[i].Function = &FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
				Strict:      t.Function.Strict,
			}
		}
	}
	return out
}

// FromLangchainMessages converts langchaingo chat messages to framework ones.
func FromLangchainMessages(msgs []lcllms.ChatMessage) []ChatMessage {
	out := make([]ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		switch m.GetType() {
		case lcllms.ChatMessageTypeAI:
			out = append(out, AIChatMessage{Content: m.GetContent()})
		case lcllms.ChatMessageTypeSystem:
			out = append(out, SystemChatMessage{Content: m.GetContent()})
		default:
			out = append(out, HumanChatMessage{Content: m.GetContent()})
		}
	}
	return out
}

// ToLangchainMessage converts a framework chat message to a langchaingo one.
func ToLangchainMessage(m ChatMessage) lcllms.ChatMessage {
	switch v := m.(type) {
	case AIChatMessage:
		return lcllms.AIChatMessage{
			Content:          v.Content,
			FunctionCall:     toLLMSFuncCall(v.FunctionCall),
			ReasoningContent: v.ReasoningContent,
		}
	case SystemChatMessage:
		return lcllms.SystemChatMessage{Content: v.Content}
	case HumanChatMessage:
		return lcllms.HumanChatMessage{Content: v.Content}
	default:
		switch m.GetType() {
		case ChatMessageTypeAI:
			return lcllms.AIChatMessage{Content: m.GetContent()}
		case ChatMessageTypeSystem:
			return lcllms.SystemChatMessage{Content: m.GetContent()}
		default:
			return lcllms.HumanChatMessage{Content: m.GetContent()}
		}
	}
}

// ToLangchainMessages converts a slice of framework chat messages.
func ToLangchainMessages(msgs []ChatMessage) []lcllms.ChatMessage {
	out := make([]lcllms.ChatMessage, len(msgs))
	for i, m := range msgs {
		out[i] = ToLangchainMessage(m)
	}
	return out
}
