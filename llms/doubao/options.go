package doubao

import (
	"net/http"
	"os"
)

// ModelName represents the model identifier for Doubao (Volcengine Ark) API.
//
// IMPORTANT: You should use your custom Endpoint ID (推理接入点ID) as the model name.
// To create an endpoint and get your Endpoint ID, visit:
// https://www.volcengine.com/docs/82379/1330310
//
// Example usage:
//
//	llm, err := doubao.New(
//		doubao.WithAPIKey("your-api-key"),
//		doubao.WithModel("your-endpoint-id"), // Use your Endpoint ID directly
//	)
type ModelName string

type options struct {
	apiKey         string
	accessKey      string
	secretKey      string
	model          ModelName
	embeddingModel ModelName
	httpClient     *http.Client
	baseURL        string
	region         string
}

// Option is a function that configures an LLM.
type Option func(*options)

// WithAPIKey sets the API key for the LLM (recommended method).
func WithAPIKey(apiKey string) Option {
	return func(opts *options) {
		opts.apiKey = apiKey
	}
}

// WithAccessKey sets the Access Key for AK/SK authentication.
func WithAccessKey(accessKey string) Option {
	return func(opts *options) {
		opts.accessKey = accessKey
	}
}

// WithSecretKey sets the Secret Key for AK/SK authentication.
func WithSecretKey(secretKey string) Option {
	return func(opts *options) {
		opts.secretKey = secretKey
	}
}

// WithModel sets the model name for the LLM.
// You should use your custom Endpoint ID as the model name.
// To create an endpoint and get your Endpoint ID, visit: https://www.volcengine.com/docs/82379/1330310
func WithModel(model ModelName) Option {
	return func(opts *options) {
		opts.model = model
	}
}

// WithEmbeddingModel sets the embedding model name.
// You should use your custom Endpoint ID as the model name.
func WithEmbeddingModel(model ModelName) Option {
	return func(opts *options) {
		opts.embeddingModel = model
	}
}

// WithHTTPClient sets the HTTP client for the LLM.
func WithHTTPClient(client *http.Client) Option {
	return func(opts *options) {
		opts.httpClient = client
	}
}

// WithBaseURL sets the base URL for the LLM API.
// Default is "https://ark.cn-beijing.volces.com/api/v3".
func WithBaseURL(baseURL string) Option {
	return func(opts *options) {
		opts.baseURL = baseURL
	}
}

// WithRegion sets the region for the LLM API.
// Default is "cn-beijing".
func WithRegion(region string) Option {
	return func(opts *options) {
		opts.region = region
	}
}

// getEnvOrDefault retrieves an environment variable or returns the default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
