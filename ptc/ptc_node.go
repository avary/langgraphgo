package ptc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// PTCToolNode is a graph node that handles programmatic tool calling
// It receives code from the LLM and executes it with tool access
type PTCToolNode struct {
	Executor *CodeExecutor
}

// NewPTCToolNode creates a new PTC tool node with default execution mode (server)
func NewPTCToolNode(language ExecutionLanguage, toolList []tools.Tool) *PTCToolNode {
	return NewPTCToolNodeWithMode(language, toolList, ModeServer)
}

// NewPTCToolNodeWithMode creates a new PTC tool node with specified execution mode
func NewPTCToolNodeWithMode(language ExecutionLanguage, toolList []tools.Tool, mode ExecutionMode) *PTCToolNode {
	return &PTCToolNode{
		Executor: NewCodeExecutorWithMode(language, toolList, mode),
	}
}

// Invoke executes the PTC node logic
func (node *PTCToolNode) Invoke(ctx context.Context, state interface{}) (interface{}, error) {
	mState, ok := state.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("state must be a map[string]interface{}")
	}

	// Extract messages from state
	messagesInterface, ok := mState["messages"]
	if !ok {
		return nil, fmt.Errorf("messages not found in state")
	}

	messages, ok := messagesInterface.([]llms.MessageContent)
	if !ok {
		return nil, fmt.Errorf("messages must be []llms.MessageContent")
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages in state")
	}

	// Get the last message from the AI
	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != llms.ChatMessageTypeAI {
		return nil, fmt.Errorf("last message must be from AI")
	}

	// Extract code from the message
	code, err := extractCodeFromMessage(lastMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to extract code: %w", err)
	}

	// Note: Tool server is already started in CreatePTCAgent, no need to start again

	// Execute the code
	result, err := node.Executor.Execute(ctx, code)
	if err != nil {
		// Create error message as system message
		errorMsg := llms.MessageContent{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart(fmt.Sprintf("[Code Execution Error]\n%v\n\nOutput:\n%s", err, result.Output)),
			},
		}
		mState["messages"] = append(messages, errorMsg)
		return mState, nil
	}

	// Create success message with execution results as human message
	successMsg := llms.MessageContent{
		Role: llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{
			llms.TextPart(fmt.Sprintf("[Code Execution Result]\n%s", result.Output)),
		},
	}

	mState["messages"] = append(messages, successMsg)
	return mState, nil
}

// extractCodeFromMessage extracts code from an AI message
// Supports multiple formats:
// 1. Code in markdown code blocks (```language\ncode\n```)
// 2. Plain text code
// 3. JSON with "code" field
func extractCodeFromMessage(msg llms.MessageContent) (string, error) {
	for _, part := range msg.Parts {
		switch p := part.(type) {
		case llms.TextContent:
			code := p.Text

			// Try to extract code from markdown code blocks
			if extracted := extractFromCodeBlock(code); extracted != "" {
				return extracted, nil
			}

			// Try to parse as JSON
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(code), &jsonData); err == nil {
				if codeField, ok := jsonData["code"].(string); ok {
					return codeField, nil
				}
			}

			// Return as is
			return code, nil
		}
	}

	return "", fmt.Errorf("no code found in message")
}

// extractFromCodeBlock extracts code from markdown code blocks
func extractFromCodeBlock(text string) string {
	// Look for code blocks: ```language\ncode\n```
	start := -1
	end := -1

	// Find first ```
	for i := 0; i < len(text)-2; i++ {
		if text[i:i+3] == "```" {
			if start == -1 {
				// Find the end of the first line (language specifier)
				lineEnd := i + 3
				for lineEnd < len(text) && text[lineEnd] != '\n' {
					lineEnd++
				}
				if lineEnd < len(text) {
					start = lineEnd + 1
				}
			} else {
				end = i
				break
			}
		}
	}

	if start != -1 && end != -1 && end > start {
		return text[start:end]
	}

	return ""
}

// Close stops the tool server
func (node *PTCToolNode) Close(ctx context.Context) error {
	if node.Executor != nil {
		return node.Executor.Stop(ctx)
	}
	return nil
}
