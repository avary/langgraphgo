package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/retriever"
	"github.com/smallnest/langgraphgo/rag/splitter"
	"github.com/smallnest/langgraphgo/rag/store"
)

func main() {
	ctx := context.Background()

	fmt.Println("Initializing LLM...")
	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}
	fmt.Println("LLM Initialized.")

	// Create a larger document corpus
	documents := []rag.Document{
		{
			Content: "LangGraph is a library for building stateful, multi-actor applications with LLMs. " +
				"It extends LangChain Expression Language with the ability to coordinate multiple chains " +
				"across multiple steps of computation in a cyclic manner. LangGraph is particularly useful " +
				"for building complex agent workflows and multi-agent systems.",
			Metadata: map[string]any{
				"source":   "langgraph_intro.txt",
				"topic":    "LangGraph",
				"category": "Framework",
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
				"Chroma, and Qdrant. These databases are essential for RAG systems.",
			Metadata: map[string]any{
				"source":   "vector_db.txt",
				"topic":    "Vector Databases",
				"category": "Infrastructure",
			},
		},
		{
			Content: "Text embeddings are dense vector representations of text that capture semantic meaning. " +
				"Models like OpenAI's text-embedding-ada-002, sentence transformers, or Cohere embeddings " +
				"can generate these embeddings. Similar texts have similar embeddings in the vector space, " +
				"which enables semantic search.",
			Metadata: map[string]any{
				"source":   "embeddings.txt",
				"topic":    "Embeddings",
				"category": "Technique",
			},
		},
		{
			Content: "Document reranking is a technique to improve retrieval quality by re-scoring retrieved " +
				"documents based on their relevance to the query. Cross-encoder models are often used for " +
				"reranking as they can better capture query-document interactions compared to bi-encoders " +
				"used for initial retrieval.",
			Metadata: map[string]any{
				"source":   "reranking.txt",
				"topic":    "Reranking",
				"category": "Technique",
			},
		},
		{
			Content: "Multi-agent systems involve multiple AI agents working together to solve complex problems. " +
				"Each agent can have specialized roles and capabilities. LangGraph provides excellent support " +
				"for building multi-agent systems with its graph-based architecture and state management.",
			Metadata: map[string]any{
				"source":   "multi_agent.txt",
				"topic":    "Multi-Agent",
				"category": "Architecture",
			},
		},
	}

	// Split documents into smaller chunks
	splitter := splitter.NewSimpleTextSplitter(200, 50)
	chunks := splitter.SplitDocuments(documents)

	fmt.Printf("Split %d documents into %d chunks\n\n", len(documents), len(chunks))

	// Create embedder and vector store
	embedder := store.NewMockEmbedder(256) // Higher dimension for better quality
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

	// Create retriever and reranker
	vecRetriever := retriever.NewVectorStoreRetriever(vectorStore, embedder, 5)
	reranker := retriever.NewSimpleReranker()

	// Configure advanced RAG pipeline with reranking and citations
	config := rag.DefaultPipelineConfig()
	config.Retriever = vecRetriever
	config.Reranker = reranker
	config.LLM = llm
	config.TopK = 5
	config.UseReranking = true
	config.IncludeCitations = true
	config.SystemPrompt = "You are a knowledgeable AI assistant. Answer questions based on the provided context. " +
		"Always cite your sources using the document numbers provided. If the context doesn't contain " +
		"enough information, acknowledge the limitations and provide what you can."

	// Build advanced RAG pipeline
	pipeline := rag.NewRAGPipeline(config)
	err = pipeline.BuildAdvancedRAG()
	if err != nil {
		log.Fatalf("Failed to build advanced RAG pipeline: %v", err)
	}

	// Compile the pipeline
	runnable, err := pipeline.Compile()
	if err != nil {
		log.Fatalf("Failed to compile pipeline: %v", err)
	}

	// Visualize the pipeline
	exporter := graph.GetGraphForRunnable(runnable)
	fmt.Println("=== Advanced RAG Pipeline Visualization (Mermaid) ===")
	fmt.Println(exporter.DrawMermaid())
	fmt.Println()

	// Test queries with more complex questions
	queries := []string{
		"What is LangGraph and how is it used in multi-agent systems?",
		"Explain the RAG technique and its benefits",
		"What role do vector databases play in RAG systems?",
		"How does document reranking improve retrieval quality?",
	}

	for i, query := range queries {
		fmt.Printf("=== Query %d ===\n", i+1)
		fmt.Printf("Question: %s\n\n", query)

		result, err := runnable.Invoke(ctx, map[string]any{
			"query": query,
		})
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		finalState := result

		fmt.Println("Retrieved and Reranked Documents:")
		if docs, ok := finalState["documents"].([]rag.RAGDocument); ok {
			for j, doc := range docs {
				source := "Unknown"
				if s, ok := doc.Metadata["source"]; ok {
					source = fmt.Sprintf("%v", s)
				}
				category := "N/A"
				if c, ok := doc.Metadata["category"]; ok {
					category = fmt.Sprintf("%v", c)
				}
				fmt.Printf("  [%d] %s (Category: %s)\n", j+1, source, category)
				fmt.Printf("      %s\n", truncate(doc.Content, 120))
			}
		}

		if rankedDocs, ok := finalState["ranked_documents"].([]rag.DocumentSearchResult); ok {
			if len(rankedDocs) > 0 {
				fmt.Printf("\nRelevance Scores:\n")
				for j, rd := range rankedDocs {
					if j >= 3 {
						break // Show top 3 scores
					}
					fmt.Printf("  [%d] Score: %.4f\n", j+1, rd.Score)
				}
			}
		}

		if answer, ok := finalState["answer"].(string); ok {
			fmt.Printf("\nAnswer: %s\n", answer)
		}

		if citations, ok := finalState["citations"].([]string); ok {
			if len(citations) > 0 {
				fmt.Println("\nCitations:")
				for _, citation := range citations {
					fmt.Printf("  %s\n", citation)
				}
			}
		}

		fmt.Println("\n" + strings.Repeat("=", 100) + "\n")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
