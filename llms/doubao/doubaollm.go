package doubao

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/llms"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

var (
	ErrEmptyResponse = errors.New("no response")
	ErrNoAuth        = errors.New("no authentication provided")
)

// LLM is a client for Doubao (Volcengine Ark) LLM.
// It supports chat completions and embeddings using the volcengine-go-sdk.
type LLM struct {
	client         *arkruntime.Client
	model          ModelName
	embeddingModel ModelName
}

var _ llms.Model = (*LLM)(nil)

// New returns a new Doubao LLM client.
//
// Authentication options (choose one):
// 1. WithAPIKey(apiKey) - API Key authentication (recommended)
// 2. WithAccessKey(ak) + WithSecretKey(sk) - AK/SK authentication
//
// Model configuration:
// - WithModel(endpointID) - Set your custom Endpoint ID for chat completion
// - WithEmbeddingModel(endpointID) - Set your custom Endpoint ID for embeddings
//
// To create an endpoint and get your Endpoint ID, visit:
// https://www.volcengine.com/docs/82379/1330310
//
// Environment variables:
// - DOUBAO_API_KEY - API Key for authentication
// - DOUBAO_ACCESS_KEY - Access Key for AK/SK authentication
// - DOUBAO_SECRET_KEY - Secret Key for AK/SK authentication
//
// Example:
//
//	llm, err := doubao.New(
//		doubao.WithAPIKey("your-api-key"),
//		doubao.WithModel("your-chat-endpoint-id"),
//		doubao.WithEmbeddingModel("your-embedding-endpoint-id"),
//	)
func New(opts ...Option) (*LLM, error) {
	options := &options{
		apiKey:         getEnvOrDefault("DOUBAO_API_KEY", ""),
		accessKey:      getEnvOrDefault("DOUBAO_ACCESS_KEY", ""),
		secretKey:      getEnvOrDefault("DOUBAO_SECRET_KEY", ""),
		model:          "doubao-seed-1-8-251215", // 默认模型
		embeddingModel: "",                       // Use your Endpoint ID
		baseURL:        "https://ark.cn-beijing.volces.com/api/v3",
		region:         "cn-beijing",
	}

	for _, opt := range opts {
		opt(options)
	}

	// Validate authentication
	if options.apiKey == "" && (options.accessKey == "" || options.secretKey == "") {
		return nil, fmt.Errorf("%w: please provide API key or AccessKey/SecretKey", ErrNoAuth)
	}

	// Create client config options
	clientOpts := []arkruntime.ConfigOption{
		arkruntime.WithBaseUrl(options.baseURL),
		arkruntime.WithRegion(options.region),
	}

	if options.httpClient != nil {
		clientOpts = append(clientOpts, arkruntime.WithHTTPClient(options.httpClient))
	}

	// Create client based on authentication method
	var client *arkruntime.Client
	if options.apiKey != "" {
		// API Key authentication
		client = arkruntime.NewClientWithApiKey(options.apiKey, clientOpts...)
	} else {
		// AK/SK authentication
		client = arkruntime.NewClientWithAkSk(options.accessKey, options.secretKey, clientOpts...)
	}

	return &LLM{
		client:         client,
		model:          options.model,
		embeddingModel: options.embeddingModel,
	}, nil
}

// Call generates a response from the LLM for the given prompt.
func (o *LLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, o, prompt, options...)
}

// GenerateContent implements the Model interface.
// Uses Doubao chat completion API for text generation.
func (o *LLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if len(messages) == 0 {
		return nil, errors.New("no messages provided")
	}

	// Parse call options
	opts := llms.CallOptions{}
	for _, opt := range options {
		opt(&opts)
	}

	// Convert framework messages to arkruntime messages
	arkMessages := make([]*model.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		arkMsg, err := convertMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("convert message: %w", err)
		}
		arkMessages = append(arkMessages, arkMsg)
	}

	// Determine model to use
	modelName := o.model
	if opts.Model != "" {
		modelName = ModelName(opts.Model)
	}

	// Build chat request
	req := &model.ChatCompletionRequest{ // nolint:staticcheck
		Model:    string(modelName),
		Messages: arkMessages,
	}

	// Add tools if provided
	if len(opts.Tools) > 0 {
		tools := make([]*model.Tool, 0, len(opts.Tools))
		for _, tool := range opts.Tools {
			tools = append(tools, &model.Tool{
				Type: model.ToolTypeFunction,
				Function: &model.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			})
		}
		req.Tools = tools
	}

	// Set tool choice if specified
	if opts.ToolChoice != nil {
		switch v := opts.ToolChoice.(type) {
		case string:
			// String type: "none", "auto", "required"
			switch v {
			case "none":
				req.ToolChoice = model.ToolChoiceStringTypeNone
			case "required":
				req.ToolChoice = model.ToolChoiceStringTypeRequired
			case "auto":
				req.ToolChoice = model.ToolChoiceStringTypeAuto
			default:
				req.ToolChoice = model.ToolChoiceStringTypeAuto
			}
		case llms.ToolChoice:
			// ToolChoice struct
			switch v.Type {
			case "none":
				req.ToolChoice = model.ToolChoiceStringTypeNone
			case "required":
				req.ToolChoice = model.ToolChoiceStringTypeRequired
			case "auto":
				req.ToolChoice = model.ToolChoiceStringTypeAuto
			case "function":
				if v.Function != nil {
					req.ToolChoice = &model.ToolChoice{
						Type:     model.ToolTypeFunction,
						Function: model.ToolChoiceFunction{Name: v.Function.Name},
					}
				} else {
					req.ToolChoice = model.ToolChoiceStringTypeAuto
				}
			default:
				req.ToolChoice = model.ToolChoiceStringTypeAuto
			}
		default:
			req.ToolChoice = model.ToolChoiceStringTypeAuto
		}
	}

	// Set optional parameters
	if opts.Temperature > 0 {
		req.Temperature = float32(opts.Temperature)
	}
	if opts.TopP > 0 {
		req.TopP = float32(opts.TopP)
	}
	if opts.MaxTokens > 0 {
		req.MaxTokens = int(opts.MaxTokens)
	}

	// Non-streaming request
	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, ErrEmptyResponse
	}

	// Convert response to framework format
	choices := make([]*llms.ContentChoice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		content := getContentString(choice.Message.Content)
		stopReason := string(choice.FinishReason)

		contentChoice := &llms.ContentChoice{
			Content:    content,
			StopReason: stopReason,
		}

		// Handle ToolCalls
		if len(choice.Message.ToolCalls) > 0 {
			toolCalls := make([]llms.ToolCall, 0, len(choice.Message.ToolCalls))
			for _, tc := range choice.Message.ToolCalls {
				toolCalls = append(toolCalls, llms.ToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					FunctionCall: &llms.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			contentChoice.ToolCalls = toolCalls
		}

		// Handle legacy FunctionCall (for backward compatibility)
		if choice.Message.FunctionCall != nil {
			contentChoice.FuncCall = &llms.FunctionCall{
				Name:      choice.Message.FunctionCall.Name,
				Arguments: choice.Message.FunctionCall.Arguments,
			}
		}

		choices = append(choices, contentChoice)
	}

	return &llms.ContentResponse{
		Choices: choices,
	}, nil
}

// CreateEmbedding generates embeddings for the given texts using Doubao embedding models.
//
// Supported embedding models:
//   - doubao-embedding: 基础向量化模型
//   - doubao-embedding-large: 大型向量化模型
//   - doubao-embedding-vision: 多模态向量化模型
//
// API documentation: https://www.volcengine.com/docs/82379/1521766
func (o *LLM) CreateEmbedding(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, errors.New("texts cannot be empty")
	}

	// Build embedding request
	req := model.EmbeddingRequestStrings{
		Model: string(o.embeddingModel),
		Input: texts,
	}

	// Create embeddings
	resp, err := o.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create embeddings: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, ErrEmptyResponse
	}

	// Convert response to [][]float32 format
	embeddings := make([][]float32, 0, len(resp.Data))
	for i := range resp.Data {
		// Sort by index to ensure correct order
		idx := resp.Data[i].Index
		if idx >= len(embeddings) {
			// Extend slice if needed
			newEmbeddings := make([][]float32, idx+1)
			copy(newEmbeddings, embeddings)
			embeddings = newEmbeddings
		}
		embeddings[idx] = resp.Data[i].Embedding
	}

	return embeddings, nil
}

// convertMessage converts an llms.MessageContent to an arkruntime ChatCompletionMessage.
func convertMessage(msg llms.MessageContent) (*model.ChatCompletionMessage, error) {
	// Get role as string
	role := string(msg.Role)

	if len(msg.Parts) == 0 {
		return nil, errors.New("message has no parts")
	}

	// Create ChatCompletionMessage
	arkMsg := &model.ChatCompletionMessage{
		Role: role,
	}

	// Process parts based on role
	for _, part := range msg.Parts {
		switch p := part.(type) {
		case llms.TextContent:
			// Text content
			if arkMsg.Content == nil {
				arkMsg.Content = createMessageContent(p.Text)
			}
		case llms.ToolCallResponse:
			// Tool response (from tool role)
			if role == "tool" {
				arkMsg.Content = createMessageContent(p.Content)
				// Also set ToolCallID if available
				if p.ToolCallID != "" {
					arkMsg.ToolCallID = p.ToolCallID
				}
			}
		}
	}

	// For tool messages, ensure content exists
	if role == "tool" && arkMsg.Content == nil {
		var content strings.Builder
		for _, part := range msg.Parts {
			if text, ok := part.(llms.TextContent); ok {
				content.WriteString(text.Text)
			} else if tr, ok := part.(llms.ToolCallResponse); ok {
				content.WriteString(tr.Content)
				if tr.ToolCallID != "" {
					arkMsg.ToolCallID = tr.ToolCallID
				}
			}
		}
		if content.String() != "" {
			arkMsg.Content = createMessageContent(content.String())
		}
	}

	// Ensure content is set for non-tool messages
	if arkMsg.Content == nil && role != "tool" {
		var content strings.Builder
		for _, part := range msg.Parts {
			if text, ok := part.(llms.TextContent); ok {
				content.WriteString(text.Text)
			}
		}
		contentStr := content.String()
		if contentStr == "" {
			return nil, errors.New("empty message content")
		}
		arkMsg.Content = createMessageContent(contentStr)
	}

	return arkMsg, nil
}

// createMessageContent creates a ChatCompletionMessageContent from a string.
func createMessageContent(s string) *model.ChatCompletionMessageContent {
	return &model.ChatCompletionMessageContent{
		StringValue: &s,
		ListValue:   nil,
	}
}

// getContentString extracts the string content from ChatCompletionMessageContent.
func getContentString(content *model.ChatCompletionMessageContent) string {
	if content == nil {
		return ""
	}
	if content.StringValue != nil {
		return *content.StringValue
	}
	if len(content.ListValue) > 0 {
		var parts []string
		for _, part := range content.ListValue {
			if part.Type == model.ChatCompletionMessageContentPartTypeText {
				parts = append(parts, part.Text)
			}
		}
		return strings.Join(parts, "")
	}
	return ""
}
