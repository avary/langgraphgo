// Package message defines the canonical message, tool, and model vocabulary
// used throughout langgraphgo. It acts as an anti-corruption layer between the
// framework and its underlying LLM provider library.
//
// At present every identifier is a thin alias over github.com/tmc/langchaingo.
// This flips the dependency direction — framework code imports this package
// instead of langchaingo directly — so the upstream types can later be replaced
// with owned structs plus an adapter, without touching call sites.
package llmtypes

import "github.com/tmc/langchaingo/llms"

// Core message model.
type (
	// MessageContent is a single role-tagged message made up of one or more parts.
	MessageContent = llms.MessageContent
	// ContentPart is one piece of a message (text, image, tool call, ...).
	ContentPart = llms.ContentPart
	// TextContent is a plain-text content part.
	TextContent = llms.TextContent
	// ContentResponse is a model's reply, possibly with multiple choices.
	ContentResponse = llms.ContentResponse
	// ContentChoice is a single candidate within a ContentResponse.
	ContentChoice = llms.ContentChoice
	// ChatMessageType is the role of a message (human, ai, system, tool).
	ChatMessageType = llms.ChatMessageType
	// ChatMessage is the interface for a single typed chat message.
	ChatMessage = llms.ChatMessage
	// AIChatMessage is a message authored by the model.
	AIChatMessage = llms.AIChatMessage
	// HumanChatMessage is a message authored by the user.
	HumanChatMessage = llms.HumanChatMessage
	// SystemChatMessage is a system-role instruction message.
	SystemChatMessage = llms.SystemChatMessage
)

// Tool-calling model.
type (
	// Tool describes a tool that a model may invoke.
	Tool = llms.Tool
	// ToolCall is a model's request to invoke a tool.
	ToolCall = llms.ToolCall
	// ToolCallResponse carries the result of a tool invocation back to the model.
	ToolCallResponse = llms.ToolCallResponse
	// ToolChoice constrains which tool the model may choose.
	ToolChoice = llms.ToolChoice
	// FunctionCall is the function name and arguments of a ToolCall.
	FunctionCall = llms.FunctionCall
	// FunctionDefinition declares a callable function's schema.
	FunctionDefinition = llms.FunctionDefinition
	// FunctionReference names a specific function (e.g. in a ToolChoice).
	FunctionReference = llms.FunctionReference
)

// Model and call options.
type (
	// Model is the interface implemented by every LLM provider.
	Model = llms.Model
	// CallOption configures a single generation call.
	CallOption = llms.CallOption
	// CallOptions is the resolved set of options for a generation call.
	CallOptions = llms.CallOptions
)

// Role constants for ChatMessageType. Names mirror langchaingo so that call
// sites migrate with a pure qualifier substitution (llms. -> message.).
const (
	ChatMessageTypeHuman  = llms.ChatMessageTypeHuman
	ChatMessageTypeAI     = llms.ChatMessageTypeAI
	ChatMessageTypeSystem = llms.ChatMessageTypeSystem
	ChatMessageTypeTool   = llms.ChatMessageTypeTool
)

// Constructors and call options re-exported as variables so callers depend on
// this package rather than langchaingo directly.
var (
	// TextPart builds a plain-text content part.
	TextPart = llms.TextPart
	// TextParts builds a Message from a role and one or more text strings.
	TextParts = llms.TextParts

	// WithTools supplies the tools available to a generation call.
	WithTools = llms.WithTools
	// WithToolChoice constrains tool selection for a generation call.
	WithToolChoice = llms.WithToolChoice
	// WithTemperature sets the sampling temperature.
	WithTemperature = llms.WithTemperature
	// WithTopP sets nucleus-sampling probability mass.
	WithTopP = llms.WithTopP
	// WithMaxTokens caps the number of generated tokens.
	WithMaxTokens = llms.WithMaxTokens
	// WithStreamingFunc registers a callback for streamed output chunks.
	WithStreamingFunc = llms.WithStreamingFunc

	// GenerateFromSinglePrompt is a convenience wrapper for single-prompt calls.
	GenerateFromSinglePrompt = llms.GenerateFromSinglePrompt
)
