package prebuilt

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/tooltypes"
)

// ChatAgent represents a session with a user and can handle multi-turn conversations.
type ChatAgent struct {
	// The underlying agent runnable
	Runnable *graph.StateRunnable[map[string]any]
	// The session ID for this conversation
	threadID string
	// Conversation history
	messages []llmtypes.MessageContent
	// Dynamic tools that can be updated at runtime
	dynamicTools []tooltypes.Tool
	// Model reference for streaming (optional)
	model llmtypes.Model
	// Options used when creating the agent
	options *CreateAgentOptions
}

// NewChatAgent creates a new ChatAgent.
// It wraps the underlying agent graph and manages conversation history automatically.
func NewChatAgent(model llmtypes.Model, inputTools []tooltypes.Tool, opts ...CreateAgentOption) (*ChatAgent, error) {
	// Parse options
	options := &CreateAgentOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Create the agent with options
	agent, err := CreateAgentMap(model, inputTools, options.MaxIterations, opts...)
	if err != nil {
		return nil, err
	}

	// Generate a random thread ID for this session
	threadID := uuid.New().String()

	return &ChatAgent{
		Runnable:     agent,
		threadID:     threadID,
		messages:     make([]llmtypes.MessageContent, 0),
		dynamicTools: make([]tooltypes.Tool, 0),
		model:        model,
		options:      options,
	}, nil
}

// ThreadID returns the current session ID.
func (c *ChatAgent) ThreadID() string {
	return c.threadID
}

// Chat sends a message to the agent and returns the response.
// It maintains the conversation context by accumulating message history.
func (c *ChatAgent) Chat(ctx context.Context, message string) (string, error) {
	// 1. Add user message to history
	userMsg := llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, message)
	c.messages = append(c.messages, userMsg)

	// 2. Construct input with full conversation history and dynamic tools
	input := map[string]any{
		"messages": c.messages,
	}

	// Add dynamic tools if any
	if len(c.dynamicTools) > 0 {
		input["extra_tools"] = c.dynamicTools
	}

	// 3. Create config with thread_id
	config := &graph.Config{
		Configurable: map[string]any{
			"thread_id": c.threadID,
		},
	}

	// 4. Invoke the agent
	resp, err := c.Runnable.InvokeWithConfig(ctx, input, config)
	if err != nil {
		return "", err
	}

	// 5. Extract messages from response
	// resp is already map[string]any from StateRunnable[map[string]any]
	messages, ok := resp["messages"].([]llmtypes.MessageContent)
	if !ok || len(messages) == 0 {
		return "", fmt.Errorf("no messages in response")
	}

	// 6. Update conversation history with all new messages
	c.messages = messages

	// 7. Extract the last message for return value
	lastMsg := messages[len(messages)-1]
	if len(lastMsg.Parts) == 0 {
		return "", nil
	}

	switch part := lastMsg.Parts[0].(type) {
	case llmtypes.TextContent:
		return part.Text, nil
	default:
		return fmt.Sprintf("%v", part), nil
	}
}

// PrintStream prints the agent's response to the provided writer (e.g., os.Stdout).
// Note: This is a simplified version that uses Chat internally.
// For true streaming support, you would need to use a graph that supports streaming.
func (c *ChatAgent) PrintStream(ctx context.Context, message string, w io.Writer) error {
	response, err := c.Chat(ctx, message)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "Response: %s\n", response)
	return nil
}

// SetTools replaces all dynamic tools with the provided tools.
// Note: This does not affect the base tools provided when creating the agent.
func (c *ChatAgent) SetTools(newTools []tooltypes.Tool) {
	c.dynamicTools = make([]tooltypes.Tool, len(newTools))
	copy(c.dynamicTools, newTools)
}

// AddTool adds a new tool to the dynamic tools list.
// If a tool with the same name already exists, it will be replaced.
func (c *ChatAgent) AddTool(tool tooltypes.Tool) {
	// Check if tool with same name exists
	for i, t := range c.dynamicTools {
		if t.Name() == tool.Name() {
			c.dynamicTools[i] = tool
			return
		}
	}
	// Add new tool
	c.dynamicTools = append(c.dynamicTools, tool)
}

// RemoveTool removes a tool by name from the dynamic tools list.
// Returns true if the tool was found and removed, false otherwise.
func (c *ChatAgent) RemoveTool(toolName string) bool {
	for i, t := range c.dynamicTools {
		if t.Name() == toolName {
			// Remove tool by slicing
			c.dynamicTools = append(c.dynamicTools[:i], c.dynamicTools[i+1:]...)
			return true
		}
	}
	return false
}

// GetTools returns a copy of the current dynamic tools list.
// Note: This does not include the base tools provided when creating the agent.
func (c *ChatAgent) GetTools() []tooltypes.Tool {
	toolsCopy := make([]tooltypes.Tool, len(c.dynamicTools))
	copy(toolsCopy, c.dynamicTools)
	return toolsCopy
}

// ClearTools removes all dynamic tools.
func (c *ChatAgent) ClearTools() {
	c.dynamicTools = make([]tooltypes.Tool, 0)
}

// AsyncChat sends a message to the agent and returns a channel for streaming the response.
// This method provides TRUE streaming by using the LLM's streaming API.
// Chunks are sent to the channel as they're generated by the LLM in real-time.
// The channel will be closed when the response is complete or an error occurs.
func (c *ChatAgent) AsyncChat(ctx context.Context, message string) (<-chan string, error) {
	// Create output channel
	outputChan := make(chan string, 100)

	// Add user message to history
	userMsg := llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, message)
	c.messages = append(c.messages, userMsg)

	// Prepare messages to send
	msgsToSend := c.messages

	// Apply system message if provided
	if c.options != nil && c.options.SystemMessage != "" {
		sysMsg := llmtypes.TextParts(llmtypes.ChatMessageTypeSystem, c.options.SystemMessage)
		msgsToSend = append([]llmtypes.MessageContent{sysMsg}, msgsToSend...)
	}

	// Apply state modifier if provided
	if c.options != nil && c.options.StateModifier != nil {
		msgsToSend = c.options.StateModifier(msgsToSend)
	}

	// Start goroutine to handle streaming
	go func() {
		defer close(outputChan)

		var fullResponse string

		// Create streaming function that sends chunks to the channel
		streamingFunc := func(ctx context.Context, chunk []byte) error {
			chunkStr := string(chunk)
			fullResponse += chunkStr

			select {
			case <-ctx.Done():
				return ctx.Err()
			case outputChan <- chunkStr:
				return nil
			}
		}

		// Call model with streaming enabled
		_, err := c.model.GenerateContent(ctx, msgsToSend, llmtypes.WithStreamingFunc(streamingFunc))
		if err != nil {
			// Error during streaming, channel will be closed
			return
		}

		// Add AI response to history
		aiMsg := llmtypes.MessageContent{
			Role:  llmtypes.ChatMessageTypeAI,
			Parts: []llmtypes.ContentPart{llmtypes.TextPart(fullResponse)},
		}
		c.messages = append(c.messages, aiMsg)
	}()

	return outputChan, nil
}

// AsyncChatWithChunks sends a message to the agent and returns a channel for streaming the response.
// Unlike AsyncChat, this streams the response in word-sized chunks for better readability.
// The channel will be closed when the response is complete.
func (c *ChatAgent) AsyncChatWithChunks(ctx context.Context, message string) (<-chan string, error) {
	// Create output channel
	outputChan := make(chan string, 100)

	// Start goroutine to handle the chat
	go func() {
		defer close(outputChan)

		// Call the regular Chat method
		response, err := c.Chat(ctx, message)
		if err != nil {
			// If there's an error, we can't send it through the string channel
			// Just close the channel
			return
		}

		// Split response into words and stream them
		words := splitIntoWords(response)
		for i, word := range words {
			select {
			case <-ctx.Done():
				// Context cancelled, stop streaming
				return
			case outputChan <- word:
				// Add space after word (except for last word)
				if i < len(words)-1 {
					select {
					case <-ctx.Done():
						return
					case outputChan <- " ":
					}
				}
			}
		}
	}()

	return outputChan, nil
}

// splitIntoWords splits a string into words while preserving punctuation
func splitIntoWords(text string) []string {
	if text == "" {
		return []string{}
	}

	var words []string
	var currentWord string

	for _, char := range text {
		if char == ' ' || char == '\n' || char == '\t' {
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
		} else {
			currentWord += string(char)
		}
	}

	// Don't forget the last word
	if currentWord != "" {
		words = append(words, currentWord)
	}

	return words
}
