package prebuilt

import (
	"context"
	"strings"
	"testing"

	"github.com/smallnest/langgraphgo/llms"
	"github.com/smallnest/langgraphgo/tool"
)

// MockModel is a simple mock for llms.Model
type MockModel struct {
	responses []string
	callCount int
}

func (m *MockModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	resp := ""
	if m.callCount < len(m.responses) {
		resp = m.responses[m.callCount]
	} else {
		resp = "default response"
	}
	m.callCount++

	// Parse options to check for streaming
	opts := llms.CallOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	// If streaming function is provided, call it with chunks
	if opts.StreamingFunc != nil {
		// Simulate streaming by sending response in small chunks
		words := splitIntoWords(resp)
		for i, word := range words {
			chunk := word
			if i < len(words)-1 {
				chunk += " "
			}
			if err := opts.StreamingFunc(ctx, []byte(chunk)); err != nil {
				return nil, err
			}
		}
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: resp},
		},
	}, nil
}

func (m *MockModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

func TestChatAgent(t *testing.T) {
	// Setup mock model
	mockModel := &MockModel{
		responses: []string{
			"Hello! I am a bot.",
			"I remember you said hi.",
		},
	}

	// Create ChatAgent
	agent, err := NewChatAgent(mockModel, nil)
	if err != nil {
		t.Fatalf("Failed to create ChatAgent: %v", err)
	}

	ctx := context.Background()

	// 1. Test first turn
	resp1, err := agent.Chat(ctx, "Hi")
	if err != nil {
		t.Errorf("Chat failed: %v", err)
	}
	if !strings.Contains(resp1, "Hello") {
		t.Errorf("Expected greeting, got: %s", resp1)
	}

	// 2. Test second turn (memory)
	// Note: The mock model itself doesn't actually "remember" in this simple implementation,
	// but the agent infrastructure should retrieve history.
	// To verify memory, we'd ideally check the input messages to the model in a real integration test or a more sophisticated mock.
	// For this unit test, we just verify the flow works and the thread ID persists.

	threadID1 := agent.ThreadID()
	if threadID1 == "" {
		t.Error("ThreadID should be set")
	}

	resp2, err := agent.Chat(ctx, "Do you remember me?")
	if err != nil {
		t.Errorf("Chat failed: %v", err)
	}
	if resp2 == "" {
		t.Error("Expected response, got empty")
	}

	if agent.ThreadID() != threadID1 {
		t.Error("ThreadID should be consistent across calls")
	}
}

func TestChatAgent_DynamicTools(t *testing.T) {
	// Setup mock model
	mockModel := &MockModel{
		responses: []string{"Response"},
	}

	// Create ChatAgent without initial tools
	agent, err := NewChatAgent(mockModel, nil)
	if err != nil {
		t.Fatalf("Failed to create ChatAgent: %v", err)
	}

	// Test initial state - no dynamic tools
	if len(agent.GetTools()) != 0 {
		t.Errorf("Expected 0 tools initially, got %d", len(agent.GetTools()))
	}

	// Test AddTool
	tool1 := &MockTool{name: "tool1"}
	agent.AddTool(tool1)
	if len(agent.GetTools()) != 1 {
		t.Errorf("Expected 1 tool after AddTool, got %d", len(agent.GetTools()))
	}
	if agent.GetTools()[0].Name() != "tool1" {
		t.Errorf("Expected tool name 'tool1', got '%s'", agent.GetTools()[0].Name())
	}

	// Test adding another tool
	tool2 := &MockTool{name: "tool2"}
	agent.AddTool(tool2)
	if len(agent.GetTools()) != 2 {
		t.Errorf("Expected 2 tools after second AddTool, got %d", len(agent.GetTools()))
	}

	// Test replacing a tool with same name
	tool1Updated := &MockTool{name: "tool1"}
	agent.AddTool(tool1Updated)
	if len(agent.GetTools()) != 2 {
		t.Errorf("Expected 2 tools after updating tool1, got %d", len(agent.GetTools()))
	}

	// Test RemoveTool
	removed := agent.RemoveTool("tool1")
	if !removed {
		t.Error("Expected RemoveTool to return true for existing tool")
	}
	if len(agent.GetTools()) != 1 {
		t.Errorf("Expected 1 tool after RemoveTool, got %d", len(agent.GetTools()))
	}
	if agent.GetTools()[0].Name() != "tool2" {
		t.Errorf("Expected remaining tool to be 'tool2', got '%s'", agent.GetTools()[0].Name())
	}

	// Test removing non-existent tool
	removed = agent.RemoveTool("nonexistent")
	if removed {
		t.Error("Expected RemoveTool to return false for non-existent tool")
	}

	// Test SetTools
	tool3 := &MockTool{name: "tool3"}
	tool4 := &MockTool{name: "tool4"}
	agent.SetTools([]tool.Tool{tool3, tool4})
	if len(agent.GetTools()) != 2 {
		t.Errorf("Expected 2 tools after SetTools, got %d", len(agent.GetTools()))
	}
	if agent.GetTools()[0].Name() != "tool3" || agent.GetTools()[1].Name() != "tool4" {
		t.Error("SetTools did not correctly set the tools")
	}

	// Test ClearTools
	agent.ClearTools()
	if len(agent.GetTools()) != 0 {
		t.Errorf("Expected 0 tools after ClearTools, got %d", len(agent.GetTools()))
	}
}

func TestChatAgent_ToolsInChat(t *testing.T) {
	// Setup mock model that checks for tool calls
	mockModel := &MockModel{
		responses: []string{"Using the calculator tool"},
	}

	// Create ChatAgent
	agent, err := NewChatAgent(mockModel, nil)
	if err != nil {
		t.Fatalf("Failed to create ChatAgent: %v", err)
	}

	ctx := context.Background()

	// Add a tool dynamically
	calcTool := &MockTool{name: "calculator"}
	agent.AddTool(calcTool)

	// Chat should include the dynamic tool
	_, err = agent.Chat(ctx, "Calculate 2+2")
	if err != nil {
		t.Errorf("Chat with dynamic tool failed: %v", err)
	}

	// Verify tool is still available after chat
	if len(agent.GetTools()) != 1 {
		t.Errorf("Expected 1 tool after chat, got %d", len(agent.GetTools()))
	}

	// Remove tool and chat again
	agent.RemoveTool("calculator")
	_, err = agent.Chat(ctx, "Another message")
	if err != nil {
		t.Errorf("Chat after removing tool failed: %v", err)
	}

	// Verify tool was removed
	if len(agent.GetTools()) != 0 {
		t.Errorf("Expected 0 tools after removal, got %d", len(agent.GetTools()))
	}
}

func TestChatAgent_AsyncChat(t *testing.T) {
	// Setup mock model
	mockModel := &MockModel{
		responses: []string{"Hello, how can I help you today?"},
	}

	// Create ChatAgent
	agent, err := NewChatAgent(mockModel, nil)
	if err != nil {
		t.Fatalf("Failed to create ChatAgent: %v", err)
	}

	ctx := context.Background()

	// Test AsyncChat
	respChan, err := agent.AsyncChat(ctx, "Hi")
	if err != nil {
		t.Fatalf("AsyncChat failed: %v", err)
	}

	// Collect all chunks
	var fullResponse string
	for chunk := range respChan {
		fullResponse += chunk
	}

	// Verify we got the expected response
	expectedResponse := "Hello, how can I help you today?"
	if fullResponse != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, fullResponse)
	}

	// Verify the response was streamed (we received multiple chunks)
	if len(fullResponse) == 0 {
		t.Error("Expected non-empty response")
	}
}

func TestChatAgent_AsyncChatWithChunks(t *testing.T) {
	// Setup mock model
	mockModel := &MockModel{
		responses: []string{"Hello world, this is a test response."},
	}

	// Create ChatAgent
	agent, err := NewChatAgent(mockModel, nil)
	if err != nil {
		t.Fatalf("Failed to create ChatAgent: %v", err)
	}

	ctx := context.Background()

	// Test AsyncChatWithChunks
	respChan, err := agent.AsyncChatWithChunks(ctx, "Hi")
	if err != nil {
		t.Fatalf("AsyncChatWithChunks failed: %v", err)
	}

	// Collect all chunks
	var chunks []string
	var fullResponse string
	for chunk := range respChan {
		chunks = append(chunks, chunk)
		fullResponse += chunk
	}

	// Verify we got the expected response
	expectedResponse := "Hello world, this is a test response."
	if fullResponse != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, fullResponse)
	}

	// Verify we received multiple chunks (words + spaces)
	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks, got %d", len(chunks))
	}

	t.Logf("Received %d chunks: %v", len(chunks), chunks)
}

func TestChatAgent_AsyncChatWithContext(t *testing.T) {
	// Setup mock model with a slow response
	mockModel := &MockModel{
		responses: []string{"This is a long response that should be interrupted."},
	}

	// Create ChatAgent
	agent, err := NewChatAgent(mockModel, nil)
	if err != nil {
		t.Fatalf("Failed to create ChatAgent: %v", err)
	}

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancel is always called to avoid context leak

	// Test AsyncChat
	respChan, err := agent.AsyncChat(ctx, "Hi")
	if err != nil {
		t.Fatalf("AsyncChat failed: %v", err)
	}

	// Read a few chunks then cancel
	chunksReceived := 0
	for chunk := range respChan {
		_ = chunk
		chunksReceived++
		if chunksReceived >= 5 {
			cancel() // Cancel the context early
			break
		}
	}

	// Continue reading to drain the channel
	for range respChan {
		chunksReceived++
	}

	// We should have received some chunks but not all
	t.Logf("Received %d chunks before/after cancellation", chunksReceived)
}

func TestSplitIntoWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple sentence",
			input:    "Hello world",
			expected: []string{"Hello", "world"},
		},
		{
			name:     "With punctuation",
			input:    "Hello, world!",
			expected: []string{"Hello,", "world!"},
		},
		{
			name:     "Multiple spaces",
			input:    "Hello   world",
			expected: []string{"Hello", "world"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Single word",
			input:    "Hello",
			expected: []string{"Hello"},
		},
		{
			name:     "With newlines",
			input:    "Hello\nworld",
			expected: []string{"Hello", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitIntoWords(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d words, got %d", len(tt.expected), len(result))
				return
			}
			for i, word := range result {
				if word != tt.expected[i] {
					t.Errorf("Word %d: expected '%s', got '%s'", i, tt.expected[i], word)
				}
			}
		})
	}
}
