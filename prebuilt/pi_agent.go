// Package prebuilt provides prebuilt agent implementations.
// This file implements PiAgent, inspired by pi-mono/packages/agent.
package prebuilt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// =============================================================================
// PiAgent State - Inspired by pi-mono/packages/agent/src/types.ts
// =============================================================================

// MessageQueueMode defines how messages are delivered from queues
type MessageQueueMode string

const (
	// QueueModeAll delivers all queued messages at once
	QueueModeAll MessageQueueMode = "all"
	// QueueModeOneAtATime delivers messages one at a time
	QueueModeOneAtATime MessageQueueMode = "one-at-a-time"
)

// PiAgentState represents the current state of the agent
// Inspired by pi-mono's AgentState interface
type PiAgentState struct {
	// Core state
	SystemPrompt  string                `json:"system_prompt"`
	Model         string                `json:"model"`
	ThinkingLevel string                `json:"thinking_level"` // off, minimal, low, medium, high, xhigh
	Messages      []llms.MessageContent `json:"messages"`
	Tools         []tools.Tool          `json:"-"`

	// Streaming state
	IsStreaming   bool                 `json:"is_streaming"`
	StreamMessage *llms.MessageContent `json:"stream_message,omitempty"`

	// Tool execution
	PendingToolCalls map[string]bool `json:"pending_tool_calls"`
	Error            error           `json:"error,omitempty"`

	// Message queues for steering and follow-up
	SteeringQueue []llms.MessageContent `json:"steering_queue,omitempty"`
	SteeringMode  MessageQueueMode      `json:"steering_mode"`
	FollowUpQueue []llms.MessageContent `json:"follow_up_queue,omitempty"`
	FollowUpMode  MessageQueueMode      `json:"follow_up_mode"`

	// Session info
	SessionKey string `json:"session_key,omitempty"`

	// mu protects concurrent access to state
	mu sync.RWMutex
}

// NewPiAgentState creates a new agent state
func NewPiAgentState() *PiAgentState {
	return &PiAgentState{
		Messages:         make([]llms.MessageContent, 0),
		Tools:            make([]tools.Tool, 0),
		PendingToolCalls: make(map[string]bool),
		SteeringQueue:    make([]llms.MessageContent, 0),
		SteeringMode:     QueueModeAll,
		FollowUpQueue:    make([]llms.MessageContent, 0),
		FollowUpMode:     QueueModeAll,
		ThinkingLevel:    "off",
	}
}

// AddMessage adds a message to the state
func (s *PiAgentState) AddMessage(msg llms.MessageContent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, msg)
}

// AddMessages adds multiple messages to the state
func (s *PiAgentState) AddMessages(msgs []llms.MessageContent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, msgs...)
}

// Steer adds a steering message to interrupt the agent mid-run
func (s *PiAgentState) Steer(msg llms.MessageContent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SteeringQueue = append(s.SteeringQueue, msg)
}

// FollowUp adds a follow-up message to be processed after agent finishes
func (s *PiAgentState) FollowUp(msg llms.MessageContent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FollowUpQueue = append(s.FollowUpQueue, msg)
}

// DequeueSteeringMessages gets and clears steering messages based on queue mode
func (s *PiAgentState) DequeueSteeringMessages() []llms.MessageContent {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.SteeringQueue) == 0 {
		return []llms.MessageContent{}
	}

	switch s.SteeringMode {
	case QueueModeOneAtATime:
		// Return only the first message
		msg := s.SteeringQueue[0]
		s.SteeringQueue = s.SteeringQueue[1:]
		return []llms.MessageContent{msg}
	default: // QueueModeAll
		// Return all messages
		msgs := s.SteeringQueue
		s.SteeringQueue = make([]llms.MessageContent, 0)
		return msgs
	}
}

// DequeueFollowUpMessages gets and clears follow-up messages based on queue mode
func (s *PiAgentState) DequeueFollowUpMessages() []llms.MessageContent {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.FollowUpQueue) == 0 {
		return []llms.MessageContent{}
	}

	switch s.FollowUpMode {
	case QueueModeOneAtATime:
		// Return only the first message
		msg := s.FollowUpQueue[0]
		s.FollowUpQueue = s.FollowUpQueue[1:]
		return []llms.MessageContent{msg}
	default: // QueueModeAll
		// Return all messages
		msgs := s.FollowUpQueue
		s.FollowUpQueue = make([]llms.MessageContent, 0)
		return msgs
	}
}

// AddPendingTool adds a tool call to the pending set
func (s *PiAgentState) AddPendingTool(toolCallID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.PendingToolCalls == nil {
		s.PendingToolCalls = make(map[string]bool)
	}
	s.PendingToolCalls[toolCallID] = true
}

// RemovePendingTool removes a tool call from the pending set
func (s *PiAgentState) RemovePendingTool(toolCallID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.PendingToolCalls, toolCallID)
}

// HasPendingTools checks if there are pending tool calls
func (s *PiAgentState) HasPendingTools() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.PendingToolCalls) > 0
}

// =============================================================================
// PiAgentEvent - Event types matching pi-mono's AgentEvent
// =============================================================================

// PiAgentEventType represents the type of event
type PiAgentEventType string

const (
	// Agent lifecycle
	EventAgentStart PiAgentEventType = "agent_start"
	EventAgentEnd   PiAgentEventType = "agent_end"

	// Turn lifecycle - a turn is one assistant response + any tool calls/results
	EventTurnStart PiAgentEventType = "turn_start"
	EventTurnEnd   PiAgentEventType = "turn_end"

	// Message lifecycle
	EventMessageStart  PiAgentEventType = "message_start"
	EventMessageUpdate PiAgentEventType = "message_update"
	EventMessageEnd    PiAgentEventType = "message_end"

	// Tool execution lifecycle
	EventToolExecutionStart  PiAgentEventType = "tool_execution_start"
	EventToolExecutionUpdate PiAgentEventType = "tool_execution_update"
	EventToolExecutionEnd    PiAgentEventType = "tool_execution_end"
)

// PiAgentEvent represents an event from the agent
// Inspired by pi-mono's AgentEvent type
type PiAgentEvent struct {
	Type      PiAgentEventType `json:"type"`
	Timestamp int64            `json:"timestamp"`

	// Message fields
	Message llms.MessageContent `json:"message,omitempty"`

	// Turn end fields
	TurnMessage llms.MessageContent `json:"turn_message,omitempty"`
	ToolResults []ToolResultMsg     `json:"tool_results,omitempty"`

	// Tool execution fields
	ToolCallID    string         `json:"tool_call_id,omitempty"`
	ToolName      string         `json:"tool_name,omitempty"`
	ToolArgs      map[string]any `json:"tool_args,omitempty"`
	ToolResult    any            `json:"tool_result,omitempty"`
	ToolError     bool           `json:"tool_error,omitempty"`
	PartialResult any            `json:"partial_result,omitempty"`

	// Agent end fields
	FinalMessages []llms.MessageContent `json:"final_messages,omitempty"`
}

// ToolResultMsg represents a tool result message
type ToolResultMsg struct {
	ToolCallID string         `json:"tool_call_id"`
	ToolName   string         `json:"tool_name"`
	Content    string         `json:"content"`
	IsError    bool           `json:"is_error"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// =============================================================================
// PiAgent - High-level API matching pi-mono's Agent class
// =============================================================================

// PiAgent is a high-level agent API inspired by pi-mono/packages/agent
// It provides state management, message queues, and event subscription
type PiAgent struct {
	state         *PiAgentState
	model         llms.Model
	listeners     []func(PiAgentEvent)
	listenersMu   sync.RWMutex
	runnable      *graph.StateRunnable[*PiAgentState]
	cancelFunc    context.CancelFunc
	streamMode    graph.StreamMode
	convertToLLM  func([]llms.MessageContent) ([]llms.MessageContent, error)
	transformCtx  func([]llms.MessageContent) ([]llms.MessageContent, error)
	maxIterations int
}

// PiAgentOptions configures a PiAgent
type PiAgentOption func(*PiAgent)

// WithSystemPrompt sets the system prompt
func WithPiSystemPrompt(prompt string) PiAgentOption {
	return func(a *PiAgent) {
		a.state.SystemPrompt = prompt
	}
}

// WithThinkingLevel sets the thinking/reasoning level
func WithPiThinkingLevel(level string) PiAgentOption {
	return func(a *PiAgent) {
		a.state.ThinkingLevel = level
	}
}

// WithSteeringMode sets how steering messages are delivered
func WithPiSteeringMode(mode MessageQueueMode) PiAgentOption {
	return func(a *PiAgent) {
		a.state.SteeringMode = mode
	}
}

// WithFollowUpMode sets how follow-up messages are delivered
func WithPiFollowUpMode(mode MessageQueueMode) PiAgentOption {
	return func(a *PiAgent) {
		a.state.FollowUpMode = mode
	}
}

// WithStreamMode sets the stream mode for execution
func WithPiStreamMode(mode graph.StreamMode) PiAgentOption {
	return func(a *PiAgent) {
		a.streamMode = mode
	}
}

// WithConvertToLLM sets the message conversion function
func WithPiConvertToLLM(fn func([]llms.MessageContent) ([]llms.MessageContent, error)) PiAgentOption {
	return func(a *PiAgent) {
		a.convertToLLM = fn
	}
}

// WithTransformContext sets the context transformation function
func WithPiTransformContext(fn func([]llms.MessageContent) ([]llms.MessageContent, error)) PiAgentOption {
	return func(a *PiAgent) {
		a.transformCtx = fn
	}
}

// WithPiMaxIterations sets the maximum number of iterations
func WithPiMaxIterations(max int) PiAgentOption {
	return func(a *PiAgent) {
		a.maxIterations = max
	}
}

// NewPiAgent creates a new PiAgent
// Inspired by pi-mono's Agent class constructor
func NewPiAgent(model llms.Model, inputTools []tools.Tool, opts ...PiAgentOption) (*PiAgent, error) {
	state := NewPiAgentState()
	state.Model = "model"
	state.Tools = inputTools

	agent := &PiAgent{
		state:         state,
		model:         model,
		listeners:     make([]func(PiAgentEvent), 0),
		streamMode:    graph.StreamModeValues,
		maxIterations: 20,
	}

	for _, opt := range opts {
		opt(agent)
	}

	// Build the agent graph
	runnable, err := buildPiAgentGraph(agent, model, inputTools)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent graph: %w", err)
	}

	agent.runnable = runnable

	return agent, nil
}

// Subscribe adds an event listener and returns an unsubscribe function
// Inspired by pi-mono's Agent.subscribe()
func (a *PiAgent) Subscribe(fn func(PiAgentEvent)) func() {
	a.listenersMu.Lock()
	defer a.listenersMu.Unlock()

	a.listeners = append(a.listeners, fn)

	idx := len(a.listeners) - 1
	return func() {
		a.listenersMu.Lock()
		defer a.listenersMu.Unlock()
		// Remove listener by setting to nil (will be cleaned up later)
		if idx < len(a.listeners) {
			a.listeners[idx] = nil
		}
	}
}

// emit sends an event to all listeners
func (a *PiAgent) emit(event PiAgentEvent) {
	a.listenersMu.RLock()
	defer a.listenersMu.RUnlock()

	for _, fn := range a.listeners {
		if fn != nil {
			fn(event)
		}
	}
}

// SetSystemPrompt updates the system prompt
// Inspired by pi-mono's Agent.setSystemPrompt()
func (a *PiAgent) SetSystemPrompt(prompt string) {
	a.state.SystemPrompt = prompt
}

// SetTools updates the available tools
// Inspired by pi-mono's Agent.setTools()
func (a *PiAgent) SetTools(tools []tools.Tool) {
	a.state.Tools = tools
}

// Steer adds a steering message to interrupt the agent mid-run
// Inspired by pi-mono's Agent.steer()
func (a *PiAgent) Steer(msg llms.MessageContent) {
	a.state.Steer(msg)
}

// FollowUp adds a follow-up message to be processed after agent finishes
// Inspired by pi-mono's Agent.followUp()
func (a *PiAgent) FollowUp(msg llms.MessageContent) {
	a.state.FollowUp(msg)
}

// ReplaceMessages replaces the message history
// Inspired by pi-mono's Agent.replaceMessages()
func (a *PiAgent) ReplaceMessages(msgs []llms.MessageContent) {
	a.state.Messages = make([]llms.MessageContent, len(msgs))
	copy(a.state.Messages, msgs)
}

// GetState returns a copy of the current agent state
func (a *PiAgent) GetState() *PiAgentState {
	a.state.mu.RLock()
	defer a.state.mu.RUnlock()

	// Deep copy the state
	stateCopy := &PiAgentState{
		SystemPrompt:     a.state.SystemPrompt,
		Model:            a.state.Model,
		ThinkingLevel:    a.state.ThinkingLevel,
		Messages:         make([]llms.MessageContent, len(a.state.Messages)),
		IsStreaming:      a.state.IsStreaming,
		StreamMessage:    a.state.StreamMessage,
		PendingToolCalls: make(map[string]bool),
		Error:            a.state.Error,
		SteeringMode:     a.state.SteeringMode,
		FollowUpMode:     a.state.FollowUpMode,
		SessionKey:       a.state.SessionKey,
	}

	copy(stateCopy.Messages, a.state.Messages)
	for k, v := range a.state.PendingToolCalls {
		stateCopy.PendingToolCalls[k] = v
	}

	return stateCopy
}

// WaitForIdle waits until the agent is not streaming
// Inspired by pi-mono's Agent.waitForIdle()
func (a *PiAgent) WaitForIdle(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			a.state.mu.RLock()
			isStreaming := a.state.IsStreaming
			a.state.mu.RUnlock()
			if !isStreaming {
				return nil
			}
		}
	}
}

// Abort aborts the current agent execution
// Inspired by pi-mono's Agent.abort()
func (a *PiAgent) Abort() {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
}

// Reset resets the agent state
// Inspired by pi-mono's Agent.reset()
func (a *PiAgent) Reset() {
	tools := a.state.Tools // preserve registered tools across reset
	a.state = NewPiAgentState()
	a.state.SystemPrompt = ""
	a.state.Model = "model"
	a.state.Tools = tools
}

// Prompt sends a prompt to the agent and executes it
// Inspired by pi-mono's Agent.prompt()
func (a *PiAgent) Prompt(ctx context.Context, msg llms.MessageContent) error {
	// Create child context with cancel
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel
	defer func() {
		a.cancelFunc = nil
	}()

	// Emit agent_start event
	a.emit(PiAgentEvent{
		Type:      EventAgentStart,
		Timestamp: time.Now().UnixMilli(),
	})

	// Add the user message
	a.state.AddMessage(msg)

	// Emit message events
	a.emit(PiAgentEvent{
		Type:      EventMessageStart,
		Message:   msg,
		Timestamp: time.Now().UnixMilli(),
	})
	a.emit(PiAgentEvent{
		Type:      EventMessageEnd,
		Message:   msg,
		Timestamp: time.Now().UnixMilli(),
	})

	// Execute the agent graph
	finalState, err := a.runnable.Invoke(ctx, a.state)

	// Emit agent_end event
	finalMessages := []llms.MessageContent{}
	if finalState != nil {
		finalMessages = finalState.Messages
	}
	a.emit(PiAgentEvent{
		Type:          EventAgentEnd,
		Timestamp:     time.Now().UnixMilli(),
		FinalMessages: finalMessages,
	})

	if err != nil {
		a.state.Error = err
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Update our state with the final state
	if finalState != nil {
		a.state.Messages = finalState.Messages
	}
	a.state.IsStreaming = false

	return nil
}

// PromptWithStream sends a prompt and returns a stream of events
// Inspired by pi-mono's streaming agent loop
func (a *PiAgent) PromptWithStream(ctx context.Context, msg llms.MessageContent) (<-chan PiAgentEvent, <-chan error, func()) {
	// Create child context with cancel
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	eventChan := make(chan PiAgentEvent, 100)
	errorChan := make(chan error, 1)

	// Start execution in goroutine
	go func() {
		defer close(eventChan)
		defer close(errorChan)

		// Emit agent_start event
		eventChan <- PiAgentEvent{
			Type:      EventAgentStart,
			Timestamp: time.Now().UnixMilli(),
		}

		// Add the user message
		a.state.AddMessage(msg)

		// Emit message events
		eventChan <- PiAgentEvent{
			Type:      EventMessageStart,
			Message:   msg,
			Timestamp: time.Now().UnixMilli(),
		}
		eventChan <- PiAgentEvent{
			Type:      EventMessageEnd,
			Message:   msg,
			Timestamp: time.Now().UnixMilli(),
		}

		// Execute the agent graph
		finalState, err := a.runnable.Invoke(ctx, a.state)

		// Emit agent_end event
		finalMessages := []llms.MessageContent{}
		if finalState != nil {
			finalMessages = finalState.Messages
		}
		eventChan <- PiAgentEvent{
			Type:          EventAgentEnd,
			Timestamp:     time.Now().UnixMilli(),
			FinalMessages: finalMessages,
		}

		if err != nil {
			errorChan <- err
		}
	}()

	return eventChan, errorChan, cancel
}

// =============================================================================
// PiAgent Graph Builder - Builds the agent execution graph
// =============================================================================

// buildPiAgentGraph builds the agent execution graph
// Inspired by pi-mono's agentLoop function
func buildPiAgentGraph(agent *PiAgent, model llms.Model, inputTools []tools.Tool) (*graph.StateRunnable[*PiAgentState], error) {
	// Create state graph with schema
	g := graph.NewStateGraph[*PiAgentState]()

	// Define state schema. The merge function only appends new.Messages to
	// current.Messages; nodes must return only NEW messages, not full history.
	g.SetSchema(graph.NewStructSchema(NewPiAgentState(), mergePiAgentState))

	// Set entry point and wire nodes/edges.
	g.AddNode("agent", "Agent node that calls LLM and generates response", newPiAgentNode(agent, model, inputTools))
	g.AddNode("tools", "Tools node that executes tool calls", newPiToolsNode(agent, inputTools))
	g.SetEntryPoint("agent")
	g.AddConditionalEdge("agent", routePiAgent)
	g.AddEdge("tools", "agent")

	return g.Compile()
}

// mergePiAgentState merges a node's returned state (new) into the accumulated
// state (current). Messages and queues are appended; scalar fields overwrite
// current only when non-zero.
func mergePiAgentState(current, new *PiAgentState) (*PiAgentState, error) {
	if current == nil {
		if new == nil {
			return NewPiAgentState(), nil
		}
		return new, nil
	}
	if new == nil {
		return current, nil
	}

	if len(new.Messages) > 0 {
		current.Messages = append(current.Messages, new.Messages...)
	}
	if new.SystemPrompt != "" {
		current.SystemPrompt = new.SystemPrompt
	}
	if new.Model != "" {
		current.Model = new.Model
	}
	if new.ThinkingLevel != "" {
		current.ThinkingLevel = new.ThinkingLevel
	}
	if new.IsStreaming {
		current.IsStreaming = true
	}
	if new.StreamMessage != nil {
		current.StreamMessage = new.StreamMessage
	}
	if len(new.PendingToolCalls) > 0 {
		if current.PendingToolCalls == nil {
			current.PendingToolCalls = make(map[string]bool)
		}
		for k, v := range new.PendingToolCalls {
			current.PendingToolCalls[k] = v
		}
	}
	if new.Error != nil {
		current.Error = new.Error
	}
	if len(new.SteeringQueue) > 0 {
		current.SteeringQueue = append(current.SteeringQueue, new.SteeringQueue...)
	}
	if new.SteeringMode != "" {
		current.SteeringMode = new.SteeringMode
	}
	if len(new.FollowUpQueue) > 0 {
		current.FollowUpQueue = append(current.FollowUpQueue, new.FollowUpQueue...)
	}
	if new.FollowUpMode != "" {
		current.FollowUpMode = new.FollowUpMode
	}
	if new.SessionKey != "" {
		current.SessionKey = new.SessionKey
	}
	return current, nil
}

// newPiAgentNode builds the "agent" node function that calls the LLM and
// returns a single new AI message.
func newPiAgentNode(agent *PiAgent, model llms.Model, inputTools []tools.Tool) func(context.Context, *PiAgentState) (*PiAgentState, error) {
	maxIterations := agent.maxIterations
	return func(ctx context.Context, state *PiAgentState) (*PiAgentState, error) {
		iterationCount := 0
		if state.Messages != nil {
			for _, msg := range state.Messages {
				if msg.Role == llms.ChatMessageTypeAI {
					iterationCount++
				}
			}
		}

		if iterationCount >= maxIterations {
			finalMsg := llms.TextParts(llms.ChatMessageTypeAI, "Maximum iterations reached. Please try a simpler query.")
			return &PiAgentState{Messages: []llms.MessageContent{finalMsg}}, nil
		}

		messagesToUse := state.Messages
		if agent.transformCtx != nil {
			if transformed, err := agent.transformCtx(state.Messages); err == nil {
				messagesToUse = transformed
			}
		}
		if agent.convertToLLM != nil {
			if converted, err := agent.convertToLLM(messagesToUse); err == nil {
				messagesToUse = converted
			}
		}

		msgsToSend := make([]llms.MessageContent, 0, len(messagesToUse))
		if agent.state.SystemPrompt != "" {
			msgsToSend = append(msgsToSend, llms.TextParts(llms.ChatMessageTypeSystem, agent.state.SystemPrompt))
		}
		msgsToSend = append(msgsToSend, messagesToUse...)

		var toolDefs []llms.Tool
		for _, t := range inputTools {
			toolDefs = append(toolDefs, llms.Tool{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  getToolSchema(t),
				},
			})
		}

		resp, err := model.GenerateContent(ctx, msgsToSend, llms.WithTools(toolDefs), llms.WithToolChoice("auto"))
		if err != nil {
			return &PiAgentState{}, fmt.Errorf("LLM call failed: %w", err)
		}

		choice := resp.Choices[0]
		aiMsg := llms.MessageContent{Role: llms.ChatMessageTypeAI}
		if choice.Content != "" {
			aiMsg.Parts = append(aiMsg.Parts, llms.TextPart(choice.Content))
		}
		for _, tc := range choice.ToolCalls {
			aiMsg.Parts = append(aiMsg.Parts, tc)
		}

		return &PiAgentState{Messages: []llms.MessageContent{aiMsg}}, nil
	}
}

// newPiToolsNode builds the "tools" node function that executes the tool calls
// from the last AI message and returns the resulting tool messages.
func newPiToolsNode(agent *PiAgent, inputTools []tools.Tool) func(context.Context, *PiAgentState) (*PiAgentState, error) {
	return func(ctx context.Context, state *PiAgentState) (*PiAgentState, error) {
		if len(state.Messages) == 0 {
			return &PiAgentState{}, nil
		}

		lastMsg := state.Messages[len(state.Messages)-1]
		if lastMsg.Role != llms.ChatMessageTypeAI {
			return &PiAgentState{}, nil
		}

		toolExecutor := NewToolExecutor(inputTools)

		var toolMessages []llms.MessageContent
		for _, part := range lastMsg.Parts {
			tc, ok := part.(llms.ToolCall)
			if !ok {
				continue
			}

			agent.emit(PiAgentEvent{
				Type:       EventToolExecutionStart,
				Timestamp:  time.Now().UnixMilli(),
				ToolCallID: tc.ID,
				ToolName:   tc.FunctionCall.Name,
			})

			res, err := toolExecutor.Execute(ctx, ToolInvocation{
				Tool:      tc.FunctionCall.Name,
				ToolInput: tc.FunctionCall.Arguments,
			})
			if err != nil {
				res = fmt.Sprintf("Error: %v", err)
			}
			agent.emit(PiAgentEvent{
				Type:       EventToolExecutionEnd,
				Timestamp:  time.Now().UnixMilli(),
				ToolCallID: tc.ID,
				ToolName:   tc.FunctionCall.Name,
				ToolResult: res,
				ToolError:  err != nil,
			})

			toolMessages = append(toolMessages, llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{
					llms.ToolCallResponse{
						ToolCallID: tc.ID,
						Name:       tc.FunctionCall.Name,
						Content:    res,
					},
				},
			})

			// Steering messages injected mid-execution stop further tool calls.
			if steering := state.DequeueSteeringMessages(); len(steering) > 0 {
				toolMessages = append(toolMessages, steering...)
				break
			}
		}

		return &PiAgentState{Messages: toolMessages}, nil
	}
}

// routePiAgent decides whether the agent's last message requires tool execution
// or ends the run.
func routePiAgent(ctx context.Context, state *PiAgentState) string {
	if len(state.Messages) == 0 {
		return graph.END
	}

	lastMsg := state.Messages[len(state.Messages)-1]
	if lastMsg.Role != llms.ChatMessageTypeAI {
		return graph.END
	}

	for _, part := range lastMsg.Parts {
		if _, ok := part.(llms.ToolCall); ok {
			return "tools"
		}
	}

	return graph.END
}
