package openairesponses

import "github.com/smallnest/langgraphgo/llmtypes"

// --- Request wire types (subset of the Responses API request body) ---

type request struct {
	Model           string         `json:"model"`
	Input           []inputItem    `json:"input"`
	Instructions    string         `json:"instructions,omitempty"`
	Temperature     *float64       `json:"temperature,omitempty"`
	TopP            *float64       `json:"top_p,omitempty"`
	MaxOutputTokens int            `json:"max_output_tokens,omitempty"`
	Tools           []requestTool  `json:"tools,omitempty"`
	ToolChoice      any            `json:"tool_choice,omitempty"`
	Text            *textFormat    `json:"text,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	Stream          bool           `json:"stream,omitempty"`
}

type textFormat struct {
	Format formatSpec `json:"format"`
}

type formatSpec struct {
	Type string `json:"type"`
}

// inputItem is a single element of the Responses "input" array. It is either a
// role-tagged message, a function_call, or a function_call_output.
type inputItem struct {
	Type    string        `json:"type,omitempty"`
	Role    string        `json:"role,omitempty"`
	Content []contentPart `json:"content,omitempty"`

	// function_call
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`

	// function_call_output
	Output string `json:"output,omitempty"`
}

type contentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type requestTool struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
	Strict      bool   `json:"strict,omitempty"`
}

func (o *LLM) buildRequest(messages []llmtypes.MessageContent, opts *llmtypes.CallOptions, stream bool) request {
	req := request{
		Model:           opts.Model,
		Input:           toInputItems(messages),
		MaxOutputTokens: opts.MaxTokens,
		Stream:          stream,
	}
	if opts.Temperature != 0 {
		t := opts.Temperature
		req.Temperature = &t
	}
	if opts.TopP != 0 {
		p := opts.TopP
		req.TopP = &p
	}
	if opts.JSONMode {
		req.Text = &textFormat{Format: formatSpec{Type: "json_object"}}
	}
	if len(opts.Tools) > 0 {
		req.Tools = toResponsesTools(opts.Tools)
	}
	if opts.ToolChoice != nil {
		req.ToolChoice = toResponsesToolChoice(opts.ToolChoice)
	}
	if len(opts.Metadata) > 0 {
		req.Metadata = opts.Metadata
	}
	return req
}

// toInputItems flattens framework messages into Responses input items. Tool
// calls and tool results become their own top-level items, as the API expects.
func toInputItems(messages []llmtypes.MessageContent) []inputItem {
	items := make([]inputItem, 0, len(messages))
	for _, m := range messages {
		role := toResponsesRole(m.Role)
		textType := "input_text"
		if m.Role == llmtypes.ChatMessageTypeAI {
			textType = "output_text"
		}

		var parts []contentPart
		flush := func() {
			if len(parts) > 0 {
				items = append(items, inputItem{Type: "message", Role: role, Content: parts})
				parts = nil
			}
		}
		for _, part := range m.Parts {
			switch p := part.(type) {
			case llmtypes.TextContent:
				parts = append(parts, contentPart{Type: textType, Text: p.Text})
			case llmtypes.ImageURLContent:
				parts = append(parts, contentPart{Type: "input_image", ImageURL: p.URL})
			case llmtypes.ToolCall:
				flush()
				items = append(items, toolCallItem(p))
			case llmtypes.ToolCallResponse:
				flush()
				items = append(items, inputItem{
					Type:   "function_call_output",
					CallID: p.ToolCallID,
					Output: p.Content,
				})
			}
		}
		flush()
	}
	return items
}

func toolCallItem(tc llmtypes.ToolCall) inputItem {
	item := inputItem{Type: "function_call", CallID: tc.ID}
	if tc.FunctionCall != nil {
		item.Name = tc.FunctionCall.Name
		item.Arguments = tc.FunctionCall.Arguments
	}
	return item
}

func toResponsesRole(role llmtypes.ChatMessageType) string {
	switch role {
	case llmtypes.ChatMessageTypeSystem:
		return "system"
	case llmtypes.ChatMessageTypeAI:
		return "assistant"
	default:
		return "user"
	}
}

func toResponsesTools(tools []llmtypes.Tool) []requestTool {
	out := make([]requestTool, 0, len(tools))
	for _, t := range tools {
		rt := requestTool{Type: t.Type}
		if t.Function != nil {
			rt.Name = t.Function.Name
			rt.Description = t.Function.Description
			rt.Parameters = t.Function.Parameters
			rt.Strict = t.Function.Strict
		}
		out = append(out, rt)
	}
	return out
}

func toResponsesToolChoice(choice any) any {
	tc, ok := choice.(llmtypes.ToolChoice)
	if !ok {
		return choice
	}
	if tc.Function != nil {
		return map[string]any{"type": "function", "name": tc.Function.Name}
	}
	return tc.Type
}
