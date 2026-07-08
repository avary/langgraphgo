package llmtypes

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

// Wrap adapts a langchaingo llms.Model into the framework's Model interface.
func Wrap(m llms.Model) Model {
	if m == nil {
		return nil
	}
	return &langchainModel{inner: m}
}

// ToLangchain converts a framework Model back into a langchaingo llms.Model.
// If m was produced by Wrap, the original provider is returned unchanged.
func ToLangchain(m Model) llms.Model {
	if m == nil {
		return nil
	}
	if lm, ok := m.(*langchainModel); ok {
		return lm.inner
	}
	return &reverseModel{inner: m}
}

// langchainModel adapts an llms.Model to the framework Model interface.
type langchainModel struct {
	inner llms.Model
}

func (m *langchainModel) Unwrap() llms.Model { return m.inner }

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

// reverseModel adapts a framework Model to the llms.Model interface, for code
// paths that still hand a model to langchaingo.
type reverseModel struct {
	inner Model
}

func (m *reverseModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	resp, err := m.inner.GenerateContent(ctx, fromLLMSMessages(messages), fromLLMSOptions(options)...)
	if err != nil {
		return nil, err
	}
	return toLLMSResponse(resp), nil
}

func (m *reverseModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return m.inner.Call(ctx, prompt, fromLLMSOptions(options)...)
}

// --- message conversion ---

func toLLMSMessages(msgs []MessageContent) []llms.MessageContent {
	out := make([]llms.MessageContent, len(msgs))
	for i, m := range msgs {
		out[i] = llms.MessageContent{
			Role:  llms.ChatMessageType(m.Role),
			Parts: toLLMSParts(m.Parts),
		}
	}
	return out
}

func fromLLMSMessages(msgs []llms.MessageContent) []MessageContent {
	out := make([]MessageContent, len(msgs))
	for i, m := range msgs {
		out[i] = MessageContent{
			Role:  ChatMessageType(m.Role),
			Parts: fromLLMSParts(m.Parts),
		}
	}
	return out
}

func toLLMSParts(parts []ContentPart) []llms.ContentPart {
	out := make([]llms.ContentPart, len(parts))
	for i, p := range parts {
		switch v := p.(type) {
		case TextContent:
			out[i] = llms.TextContent{Text: v.Text}
		case ImageURLContent:
			out[i] = llms.ImageURLContent{URL: v.URL, Detail: v.Detail}
		case BinaryContent:
			out[i] = llms.BinaryContent{MIMEType: v.MIMEType, Data: v.Data}
		case ToolCall:
			out[i] = toLLMSToolCall(v)
		case ToolCallResponse:
			out[i] = llms.ToolCallResponse{ToolCallID: v.ToolCallID, Name: v.Name, Content: v.Content}
		default:
			out[i] = llms.TextContent{Text: ""}
		}
	}
	return out
}

func fromLLMSParts(parts []llms.ContentPart) []ContentPart {
	out := make([]ContentPart, len(parts))
	for i, p := range parts {
		switch v := p.(type) {
		case llms.TextContent:
			out[i] = TextContent{Text: v.Text}
		case llms.ImageURLContent:
			out[i] = ImageURLContent{URL: v.URL, Detail: v.Detail}
		case llms.BinaryContent:
			out[i] = BinaryContent{MIMEType: v.MIMEType, Data: v.Data}
		case llms.ToolCall:
			out[i] = fromLLMSToolCall(v)
		case llms.ToolCallResponse:
			out[i] = ToolCallResponse{ToolCallID: v.ToolCallID, Name: v.Name, Content: v.Content}
		default:
			out[i] = TextContent{Text: ""}
		}
	}
	return out
}

// --- tool call / function conversion ---

func toLLMSToolCall(tc ToolCall) llms.ToolCall {
	return llms.ToolCall{ID: tc.ID, Type: tc.Type, FunctionCall: toLLMSFuncCall(tc.FunctionCall)}
}

func fromLLMSToolCall(tc llms.ToolCall) ToolCall {
	return ToolCall{ID: tc.ID, Type: tc.Type, FunctionCall: fromLLMSFuncCall(tc.FunctionCall)}
}

func toLLMSFuncCall(fc *FunctionCall) *llms.FunctionCall {
	if fc == nil {
		return nil
	}
	return &llms.FunctionCall{Name: fc.Name, Arguments: fc.Arguments}
}

func fromLLMSFuncCall(fc *llms.FunctionCall) *FunctionCall {
	if fc == nil {
		return nil
	}
	return &FunctionCall{Name: fc.Name, Arguments: fc.Arguments}
}

func toLLMSTools(tools []Tool) []llms.Tool {
	if tools == nil {
		return nil
	}
	out := make([]llms.Tool, len(tools))
	for i, t := range tools {
		out[i] = llms.Tool{Type: t.Type}
		if t.Function != nil {
			out[i].Function = &llms.FunctionDefinition{
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
	out := llms.ToolChoice{Type: tc.Type}
	if tc.Function != nil {
		out.Function = &llms.FunctionReference{Name: tc.Function.Name}
	}
	return out
}

// --- response conversion ---

func fromLLMSResponse(r *llms.ContentResponse) *ContentResponse {
	if r == nil {
		return nil
	}
	out := &ContentResponse{Choices: make([]*ContentChoice, len(r.Choices))}
	for i, c := range r.Choices {
		out.Choices[i] = fromLLMSChoice(c)
	}
	return out
}

func toLLMSResponse(r *ContentResponse) *llms.ContentResponse {
	if r == nil {
		return nil
	}
	out := &llms.ContentResponse{Choices: make([]*llms.ContentChoice, len(r.Choices))}
	for i, c := range r.Choices {
		out.Choices[i] = toLLMSChoice(c)
	}
	return out
}

func fromLLMSChoice(c *llms.ContentChoice) *ContentChoice {
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

func toLLMSChoice(c *ContentChoice) *llms.ContentChoice {
	if c == nil {
		return nil
	}
	out := &llms.ContentChoice{
		Content:          c.Content,
		StopReason:       c.StopReason,
		GenerationInfo:   c.GenerationInfo,
		FuncCall:         toLLMSFuncCall(c.FuncCall),
		ReasoningContent: c.ReasoningContent,
	}
	if c.ToolCalls != nil {
		out.ToolCalls = make([]llms.ToolCall, len(c.ToolCalls))
		for i, tc := range c.ToolCalls {
			out.ToolCalls[i] = toLLMSToolCall(tc)
		}
	}
	return out
}

// --- option conversion ---

func toLLMSOptions(opts []CallOption) []llms.CallOption {
	var co CallOptions
	for _, o := range opts {
		o(&co)
	}
	return []llms.CallOption{func(lo *llms.CallOptions) {
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

func fromLLMSOptions(opts []llms.CallOption) []CallOption {
	var lo llms.CallOptions
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

func fromLLMSTools(tools []llms.Tool) []Tool {
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
func FromLangchainMessages(msgs []llms.ChatMessage) []ChatMessage {
	out := make([]ChatMessage, 0, len(msgs))
	for _, m := range msgs {
		switch m.GetType() {
		case llms.ChatMessageTypeAI:
			out = append(out, AIChatMessage{Content: m.GetContent()})
		case llms.ChatMessageTypeSystem:
			out = append(out, SystemChatMessage{Content: m.GetContent()})
		default:
			out = append(out, HumanChatMessage{Content: m.GetContent()})
		}
	}
	return out
}

// ToLangchainMessage converts a framework chat message to a langchaingo one.
func ToLangchainMessage(m ChatMessage) llms.ChatMessage {
	switch v := m.(type) {
	case AIChatMessage:
		return llms.AIChatMessage{
			Content:          v.Content,
			FunctionCall:     toLLMSFuncCall(v.FunctionCall),
			ReasoningContent: v.ReasoningContent,
		}
	case SystemChatMessage:
		return llms.SystemChatMessage{Content: v.Content}
	case HumanChatMessage:
		return llms.HumanChatMessage{Content: v.Content}
	default:
		switch m.GetType() {
		case ChatMessageTypeAI:
			return llms.AIChatMessage{Content: m.GetContent()}
		case ChatMessageTypeSystem:
			return llms.SystemChatMessage{Content: m.GetContent()}
		default:
			return llms.HumanChatMessage{Content: m.GetContent()}
		}
	}
}

// ToLangchainMessages converts a slice of framework chat messages.
func ToLangchainMessages(msgs []ChatMessage) []llms.ChatMessage {
	out := make([]llms.ChatMessage, len(msgs))
	for i, m := range msgs {
		out[i] = ToLangchainMessage(m)
	}
	return out
}
