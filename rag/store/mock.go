package store

import (
	"context"
	"math"
)

// MockEmbedder is a simple mock embedder for testing
type MockEmbedder struct {
	Dimension int
}

// NewMockEmbedder creates a new MockEmbedder
func NewMockEmbedder(dimension int) *MockEmbedder {
	return &MockEmbedder{
		Dimension: dimension,
	}
}

// EmbedDocument generates mock embedding for a document
func (e *MockEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return e.generateEmbedding(text), nil
}

// EmbedDocuments generates mock embeddings for documents
func (e *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = e.generateEmbedding(text)
	}
	return embeddings, nil
}

// GetDimension returns the embedding dimension
func (e *MockEmbedder) GetDimension() int {
	return e.Dimension
}

func (e *MockEmbedder) generateEmbedding(text string) []float32 {
	// Simple deterministic embedding based on text content
	embedding := make([]float32, e.Dimension)

	for i := 0; i < e.Dimension; i++ {
		var sum float64
		for j, char := range text {
			sum += float64(char) * float64(i+j+1)
		}
		embedding[i] = float32(math.Sin(sum / 1000.0))
	}

	// Normalize
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}
