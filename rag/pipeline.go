package rag

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
)

// RAGState represents the state flowing through a RAG pipeline
type RAGState struct {
	Query              string
	Documents          []RAGDocument
	RetrievedDocuments []RAGDocument
	RankedDocuments    []DocumentSearchResult
	Context            string
	Answer             string
	Citations          []string
	Metadata           map[string]any
}

// PipelineConfig configures a RAG pipeline
type PipelineConfig struct {
	// Retrieval configuration
	TopK           int     // Number of documents to retrieve
	ScoreThreshold float64 // Minimum relevance score
	UseReranking   bool    // Whether to use reranking
	UseFallback    bool    // Whether to use fallback search

	// Generation configuration
	SystemPrompt     string
	IncludeCitations bool
	MaxTokens        int
	Temperature      float64

	// Components
	Loader      RAGDocumentLoader
	Splitter    RAGTextSplitter
	Embedder    Embedder
	VectorStore VectorStore
	Retriever   Retriever
	Reranker    Reranker
	LLM         llmtypes.Model
}

// DefaultPipelineConfig returns a default RAG configuration
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		TopK:             4,
		ScoreThreshold:   0.7,
		UseReranking:     false,
		UseFallback:      false,
		SystemPrompt:     "You are a helpful assistant. Answer the question based on the provided context. If you cannot answer based on the context, say so.",
		IncludeCitations: true,
		MaxTokens:        1000,
		Temperature:      0.0,
	}
}

// RAGPipeline represents a complete RAG pipeline
type RAGPipeline struct {
	config *PipelineConfig
	graph  *graph.StateGraph[map[string]any]
}

// NewRAGPipeline creates a new RAG pipeline with the given configuration
func NewRAGPipeline(config *PipelineConfig) *RAGPipeline {
	if config == nil {
		config = DefaultPipelineConfig()
	}

	g := graph.NewStateGraph[map[string]any]()
	g.SetSchema(&ragStateSchema{})

	return &RAGPipeline{
		config: config,
		graph:  g,
	}
}

type ragStateSchema struct{}

func (s *ragStateSchema) Init() map[string]any {
	return map[string]any{
		"query":               "",
		"context":             "",
		"answer":              "",
		"documents":           []Document{},
		"retrieved_documents": []Document{},
		"ranked_documents":    []Document{},
		"citations":           []string{},
		"metadata":            make(map[string]any),
	}
}

func (s *ragStateSchema) Update(current, new map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	maps.Copy(result, current)
	maps.Copy(result, new)
	return result, nil
}

// BuildBasicRAG builds a basic RAG pipeline: Retrieve -> Generate
func (p *RAGPipeline) BuildBasicRAG() error {
	if p.config.Retriever == nil {
		return fmt.Errorf("retriever is required for basic RAG")
	}
	if p.config.LLM == nil {
		return fmt.Errorf("LLM is required for basic RAG")
	}

	// Add retrieval node
	p.graph.AddNode("retrieve", "Document retrieval node", p.retrieveNode)

	// Add generation node
	p.graph.AddNode("generate", "Answer generation node", p.generateNode)

	// Build pipeline
	p.graph.SetEntryPoint("retrieve")
	p.graph.AddEdge("retrieve", "generate")
	p.graph.AddEdge("generate", graph.END)

	return nil
}

// BuildAdvancedRAG builds an advanced RAG pipeline: Retrieve -> Rerank -> Generate
func (p *RAGPipeline) BuildAdvancedRAG() error {
	if p.config.Retriever == nil {
		return fmt.Errorf("retriever is required for advanced RAG")
	}
	if p.config.LLM == nil {
		return fmt.Errorf("LLM is required for advanced RAG")
	}

	// Add retrieval node
	p.graph.AddNode("retrieve", "Document retrieval node", p.retrieveNode)

	// Add reranking node if enabled
	if p.config.UseReranking && p.config.Reranker != nil {
		p.graph.AddNode("rerank", "Document reranking node", p.rerankNode)
	}

	// Add generation node
	p.graph.AddNode("generate", "Answer generation node", p.generateNode)

	// Add citation formatting node if enabled
	if p.config.IncludeCitations {
		p.graph.AddNode("format_citations", "Citation formatting node", p.formatCitationsNode)
	}

	// Build pipeline
	p.graph.SetEntryPoint("retrieve")

	if p.config.UseReranking && p.config.Reranker != nil {
		p.graph.AddEdge("retrieve", "rerank")
		p.graph.AddEdge("rerank", "generate")
	} else {
		p.graph.AddEdge("retrieve", "generate")
	}

	if p.config.IncludeCitations {
		p.graph.AddEdge("generate", "format_citations")
		p.graph.AddEdge("format_citations", graph.END)
	} else {
		p.graph.AddEdge("generate", graph.END)
	}

	return nil
}

// BuildConditionalRAG builds a RAG pipeline with conditional routing based on relevance
func (p *RAGPipeline) BuildConditionalRAG() error {
	if p.config.Retriever == nil {
		return fmt.Errorf("retriever is required for conditional RAG")
	}
	if p.config.LLM == nil {
		return fmt.Errorf("LLM is required for conditional RAG")
	}

	// Add retrieval node
	p.graph.AddNode("retrieve", "Document retrieval node", p.retrieveNode)

	// Add reranking node
	p.graph.AddNode("rerank", "Document reranking node", p.rerankNode)

	// Add fallback search node if enabled
	if p.config.UseFallback {
		p.graph.AddNode("fallback_search", "Fallback search node", p.fallbackSearchNode)
	}

	// Add generation node
	p.graph.AddNode("generate", "Answer generation node", p.generateNode)

	// Add citation formatting node
	if p.config.IncludeCitations {
		p.graph.AddNode("format_citations", "Citation formatting node", p.formatCitationsNode)
	}

	// Build pipeline with conditional routing
	p.graph.SetEntryPoint("retrieve")
	p.graph.AddEdge("retrieve", "rerank")

	// Conditional edge based on relevance score
	p.graph.AddConditionalEdge("rerank", func(ctx context.Context, state map[string]any) string {
		rankedDocs, _ := state["ranked_documents"].([]DocumentSearchResult)
		if len(rankedDocs) > 0 && rankedDocs[0].Score >= p.config.ScoreThreshold {
			return "generate"
		}
		if p.config.UseFallback {
			return "fallback_search"
		}
		return "generate"
	})

	if p.config.UseFallback {
		p.graph.AddEdge("fallback_search", "generate")
	}

	if p.config.IncludeCitations {
		p.graph.AddEdge("generate", "format_citations")
		p.graph.AddEdge("format_citations", graph.END)
	} else {
		p.graph.AddEdge("generate", graph.END)
	}

	return nil
}

// Compile compiles the RAG pipeline into a runnable graph
func (p *RAGPipeline) Compile() (*graph.StateRunnable[map[string]any], error) {
	return p.graph.Compile()
}

// GetGraph returns the underlying graph for visualization
func (p *RAGPipeline) GetGraph() *graph.StateGraph[map[string]any] {
	return p.graph
}

// Node implementations

func (p *RAGPipeline) retrieveNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	query, _ := state["query"].(string)

	docs, err := p.config.Retriever.Retrieve(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	state["retrieved_documents"] = convertToRAGDocuments(docs)
	state["documents"] = convertToRAGDocuments(docs)

	return state, nil
}

func (p *RAGPipeline) rerankNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	query, _ := state["query"].(string)
	retrievedDocs, _ := state["retrieved_documents"].([]RAGDocument)

	if p.config.Reranker == nil {
		// If no reranker, just assign scores based on order
		rankedDocs := make([]DocumentSearchResult, len(retrievedDocs))
		for i, doc := range retrievedDocs {
			rankedDocs[i] = DocumentSearchResult{
				Document: doc.Document(),
				Score:    1.0 - float64(i)*0.1, // Simple decreasing score
			}
		}
		state["ranked_documents"] = rankedDocs
		return state, nil
	}

	// Convert to DocumentSearchResult for reranking
	searchResults := make([]DocumentSearchResult, len(retrievedDocs))
	for i, doc := range retrievedDocs {
		searchResults[i] = DocumentSearchResult{
			Document: doc.Document(),
			Score:    1.0 - float64(i)*0.1,
		}
	}

	rerankedResults, err := p.config.Reranker.Rerank(ctx, query, searchResults)
	if err != nil {
		return nil, fmt.Errorf("reranking failed: %w", err)
	}

	state["ranked_documents"] = rerankedResults

	// Update documents with reranked order
	docs := make([]RAGDocument, len(rerankedResults))
	for i, rd := range rerankedResults {
		docs[i] = DocumentFromRAGDocument(rd.Document)
	}
	state["documents"] = docs

	return state, nil
}

func (p *RAGPipeline) fallbackSearchNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	// Placeholder for fallback search (e.g., web search)
	// In a real implementation, this would call an external search API
	state["metadata"] = map[string]any{
		"fallback_used": true,
	}

	return state, nil
}

func (p *RAGPipeline) generateNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	query, _ := state["query"].(string)
	documents, _ := state["documents"].([]RAGDocument)

	// Build context from retrieved documents
	var contextParts []string
	for i, doc := range documents {
		source := "Unknown"
		if s, ok := doc.Metadata["source"]; ok {
			source = fmt.Sprintf("%v", s)
		}
		contextParts = append(contextParts, fmt.Sprintf("[%d] Source: %s\nContent: %s", i+1, source, doc.Content))
	}
	contextStr := strings.Join(contextParts, "\n\n")

	// Build prompt
	prompt := fmt.Sprintf("Context:\n%s\n\nQuestion: %s\n\nAnswer:", contextStr, query)

	messages := []llmtypes.MessageContent{
		llmtypes.TextParts("system", p.config.SystemPrompt),
		llmtypes.TextParts("human", prompt),
	}

	// Generate answer
	response, err := p.config.LLM.GenerateContent(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	if len(response.Choices) > 0 {
		state["answer"] = response.Choices[0].Content
	}
	state["context"] = contextStr

	return state, nil
}

func (p *RAGPipeline) formatCitationsNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	documents, _ := state["documents"].([]RAGDocument)

	// Extract citations from documents
	citations := make([]string, len(documents))
	for i, doc := range documents {
		source := "Unknown"
		if s, ok := doc.Metadata["source"]; ok {
			source = fmt.Sprintf("%v", s)
		}
		citations[i] = fmt.Sprintf("[%d] %s", i+1, source)
	}
	state["citations"] = citations

	return state, nil
}

// RAGDocument represents a document with content and metadata (for pipeline compatibility)
type RAGDocument struct {
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// ConvertToDocument converts RAGDocument to Document
func (d RAGDocument) Document() Document {
	return Document{
		Content:   d.Content,
		Metadata:  d.Metadata,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// DocumentFromRAGDocument converts Document to RAGDocument
func DocumentFromRAGDocument(doc Document) RAGDocument {
	return RAGDocument{
		Content:   doc.Content,
		Metadata:  doc.Metadata,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
}

// RAGDocumentLoader represents a document loader for RAG pipelines
type RAGDocumentLoader interface {
	Load(ctx context.Context) ([]RAGDocument, error)
}

// RAGTextSplitter represents a text splitter for RAG pipelines
type RAGTextSplitter interface {
	SplitDocuments(documents []RAGDocument) ([]RAGDocument, error)
}

// convertToRAGDocuments converts Document to RAGDocument
func convertToRAGDocuments(docs []Document) []RAGDocument {
	result := make([]RAGDocument, len(docs))
	for i, doc := range docs {
		result[i] = DocumentFromRAGDocument(doc)
	}
	return result
}
