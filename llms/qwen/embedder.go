package qwen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// NewEmbedder creates a new Qwen embedder that supports encoding_format parameter.
// This is required for Qwen3-Embedding-4B on ModelScope API.
//
// The encoding_format parameter must be "float" or "base64" (not "float32").
func NewEmbedder(baseURL, apiKey, model string) *Embedder {
	return &Embedder{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
	}
}

// Embedder is a custom embedder that supports encoding_format for Qwen models.
type Embedder struct {
	baseURL string
	apiKey  string
	model   string
}

// EmbedQuery embeds a single query text.
func (e *Embedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	emb, err := e.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return emb[0], nil
}

// EmbedDocument embeds a single document text.
func (e *Embedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	emb, err := e.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return emb[0], nil
}

// EmbedDocuments embeds multiple documents with retry logic for rate limiting.
func (e *Embedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Create request payload with encoding_format for Qwen
	payload := map[string]any{
		"model":           e.model,
		"input":           texts,
		"encoding_format": "float", // Required for Qwen3-Embedding-4B (valid values: "float" or "base64")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// Build URL
	baseURL := strings.TrimSuffix(e.baseURL, "/")
	url := baseURL + "/embeddings"

	// Retry logic for rate limiting
	maxRetries := 5
	retryDelay := 2 * time.Second

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d after %v delay", attempt, maxRetries, retryDelay)
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			// Exponential backoff
			retryDelay *= 2
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadBytes)))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+e.apiKey)

		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("send request: %w", err)
			continue
		}

		// Read response body for error details
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			// Parse response
			var result embeddingResponse
			if err := json.Unmarshal(bodyBytes, &result); err != nil {
				return nil, fmt.Errorf("decode response: %w", err)
			}

			// Extract embeddings
			emb := make([][]float32, len(result.Data))
			for i, item := range result.Data {
				emb[i] = item.Embedding
			}
			return emb, nil
		}

		// Handle rate limiting (429) and server errors (5xx)
		if resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode < 600) {
			// Try to parse error response for logging
			var errResp map[string]any
			if json.Unmarshal(bodyBytes, &errResp) == nil {
				log.Printf("API returned status %d (will retry): %v", resp.StatusCode, errResp)
			}
			lastErr = fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
			continue
		}

		// For other errors, don't retry
		var errResp map[string]any
		if json.Unmarshal(bodyBytes, &errResp) == nil {
			return nil, fmt.Errorf("API returned status %d: %v", resp.StatusCode, errResp)
		}
		return nil, fmt.Errorf("API returned unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// GetDimension returns the dimension of the embeddings.
func (e *Embedder) GetDimension() int {
	// Qwen3-Embedding-4B outputs 2560 dimensions
	// You can also make an API call to detect this dynamically
	return 2560
}

// Dimension returns the dimension of the embeddings.
func (e *Embedder) Dimension() int {
	return e.GetDimension()
}

type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}
