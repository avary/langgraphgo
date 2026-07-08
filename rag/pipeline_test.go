package rag

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/stretchr/testify/assert"
)

type mockLLM struct{}

func (m *mockLLM) GenerateContent(ctx context.Context, messages []llmtypes.MessageContent, options ...llmtypes.CallOption) (*llmtypes.ContentResponse, error) {
	return &llmtypes.ContentResponse{
		Choices: []*llmtypes.ContentChoice{
			{Content: "Mock Answer"},
		},
	}, nil
}

func (m *mockLLM) Call(ctx context.Context, prompt string, options ...llmtypes.CallOption) (string, error) {
	return "Mock Answer", nil
}

type mockRetriever struct {
	docs []Document
}

func (m *mockRetriever) Retrieve(ctx context.Context, query string) ([]Document, error) {
	return m.docs, nil
}

func (m *mockRetriever) RetrieveWithK(ctx context.Context, query string, k int) ([]Document, error) {
	return m.docs, nil
}

func (m *mockRetriever) RetrieveWithConfig(ctx context.Context, query string, config *RetrievalConfig) ([]DocumentSearchResult, error) {
	res := make([]DocumentSearchResult, len(m.docs))
	for i, d := range m.docs {
		res[i] = DocumentSearchResult{Document: d, Score: 0.9}
	}
	return res, nil
}

type mockEmbedder struct{}

func (m *mockEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}
func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{{0.1, 0.2}}, nil
}
func (m *mockEmbedder) GetDimension() int { return 2 }

func TestRAGPipelineNodes(t *testing.T) {
	ctx := context.Background()
	llm := &mockLLM{}
	retriever := &mockRetriever{
		docs: []Document{
			{Content: "Context doc 1", Metadata: map[string]any{"source": "src1"}},
		},
	}

	config := DefaultPipelineConfig()
	config.LLM = llm
	config.Retriever = retriever

	p := NewRAGPipeline(config)

	t.Run("Retrieve Node", func(t *testing.T) {
		state := map[string]any{"query": "test", "documents": []Document{}}
		res, err := p.retrieveNode(ctx, state)
		assert.NoError(t, err)
		docs, _ := res["documents"].([]RAGDocument)
		assert.Len(t, docs, 1)
	})

	t.Run("Generate Node", func(t *testing.T) {
		state := map[string]any{
			"query":     "test",
			"documents": []RAGDocument{{Content: "context", Metadata: map[string]any{"source": "src1"}}},
		}
		res, err := p.generateNode(ctx, state)
		assert.NoError(t, err)
		answer, _ := res["answer"].(string)
		assert.Equal(t, "Mock Answer", answer)
	})

	t.Run("Format Citations Node", func(t *testing.T) {
		state := map[string]any{
			"documents": []RAGDocument{{Metadata: map[string]any{"source": "src1"}}},
		}
		res, err := p.formatCitationsNode(ctx, state)
		assert.NoError(t, err)
		citations, _ := res["citations"].([]string)
		assert.Len(t, citations, 1)
		assert.Contains(t, citations[0], "src1")
	})
}

func TestRAGPipelineBuilds(t *testing.T) {
	config := DefaultPipelineConfig()
	config.LLM = &mockLLM{}
	config.Retriever = &mockRetriever{}
	p := NewRAGPipeline(config)

	assert.NoError(t, p.BuildBasicRAG())
	assert.NoError(t, p.BuildAdvancedRAG())
	assert.NoError(t, p.BuildConditionalRAG())
}

func TestRerankNode(t *testing.T) {
	ctx := context.Background()
	p := NewRAGPipeline(nil)
	state := map[string]any{
		"retrieved_documents": []RAGDocument{{Content: "doc1"}},
	}
	res, err := p.rerankNode(ctx, state)
	assert.NoError(t, err)
	rankedDocs, _ := res["ranked_documents"].([]DocumentSearchResult)
	assert.Len(t, rankedDocs, 1)
}

func TestRAGStateSchema(t *testing.T) {
	s := &ragStateSchema{}
	init := s.Init()
	assert.NotNil(t, init["metadata"])

	update := map[string]any{
		"query":    "new query",
		"metadata": map[string]any{"key": "val"},
	}
	merged, err := s.Update(init, update)
	assert.NoError(t, err)
	query, _ := merged["query"].(string)
	metadata, _ := merged["metadata"].(map[string]any)
	assert.Equal(t, "new query", query)
	assert.Equal(t, "val", metadata["key"])
}

func TestBaseEngine(t *testing.T) {
	ctx := context.Background()
	retriever := &mockRetriever{docs: []Document{{Content: "context"}}}
	embedder := &mockEmbedder{}

	engine := NewBaseEngine(retriever, embedder, nil)
	assert.NotNil(t, engine)

	t.Run("Engine Query", func(t *testing.T) {
		res, err := engine.Query(ctx, "test")
		assert.NoError(t, err)
		assert.NotEmpty(t, res.Context)
	})

	t.Run("Engine Search", func(t *testing.T) {
		docs, err := engine.SimilaritySearch(ctx, "test", 1)
		assert.NoError(t, err)
		assert.Len(t, docs, 1)
	})
}

func TestCompositeEngine(t *testing.T) {
	ctx := context.Background()
	retriever := &mockRetriever{docs: []Document{{ID: "1", Content: "c1"}}}
	embedder := &mockEmbedder{}
	engine1 := NewBaseEngine(retriever, embedder, nil)

	comp := NewCompositeEngine([]Engine{engine1}, nil)

	t.Run("Composite Query", func(t *testing.T) {
		res, err := comp.Query(ctx, "test")
		assert.NoError(t, err)
		assert.Len(t, res.Sources, 1)
	})

	t.Run("Aggregators", func(t *testing.T) {
		res1 := &QueryResult{Confidence: 0.5, Sources: []Document{{ID: "1"}}, Metadata: make(map[string]any)}
		res2 := &QueryResult{Confidence: 0.8, Sources: []Document{{ID: "2"}}, Metadata: make(map[string]any)}

		agg := DefaultAggregator([]*QueryResult{res1, res2})
		assert.Equal(t, 0.8, agg.Confidence)
		assert.Len(t, agg.Sources, 2)

		wAgg := WeightedAggregator([]float64{1.0, 0.1})([]*QueryResult{res1, res2})
		assert.Equal(t, 0.5, wAgg.Confidence)
	})
}
