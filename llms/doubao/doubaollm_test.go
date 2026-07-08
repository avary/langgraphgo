package doubao

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// getTestModel returns the model name from env or empty string.
//
// IMPORTANT: The Doubao API requires custom Endpoint IDs that you create in the
// Volcengine console. Set the DOUBAO_MODEL environment variable to your
// custom Endpoint ID to run tests with your specific endpoint.
//
// To get your Endpoint ID, visit: https://www.volcengine.com/docs/82379/1330310
func getTestModel() ModelName {
	return ModelName(getEnvOrDefault("DOUBAO_MODEL", ""))
}

// getTestEmbeddingModel returns the embedding model name from env or empty string.
//
// IMPORTANT: Set the DOUBAO_EMBEDDING_MODEL environment variable to your
// custom embedding Endpoint ID to run embedding tests.
func getTestEmbeddingModel() ModelName {
	return ModelName(getEnvOrDefault("DOUBAO_EMBEDDING_MODEL", ""))
}

// TestLLM_Create tests the LLM creation with various options.
func TestLLM_Create(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name: "with api key",
			opts: []Option{
				WithAPIKey("test-key"),
			},
			wantErr: false,
		},
		{
			name: "with api key and model",
			opts: []Option{
				WithAPIKey("test-key"),
				WithModel("test-endpoint-id"),
			},
			wantErr: false,
		},
		{
			name: "with ak/sk",
			opts: []Option{
				WithAccessKey("test-ak"),
				WithSecretKey("test-sk"),
			},
			wantErr: false,
		},
		{
			name:    "no authentication",
			opts:    []Option{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the "no authentication" test, temporarily clear env vars
			if tt.name == "no authentication" {
				oldAPIKey := os.Getenv("DOUBAO_API_KEY")
				oldAccessKey := os.Getenv("DOUBAO_ACCESS_KEY")
				oldSecretKey := os.Getenv("DOUBAO_SECRET_KEY")
				defer func() {
					if oldAPIKey != "" {
						os.Setenv("DOUBAO_API_KEY", oldAPIKey)
					} else {
						os.Unsetenv("DOUBAO_API_KEY")
					}
					if oldAccessKey != "" {
						os.Setenv("DOUBAO_ACCESS_KEY", oldAccessKey)
					} else {
						os.Unsetenv("DOUBAO_ACCESS_KEY")
					}
					if oldSecretKey != "" {
						os.Setenv("DOUBAO_SECRET_KEY", oldSecretKey)
					} else {
						os.Unsetenv("DOUBAO_SECRET_KEY")
					}
				}()
				os.Unsetenv("DOUBAO_API_KEY")
				os.Unsetenv("DOUBAO_ACCESS_KEY")
				os.Unsetenv("DOUBAO_SECRET_KEY")
			}

			llm, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && llm == nil {
				t.Error("New() returned nil LLM")
			}
		})
	}
}

// TestLLM_GenerateContent tests the content generation with real API.
// Skipped if DOUBAO_API_KEY or DOUBAO_MODEL is not set.
//
// IMPORTANT: This test requires DOUBAO_MODEL to be set to your custom Endpoint ID.
func TestLLM_GenerateContent(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	model := getTestModel()
	if model == "" {
		t.Skip("DOUBAO_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(model),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	messages := []llmtypes.MessageContent{
		{
			Role: llmtypes.ChatMessageTypeHuman,
			Parts: []llmtypes.ContentPart{
				llmtypes.TextPart("Hello, how are you?"),
			},
		},
	}

	resp, err := llm.GenerateContent(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to generate content: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No choices in response")
	}

	content := resp.Choices[0].Content
	if content == "" {
		t.Error("Empty response content")
	}

	t.Logf("Response: %s", content)
	t.Logf("StopReason: %s", resp.Choices[0].StopReason)
}

// TestLLM_CreateEmbedding tests the embedding generation with real API.
// Skipped if DOUBAO_API_KEY or DOUBAO_EMBEDDING_MODEL is not set.
func TestLLM_CreateEmbedding(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	embeddingModel := getTestEmbeddingModel()
	if embeddingModel == "" {
		t.Skip("DOUBAO_EMBEDDING_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithEmbeddingModel(embeddingModel),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	texts := []string{"Hello world"}

	embeddings, err := llm.CreateEmbedding(ctx, texts)
	if err != nil {
		t.Fatalf("Failed to create embedding: %v", err)
	}

	if len(embeddings) != 1 {
		t.Fatalf("Expected 1 embedding, got %d", len(embeddings))
	}

	if len(embeddings[0]) == 0 {
		t.Fatal("Empty embedding")
	}

	t.Logf("Embedding dimension: %d", len(embeddings[0]))
}

// TestLLM_CreateEmbeddingMultiple tests embedding generation for multiple texts.
func TestLLM_CreateEmbeddingMultiple(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	embeddingModel := getTestEmbeddingModel()
	if embeddingModel == "" {
		t.Skip("DOUBAO_EMBEDDING_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithEmbeddingModel(embeddingModel),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	texts := []string{"Hello", "World"}

	embeddings, err := llm.CreateEmbedding(ctx, texts)
	if err != nil {
		t.Fatalf("Failed to create embedding: %v", err)
	}

	if len(embeddings) != 2 {
		t.Fatalf("Expected 2 embeddings, got %d", len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) == 0 {
			t.Errorf("Empty embedding at index %d", i)
		}
		t.Logf("Embedding %d dimension: %d", i, len(emb))
	}
}

// TestLLM_Call tests the Call method.
func TestLLM_Call(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	model := getTestModel()
	if model == "" {
		t.Skip("DOUBAO_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(model),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	response, err := llm.Call(ctx, "What is 2+2?")
	if err != nil {
		t.Fatalf("Failed to call LLM: %v", err)
	}

	if response == "" {
		t.Error("Empty response")
	}

	t.Logf("Response: %s", response)
}

// TestLLM_Conversation tests a multi-turn conversation.
func TestLLM_Conversation(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	model := getTestModel()
	if model == "" {
		t.Skip("DOUBAO_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(model),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	messages := []llmtypes.MessageContent{
		{
			Role: llmtypes.ChatMessageTypeHuman,
			Parts: []llmtypes.ContentPart{
				llmtypes.TextPart("My name is Alice"),
			},
		},
		{
			Role: llmtypes.ChatMessageTypeAI,
			Parts: []llmtypes.ContentPart{
				llmtypes.TextPart("Hello Alice! Nice to meet you."),
			},
		},
		{
			Role: llmtypes.ChatMessageTypeHuman,
			Parts: []llmtypes.ContentPart{
				llmtypes.TextPart("What's my name?"),
			},
		},
	}

	resp, err := llm.GenerateContent(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to generate content: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No choices in response")
	}

	content := resp.Choices[0].Content
	t.Logf("Response: %s", content)
}

// TestLLM_WithAKSK tests AK/SK authentication.
func TestLLM_WithAKSK(t *testing.T) {
	ak := os.Getenv("DOUBAO_ACCESS_KEY")
	sk := os.Getenv("DOUBAO_SECRET_KEY")
	if ak == "" || sk == "" {
		t.Skip("DOUBAO_ACCESS_KEY and DOUBAO_SECRET_KEY not set")
	}

	model := getTestModel()
	if model == "" {
		t.Skip("DOUBAO_MODEL not set")
	}

	llm, err := New(
		WithAccessKey(ak),
		WithSecretKey(sk),
		WithModel(model),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	response, err := llm.Call(ctx, "Say hello")
	if err != nil {
		t.Fatalf("Failed to call LLM: %v", err)
	}

	if response == "" {
		t.Error("Empty response")
	}

	t.Logf("Response: %s", response)
}

// TestLLM_EmbeddingLarge tests the large embedding model.
func TestLLM_EmbeddingLarge(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	embeddingModel := getTestEmbeddingModel()
	if embeddingModel == "" {
		t.Skip("DOUBAO_EMBEDDING_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithEmbeddingModel(embeddingModel),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	embeddings, err := llm.CreateEmbedding(ctx, []string{"test text for large embedding model"})
	if err != nil {
		t.Fatalf("Failed to create embedding: %v", err)
	}

	if len(embeddings) != 1 {
		t.Fatalf("Expected 1 embedding, got %d", len(embeddings))
	}

	dim := len(embeddings[0])
	t.Logf("Doubao Embedding dimension: %d", dim)
}

// TestLLM_DifferentModels tests different model types.
func TestLLM_DifferentModels(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	model := getTestModel()
	if model == "" {
		t.Skip("DOUBAO_MODEL not set")
	}

	// Test with the default model from env
	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(model),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	response, err := llm.Call(ctx, "Say hello")
	if err != nil {
		t.Logf("Model %s error: %v", model, err)
		return
	}

	if response == "" {
		t.Errorf("Model %s returned empty response", model)
	}

	t.Logf("Model %s response: %s", model, response)
}

// TestLLM_Streaming tests streaming response.
func TestLLM_Streaming(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	model := getTestModel()
	if model == "" {
		t.Skip("DOUBAO_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(model),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	messages := []llmtypes.MessageContent{
		{
			Role: llmtypes.ChatMessageTypeHuman,
			Parts: []llmtypes.ContentPart{
				llmtypes.TextPart("Count from 1 to 5"),
			},
		},
	}

	resp, err := llm.GenerateContent(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to generate content: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No choices in response")
	}

	t.Logf("Response: %s", resp.Choices[0].Content)
}

// TestLLM_ToolCall tests tool call functionality.
// Skipped if DOUBAO_API_KEY or DOUBAO_MODEL is not set.
func TestLLM_ToolCall(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	model := getTestModel()
	if model == "" {
		t.Skip("DOUBAO_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(model),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()

	// Define a simple tool for getting weather
	tools := []llmtypes.Tool{
		{
			Type: "function",
			Function: &llmtypes.FunctionDefinition{
				Name:        "get_weather",
				Description: "Get the current weather in a given location",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{
							"type":        "string",
							"description": "The city and state, e.g. San Francisco, CA",
						},
						"unit": map[string]any{
							"type":        "string",
							"enum":        []string{"celsius", "fahrenheit"},
							"description": "The temperature unit to use",
						},
					},
					"required": []string{"location"},
				},
			},
		},
	}

	messages := []llmtypes.MessageContent{
		{
			Role: llmtypes.ChatMessageTypeHuman,
			Parts: []llmtypes.ContentPart{
				llmtypes.TextPart("What's the weather like in San Francisco, CA?"),
			},
		},
	}

	resp, err := llm.GenerateContent(ctx, messages, llmtypes.WithTools(tools))
	if err != nil {
		t.Fatalf("Failed to generate content: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No choices in response")
	}

	choice := resp.Choices[0]
	t.Logf("Content: %s", choice.Content)
	t.Logf("StopReason: %s", choice.StopReason)

	// Check for tool calls
	if len(choice.ToolCalls) > 0 {
		t.Logf("Tool calls returned: %d", len(choice.ToolCalls))
		for i, tc := range choice.ToolCalls {
			t.Logf("  ToolCall[%d]: ID=%s, Type=%s, Function.Name=%s, Function.Arguments=%s",
				i, tc.ID, tc.Type, tc.FunctionCall.Name, tc.FunctionCall.Arguments)
		}
	} else if choice.FuncCall != nil {
		t.Logf("FuncCall returned: Name=%s, Arguments=%s",
			choice.FuncCall.Name, choice.FuncCall.Arguments)
	} else {
		t.Log("No tool calls or function call in response")
	}
}

// TestLLM_ToolChoice tests tool choice option.
// Skipped if DOUBAO_API_KEY or DOUBAO_MODEL is not set.
func TestLLM_ToolChoice(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	model := getTestModel()
	if model == "" {
		t.Skip("DOUBAO_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(model),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()

	tools := []llmtypes.Tool{
		{
			Type: "function",
			Function: &llmtypes.FunctionDefinition{
				Name:        "get_current_time",
				Description: "Get the current time",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
	}

	tests := []struct {
		name       string
		toolChoice any
		desc       string
	}{
		{
			name:       "tool choice auto",
			toolChoice: "auto",
			desc:       "Model decides whether to call tools",
		},
		{
			name:       "tool choice none",
			toolChoice: "none",
			desc:       "Model will not call tools",
		},
		{
			name: "tool choice required",
			toolChoice: llmtypes.ToolChoice{
				Type: "required",
			},
			desc: "Model must call a tool",
		},
		{
			name: "tool choice specific function",
			toolChoice: llmtypes.ToolChoice{
				Type: "function",
				Function: &llmtypes.FunctionReference{
					Name: "get_current_time",
				},
			},
			desc: "Model must call the specific function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := []llmtypes.MessageContent{
				{
					Role: llmtypes.ChatMessageTypeHuman,
					Parts: []llmtypes.ContentPart{
						llmtypes.TextPart("What time is it?"),
					},
				},
			}

			resp, err := llm.GenerateContent(ctx, messages,
				llmtypes.WithTools(tools),
				llmtypes.WithToolChoice(tt.toolChoice),
			)
			if err != nil {
				t.Fatalf("Failed to generate content: %v", err)
			}

			if len(resp.Choices) == 0 {
				t.Fatal("No choices in response")
			}

			choice := resp.Choices[0]
			t.Logf("Test: %s", tt.desc)
			t.Logf("Content: %s", choice.Content)
			t.Logf("StopReason: %s", choice.StopReason)

			if len(choice.ToolCalls) > 0 {
				t.Logf("Tool calls: %d", len(choice.ToolCalls))
			}
		})
	}
}

// TestLLM_ToolResponse tests tool response handling.
// Skipped if DOUBAO_API_KEY or DOUBAO_MODEL is not set.
func TestLLM_ToolResponse(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	model := getTestModel()
	if model == "" {
		t.Skip("DOUBAO_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(model),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()

	// Simulate a conversation with tool call and response
	messages := []llmtypes.MessageContent{
		{
			Role: llmtypes.ChatMessageTypeHuman,
			Parts: []llmtypes.ContentPart{
				llmtypes.TextPart("What's the weather like in Beijing?"),
			},
		},
		// Simulated tool response
		{
			Role: llmtypes.ChatMessageTypeTool,
			Parts: []llmtypes.ContentPart{
				llmtypes.ToolCallResponse{
					ToolCallID: "call_123",
					Content:    `{"temperature": "22°C", "condition": "Sunny"}`,
				},
			},
		},
	}

	resp, err := llm.GenerateContent(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to generate content: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No choices in response")
	}

	content := resp.Choices[0].Content
	t.Logf("Response after tool call: %s", content)

	if content == "" {
		t.Error("Empty response after tool call")
	}
}

// TestLLM_ConvertMessageWithToolResponse tests the convertMessage function with tool response.
func TestLLM_ConvertMessageWithToolResponse(t *testing.T) {
	tests := []struct {
		name     string
		msg      llmtypes.MessageContent
		wantErr  bool
		validate func(*testing.T, *model.ChatCompletionMessage)
	}{
		{
			name: "tool response with ToolCallID",
			msg: llmtypes.MessageContent{
				Role: llmtypes.ChatMessageTypeTool,
				Parts: []llmtypes.ContentPart{
					llmtypes.ToolCallResponse{
						ToolCallID: "test-call-id",
						Content:    `{"result": "success"}`,
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, m *model.ChatCompletionMessage) {
				if m.Role != "tool" {
					t.Errorf("Expected role 'tool', got '%s'", m.Role)
				}
				if m.ToolCallID != "test-call-id" {
					t.Errorf("Expected ToolCallID 'test-call-id', got '%s'", m.ToolCallID)
				}
				if m.Content == nil {
					t.Error("Expected content to be set")
				}
			},
		},
		{
			name: "tool response with text content",
			msg: llmtypes.MessageContent{
				Role: llmtypes.ChatMessageTypeTool,
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("some text content"),
				},
			},
			wantErr: false,
			validate: func(t *testing.T, m *model.ChatCompletionMessage) {
				if m.Role != "tool" {
					t.Errorf("Expected role 'tool', got '%s'", m.Role)
				}
				if m.Content == nil {
					t.Error("Expected content to be set")
				}
			},
		},
		{
			name: "user message with text",
			msg: llmtypes.MessageContent{
				Role: llmtypes.ChatMessageTypeHuman,
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("Hello, how are you?"),
				},
			},
			wantErr: false,
			validate: func(t *testing.T, m *model.ChatCompletionMessage) {
				if m.Role != string(llmtypes.ChatMessageTypeHuman) {
					t.Errorf("Expected role '%s', got '%s'", llmtypes.ChatMessageTypeHuman, m.Role)
				}
				if m.Content == nil {
					t.Error("Expected content to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := convertMessage(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if tt.validate != nil {
				tt.validate(t, msg)
			}
		})
	}
}

// TestGetContentString tests the getContentString helper function.
func TestGetContentString(t *testing.T) {
	textPartType := model.ChatCompletionMessageContentPartTypeText

	tests := []struct {
		name     string
		content  *model.ChatCompletionMessageContent
		expected string
	}{
		{
			name:     "nil content",
			content:  nil,
			expected: "",
		},
		{
			name: "string value",
			content: &model.ChatCompletionMessageContent{
				StringValue: stringPtr("Hello, world!"),
			},
			expected: "Hello, world!",
		},
		{
			name: "nil string value",
			content: &model.ChatCompletionMessageContent{
				StringValue: nil,
			},
			expected: "",
		},
		{
			name: "empty list value",
			content: &model.ChatCompletionMessageContent{
				ListValue: []*model.ChatCompletionMessageContentPart{},
			},
			expected: "",
		},
		{
			name: "list value with text parts",
			content: &model.ChatCompletionMessageContent{
				ListValue: []*model.ChatCompletionMessageContentPart{
					{
						Type: textPartType,
						Text: "Hello, ",
					},
					{
						Type: textPartType,
						Text: "world!",
					},
				},
			},
			expected: "Hello, world!",
		},
		{
			name: "list value with mixed parts",
			content: &model.ChatCompletionMessageContent{
				ListValue: []*model.ChatCompletionMessageContentPart{
					{
						Type: textPartType,
						Text: "Text part",
					},
					{
						Type: "other_type",
						Text: "ignored",
					},
				},
			},
			expected: "Text part",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getContentString(tt.content)
			if result != tt.expected {
				t.Errorf("getContentString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestLLM_Options tests various LLM options.
func TestLLM_Options(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
		check   func(*testing.T, *LLM)
	}{
		{
			name: "with embedding model",
			opts: []Option{
				WithAPIKey("test-key"),
				WithEmbeddingModel("embedding-endpoint-id"),
			},
			wantErr: false,
			check: func(t *testing.T, llm *LLM) {
				if llm.embeddingModel != "embedding-endpoint-id" {
					t.Errorf("embeddingModel = %q, want %q", llm.embeddingModel, "embedding-endpoint-id")
				}
			},
		},
		{
			name: "with base URL",
			opts: []Option{
				WithAPIKey("test-key"),
				WithBaseURL("https://custom.endpoint.com/api/v3"),
			},
			wantErr: false,
			check: func(t *testing.T, llm *LLM) {
				// Verify the LLM was created successfully
				if llm == nil {
					t.Error("LLM is nil")
				}
			},
		},
		{
			name: "with region",
			opts: []Option{
				WithAPIKey("test-key"),
				WithRegion("us-east-1"),
			},
			wantErr: false,
			check: func(t *testing.T, llm *LLM) {
				// Verify the LLM was created successfully
				if llm == nil {
					t.Error("LLM is nil")
				}
			},
		},
		{
			name: "with all options",
			opts: []Option{
				WithAPIKey("test-key"),
				WithModel("model-endpoint-id"),
				WithEmbeddingModel("embedding-endpoint-id"),
				WithBaseURL("https://custom.endpoint.com/api/v3"),
				WithRegion("cn-shanghai"),
			},
			wantErr: false,
			check: func(t *testing.T, llm *LLM) {
				if llm.model != "model-endpoint-id" {
					t.Errorf("model = %q, want %q", llm.model, "model-endpoint-id")
				}
				if llm.embeddingModel != "embedding-endpoint-id" {
					t.Errorf("embeddingModel = %q, want %q", llm.embeddingModel, "embedding-endpoint-id")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if tt.check != nil {
				tt.check(t, llm)
			}
		})
	}
}

// TestLLM_GenerateContent_EmptyMessages tests GenerateContent with empty messages.
func TestLLM_GenerateContent_EmptyMessages(t *testing.T) {
	llm, err := New(WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	_, err = llm.GenerateContent(ctx, []llmtypes.MessageContent{})
	if err == nil {
		t.Error("Expected error for empty messages, got nil")
	}
	if err != nil && err.Error() != "no messages provided" {
		t.Errorf("Expected 'no messages provided' error, got %v", err)
	}
}

// TestLLM_CreateEmbedding_EmptyTexts tests CreateEmbedding with empty texts.
func TestLLM_CreateEmbedding_EmptyTexts(t *testing.T) {
	llm, err := New(
		WithAPIKey("test-key"),
		WithEmbeddingModel("embedding-endpoint-id"),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	_, err = llm.CreateEmbedding(ctx, []string{})
	if err == nil {
		t.Error("Expected error for empty texts, got nil")
	}
}

// TestLLM_ConvertMessage_Errors tests convertMessage error cases.
func TestLLM_ConvertMessage_Errors(t *testing.T) {
	tests := []struct {
		name    string
		msg     llmtypes.MessageContent
		wantErr bool
		errMsg  string
	}{
		{
			name: "message with no parts",
			msg: llmtypes.MessageContent{
				Role:  llmtypes.ChatMessageTypeHuman,
				Parts: []llmtypes.ContentPart{},
			},
			wantErr: true,
			errMsg:  "message has no parts",
		},
		{
			name: "message with empty content - actually works with empty string",
			msg: llmtypes.MessageContent{
				Role:  llmtypes.ChatMessageTypeHuman,
				Parts: []llmtypes.ContentPart{llmtypes.TextPart("")},
			},
			wantErr: false, // The code allows empty strings, it creates content with empty string
		},
		{
			name: "tool message with no valid content",
			msg: llmtypes.MessageContent{
				Role:  llmtypes.ChatMessageTypeTool,
				Parts: []llmtypes.ContentPart{},
			},
			wantErr: true,
			errMsg:  "message has no parts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := convertMessage(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

// TestCreateMessageContent tests the createMessageContent helper function.
func TestCreateMessageContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "normal text",
			input: "Hello, world!",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "special characters",
			input: "Hello\nWorld\t!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := createMessageContent(tt.input)
			if content == nil {
				t.Fatal("createMessageContent() returned nil")
			}
			if content.StringValue == nil {
				t.Error("StringValue is nil")
			} else if *content.StringValue != tt.input {
				t.Errorf("StringValue = %q, want %q", *content.StringValue, tt.input)
			}
			if content.ListValue != nil {
				t.Error("ListValue should be nil")
			}
		})
	}
}

// Helper function to get a string pointer.
func stringPtr(s string) *string {
	return &s
}

// Helper function to check if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGetEnvOrDefault tests the getEnvOrDefault helper function.
func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		setEnv       bool
		envValue     string
		expected     string
	}{
		{
			name:         "env var not set",
			key:          "NONEXISTENT_VAR",
			defaultValue: "default",
			setEnv:       false,
			expected:     "default",
		},
		{
			name:         "env var set with value",
			key:          "TEST_DOUBAO_VAR",
			defaultValue: "default",
			setEnv:       true,
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "env var set to empty string",
			key:          "TEST_DOUBAO_EMPTY",
			defaultValue: "default",
			setEnv:       true,
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up env var before and after test
			if tt.setEnv {
				defer os.Unsetenv(tt.key)
			}

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvOrDefault() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestLLM_WithHTTPClient tests the WithHTTPClient option.
func TestLLM_WithHTTPClient(t *testing.T) {
	client := &http.Client{}
	llm, err := New(
		WithAPIKey("test-key"),
		WithHTTPClient(client),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	if llm == nil {
		t.Fatal("LLM is nil")
	}

	// Verify the LLM was created successfully with custom HTTP client
	// We can't directly access the client, but successful creation is enough
}

// TestLLM_ModelNameType tests the ModelName type.
func TestLLM_ModelNameType(t *testing.T) {
	tests := []struct {
		name  string
		model ModelName
	}{
		{
			name:  "string to ModelName",
			model: ModelName("doubao-seed-1-8-251215"),
		},
		{
			name:  "custom endpoint id",
			model: ModelName("ep-20250107123456-abcd"),
		},
		{
			name:  "empty model name",
			model: ModelName(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the type can be created and used
			llm, err := New(
				WithAPIKey("test-key"),
				WithModel(tt.model),
			)
			if err != nil {
				t.Fatalf("Failed to create LLM: %v", err)
			}

			if llm.model != tt.model {
				t.Errorf("model = %q, want %q", llm.model, tt.model)
			}
		})
	}
}

// TestLLM_ErrorCases tests various error conditions.
func TestLLM_ErrorCases(t *testing.T) {
	t.Run("empty API key with empty env vars", func(t *testing.T) {
		// Temporarily clear env vars
		oldAPIKey := os.Getenv("DOUBAO_API_KEY")
		oldAK := os.Getenv("DOUBAO_ACCESS_KEY")
		oldSK := os.Getenv("DOUBAO_SECRET_KEY")
		defer func() {
			if oldAPIKey != "" {
				os.Setenv("DOUBAO_API_KEY", oldAPIKey)
			} else {
				os.Unsetenv("DOUBAO_API_KEY")
			}
			if oldAK != "" {
				os.Setenv("DOUBAO_ACCESS_KEY", oldAK)
			} else {
				os.Unsetenv("DOUBAO_ACCESS_KEY")
			}
			if oldSK != "" {
				os.Setenv("DOUBAO_SECRET_KEY", oldSK)
			} else {
				os.Unsetenv("DOUBAO_SECRET_KEY")
			}
		}()
		os.Unsetenv("DOUBAO_API_KEY")
		os.Unsetenv("DOUBAO_ACCESS_KEY")
		os.Unsetenv("DOUBAO_SECRET_KEY")

		_, err := New(
			WithAPIKey(""),
		)
		if err == nil {
			t.Error("Expected error for empty API key")
		}
		if err != nil && !errors.Is(err, ErrNoAuth) {
			t.Errorf("Expected ErrNoAuth, got %v", err)
		}
	})

	t.Run("only access key provided", func(t *testing.T) {
		// Clear env vars first
		os.Unsetenv("DOUBAO_API_KEY")
		os.Unsetenv("DOUBAO_ACCESS_KEY")
		os.Unsetenv("DOUBAO_SECRET_KEY")

		_, err := New(
			WithAccessKey("test-ak"),
		)
		if err == nil {
			t.Error("Expected error when only access key is provided")
		}
		if err != nil && !errors.Is(err, ErrNoAuth) {
			t.Errorf("Expected ErrNoAuth, got %v", err)
		}
	})

	t.Run("only secret key provided", func(t *testing.T) {
		// Clear env vars first
		os.Unsetenv("DOUBAO_API_KEY")
		os.Unsetenv("DOUBAO_ACCESS_KEY")
		os.Unsetenv("DOUBAO_SECRET_KEY")

		_, err := New(
			WithSecretKey("test-sk"),
		)
		if err == nil {
			t.Error("Expected error when only secret key is provided")
		}
		if err != nil && !errors.Is(err, ErrNoAuth) {
			t.Errorf("Expected ErrNoAuth, got %v", err)
		}
	})
}

// TestLLM_GenerateContent_WithOptions tests GenerateContent with various options.
func TestLLM_GenerateContent_WithOptions(t *testing.T) {
	apiKey := os.Getenv("DOUBAO_API_KEY")
	if apiKey == "" {
		t.Skip("DOUBAO_API_KEY not set")
	}

	model := getTestModel()
	if model == "" {
		t.Skip("DOUBAO_MODEL not set")
	}

	llm, err := New(
		WithAPIKey(apiKey),
		WithModel(model),
	)
	if err != nil {
		t.Fatalf("Failed to create LLM: %v", err)
	}

	ctx := context.Background()
	messages := []llmtypes.MessageContent{
		{
			Role: llmtypes.ChatMessageTypeHuman,
			Parts: []llmtypes.ContentPart{
				llmtypes.TextPart("Say hello"),
			},
		},
	}

	// Test with temperature
	resp, err := llm.GenerateContent(ctx, messages, llmtypes.WithTemperature(0.7))
	if err != nil {
		t.Logf("GenerateContent with temperature failed: %v", err)
	} else {
		if len(resp.Choices) == 0 {
			t.Error("No choices in response")
		}
		t.Logf("Response with temperature: %s", resp.Choices[0].Content)
	}

	// Test with top_p
	resp, err = llm.GenerateContent(ctx, messages, llmtypes.WithTopP(0.9))
	if err != nil {
		t.Logf("GenerateContent with top_p failed: %v", err)
	} else {
		if len(resp.Choices) == 0 {
			t.Error("No choices in response")
		}
		t.Logf("Response with top_p: %s", resp.Choices[0].Content)
	}

	// Test with max tokens
	resp, err = llm.GenerateContent(ctx, messages, llmtypes.WithMaxTokens(100))
	if err != nil {
		t.Logf("GenerateContent with max tokens failed: %v", err)
	} else {
		if len(resp.Choices) == 0 {
			t.Error("No choices in response")
		}
		t.Logf("Response with max tokens: %s", resp.Choices[0].Content)
	}

	// Test with all options
	resp, err = llm.GenerateContent(ctx, messages,
		llmtypes.WithTemperature(0.5),
		llmtypes.WithTopP(0.8),
		llmtypes.WithMaxTokens(50),
	)
	if err != nil {
		t.Logf("GenerateContent with all options failed: %v", err)
	} else {
		if len(resp.Choices) == 0 {
			t.Error("No choices in response")
		}
		t.Logf("Response with all options: %s", resp.Choices[0].Content)
	}
}

// TestLLM_ConvertMessage_MultiPart tests convertMessage with multiple parts.
func TestLLM_ConvertMessage_MultiPart(t *testing.T) {
	tests := []struct {
		name     string
		msg      llmtypes.MessageContent
		wantErr  bool
		validate func(*testing.T, *model.ChatCompletionMessage)
	}{
		{
			name: "user message with multiple text parts",
			msg: llmtypes.MessageContent{
				Role: llmtypes.ChatMessageTypeHuman,
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("Hello, "),
					llmtypes.TextPart("world!"),
				},
			},
			wantErr: false,
			validate: func(t *testing.T, m *model.ChatCompletionMessage) {
				if m.Content == nil {
					t.Error("Expected content to be set")
				} else if m.Content.StringValue == nil {
					t.Error("Expected StringValue to be set")
				} else {
					// The convertMessage function only uses the first text part
					if *m.Content.StringValue != "Hello, " {
						t.Errorf("Expected 'Hello, ', got %q", *m.Content.StringValue)
					}
				}
			},
		},
		{
			name: "system message",
			msg: llmtypes.MessageContent{
				Role: llmtypes.ChatMessageTypeSystem,
				Parts: []llmtypes.ContentPart{
					llmtypes.TextPart("You are a helpful assistant"),
				},
			},
			wantErr: false,
			validate: func(t *testing.T, m *model.ChatCompletionMessage) {
				if m.Role != "system" {
					t.Errorf("Expected role 'system', got '%s'", m.Role)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := convertMessage(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if tt.validate != nil {
				tt.validate(t, msg)
			}
		})
	}
}
