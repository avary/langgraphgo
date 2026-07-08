package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/retriever"
	"github.com/smallnest/langgraphgo/rag/splitter"
	"github.com/smallnest/langgraphgo/rag/store"
)

func main() {
	// Check for API keys
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		fmt.Println("OPENAI_API_KEY not set. Skipping execution.")
		fmt.Println("\nTo run this example:")
		fmt.Println("1. Set OPENAI_API_KEY for LLM-based reranking")
		fmt.Println("2. Optionally set COHERE_API_KEY for Cohere reranking")
		fmt.Println("3. Optionally set JINA_API_KEY for Jina reranking")
		return
	}

	ctx := context.Background()

	fmt.Println("=== RAG Reranker Comparison Example ===")

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Create sample documents
	documents := createSampleDocuments()

	// Split documents
	splitter := splitter.NewSimpleTextSplitter(200, 50)
	chunks := splitter.SplitDocuments(documents)
	fmt.Printf("Split %d documents into %d chunks\n\n", len(documents), len(chunks))

	// Create embedder and vector store
	embedder := store.NewMockEmbedder(256)
	vectorStore := store.NewInMemoryVectorStore(embedder)

	// Generate embeddings and add chunks to vector store
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	err = vectorStore.AddBatch(ctx, chunks, embeddings)
	if err != nil {
		log.Fatalf("Failed to add documents to vector store: %v", err)
	}

	// Test query
	query := "What is LangGraph and how does it help with multi-agent systems?"
	fmt.Printf("Query: %s\n\n", query)

	// Create base retriever
	baseRetriever := retriever.NewVectorStoreRetriever(vectorStore, embedder, 5)

	// Define rerankers to test
	rerankers := []struct {
		name     string
		reranker rag.Reranker
	}{
		{
			name:     "SimpleReranker (keyword-based)",
			reranker: retriever.NewSimpleReranker(),
		},
		{
			name:     "LLMReranker",
			reranker: retriever.NewLLMReranker(llm, retriever.DefaultLLMRerankerConfig()),
		},
	}

	// Add Cohere reranker if API key is available
	if os.Getenv("COHERE_API_KEY") != "" {
		cohereReranker := retriever.NewCohereReranker("", retriever.DefaultCohereRerankerConfig())
		rerankers = append(rerankers, struct {
			name     string
			reranker rag.Reranker
		}{
			name:     "CohereReranker",
			reranker: cohereReranker,
		})
		fmt.Println("Added CohereReranker")
	}

	// Add Jina reranker if API key is available
	if os.Getenv("JINA_API_KEY") != "" {
		jinaReranker := retriever.NewJinaReranker("", retriever.DefaultJinaRerankerConfig())
		rerankers = append(rerankers, struct {
			name     string
			reranker rag.Reranker
		}{
			name:     "JinaReranker",
			reranker: jinaReranker,
		})
		fmt.Println("Added JinaReranker")
	}

	// Test each reranker
	for _, rr := range rerankers {
		fmt.Printf("\n--- %s ---\n", rr.name)

		// Create pipeline with this reranker
		config := rag.DefaultPipelineConfig()
		config.Retriever = baseRetriever
		config.Reranker = rr.reranker
		config.LLM = llm
		config.TopK = 3
		config.UseReranking = true

		pipeline := rag.NewRAGPipeline(config)
		err = pipeline.BuildAdvancedRAG()
		if err != nil {
			log.Printf("Failed to build pipeline: %v", err)
			continue
		}

		runnable, err := pipeline.Compile()
		if err != nil {
			log.Printf("Failed to compile pipeline: %v", err)
			continue
		}

		// Run query
		result, err := runnable.Invoke(ctx, map[string]any{
			"query": query,
		})
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		// Display results
		displayResults(result)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("\nNotes:")
	fmt.Println("- SimpleReranker: Fast, keyword-based, no API calls")
	fmt.Println("- LLMReranker: Uses LLM to score documents, slower but good semantic understanding")
	fmt.Println("- CohereReranker: High-quality results, requires Cohere API key")
	fmt.Println("- JinaReranker: High-quality results, supports multiple languages")
	fmt.Println("\nFor cross-encoder reranking, see scripts/cross_encoder_server.py")
}

func createSampleDocuments() []rag.Document {
	return []rag.Document{
		{
			Content: "LangGraph is a library for building stateful, multi-actor applications with LLMs. " +
				"It extends LangChain Expression Language with the ability to coordinate multiple chains " +
				"across multiple steps of computation in a cyclic manner. LangGraph is particularly useful " +
				"for building complex agent workflows and multi-agent systems where agents can communicate " +
				"with each other and maintain state across interactions.",
			Metadata: map[string]any{
				"source":   "langgraph_intro.txt",
				"topic":    "LangGraph",
				"category": "Framework",
			},
		},
		{
			Content: "Multi-agent systems in LangGraph enable multiple AI agents to work together on complex tasks. " +
				"Each agent can have specialized roles, tools, and objectives. The graph-based architecture allows " +
				"agents to pass messages, share state, and coordinate their actions. This enables sophisticated " +
				"workflows like research teams, code generation pipelines, and decision-making systems.",
			Metadata: map[string]any{
				"source":   "multi_agent.txt",
				"topic":    "Multi-Agent",
				"category": "Architecture",
			},
		},
		{
			Content: "RAG (Retrieval-Augmented Generation) is a technique that combines information retrieval " +
				"with text generation. It retrieves relevant documents from a knowledge base and uses them " +
				"to augment the context provided to a language model for generation. This approach helps " +
				"reduce hallucinations and provides more factual, grounded responses.",
			Metadata: map[string]any{
				"source":   "rag_overview.txt",
				"topic":    "RAG",
				"category": "Technique",
			},
		},
		{
			Content: "Vector databases store embeddings, which are numerical representations of text. " +
				"They enable efficient similarity search by comparing vector distances using metrics like " +
				"cosine similarity or Euclidean distance. Popular vector databases include Pinecone, Weaviate, " +
				"Chroma, and Qdrant.",
			Metadata: map[string]any{
				"source":   "vector_db.txt",
				"topic":    "Vector Databases",
				"category": "Infrastructure",
			},
		},
		{
			Content: "Document reranking is a technique to improve retrieval quality by re-scoring retrieved " +
				"documents based on their relevance to the query. Cross-encoder models are often used for " +
				"reranking as they can better capture query-document interactions compared to bi-encoders " +
				"used for initial retrieval. Popular reranking services include Cohere Rerank and Jina Rerank.",
			Metadata: map[string]any{
				"source":   "reranking.txt",
				"topic":    "Reranking",
				"category": "Technique",
			},
		},
		{
			Content: "State management in LangGraph is handled through a stateful graph where each node " +
				"can read and modify the state. The state flows through the graph and evolves at each step. " +
				"This allows agents to maintain context, remember previous interactions, and make decisions " +
				"based on accumulated information.",
			Metadata: map[string]any{
				"source":   "state_management.txt",
				"topic":    "State Management",
				"category": "Core Concept",
			},
		},
	}
}

func displayResults(result map[string]any) {
	// Display retrieved documents
	if docs, ok := result["documents"].([]rag.RAGDocument); ok && len(docs) > 0 {
		fmt.Println("Top Retrieved Documents:")
		for i, doc := range docs {
			source := "Unknown"
			if s, ok := doc.Metadata["source"].(string); ok {
				source = s
			}
			topic := "N/A"
			if t, ok := doc.Metadata["topic"].(string); ok {
				topic = t
			}
			fmt.Printf("  [%d] %s (Topic: %s)\n", i+1, source, topic)
			fmt.Printf("      %s\n", truncate(doc.Content, 100))
		}
	}

	// Display reranked scores if available
	if rankedDocs, ok := result["ranked_documents"].([]rag.DocumentSearchResult); ok && len(rankedDocs) > 0 {
		fmt.Println("\nRelevance Scores:")
		for i, rd := range rankedDocs {
			if i >= 3 {
				break
			}
			method := "original"
			if m, ok := rd.Metadata["reranking_method"].(string); ok {
				method = m
			}
			fmt.Printf("  [%d] Score: %.4f (Method: %s)\n", i+1, rd.Score, method)
		}
	}

	// Display answer if available
	if answer, ok := result["answer"].(string); ok {
		fmt.Printf("\nAnswer: %s\n", truncate(answer, 200))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
