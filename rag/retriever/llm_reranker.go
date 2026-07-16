package retriever

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/smallnest/langgraphgo/llms"
	"github.com/smallnest/langgraphgo/rag"
)

// LLMRerankerConfig configures the LLM-based reranker
type LLMRerankerConfig struct {
	// TopK is the number of documents to return
	TopK int
	// ScoreThreshold is the minimum relevance score (0-1)
	ScoreThreshold float64
	// SystemPrompt is a custom system prompt for scoring
	SystemPrompt string
	// BatchSize is the number of documents to score in a single request (for efficiency)
	BatchSize int
}

// DefaultLLMRerankerConfig returns the default configuration for LLM reranker
func DefaultLLMRerankerConfig() LLMRerankerConfig {
	return LLMRerankerConfig{
		TopK:           5,
		ScoreThreshold: 0.0,
		SystemPrompt: "You are a relevance scoring assistant. Rate how well each document answers " +
			"the query on a scale of 0.0 to 1.0, where 1.0 is perfectly relevant and 0.0 is not relevant. " +
			"Consider semantic meaning, factual accuracy, and completeness.",
		BatchSize: 5,
	}
}

// LLMReranker uses an LLM to score query-document pairs for reranking
type LLMReranker struct {
	llm    llms.Model
	config LLMRerankerConfig
}

// NewLLMReranker creates a new LLM-based reranker
func NewLLMReranker(llm llms.Model, config LLMRerankerConfig) *LLMReranker {
	if config.TopK <= 0 {
		config.TopK = 5
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 5
	}
	return &LLMReranker{
		llm:    llm,
		config: config,
	}
}

// Rerank reranks documents based on query relevance using LLM scoring
func (r *LLMReranker) Rerank(ctx context.Context, query string, documents []rag.DocumentSearchResult) ([]rag.DocumentSearchResult, error) {
	if len(documents) == 0 {
		return []rag.DocumentSearchResult{}, nil
	}

	// Score all documents
	scores := make([]float64, len(documents))

	// Score documents in batches for efficiency
	for i := 0; i < len(documents); i += r.config.BatchSize {
		end := min(i+r.config.BatchSize, len(documents))
		batch := documents[i:end]

		batchScores, err := r.scoreBatch(ctx, query, batch)
		if err != nil {
			// If batch scoring fails, use original scores
			for j := i; j < end; j++ {
				scores[j] = documents[j].Score
			}
			continue
		}

		copy(scores[i:end], batchScores)
	}

	// Combine original scores with LLM scores (weighted average)
	type docScore struct {
		doc   rag.DocumentSearchResult
		score float64
	}

	combinedScores := make([]docScore, len(documents))
	for i, doc := range documents {
		// Weight LLM score higher than original retrieval score
		llmWeight := 0.7
		originalWeight := 0.3
		finalScore := llmWeight*scores[i] + originalWeight*doc.Score

		combinedScores[i] = docScore{
			doc: rag.DocumentSearchResult{
				Document: doc.Document,
				Score:    finalScore,
				Metadata: r.mergeMetadata(doc.Metadata, map[string]any{
					"llm_rerank_score": scores[i],
					"original_score":   doc.Score,
					"reranking_method": "llm",
				}),
			},
			score: finalScore,
		}
	}

	// Sort by score (descending)
	for i := range combinedScores {
		for j := i + 1; j < len(combinedScores); j++ {
			if combinedScores[j].score > combinedScores[i].score {
				combinedScores[i], combinedScores[j] = combinedScores[j], combinedScores[i]
			}
		}
	}

	// Filter by score threshold
	var filtered []docScore
	if r.config.ScoreThreshold > 0 {
		for _, ds := range combinedScores {
			if ds.score >= r.config.ScoreThreshold {
				filtered = append(filtered, ds)
			}
		}
	} else {
		filtered = combinedScores
	}

	// Limit to TopK
	if len(filtered) > r.config.TopK {
		filtered = filtered[:r.config.TopK]
	}

	// Extract results
	results := make([]rag.DocumentSearchResult, len(filtered))
	for i, ds := range filtered {
		results[i] = ds.doc
	}

	return results, nil
}

// scoreBatch scores a batch of documents using a single LLM call
func (r *LLMReranker) scoreBatch(ctx context.Context, query string, documents []rag.DocumentSearchResult) ([]float64, error) {
	// Build prompt with all documents
	var promptParts []string
	promptParts = append(promptParts, fmt.Sprintf("Query: %s\n\n", query))
	promptParts = append(promptParts, "Rate the relevance of each document to the query. Return scores in JSON format.\n\n")
	promptParts = append(promptParts, "Documents:\n")

	for i, doc := range documents {
		// Truncate content to avoid token limits
		content := doc.Document.Content
		maxContentLen := 500
		if len(content) > maxContentLen {
			content = content[:maxContentLen] + "..."
		}
		promptParts = append(promptParts, fmt.Sprintf("[%d] %s\n", i+1, content))
	}

	promptParts = append(promptParts, "\nReturn scores in format: [score1, score2, ...] where each score is between 0.0 and 1.0")

	prompt := strings.Join(promptParts, "")

	messages := []llms.MessageContent{
		llms.TextParts("system", r.config.SystemPrompt),
		llms.TextParts("human", prompt),
	}

	// Generate response
	response, err := r.llm.GenerateContent(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse scores from response
	scores, err := r.parseScores(response.Choices[0].Content, len(documents))
	if err != nil {
		return nil, fmt.Errorf("failed to parse scores: %w", err)
	}

	return scores, nil
}

// parseScores parses LLM response to extract scores
func (r *LLMReranker) parseScores(response string, expectedCount int) ([]float64, error) {
	// Try to parse JSON array
	response = strings.TrimSpace(response)

	// Look for JSON array pattern
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")

	if startIdx == -1 || endIdx == -1 {
		// Fallback: extract numbers from text
		return r.extractNumbers(response, expectedCount)
	}

	arrayStr := response[startIdx+1 : endIdx]
	parts := strings.Split(arrayStr, ",")

	scores := make([]float64, 0, expectedCount)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var score float64
		_, err := fmt.Sscanf(part, "%f", &score)
		if err == nil {
			scores = append(scores, score)
		}
	}

	// If we didn't get the expected count, try alternative parsing
	if len(scores) != expectedCount {
		return r.extractNumbers(response, expectedCount)
	}

	return scores, nil
}

// extractNumbers extracts numbers from text as fallback
func (r *LLMReranker) extractNumbers(text string, expectedCount int) ([]float64, error) {
	// Simple number extraction
	scores := make([]float64, 0, expectedCount)
	var num float64
	for s := range strings.FieldsSeq(text) {
		_, err := fmt.Sscanf(s, "%f", &num)
		if err == nil && num >= 0 && num <= 1 {
			scores = append(scores, num)
			if len(scores) == expectedCount {
				break
			}
		}
	}

	// If we still don't have enough scores, return default scores
	if len(scores) < expectedCount {
		defaultScores := make([]float64, expectedCount)
		for i := range defaultScores {
			defaultScores[i] = 0.5 // Default middle score
		}
		// Copy whatever scores we got
		copy(defaultScores, scores)
		return defaultScores, nil
	}

	return scores, nil
}

// mergeMetadata merges two metadata maps
func (r *LLMReranker) mergeMetadata(m1, m2 map[string]any) map[string]any {
	result := make(map[string]any)
	maps.Copy(result, m1)
	maps.Copy(result, m2)
	return result
}
