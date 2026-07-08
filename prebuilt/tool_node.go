package prebuilt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/smallnest/langgraphgo/llmtypes"
)

// ToolNodeMap is a reusable node that executes tool calls from the last AI message
// for map[string]any state.
func ToolNodeMap(executor *ToolExecutor) func(context.Context, map[string]any) (map[string]any, error) {
	return func(ctx context.Context, state map[string]any) (map[string]any, error) {
		messages, ok := state["messages"].([]llmtypes.MessageContent)
		if !ok || len(messages) == 0 {
			return nil, fmt.Errorf("no messages found in state")
		}

		lastMsg := messages[len(messages)-1]
		if lastMsg.Role != llmtypes.ChatMessageTypeAI {
			return nil, fmt.Errorf("last message is not an AI message")
		}

		var toolMessages []llmtypes.MessageContent
		for _, part := range lastMsg.Parts {
			if tc, ok := part.(llmtypes.ToolCall); ok {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)

				inputVal := ""
				if val, ok := args["input"].(string); ok {
					inputVal = val
				} else {
					inputVal = tc.FunctionCall.Arguments
				}

				res, err := executor.Execute(ctx, ToolInvocation{
					Tool:      tc.FunctionCall.Name,
					ToolInput: inputVal,
				})
				if err != nil {
					res = fmt.Sprintf("Error: %v", err)
				}

				toolMessages = append(toolMessages, llmtypes.MessageContent{
					Role: llmtypes.ChatMessageTypeTool,
					Parts: []llmtypes.ContentPart{
						llmtypes.ToolCallResponse{
							ToolCallID: tc.ID,
							Name:       tc.FunctionCall.Name,
							Content:    res,
						},
					},
				})
			}
		}

		return map[string]any{
			"messages": toolMessages,
		}, nil
	}
}

// ToolNode creates a generic tool execution node
func ToolNode[S any](
	executor *ToolExecutor,
	getMessages func(S) []llmtypes.MessageContent,
	setMessages func(S, []llmtypes.MessageContent) S,
) func(context.Context, S) (S, error) {
	return func(ctx context.Context, state S) (S, error) {
		messages := getMessages(state)
		if len(messages) == 0 {
			return state, fmt.Errorf("no messages")
		}

		lastMsg := messages[len(messages)-1]
		if lastMsg.Role != llmtypes.ChatMessageTypeAI {
			return state, fmt.Errorf("not an AI message")
		}

		var toolMessages []llmtypes.MessageContent
		for _, part := range lastMsg.Parts {
			if tc, ok := part.(llmtypes.ToolCall); ok {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args)

				inputVal := ""
				if val, ok := args["input"].(string); ok {
					inputVal = val
				} else {
					inputVal = tc.FunctionCall.Arguments
				}

				res, err := executor.Execute(ctx, ToolInvocation{
					Tool:      tc.FunctionCall.Name,
					ToolInput: inputVal,
				})
				if err != nil {
					res = fmt.Sprintf("Error: %v", err)
				}

				toolMessages = append(toolMessages, llmtypes.MessageContent{
					Role: llmtypes.ChatMessageTypeTool,
					Parts: []llmtypes.ContentPart{
						llmtypes.ToolCallResponse{
							ToolCallID: tc.ID,
							Name:       tc.FunctionCall.Name,
							Content:    res,
						},
					},
				})
			}
		}

		return setMessages(state, append(messages, toolMessages...)), nil
	}
}
