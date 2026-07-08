package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/retriever"
	"github.com/smallnest/langgraphgo/rag/store"
)

func main() {
	ctx := context.Background()

	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	documents := []rag.Document{
		{
			Content:  "The company policy allows remote work for 3 days a week.",
			Metadata: map[string]any{"source": "policy"},
		},
		{
			Content:  "Employees must be in the office on Mondays and Fridays.",
			Metadata: map[string]any{"source": "policy"},
		},
	}

	embedder := store.NewMockEmbedder(128)
	vectorStore := store.NewInMemoryVectorStore(embedder)

	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.Content
	}
	embeddings, _ := embedder.EmbedDocuments(ctx, texts)
	vectorStore.AddBatch(ctx, documents, embeddings)

	retriever := retriever.NewVectorStoreRetriever(vectorStore, embedder, 2)

	config := rag.DefaultPipelineConfig()
	config.Retriever = retriever
	config.LLM = llm
	config.ScoreThreshold = 0.5
	config.UseFallback = true

	pipeline := rag.NewRAGPipeline(config)
	err = pipeline.BuildConditionalRAG()
	if err != nil {
		log.Fatalf("Failed to build RAG pipeline: %v", err)
	}

	runnable, err := pipeline.Compile()
	if err != nil {
		log.Fatalf("Failed to compile pipeline: %v", err)
	}

	exporter := graph.GetGraphForRunnable(runnable)
	fmt.Println(exporter.DrawASCII())

	query := "Can I work from home on Tuesday?"
	fmt.Printf("\nQuery: %s\n", query)

	result, err := runnable.Invoke(ctx, map[string]any{
		"query": query,
	})
	if err != nil {
		log.Fatalf("Failed to process query: %v", err)
	}

	if answer, ok := result["answer"].(string); ok {
		fmt.Printf("Answer: %s\n", answer)
	}
}
