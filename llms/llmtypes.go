// Package llms defines langgraphgo's own LLM message, tool, and model
// vocabulary. These types are owned by the framework and carry no dependency on
// any provider library.
//
// Interop with github.com/tmc/langchaingo lives in langchain.go, the only file
// in this package that imports langchaingo. Provider implementations are bridged
// into the framework with Wrap; the reverse conversion is ToLangchain.
package llms

import (
	"context"
	"encoding/base64"
	"errors"
)

// ErrEmptyResponse is returned when a model produces no choices.
var ErrEmptyResponse = errors.New("empty response from model")

// ChatMessageType is the role of a message.
type ChatMessageType string

// Message roles.
const (
	ChatMessageTypeAI       ChatMessageType = "ai"
	ChatMessageTypeHuman    ChatMessageType = "human"
	ChatMessageTypeSystem   ChatMessageType = "system"
	ChatMessageTypeGeneric  ChatMessageType = "generic"
	ChatMessageTypeFunction ChatMessageType = "function"
	ChatMessageTypeTool     ChatMessageType = "tool"
)

// ContentPart is one part of a message's content.
type ContentPart interface {
	isPart()
}

// TextContent is a plain-text content part.
type TextContent struct {
	Text string
}

func (tc TextContent) String() string { return tc.Text }
func (TextContent) isPart()           {}

// ImageURLContent is a content part referencing an image by URL.
type ImageURLContent struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

func (iuc ImageURLContent) String() string { return iuc.URL }
func (ImageURLContent) isPart()            {}

// BinaryContent is a content part holding binary data with a MIME type.
type BinaryContent struct {
	MIMEType string
	Data     []byte
}

func (bc BinaryContent) String() string {
	return "data:" + bc.MIMEType + ";base64," + base64.StdEncoding.EncodeToString(bc.Data)
}
func (BinaryContent) isPart() {}

// FunctionCall is the name and JSON arguments of a function invocation.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall is a model's request to invoke a tool.
type ToolCall struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	FunctionCall *FunctionCall `json:"function,omitempty"`
}

func (ToolCall) isPart() {}

// ToolCallResponse carries the result of a tool invocation back to the model.
type ToolCallResponse struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
}

func (ToolCallResponse) isPart() {}

// MessageContent is a role-tagged message made up of one or more parts.
type MessageContent struct {
	Role  ChatMessageType
	Parts []ContentPart
}

// ContentChoice is a single candidate within a ContentResponse.
type ContentChoice struct {
	Content          string
	StopReason       string
	GenerationInfo   map[string]any
	FuncCall         *FunctionCall
	ToolCalls        []ToolCall
	ReasoningContent string
}

// ContentResponse is a model's reply, possibly with multiple choices.
type ContentResponse struct {
	Choices []*ContentChoice
}

// Tool describes a tool the model may invoke.
type Tool struct {
	Type     string              `json:"type"`
	Function *FunctionDefinition `json:"function,omitempty"`
}

// FunctionDefinition declares a callable function's schema.
type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters,omitempty"`
	Strict      bool   `json:"strict,omitempty"`
}

// ToolChoice constrains which tool the model may choose.
type ToolChoice struct {
	Type     string             `json:"type"`
	Function *FunctionReference `json:"function,omitempty"`
}

// FunctionReference names a specific function (e.g. within a ToolChoice).
type FunctionReference struct {
	Name string `json:"name"`
}

// ChatMessage is a single typed chat message.
type ChatMessage interface {
	GetType() ChatMessageType
	GetContent() string
}

// AIChatMessage is a message authored by the model.
type AIChatMessage struct {
	Content          string        `json:"content,omitempty"`
	FunctionCall     *FunctionCall `json:"function_call,omitempty"`
	ToolCalls        []ToolCall    `json:"tool_calls,omitempty"`
	ReasoningContent string        `json:"reasoning_content,omitempty"`
}

func (m AIChatMessage) GetType() ChatMessageType { return ChatMessageTypeAI }
func (m AIChatMessage) GetContent() string       { return m.Content }

// HumanChatMessage is a message authored by the user.
type HumanChatMessage struct {
	Content string
}

func (m HumanChatMessage) GetType() ChatMessageType { return ChatMessageTypeHuman }
func (m HumanChatMessage) GetContent() string       { return m.Content }

// SystemChatMessage is a system-role instruction message.
type SystemChatMessage struct {
	Content string
}

func (m SystemChatMessage) GetType() ChatMessageType { return ChatMessageTypeSystem }
func (m SystemChatMessage) GetContent() string       { return m.Content }

// Model is the interface implemented by every LLM provider used by the
// framework. Provider libraries are adapted to this interface via Wrap.
type Model interface {
	GenerateContent(ctx context.Context, messages []MessageContent, options ...CallOption) (*ContentResponse, error)
	Call(ctx context.Context, prompt string, options ...CallOption) (string, error)
}

// CallOptions is the resolved set of options for a generation call.
type CallOptions struct {
	Model                  string
	CandidateCount         int
	MaxTokens              int
	Temperature            float64
	StopWords              []string
	StreamingFunc          func(ctx context.Context, chunk []byte) error
	StreamingReasoningFunc func(ctx context.Context, reasoningChunk, chunk []byte) error
	TopK                   int
	TopP                   float64
	Seed                   int
	MinLength              int
	MaxLength              int
	N                      int
	RepetitionPenalty      float64
	FrequencyPenalty       float64
	PresencePenalty        float64
	JSONMode               bool
	Tools                  []Tool
	ToolChoice             any
	Metadata               map[string]any
	ResponseMIMEType       string
}

// CallOption configures a single generation call.
type CallOption func(*CallOptions)

// WithModel sets the model name.
func WithModel(model string) CallOption {
	return func(o *CallOptions) { o.Model = model }
}

// WithMaxTokens caps the number of generated tokens.
func WithMaxTokens(maxTokens int) CallOption {
	return func(o *CallOptions) { o.MaxTokens = maxTokens }
}

// WithTemperature sets the sampling temperature.
func WithTemperature(temperature float64) CallOption {
	return func(o *CallOptions) { o.Temperature = temperature }
}

// WithTopP sets nucleus-sampling probability mass.
func WithTopP(topP float64) CallOption {
	return func(o *CallOptions) { o.TopP = topP }
}

// WithTopK sets top-k sampling.
func WithTopK(topK int) CallOption {
	return func(o *CallOptions) { o.TopK = topK }
}

// WithStopWords sets stop sequences.
func WithStopWords(stopWords []string) CallOption {
	return func(o *CallOptions) { o.StopWords = stopWords }
}

// WithSeed sets a deterministic sampling seed.
func WithSeed(seed int) CallOption {
	return func(o *CallOptions) { o.Seed = seed }
}

// WithN sets how many choices to generate.
func WithN(n int) CallOption {
	return func(o *CallOptions) { o.N = n }
}

// WithJSONMode enables JSON output mode.
func WithJSONMode() CallOption {
	return func(o *CallOptions) { o.JSONMode = true }
}

// WithTools supplies the tools available to the call.
func WithTools(tools []Tool) CallOption {
	return func(o *CallOptions) { o.Tools = tools }
}

// WithToolChoice constrains tool selection.
func WithToolChoice(choice any) CallOption {
	return func(o *CallOptions) { o.ToolChoice = choice }
}

// WithStreamingFunc registers a callback for streamed output chunks.
func WithStreamingFunc(fn func(ctx context.Context, chunk []byte) error) CallOption {
	return func(o *CallOptions) { o.StreamingFunc = fn }
}

// TextPart builds a plain-text content part.
func TextPart(s string) TextContent {
	return TextContent{Text: s}
}

// TextParts builds a MessageContent from a role and one or more text strings.
func TextParts(role ChatMessageType, parts ...string) MessageContent {
	result := MessageContent{Role: role, Parts: []ContentPart{}}
	for _, part := range parts {
		result.Parts = append(result.Parts, TextPart(part))
	}
	return result
}

// GenerateFromSinglePrompt calls a model with a single string prompt and returns
// the first text choice.
func GenerateFromSinglePrompt(ctx context.Context, llm Model, prompt string, options ...CallOption) (string, error) {
	msg := MessageContent{
		Role:  ChatMessageTypeHuman,
		Parts: []ContentPart{TextContent{Text: prompt}},
	}
	resp, err := llm.GenerateContent(ctx, []MessageContent{msg}, options...)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) < 1 {
		return "", ErrEmptyResponse
	}
	return resp.Choices[0].Content, nil
}
