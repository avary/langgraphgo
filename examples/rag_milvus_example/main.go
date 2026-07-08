package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/retriever"
	"github.com/smallnest/langgraphgo/rag/store"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	milvusv2 "github.com/tmc/langchaingo/vectorstores/milvus/v2"
)

// This example demonstrates how to use Milvus as a vector store with LangGraphGo's RAG pipeline.
//
// Prerequisites:
// 1. Install dependencies: go get github.com/tmc/langchaingo/vectorstores/milvus/v2
// 2. Start Milvus server: docker run -d --name milvus-standalone -p 19530:19530 milvusdb/milvus:latest
//
// Run the example:
//   cd examples/rag_milvus_example
//   go run main.go

func main() {
	ctx := context.Background()

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Initialize embeddings
	// Configure embedding model based on your provider:
	// - OpenAI: use "text-embedding-3-small" or "text-embedding-3-large"
	// - Qwen (DashScope): use "text-embedding-v3"
	// - Doubao: use your custom embedding endpoint ID
	// - Other providers: use the appropriate model name
	// You can also set OPENAI_EMBEDDING_MODEL environment variable
	embeddingModel := os.Getenv("OPENAI_EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = "embedding-v1" // Default to Qwen compatible model
	}

	llmForEmbeddings, err := openai.New(
		openai.WithEmbeddingModel(embeddingModel),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM for embeddings: %v", err)
	}
	openaiEmbedder, err := embeddings.NewEmbedder(llmForEmbeddings)
	if err != nil {
		log.Fatalf("Failed to create OpenAI embedder: %v", err)
	}
	embedder := rag.NewLangChainEmbedder(openaiEmbedder)

	fmt.Println("=== RAG with Milvus VectorStore Example ===")
	fmt.Println()

	// Create sample documents about Milvus
	documents := []rag.Document{
		{
			ID: "doc1",
			Content: "Milvus is an open-source vector database built to power embedding similarity search " +
				"and AI applications. It provides high-performance search capabilities for billion-scale vector data.",
			Metadata: map[string]any{"source": "milvus_docs", "category": "introduction"},
		},
		{
			ID: "doc2",
			Content: "Key features of Milvus include: billion-scale vector indexing, real-time search performance, " +
				"multiple index types (HNSW, IVF, Flat), and support for various distance metrics (L2, IP, COSINE).",
			Metadata: map[string]any{"source": "milvus_docs", "category": "features"},
		},
		{
			ID: "doc3",
			Content: "Milvus storage supports multiple backends including local disk, distributed storage, " +
				"and cloud-native solutions. It uses object storage like MinIO or S3 for data persistence.",
			Metadata: map[string]any{"source": "milvus_docs", "category": "storage"},
		},
		{
			ID: "doc4",
			Content: "Milvus provides SDKs for multiple programming languages including Go, Python, Java, and Node.js. " +
				"The Go SDK v2 offers improved performance and modern API design.",
			Metadata: map[string]any{"source": "milvus_docs", "category": "integration"},
		},
		{
			ID: "doc5",
			Content: "LangGraphGo integrates with Milvus through the LangChain adapter, providing enterprise-grade " +
				"vector search capabilities for production RAG applications.",
			Metadata: map[string]any{"source": "langgraphgo_docs", "category": "integration"},
		},
	}

	fmt.Println("Initializing Milvus vector store...")

	// Milvus connection configuration
	milvusAddr := os.Getenv("MILVUS_ADDRESS")
	if milvusAddr == "" {
		milvusAddr = "localhost:19530"
		fmt.Printf("Using default Milvus address: %s\n", milvusAddr)
		fmt.Println("Ensure Milvus is running: docker run -d --name milvus-standalone -p 19530:19530 milvusdb/milvus:latest")
		fmt.Println()
	}

	// Create Milvus vector store
	// Note: This will fail if Milvus is not running
	milvusConfig := milvusclient.ClientConfig{
		Address: milvusAddr,
	}
	milvusStore, err := milvusv2.New(
		ctx,
		milvusConfig,
		milvusv2.WithEmbedder(openaiEmbedder),
		milvusv2.WithCollectionName("langgraphgo_example"),
		milvusv2.WithDropOld(),
		milvusv2.WithMetricType(entity.COSINE), // Use COSINE similarity for embeddings
	)
	if err != nil {
		log.Printf("Failed to create Milvus store: %v", err)
		fmt.Println(`
Milvus connection failed. Please ensure:

1. Milvus is running: docker run -d --name milvus-standalone -p 19530:19530 milvusdb/milvus:latest
2. Check connection: telnet localhost 19530
3. Check Milvus logs: docker logs milvus-standalone

For setup instructions, see README.md

Using mock embedder for demonstration purposes...
`)
		// Fall back to demonstration mode with mock embedder
		demonstrationMode(ctx, llm)
		return
	}

	fmt.Printf("Connected to Milvus at: %s\n", milvusAddr)

	// Add documents to Milvus
	fmt.Println("\nAdding documents to Milvus...")
	langchainDocs := make([]schema.Document, len(documents))
	for i, doc := range documents {
		langchainDocs[i] = schema.Document{
			PageContent: doc.Content,
			Metadata:    doc.Metadata,
		}
	}

	_, err = milvusStore.AddDocuments(ctx, langchainDocs)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("Successfully added %d documents\n\n", len(documents))

	// Wrap with LangGraphGo adapter
	langGraphVectorStore := rag.NewLangChainVectorStore(milvusStore)

	// Create retriever
	vectorRetriever := retriever.NewVectorStoreRetriever(langGraphVectorStore, embedder, 2)

	// Configure RAG pipeline
	config := rag.DefaultPipelineConfig()
	config.Retriever = vectorRetriever
	config.LLM = llmtypes.Wrap(llm)

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
		"What is Milvus?",
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

	// Demonstrate similarity search with score threshold
	fmt.Println("================================================================================")
	fmt.Println("Similarity Search with Score Threshold")
	fmt.Println("--------------------------------------------------------------------------------")

	searchResults, err := milvusStore.SimilaritySearch(
		ctx,
		"vector database features",
		3,
		vectorstores.WithScoreThreshold(0.7),
	)
	if err != nil {
		log.Printf("Failed to search with threshold: %v", err)
	} else {
		fmt.Printf("\nFound %d results with score >= 0.7:\n", len(searchResults))
		for i, result := range searchResults {
			fmt.Printf("  [%d] %s\n", i+1, truncate(result.PageContent, 80))
			fmt.Printf("      Score: %.4f\n", result.Score)
			fmt.Printf("      Metadata: %v\n", result.Metadata)
		}
	}
	fmt.Println()

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nKey Milvus Configuration Options:")
	fmt.Println(`
Collection Options:
  - WithCollectionName(name)        // Set collection name
  - WithPartitionName(name)         // Enable multi-tenancy with partitions
  - WithDropOld()                   // Replace existing collection

Index Options:
  - WithIndex(index)                // Set index type (HNSW, IVF, Flat, Auto)
  - WithMetricType(entity.COSINE)   // L2, IP, COSINE, HAMMING, JACCARD

Field Options:
  - WithTextField("content")        // Default: "text"
  - WithMetaField("metadata")        // Default: "meta"
  - WithVectorField("embedding")     // Default: "vector"
  - WithPrimaryField("id")           // Default: "pk"

Performance Options:
  - WithShards(2)                   // Number of shards for parallel processing
  - WithMaxTextLength(1000)          // Max text field length
  - WithSkipFlushOnWrite()          // Skip immediate flush for bulk inserts
`)

	fmt.Println("\nFor detailed setup instructions, see README.md")
}

// demonstrationMode runs a simplified demo when Milvus is not available
func demonstrationMode(ctx context.Context, llm *openai.LLM) {
	// Use mock embedder for demonstration
	mockEmbedder := store.NewMockEmbedder(128)
	mockStore := store.NewInMemoryVectorStore(mockEmbedder)

	// Add sample documents
	docs := []rag.Document{
		{
			ID:       "doc1",
			Content:  "Milvus is an open-source vector database built for AI applications with billion-scale vector indexing.",
			Metadata: map[string]any{"source": "milvus_docs", "category": "introduction"},
		},
		{
			ID:       "doc2",
			Content:  "Key features include real-time search, multiple index types (HNSW, IVF, Flat), and various distance metrics.",
			Metadata: map[string]any{"source": "milvus_docs", "category": "features"},
		},
	}
	mockStore.Add(ctx, docs)

	// Create retriever
	vectorRetriever := retriever.NewVectorStoreRetriever(mockStore, mockEmbedder, 2)

	// Configure RAG pipeline
	config := rag.DefaultPipelineConfig()
	config.Retriever = vectorRetriever
	config.LLM = llmtypes.Wrap(llm)

	// Build basic RAG pipeline
	fmt.Println("Building RAG pipeline with mock embedder...")
	pipeline := rag.NewRAGPipeline(config)
	if err := pipeline.BuildBasicRAG(); err != nil {
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

	// Run a simple demonstration query
	fmt.Println("Running demonstration query...")
	result, err := runnable.Invoke(ctx, map[string]any{
		"query": "What is Milvus?",
	})
	if err != nil {
		log.Printf("Failed to process query: %v", err)
	} else {
		if answer, ok := result["answer"].(string); ok {
			fmt.Printf("\nAnswer:\n%s\n", answer)
		}
	}

	fmt.Println("\n=== Example completed (demo mode) ===")
	fmt.Println("\nTo use real Milvus:")
	fmt.Println("1. Install Milvus: go get github.com/tmc/langchaingo/vectorstores/milvus/v2")
	fmt.Println("2. Start Milvus server")
	fmt.Println("3. Run this example again")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
