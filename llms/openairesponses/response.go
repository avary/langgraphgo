package openairesponses

import "github.com/smallnest/langgraphgo/llmtypes"

// --- Response wire types (subset of the Responses API response body) ---

type response struct {
	ID     string         `json:"id"`
	Status string         `json:"status"`
	Output []outputItem   `json:"output"`
	Usage  *responseUsage `json:"usage,omitempty"`
	Error  *responseError `json:"error,omitempty"`
}

type outputItem struct {
	Type      string              `json:"type"`
	Role      string              `json:"role,omitempty"`
	Content   []outputContentPart `json:"content,omitempty"`
	Summary   []summaryPart       `json:"summary,omitempty"`
	CallID    string              `json:"call_id,omitempty"`
	Name      string              `json:"name,omitempty"`
	Arguments string              `json:"arguments,omitempty"`
}

type outputContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type summaryPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type responseUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type responseError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func (e *responseError) Error() string { return "openairesponses: " + e.Message }

// toContentResponse flattens the Responses output array into the framework's
// single-choice ContentResponse, collecting text, reasoning, and tool calls.
func (r *response) toContentResponse() *llmtypes.ContentResponse {
	var (
		text      string
		reasoning string
		toolCalls []llmtypes.ToolCall
	)
	for _, item := range r.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if part.Type == "output_text" {
					text += part.Text
				}
			}
		case "reasoning":
			for _, part := range item.Summary {
				reasoning += part.Text
			}
		case "function_call":
			toolCalls = append(toolCalls, llmtypes.ToolCall{
				ID:   item.CallID,
				Type: "function",
				FunctionCall: &llmtypes.FunctionCall{
					Name:      item.Name,
					Arguments: item.Arguments,
				},
			})
		}
	}

	choice := &llmtypes.ContentChoice{
		Content:          text,
		ReasoningContent: reasoning,
		ToolCalls:        toolCalls,
		StopReason:       r.Status,
	}
	if len(toolCalls) > 0 {
		choice.FuncCall = toolCalls[0].FunctionCall
	}
	if r.Usage != nil {
		choice.GenerationInfo = map[string]any{
			"prompt_tokens":     r.Usage.InputTokens,
			"completion_tokens": r.Usage.OutputTokens,
			"total_tokens":      r.Usage.TotalTokens,
		}
	}
	return &llmtypes.ContentResponse{Choices: []*llmtypes.ContentChoice{choice}}
}
