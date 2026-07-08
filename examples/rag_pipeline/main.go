package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smallnest/langgraphgo/graph"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/retriever"
	"github.com/smallnest/langgraphgo/rag/store"
)

func main() {
	// Check for OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		fmt.Println("OPENAI_API_KEY not set. Skipping execution.")
		return
	}

	ctx := context.Background()

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Create sample documents
	documents := []rag.Document{
		{
			Content:  "LangGraph is a library for building stateful, multi-actor applications with LLMs.",
			Metadata: map[string]any{"source": "docs"},
		},
		{
			Content:  "RAG (Retrieval-Augmented Generation) combines retrieval with generation.",
			Metadata: map[string]any{"source": "docs"},
		},
	}

	// Create embedder and vector store
	embedder := store.NewMockEmbedder(128)
	vectorStore := store.NewInMemoryVectorStore(embedder)

	// Add documents
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
	fmt.Println("Pipeline Structure:")
	fmt.Println(exporter.DrawASCII())

	// Run query
	query := "What is LangGraph?"
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
		fmt.Printf("Retrieved %d documents\n", len(docs))
	}
}
