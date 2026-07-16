package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/retriever"
	"github.com/smallnest/langgraphgo/rag/store"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	ctx := context.Background()

	// Initialize LLM
	llm, err := openai.New(openai.WithEmbeddingModel("embedding-v1"))
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Initialize OpenAI embedder
	openaiEmbedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatalf("Failed to create OpenAI embedder: %v", err)
	}

	// Wrap with LangGraphGo adapter
	embedder := rag.NewLangChainEmbedder(openaiEmbedder)

	fmt.Println("=== RAG with Chroma v2 API VectorStore Example ===")
	fmt.Println()

	// Check if Chroma URL is set
	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = "http://localhost:8000"
	}
	fmt.Printf("Connecting to Chroma v2 at: %s\n\n", chromaURL)

	// Create Chroma v2 vector store
	chromaStore, err := store.NewChromaV2VectorStoreSimple(chromaURL, "langgraphgo_v2_test", embedder)
	if err != nil {
		log.Fatalf("Failed to create Chroma v2 store: %v", err)
	}
	defer chromaStore.Close()

	fmt.Printf("Store created with collection: %s (ID: %s)\n\n",
		chromaStore.GetCollectionName(), chromaStore.GetCollectionID())

	// Create sample documents about Chroma v2
	documents := []rag.Document{
		{
			ID:      "doc1",
			Content: "Chroma v2 API introduces a new hierarchical structure with tenants, databases, and collections for better multi-tenancy support.",
			Metadata: map[string]any{
				"source":   "chroma_v2_docs",
				"category": "architecture",
			},
		},
		{
			ID:      "doc2",
			Content: "Key features of Chroma v2 include RESTful API design, improved authentication, granular permissions, and better performance.",
			Metadata: map[string]any{
				"source":   "chroma_v2_docs",
				"category": "features",
			},
		},
		{
			ID:      "doc3",
			Content: "Chroma v2 uses /api/v2 prefix for all endpoints and follows OpenAPI 3.1.0 specification for better integration.",
			Metadata: map[string]any{
				"source":   "chroma_v2_docs",
				"category": "api",
			},
		},
		{
			ID:      "doc4",
			Content: "LangGraphGo provides native Chroma v2 support through ChromaV2VectorStore, implementing the VectorStore interface.",
			Metadata: map[string]any{
				"source":   "langgraphgo_docs",
				"category": "integration",
			},
		},
		{
			ID:      "doc5",
			Content: "Chroma v2 separates concepts into tenant, database, and collection hierarchy for improved resource organization.",
			Metadata: map[string]any{
				"source":   "chroma_v2_docs",
				"category": "architecture",
			},
		},
	}

	fmt.Println("Adding documents to Chroma v2...")
	err = chromaStore.Add(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("Successfully added %d documents\n\n", len(documents))

	// Display store statistics
	stats, err := chromaStore.GetStats(ctx)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("Store Statistics:\n")
		fmt.Printf("  Total Documents: %d\n", stats.TotalDocuments)
		fmt.Printf("  Vector Dimension: %d\n\n", stats.Dimension)
	}

	// Create retriever
	vectorRetriever := retriever.NewVectorStoreRetriever(chromaStore, embedder, 2)

	// Configure RAG pipeline
	config := rag.DefaultPipelineConfig()
	config.Retriever = vectorRetriever
	config.LLM = llms.Wrap(llm)

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
		"What is new in Chroma v2?",
		"What are the key features?",
		"Tell me about the API structure",
	}

	for i, query := range queries {
		fmt.Println("================================================================================")
		fmt.Printf("Query %d: %s\n", i+1, query)
		fmt.Println("--------------------------------------------------------------------------------")

		// Generate query embedding
		queryEmbedding, err := embedder.EmbedDocument(ctx, query)
		if err != nil {
			log.Printf("Failed to generate query embedding: %v", err)
			continue
		}

		// Test direct similarity search from Chroma v2
		fmt.Println("\nDirect Chroma v2 similarity search results:")
		results, err := chromaStore.Search(ctx, queryEmbedding, 2)
		if err != nil {
			log.Printf("Failed to search: %v", err)
		} else {
			for j, result := range results {
				fmt.Printf("  [%d] Score: %.4f\n", j+1, result.Score)
				fmt.Printf("      %s\n", truncate(result.Document.Content, 100))
				fmt.Printf("      Metadata: %v\n", result.Document.Metadata)
			}
		}

		// Test through RAG pipeline
		result, err := runnable.Invoke(ctx, map[string]any{
			"query": query,
		})
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		if answer, ok := result["answer"].(string); ok {
			fmt.Printf("\nRAG Answer:\n%s\n", answer)
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
	fmt.Println("Metadata Filtering Example (Chroma v2 Native)")
	fmt.Println("--------------------------------------------------------------------------------")

	// Generate query embedding for filtering
	queryEmbedding, err := embedder.EmbedDocument(ctx, "architecture and features")
	if err != nil {
		log.Printf("Failed to generate query embedding: %v", err)
	} else {
		// Search with metadata filter
		filteredResults, err := chromaStore.SearchWithFilter(ctx, queryEmbedding, 10,
			map[string]any{"category": "architecture"})
		if err != nil {
			log.Printf("Failed to search with filter: %v", err)
		} else {
			fmt.Printf("\nFound %d documents with category='architecture':\n", len(filteredResults))
			for i, result := range filteredResults {
				fmt.Printf("  [%d] Score: %.4f\n", i+1, result.Score)
				fmt.Printf("      %s\n", truncate(result.Document.Content, 80))
				fmt.Printf("      Category: %v\n", result.Document.Metadata["category"])
			}
		}
	}
	fmt.Println()

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nNote: This example uses Chroma v2 API.")
	fmt.Println("Chroma v2 is the default in latest Chroma versions.")
	fmt.Println("Start Chroma with: docker run -p 8000:8000 chromadb/chroma")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
