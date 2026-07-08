package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/engine"
	"github.com/smallnest/langgraphgo/rag/store"
)

// OpenAILLMAdapter wraps langchaingo's openai.LLM to implement rag.LLMInterface
type OpenAILLMAdapter struct {
	llm *openai.LLM
}

func NewOpenAILLMAdapter(baseLLM *openai.LLM) *OpenAILLMAdapter {
	return &OpenAILLMAdapter{llm: baseLLM}
}

func (a *OpenAILLMAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	return a.llm.Call(ctx, prompt)
}

func (a *OpenAILLMAdapter) GenerateWithConfig(ctx context.Context, prompt string, config map[string]any) (string, error) {
	return a.Generate(ctx, prompt)
}

func (a *OpenAILLMAdapter) GenerateWithSystem(ctx context.Context, system, prompt string) (string, error) {
	fullPrompt := fmt.Sprintf("%s\n\n%s", system, prompt)
	return a.Generate(ctx, fullPrompt)
}

// MockLLM implements rag.LLMInterface for demonstration without API keys
type MockLLM struct{}

func (m *MockLLM) Generate(ctx context.Context, prompt string) (string, error) {
	// Return a mock response with entity extraction
	return `{
		"entities": [
			{
				"id": "entity_1",
				"name": "LangGraph",
				"type": "TECHNOLOGY",
				"description": "A library for building stateful, multi-actor applications with LLMs",
				"properties": {"category": "framework"}
			},
			{
				"id": "entity_2",
				"name": "LightRAG",
				"type": "TECHNOLOGY",
				"description": "A lightweight Retrieval-Augmented Generation framework",
				"properties": {"category": "rag"}
			}
		]
	}`, nil
}

func (m *MockLLM) GenerateWithConfig(ctx context.Context, prompt string, config map[string]any) (string, error) {
	return m.Generate(ctx, prompt)
}

func (m *MockLLM) GenerateWithSystem(ctx context.Context, system, prompt string) (string, error) {
	return m.Generate(ctx, prompt)
}

func main() {
	ctx := context.Background()

	// Check if OpenAI API key is set, not empty, and looks valid
	apiKey := os.Getenv("OPENAI_API_KEY")
	useOpenAI := apiKey != "" && len(apiKey) > 10 // Basic validation

	var llm rag.LLMInterface

	if useOpenAI {
		// Use real OpenAI LLM with explicit token
		baseLLM, err := openai.New()
		if err != nil {
			log.Printf("Failed to create OpenAI LLM: %v", err)
			log.Println("Falling back to Mock LLM")
			llm = &MockLLM{}
		} else {
			llm = NewOpenAILLMAdapter(baseLLM)
			fmt.Println("Using OpenAI LLM for entity extraction")
		}
	} else {
		// API key not set or invalid
		if apiKey != "" && len(apiKey) <= 10 {
			fmt.Println("Warning: OPENAI_API_KEY appears to be invalid (too short)")
		}
		fmt.Println("Using Mock LLM for demonstration")
		fmt.Println("Note: Set a valid OPENAI_API_KEY environment variable to use real OpenAI LLM")
		fmt.Println()
		llm = &MockLLM{}
	}

	// Create embedder
	embedder := store.NewMockEmbedder(128)

	// Create knowledge graph (in-memory)
	kg, err := store.NewKnowledgeGraph("memory://")
	if err != nil {
		log.Fatalf("Failed to create knowledge graph: %v", err)
	}

	// Create vector store (in-memory)
	vectorStore := store.NewInMemoryVectorStore(embedder)

	// Configure LightRAG
	config := rag.LightRAGConfig{
		Mode:                      "hybrid", // naive, local, global, or hybrid
		ChunkSize:                 512,
		ChunkOverlap:              50,
		MaxEntitiesPerChunk:       20,
		EntityExtractionThreshold: 0.5,
		LocalConfig: rag.LocalRetrievalConfig{
			TopK:                10,
			MaxHops:             2,
			IncludeDescriptions: true,
		},
		GlobalConfig: rag.GlobalRetrievalConfig{
			MaxCommunities:    5,
			IncludeHierarchy:  false,
			MaxHierarchyDepth: 3,
		},
		HybridConfig: rag.HybridRetrievalConfig{
			LocalWeight:  0.5,
			GlobalWeight: 0.5,
			FusionMethod: "rrf",
			RFFK:         60,
		},
		EnableCommunityDetection: true,
	}

	// Create LightRAG engine
	lightrag, err := engine.NewLightRAGEngine(config, llm, embedder, kg, vectorStore)
	if err != nil {
		log.Fatalf("Failed to create LightRAG engine: %v", err)
	}

	fmt.Println("=== LightRAG Simple Example ===")
	fmt.Println()

	// Sample documents about technology
	documents := []rag.Document{
		{
			ID: "doc1",
			Content: `LangGraph is a library for building stateful, multi-actor applications with LLMs.
It extends LangChain Expression Language with the ability to coordinate multiple chains
across multiple steps of computation in a cyclic manner. LangGraph is designed to make it
easy to build agents and multi-agent systems.`,
			Metadata: map[string]any{
				"source": "langgraph_intro.txt",
				"topic":  "LangGraph",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID: "doc2",
			Content: `LightRAG is a lightweight Retrieval-Augmented Generation framework that combines
low-level semantic chunks with high-level graph structures. It supports four retrieval modes:
naive, local, global, and hybrid. LightRAG provides a simple API for building knowledge graphs
and performing semantic search.`,
			Metadata: map[string]any{
				"source": "lightrag_overview.txt",
				"topic":  "LightRAG",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID: "doc3",
			Content: `Knowledge graphs are structured representations of knowledge that use entities
and relationships to model information. They are particularly useful for RAG systems because
they enable traversing related concepts and finding multi-hop connections between pieces of
information. Popular knowledge graph databases include Neo4j, FalkorDB, and GraphDB.`,
			Metadata: map[string]any{
				"source": "knowledge_graphs.txt",
				"topic":  "Knowledge Graphs",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID: "doc4",
			Content: `Vector databases are designed to store and query high-dimensional vectors efficiently.
They use approximate nearest neighbor (ANN) algorithms to quickly find similar vectors.
Popular vector databases include Pinecone, Weaviate, Chroma, and Qdrant. They are essential
for semantic search and RAG applications.`,
			Metadata: map[string]any{
				"source": "vector_databases.txt",
				"topic":  "Vector Databases",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID: "doc5",
			Content: `RAG (Retrieval-Augmented Generation) combines retrieval systems with language models
to improve answer quality. It retrieves relevant documents from a knowledge base and uses them
to augment the context provided to the language model. This helps reduce hallucinations and
improves factual accuracy of responses.`,
			Metadata: map[string]any{
				"source": "rag_intro.txt",
				"topic":  "RAG",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	fmt.Println("Adding documents to LightRAG...")
	err = lightrag.AddDocuments(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}

	fmt.Printf("Successfully indexed %d documents\n\n", len(documents))

	// Display configuration
	fmt.Println("=== LightRAG Configuration ===")
	fmt.Printf("Mode: %s\n", config.Mode)
	fmt.Printf("Chunk Size: %d\n", config.ChunkSize)
	fmt.Printf("Chunk Overlap: %d\n", config.ChunkOverlap)
	fmt.Printf("Local Config: TopK=%d, MaxHops=%d\n",
		config.LocalConfig.TopK, config.LocalConfig.MaxHops)
	fmt.Printf("Global Config: MaxCommunities=%d\n", config.GlobalConfig.MaxCommunities)
	fmt.Printf("Hybrid Config: LocalWeight=%.2f, GlobalWeight=%.2f, FusionMethod=%s\n",
		config.HybridConfig.LocalWeight, config.HybridConfig.GlobalWeight, config.HybridConfig.FusionMethod)
	fmt.Println()

	// Test different retrieval modes
	modes := []string{"naive", "local", "global", "hybrid"}
	queries := []string{
		"What is LightRAG and how does it work?",
		"Explain the relationship between RAG and knowledge graphs",
		"What are the benefits of using vector databases?",
	}

	for _, mode := range modes {
		fmt.Printf("=== Testing %s Mode ===\n", strings.ToUpper(mode))

		// Update configuration for this mode
		testConfig := config
		testConfig.Mode = mode

		for i, query := range queries {
			fmt.Printf("\n--- Query %d: %s ---\n", i+1, query)

			// Query with the current mode
			result, err := lightrag.QueryWithConfig(ctx, query, &rag.RetrievalConfig{
				K:              3,
				ScoreThreshold: 0.3,
				SearchType:     mode,
				IncludeScores:  true,
			})
			if err != nil {
				log.Printf("Query failed: %v", err)
				continue
			}

			// Display results
			fmt.Printf("Retrieved %d sources\n", len(result.Sources))
			fmt.Printf("Confidence: %.2f\n", result.Confidence)
			fmt.Printf("Response Time: %v\n", result.ResponseTime)

			// Show metadata
			if modeVal, ok := result.Metadata["mode"].(string); ok {
				fmt.Printf("Mode: %s\n", modeVal)
			}

			// Show first few results
			fmt.Println("\nTop Sources:")
			for j, source := range result.Sources {
				if j >= 2 {
					break
				}
				fmt.Printf("  [%d] %s\n", j+1, truncate(source.Content, 100))
			}
		}
		fmt.Println()
	}

	// Display metrics
	fmt.Println("\n=== LightRAG Metrics ===")
	metrics := lightrag.GetMetrics()
	fmt.Printf("Total Queries: %d\n", metrics.TotalQueries)
	fmt.Printf("Total Documents: %d\n", metrics.TotalDocuments)
	fmt.Printf("Average Latency: %v\n", metrics.AverageLatency)
	fmt.Printf("Indexing Latency: %v\n", metrics.IndexingLatency)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
