package retriever

import (
	"context"
	"fmt"

	"github.com/smallnest/langgraphgo/rag"
)

// VectorRetriever implements document retrieval using vector similarity
type VectorRetriever struct {
	vectorStore rag.VectorStore
	embedder    rag.Embedder
	config      rag.RetrievalConfig
}

// NewVectorRetriever creates a new vector retriever
func NewVectorRetriever(vectorStore rag.VectorStore, embedder rag.Embedder, config rag.RetrievalConfig) *VectorRetriever {
	if config.K == 0 {
		config.K = 4
	}
	if config.ScoreThreshold == 0 {
		config.ScoreThreshold = 0.5
	}
	if config.SearchType == "" {
		config.SearchType = "similarity"
	}

	return &VectorRetriever{
		vectorStore: vectorStore,
		embedder:    embedder,
		config:      config,
	}
}

// NewVectorRetrieverTopK creates a vector retriever configured only by top-K,
// using default score threshold and search type. It is the recommended
// replacement for NewVectorStoreRetriever.
func NewVectorRetrieverTopK(vectorStore rag.VectorStore, embedder rag.Embedder, topK int) *VectorRetriever {
	return NewVectorRetriever(vectorStore, embedder, rag.RetrievalConfig{K: topK})
}

// Retrieve retrieves documents based on a query
func (r *VectorRetriever) Retrieve(ctx context.Context, query string) ([]rag.Document, error) {
	return r.RetrieveWithK(ctx, query, r.config.K)
}

// RetrieveWithK retrieves exactly k documents
func (r *VectorRetriever) RetrieveWithK(ctx context.Context, query string, k int) ([]rag.Document, error) {
	config := r.config
	config.K = k
	results, err := r.RetrieveWithConfig(ctx, query, &config)
	if err != nil {
		return nil, err
	}

	docs := make([]rag.Document, len(results))
	for i, result := range results {
		docs[i] = result.Document
	}

	return docs, nil
}

// RetrieveWithConfig retrieves documents with custom configuration
func (r *VectorRetriever) RetrieveWithConfig(ctx context.Context, query string, config *rag.RetrievalConfig) ([]rag.DocumentSearchResult, error) {
	if config == nil {
		config = &r.config
	}

	// Embed the query
	queryEmbedding, err := r.embedder.EmbedDocument(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Perform vector search
	var results []rag.DocumentSearchResult

	if len(config.Filter) > 0 {
		// Search with filters
		results, err = r.vectorStore.SearchWithFilter(ctx, queryEmbedding, config.K, config.Filter)
	} else {
		// Simple search
		results, err = r.vectorStore.Search(ctx, queryEmbedding, config.K)
	}

	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Filter by score threshold
	if config.ScoreThreshold > 0 {
		filtered := make([]rag.DocumentSearchResult, 0)
		for _, result := range results {
			if result.Score >= config.ScoreThreshold {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	// Apply different search strategies
	switch config.SearchType {
	case "mmr":
		results = r.applyMMR(results, config.K)
	case "diversity":
		results = r.applyDiversitySearch(results, config.K)
	}

	return results, nil
}

// applyMMR applies Maximal Marginal Relevance to ensure diversity
func (r *VectorRetriever) applyMMR(results []rag.DocumentSearchResult, k int) []rag.DocumentSearchResult {
	if len(results) <= k {
		return results
	}

	selected := make([]rag.DocumentSearchResult, 0, k)
	selected = append(selected, results[0]) // Always select the highest scoring result

	candidates := results[1:]

	for len(selected) < k && len(candidates) > 0 {
		// Find the candidate with highest MMR score
		bestIdx := 0
		bestScore := 0.0

		for i, candidate := range candidates {
			// Calculate relevance score
			relevance := candidate.Score

			// Calculate maximal similarity to already selected documents
			maxSimilarity := 0.0
			for _, selectedDoc := range selected {
				similarity := r.calculateSimilarity(candidate.Document, selectedDoc.Document)
				if similarity > maxSimilarity {
					maxSimilarity = similarity
				}
			}

			// MMR score: λ * relevance - (1-λ) * maxSimilarity
			lambda := 0.5 // Balance between relevance and diversity
			mmrScore := lambda*relevance - (1-lambda)*maxSimilarity

			if mmrScore > bestScore {
				bestScore = mmrScore
				bestIdx = i
			}
		}

		// Add the best candidate to selected results
		selected = append(selected, candidates[bestIdx])
		// Remove from candidates
		candidates = append(candidates[:bestIdx], candidates[bestIdx+1:]...)
	}

	return selected
}

// applyDiversitySearch applies diversity-based selection
func (r *VectorRetriever) applyDiversitySearch(results []rag.DocumentSearchResult, k int) []rag.DocumentSearchResult {
	if len(results) <= k {
		return results
	}

	// Group results by content type or source to ensure diversity
	groups := make(map[string][]rag.DocumentSearchResult)

	for _, result := range results {
		// Try to group by source or type
		var groupKey string
		if result.Document.Metadata != nil {
			if source, ok := result.Document.Metadata["source"]; ok {
				groupKey = fmt.Sprintf("%v", source)
			} else if docType, ok := result.Document.Metadata["type"]; ok {
				groupKey = fmt.Sprintf("%v", docType)
			}
		}

		if groupKey == "" {
			groupKey = "default"
		}

		groups[groupKey] = append(groups[groupKey], result)
	}

	// Select top results from each group to ensure diversity
	selected := make([]rag.DocumentSearchResult, 0, k)
	for _, group := range groups {
		if len(group) > 0 {
			selected = append(selected, group[0]) // Take the best from each group
			if len(selected) >= k {
				break
			}
		}
	}

	// If we need more results to reach k, take the next best from any group
	if len(selected) < k {
		remaining := k - len(selected)
		remainingResults := make([]rag.DocumentSearchResult, 0)

		for _, group := range groups {
			if len(group) > 1 {
				remainingResults = append(remainingResults, group[1:]...)
			}
		}

		// Add the top remaining results
		for i := 0; i < remaining && i < len(remainingResults); i++ {
			selected = append(selected, remainingResults[i])
		}
	}

	return selected
}

// calculateSimilarity calculates similarity between two documents
func (r *VectorRetriever) calculateSimilarity(doc1, doc2 rag.Document) float64 {
	// Use embeddings if available
	if len(doc1.Embedding) > 0 && len(doc2.Embedding) > 0 {
		return cosineSimilarity(doc1.Embedding, doc2.Embedding)
	}

	// Fallback to content similarity
	return contentSimilarity(doc1.Content, doc2.Content)
}

// cosineSimilarity calculates cosine similarity between two embeddings
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return float64(dotProduct / (normA * normB))
}

// contentSimilarity calculates similarity between document contents
func contentSimilarity(a, b string) float64 {
	// Simple word overlap similarity
	wordsA := make(map[string]bool)
	wordsB := make(map[string]bool)

	// Extract words from both documents
	for _, word := range splitWords(a) {
		wordsA[word] = true
	}
	for _, word := range splitWords(b) {
		wordsB[word] = true
	}

	// Calculate Jaccard similarity
	intersection := 0
	for word := range wordsA {
		if wordsB[word] {
			intersection++
		}
	}

	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union)
}

// splitWords splits text into words
func splitWords(text string) []string {
	// Simple word splitting - in practice, you'd use more sophisticated tokenization
	words := make([]string, 0)
	current := ""

	for _, char := range text {
		if isAlphaNumeric(char) {
			current += string(char)
		} else {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		}
	}

	if current != "" {
		words = append(words, current)
	}

	return words
}

// isAlphaNumeric checks if a character is alphanumeric
func isAlphaNumeric(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}

// VectorStoreRetriever implements Retriever using a VectorStore with backward compatibility.
//
// Deprecated: VectorStoreRetriever is a top-K-only subset of VectorRetriever.
// Use NewVectorRetrieverTopK instead; this type will be removed in a future release.
type VectorStoreRetriever struct {
	vectorStore rag.VectorStore
	embedder    rag.Embedder
	topK        int
}

// NewVectorStoreRetriever creates a new VectorStoreRetriever.
//
// Deprecated: use NewVectorRetrieverTopK, which returns a fully-configured
// VectorRetriever with the same top-K behavior.
func NewVectorStoreRetriever(vectorStore rag.VectorStore, embedder rag.Embedder, topK int) *VectorStoreRetriever {
	if topK <= 0 {
		topK = 4
	}
	return &VectorStoreRetriever{
		vectorStore: vectorStore,
		embedder:    embedder,
		topK:        topK,
	}
}

// Retrieve retrieves relevant documents for a query
func (r *VectorStoreRetriever) Retrieve(ctx context.Context, query string) ([]rag.Document, error) {
	return r.RetrieveWithK(ctx, query, r.topK)
}

// RetrieveWithK retrieves exactly k documents
func (r *VectorStoreRetriever) RetrieveWithK(ctx context.Context, query string, k int) ([]rag.Document, error) {
	// Embed the query
	queryEmbedding, err := r.embedder.EmbedDocument(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search in vector store
	results, err := r.vectorStore.Search(ctx, queryEmbedding, k)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Extract documents from results
	docs := make([]rag.Document, len(results))
	for i, result := range results {
		docs[i] = result.Document
	}

	return docs, nil
}

// RetrieveWithConfig retrieves documents with custom configuration
func (r *VectorStoreRetriever) RetrieveWithConfig(ctx context.Context, query string, config *rag.RetrievalConfig) ([]rag.DocumentSearchResult, error) {
	if config == nil {
		config = &rag.RetrievalConfig{
			K:              r.topK,
			ScoreThreshold: 0.0,
			SearchType:     "similarity",
			IncludeScores:  false,
		}
	}

	// Embed the query
	queryEmbedding, err := r.embedder.EmbedDocument(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Perform search
	var results []rag.DocumentSearchResult

	if len(config.Filter) > 0 {
		results, err = r.vectorStore.SearchWithFilter(ctx, queryEmbedding, config.K, config.Filter)
	} else {
		results, err = r.vectorStore.Search(ctx, queryEmbedding, config.K)
	}

	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Filter by score threshold if specified
	if config.ScoreThreshold > 0 {
		filtered := make([]rag.DocumentSearchResult, 0)
		for _, result := range results {
			if result.Score >= config.ScoreThreshold {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	return results, nil
}
