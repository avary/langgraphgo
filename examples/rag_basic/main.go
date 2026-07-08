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
			Content: "LangGraph is a library for building stateful, multi-actor applications with LLMs. " +
				"It extends LangChain Expression Language with the ability to coordinate multiple chains " +
				"(or actors) across multiple steps of computation in a cyclic manner.",
			Metadata: map[string]any{
				"source": "langgraph_intro.txt",
				"topic":  "LangGraph",
			},
		},
		{
			Content: "RAG (Retrieval-Augmented Generation) is a technique that combines information retrieval " +
				"with text generation. It retrieves relevant documents from a knowledge base and uses them " +
				"to augment the context provided to a language model for generation.",
			Metadata: map[string]any{
				"source": "rag_overview.txt",
				"topic":  "RAG",
			},
		},
		{
			Content: "Vector databases store embeddings, which are numerical representations of text. " +
				"They enable efficient similarity search by comparing vector distances. " +
				"Popular vector databases include Pinecone, Weaviate, and Chroma.",
			Metadata: map[string]any{
				"source": "vector_db.txt",
				"topic":  "Vector Databases",
			},
		},
		{
			Content: "Text embeddings are dense vector representations of text that capture semantic meaning. " +
				"Models like OpenAI's text-embedding-ada-002 or sentence transformers can generate these embeddings. " +
				"Similar texts have similar embeddings in the vector space.",
			Metadata: map[string]any{
				"source": "embeddings.txt",
				"topic":  "Embeddings",
			},
		},
	}

	// Create embedder and vector store
	embedder := store.NewMockEmbedder(128)
	vectorStore := store.NewInMemoryVectorStore(embedder)

	// Generate embeddings and add documents to vector store
	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.Content
	}

	embeddings, err := embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	err = vectorStore.AddBatch(ctx, documents, embeddings)
	if err != nil {
		log.Fatalf("Failed to add documents to vector store: %v", err)
	}

	// Create retriever
	retriever := retriever.NewVectorStoreRetriever(vectorStore, embedder, 3)

	// Configure RAG pipeline
	config := rag.DefaultPipelineConfig()
	config.Retriever = retriever
	config.LLM = llm
	config.TopK = 3
	config.SystemPrompt = "You are a helpful AI assistant. Answer the question based on the provided context. " +
		"If the context doesn't contain enough information to answer the question, say so."

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

	// Visualize the pipeline
	exporter := graph.GetGraphForRunnable(runnable)
	fmt.Println("=== RAG Pipeline Visualization (Mermaid) ===")
	fmt.Println(exporter.DrawMermaid())
	fmt.Println()

	// Test queries
	queries := []string{
		"What is LangGraph?",
		"How does RAG work?",
		"What are vector databases used for?",
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

		// In map[string]any state, we need to extract documents
		if docs, ok := finalState["documents"].([]rag.RAGDocument); ok {
			fmt.Println("Retrieved Documents:")
			for j, doc := range docs {
				source := "Unknown"
				if s, ok := doc.Metadata["source"]; ok {
					source = fmt.Sprintf("%v", s)
				}
				fmt.Printf("  [%d] %s\n", j+1, source)
				fmt.Printf("      %s...\n", truncate(doc.Content, 100))
			}
		}

		if answer, ok := finalState["answer"].(string); ok {
			fmt.Printf("\nAnswer: %s\n", answer)
		}
		fmt.Println("\n" + strings.Repeat("-", 80) + "\n")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
