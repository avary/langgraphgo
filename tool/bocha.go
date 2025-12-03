package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// BochaSearch is a tool that uses the Bocha API to search the web.
type BochaSearch struct {
	APIKey    string
	BaseURL   string
	Count     int
	Freshness string
	Summary   bool
}

type BochaOption func(*BochaSearch)

// WithBochaBaseURL sets the base URL for the Bocha API.
func WithBochaBaseURL(url string) BochaOption {
	return func(b *BochaSearch) {
		b.BaseURL = url
	}
}

// WithBochaCount sets the number of results to return.
func WithBochaCount(count int) BochaOption {
	return func(b *BochaSearch) {
		b.Count = count
	}
}

// WithBochaFreshness sets the freshness filter for the search.
// Valid values: "oneDay", "oneWeek", "oneMonth", "oneYear", "noLimit".
func WithBochaFreshness(freshness string) BochaOption {
	return func(b *BochaSearch) {
		b.Freshness = freshness
	}
}

// WithBochaSummary sets whether to return a summary.
func WithBochaSummary(summary bool) BochaOption {
	return func(b *BochaSearch) {
		b.Summary = summary
	}
}

// NewBochaSearch creates a new BochaSearch tool.
// If apiKey is empty, it tries to read from BOCHA_API_KEY environment variable.
func NewBochaSearch(apiKey string, opts ...BochaOption) (*BochaSearch, error) {
	if apiKey == "" {
		apiKey = os.Getenv("BOCHA_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("BOCHA_API_KEY not set")
	}

	b := &BochaSearch{
		APIKey:    apiKey,
		BaseURL:   "https://api.bochaai.com/v1/web-search",
		Count:     10,
		Freshness: "noLimit",
		Summary:   true,
	}

	for _, opt := range opts {
		opt(b)
	}

	return b, nil
}

// Name returns the name of the tool.
func (b *BochaSearch) Name() string {
	return "Bocha_Search"
}

// Description returns the description of the tool.
func (b *BochaSearch) Description() string {
	return "A search engine powered by Bocha AI. " +
		"Useful for finding real-time information and answering questions. " +
		"Input should be a search query."
}

// Call executes the search.
func (b *BochaSearch) Call(ctx context.Context, input string) (string, error) {
	reqBody := map[string]interface{}{
		"query":     input,
		"count":     b.Count,
		"freshness": b.Freshness,
		"summary":   b.Summary,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", b.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bocha api returned status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Format the output
	// Assuming the response structure based on common patterns and search results:
	// { "data": { "webPages": { "value": [ ... ] } } } or similar.
	// Since I don't have the exact structure, I'll try to handle a generic "results" or "webPages" field if possible,
	// or dump the JSON if structure is unknown.
	// However, based on the search result, it returns structured results.
	// Let's assume a structure similar to:
	// {
	//   "data": {
	//     "webPages": {
	//       "value": [
	//         { "name": "Title", "url": "URL", "snippet": "Summary" }
	//       ]
	//     }
	//   }
	// }
	// Or maybe just a list at the top level?
	// Given the uncertainty, I will try to parse a few common fields.

	var sb strings.Builder

	// Helper function to extract and format items
	formatItems := func(items []interface{}) {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				title, _ := m["name"].(string)
				if title == "" {
					title, _ = m["title"].(string)
				}
				url, _ := m["url"].(string)
				snippet, _ := m["snippet"].(string)
				if snippet == "" {
					snippet, _ = m["summary"].(string)
				}
				if snippet == "" {
					snippet, _ = m["content"].(string)
				}
				sb.WriteString(fmt.Sprintf("Title: %s\nURL: %s\nContent: %s\n\n", title, url, snippet))
			}
		}
	}

	// Try to find the list of results
	if data, ok := result["data"].(map[string]interface{}); ok {
		if webPages, ok := data["webPages"].(map[string]interface{}); ok {
			if value, ok := webPages["value"].([]interface{}); ok {
				formatItems(value)
				return sb.String(), nil
			}
		}
	}

	// Fallback: check if "results" exists at top level (like Tavily)
	if results, ok := result["results"].([]interface{}); ok {
		formatItems(results)
		return sb.String(), nil
	}
	
	// Fallback: check if "webPages" exists at top level
	if webPages, ok := result["webPages"].(map[string]interface{}); ok {
         if value, ok := webPages["value"].([]interface{}); ok {
             formatItems(value)
             return sb.String(), nil
         }
    }

	// If we can't parse it nicely, return the raw JSON (indented)
	formattedJSON, _ := json.MarshalIndent(result, "", "  ")
	return string(formattedJSON), nil
}
