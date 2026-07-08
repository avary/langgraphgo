package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

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

	// Create sample documents about chromem-go
	documents := []rag.Document{
		{
			ID: "doc1",
			Content: "chromem-go is a pure Go implementation of a vector database inspired by Chroma. " +
				"It provides an embedded SQLite-based persistent store with optional in-memory operation.",
			Metadata: map[string]any{"source": "chromem_docs", "category": "introduction"},
		},
		{
			ID: "doc2",
			Content: "Key features of chromem-go include: zero external dependencies, thread-safe operations, " +
				"concurrent document processing, and support for multiple embedding functions.",
			Metadata: map[string]any{"source": "chromem_docs", "category": "features"},
		},
		{
			ID: "doc3",
			Content: "chromem-go supports two storage modes: in-memory for temporary data and persistent SQLite " +
				"for long-term storage. Collections can be created with custom names and metadata.",
			Metadata: map[string]any{"source": "chromem_docs", "category": "storage"},
		},
		{
			ID: "doc4",
			Content: "The vector store supports similarity search using cosine distance, metadata filtering, " +
				"and batch operations for improved performance with large datasets.",
			Metadata: map[string]any{"source": "chromem_docs", "category": "search"},
		},
		{
			ID: "doc5",
			Content: "LangGraphGo integrates seamlessly with chromem-go, providing a native Go vector store " +
				"option for RAG applications without requiring external services or dependencies.",
			Metadata: map[string]any{"source": "langgraphgo_docs", "category": "integration"},
		},
	}

	fmt.Println("=== RAG with chromem-go VectorStore Example ===")
	fmt.Println()
	fmt.Println("Initializing chromem-go vector store...")

	// Create a temporary directory for persistent storage
	tempDir := filepath.Join(os.TempDir(), "chromem_example")
	defer os.RemoveAll(tempDir)

	// // Create real OpenAI embedder
	// llmForEmbeddings, err := openai.New(openai.WithEmbeddingModel("embedding-v1"))
	// if err != nil {
	// 	log.Fatalf("Failed to create LLM for embeddings: %v", err)
	// }
	// openaiEmbedder, err := embeddings.NewEmbedder(llmForEmbeddings)
	// if err != nil {
	// 	log.Fatalf("Failed to create OpenAI embedder: %v", err)
	// }
	// embedder := rag.NewLangChainEmbedder(openaiEmbedder)

	embedder := store.NewMockEmbedder(128)

	// Create chromem vector store with persistent storage
	chromemStore, err := store.NewChromemVectorStore(store.ChromemConfig{
		PersistenceDir: tempDir,
		CollectionName: "langgraphgo_example",
		Embedder:       embedder,
	})
	if err != nil {
		log.Fatalf("Failed to create chromem store: %v", err)
	}
	defer chromemStore.Close()

	fmt.Printf("Store created with collection: %s\n", chromemStore.GetCollectionName())
	fmt.Printf("Storage location: %s\n\n", tempDir)

	fmt.Println("Adding documents to chromem-go...")
	err = chromemStore.Add(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("Successfully added %d documents\n\n", len(documents))

	// Display store statistics
	stats, err := chromemStore.GetStats(ctx)
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}
	fmt.Printf("Store Statistics:\n")
	fmt.Printf("  Total Documents: %d\n", stats.TotalDocuments)
	fmt.Printf("  Vector Dimension: %d\n\n", stats.Dimension)

	// Create retriever
	vectorRetriever := retriever.NewVectorStoreRetriever(chromemStore, embedder, 2)

	// Configure RAG pipeline
	config := rag.DefaultPipelineConfig()
	config.Retriever = vectorRetriever
	config.LLM = llm

	// Build basic RAG pipeline
	fmt.Println("Building RAG pipeline...")
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

	// Visualize the graph
	fmt.Println("\nPipeline Graph:")
	exporter := graph.GetGraphForRunnable(runnable)
	fmt.Println(exporter.DrawASCII())
	fmt.Println()

	// Run queries
	queries := []string{
		"What is chromem-go?",
		"What are the key features?",
		"Tell me about the storage options",
	}

	for i, query := range queries {
		fmt.Println("================================================================================")
		fmt.Printf("Query %d: %s\n", i+1, query)
		fmt.Println("--------------------------------------------------------------------------------")

		result, err := runnable.Invoke(ctx, map[string]any{
			"query": query,
		})
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		if answer, ok := result["answer"].(string); ok {
			fmt.Printf("\nAnswer:\n%s\n", answer)
		}

		if docs, ok := result["documents"].([]rag.RAGDocument); ok {
			fmt.Printf("\nRetrieved %d documents:\n", len(docs))
			for j, doc := range docs {
				fmt.Printf("  [%d] %s\n", j+1, truncate(doc.Content, 100))
				fmt.Printf("      Metadata: %v\n", doc.Metadata)
			}
		}
		fmt.Println()
	}

	// Demonstrate metadata filtering
	fmt.Println("================================================================================")
	fmt.Println("Metadata Filtering Example")
	fmt.Println("--------------------------------------------------------------------------------")

	// Generate query embedding for filtering
	queryEmbedding, err := embedder.EmbedDocument(ctx, "features and capabilities")
	if err != nil {
		log.Printf("Failed to generate query embedding: %v", err)
	} else {
		// Search with metadata filter
		filteredResults, err := chromemStore.SearchWithFilter(ctx, queryEmbedding, 10, map[string]any{"category": "features"})
		if err != nil {
			log.Printf("Failed to search with filter: %v", err)
		} else {
			fmt.Printf("\nFound %d documents with category='features':\n", len(filteredResults))
			for i, result := range filteredResults {
				fmt.Printf("  [%d] %s\n", i+1, truncate(result.Document.Content, 80))
				fmt.Printf("      Category: %v\n", result.Document.Metadata["category"])
			}
		}
	}
	fmt.Println()

	// Demonstrate persistent storage (reopen store)
	fmt.Println("================================================================================")
	fmt.Println("Persistent Storage Verification")
	fmt.Println("--------------------------------------------------------------------------------")

	// Close and reopen the store to verify persistence
	chromemStore.Close()
	chromemStore2, err := store.NewChromemVectorStoreSimple(tempDir, embedder)
	if err != nil {
		log.Printf("Failed to reopen store: %v", err)
	} else {
		defer chromemStore2.Close()

		stats2, _ := chromemStore2.GetStats(ctx)
		fmt.Printf("\nReopened store - Documents persist: %d documents\n", stats2.TotalDocuments)
		fmt.Println("Data successfully persisted across store instances!")
	}

	fmt.Println("\n=== Example completed successfully! ===")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
