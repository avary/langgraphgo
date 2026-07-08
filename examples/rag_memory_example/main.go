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

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Create sample documents
	documents := []rag.Document{
		{
			Content: "Chroma is an open-source vector database that allows you to store and query embeddings. " +
				"It is designed to be easy to use and integrate with LLM applications.",
			Metadata: map[string]any{"source": "chroma_docs"},
		},
		{
			Content: "LangGraphGo integrates with various vector stores including Chroma, Pinecone, and Weaviate " +
				"to enable RAG capabilities in your Go applications.",
			Metadata: map[string]any{"source": "langgraphgo_docs"},
		},
	}

	// Create embedder
	embedder := store.NewMockEmbedder(128)

	// Create Chroma vector store (using mock for example, as Chroma client might require running instance)
	vectorStore := store.NewInMemoryVectorStore(embedder)

	// Generate embeddings and add documents
	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.Content
	}
	embeddings, _ := embedder.EmbedDocuments(ctx, texts)
	vectorStore.AddBatch(ctx, documents, embeddings)

	// Create retriever
	retriever := retriever.NewVectorStoreRetriever(vectorStore, embedder, 2)

	// Configure RAG pipeline
	config := rag.DefaultPipelineConfig()
	config.Retriever = retriever
	config.LLM = llm

	// Build basic RAG pipeline

	pipeline := rag.NewRAGPipeline(config)
	err = pipeline.BuildBasicRAG()
	if err != nil {
		log.Fatalf("Failed to build RAG pipeline: %v", err)
	}

	// Compile the pipeline
	runnable, err := pipeline.Compile()
	if err != nil {
		log.Fatalf("Failed to compile pipeline: %v", err)
	}

	// Visualize
	exporter := graph.GetGraphForRunnable(runnable)
	fmt.Println(exporter.DrawASCII())

	// Run query
	query := "What is Chroma?"
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

	if docs, ok := result["documents"].([]rag.RAGDocument); ok {
		fmt.Println("Retrieved Documents:")
		for j, doc := range docs {
			fmt.Printf("  [%d] %s\n", j+1, truncate(doc.Content, 100))
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
