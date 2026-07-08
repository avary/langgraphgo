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
	return `{
		"entities": [
			{
				"id": "entity_1",
				"name": "AI",
				"type": "CONCEPT",
				"description": "Artificial Intelligence",
				"properties": {"field": "technology"}
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
		baseLLM, err := openai.New(
			openai.WithToken(apiKey),
		)
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

	fmt.Println("=== LightRAG Advanced Example ===")
	fmt.Println("This example demonstrates advanced features of LightRAG including:")
	fmt.Println("- Custom prompt templates")
	fmt.Println("- Community detection")
	fmt.Println("- Different fusion methods")
	fmt.Println("- Performance comparison between modes")
	fmt.Println()

	// Configure LightRAG with advanced options
	config := rag.LightRAGConfig{
		Mode:                      "hybrid",
		ChunkSize:                 512,
		ChunkOverlap:              50,
		MaxEntitiesPerChunk:       20,
		EntityExtractionThreshold: 0.5,
		Temperature:               0.7,

		// Local retrieval configuration
		LocalConfig: rag.LocalRetrievalConfig{
			TopK:                15,
			MaxHops:             3,
			IncludeDescriptions: true,
			EntityWeight:        0.8,
		},

		// Global retrieval configuration
		GlobalConfig: rag.GlobalRetrievalConfig{
			MaxCommunities:    10,
			IncludeHierarchy:  true,
			MaxHierarchyDepth: 5,
			CommunityWeight:   0.7,
		},

		// Hybrid retrieval configuration
		HybridConfig: rag.HybridRetrievalConfig{
			LocalWeight:  0.6,
			GlobalWeight: 0.4,
			FusionMethod: "rrf",
			RFFK:         60,
		},

		// Enable community detection
		EnableCommunityDetection:    true,
		CommunityDetectionAlgorithm: "louvain",

		// Custom prompt templates
		PromptTemplates: map[string]string{
			"entity_extraction": `Extract key entities from the following text.
Focus on: %s

Return JSON:
{
  "entities": [
    {
      "id": "unique_id",
      "name": "entity_name",
      "type": "PERSON|ORGANIZATION|PRODUCT|CONCEPT|TECHNOLOGY",
      "description": "brief description",
      "properties": {"importance": "high|medium|low"}
    }
  ]
}

Text: %s`,

			"relationship_extraction": `Extract relationships between these entities: %s

From text: %s

Return JSON:
{
  "relationships": [
    {
      "source": "entity1",
      "target": "entity2",
      "type": "RELATED_TO|PART_OF|USES|COMPETES_WITH",
      "confidence": 0.9
    }
  ]
}`,
		},
	}

	// Create LightRAG engine
	lightrag, err := engine.NewLightRAGEngine(config, llm, embedder, kg, vectorStore)
	if err != nil {
		log.Fatalf("Failed to create LightRAG engine: %v", err)
	}

	// Sample documents about AI and Machine Learning
	documents := createSampleDocuments()

	fmt.Println("Indexing documents...")
	startTime := time.Now()
	err = lightrag.AddDocuments(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	indexDuration := time.Since(startTime)

	fmt.Printf("Indexed %d documents in %v\n\n", len(documents), indexDuration)

	// Demonstrate different retrieval modes
	fmt.Println("=== Retrieval Mode Comparison ===\n")

	testQuery := "How do transformer models work and what are their applications?"

	// Test each mode
	modes := []struct {
		name string
		mode string
	}{
		{"Naive", "naive"},
		{"Local", "local"},
		{"Global", "global"},
		{"Hybrid", "hybrid"},
	}

	results := make(map[string]*rag.QueryResult)

	for _, m := range modes {
		fmt.Printf("--- %s Mode ---\n", m.name)

		start := time.Now()
		result, err := lightrag.QueryWithConfig(ctx, testQuery, &rag.RetrievalConfig{
			K:              5,
			ScoreThreshold: 0.3,
			SearchType:     m.mode,
			IncludeScores:  true,
		})
		duration := time.Since(start)

		if err != nil {
			log.Printf("Query failed: %v\n", err)
			continue
		}

		results[m.name] = result

		fmt.Printf("Response Time: %v\n", duration)
		fmt.Printf("Sources Retrieved: %d\n", len(result.Sources))
		fmt.Printf("Confidence: %.2f\n", result.Confidence)

		// Show mode-specific metadata
		switch m.mode {
		case "local":
			if queryEntities, ok := result.Metadata["query_entities"].(int); ok {
				fmt.Printf("Query Entities: %d\n", queryEntities)
			}
		case "global":
			if numCommunities, ok := result.Metadata["num_communities"].(int); ok {
				fmt.Printf("Communities: %d\n", numCommunities)
			}
		case "hybrid":
			if localConf, ok := result.Metadata["local_confidence"].(float64); ok {
				if globalConf, ok := result.Metadata["global_confidence"].(float64); ok {
					fmt.Printf("Local Confidence: %.2f\n", localConf)
					fmt.Printf("Global Confidence: %.2f\n", globalConf)
				}
			}
		}

		// Show top source
		if len(result.Sources) > 0 {
			fmt.Printf("\nTop Source:\n%s\n", truncate(result.Sources[0].Content, 150))
		}
		fmt.Println()
	}

	// Demonstrate fusion methods comparison
	fmt.Println("\n=== Fusion Method Comparison (Hybrid Mode) ===\n")

	fusionMethods := []string{"rrf", "weighted"}

	for _, method := range fusionMethods {
		fmt.Printf("--- %s Fusion ---\n", strings.ToUpper(method))

		// Update config for this fusion method
		testConfig := config
		testConfig.HybridConfig.FusionMethod = method

		// Create new engine with this config
		testEngine, err := engine.NewLightRAGEngine(testConfig, llm, embedder, kg, vectorStore)
		if err != nil {
			log.Printf("Failed to create engine: %v\n", err)
			continue
		}

		// Re-add documents
		_ = testEngine.AddDocuments(ctx, documents)

		// Query
		start := time.Now()
		result, err := testEngine.Query(ctx, testQuery)
		duration := time.Since(start)

		if err != nil {
			log.Printf("Query failed: %v\n", err)
			continue
		}

		fmt.Printf("Response Time: %v\n", duration)
		fmt.Printf("Sources: %d\n", len(result.Sources))
		fmt.Printf("Confidence: %.2f\n", result.Confidence)
		fmt.Println()
	}

	// Demonstrate knowledge graph traversal
	fmt.Println("\n=== Knowledge Graph Traversal ===\n")

	graphKg := lightrag.GetKnowledgeGraph()

	// Query the knowledge graph
	graphResult, err := graphKg.Query(ctx, &rag.GraphQuery{
		EntityTypes: []string{"CONCEPT", "TECHNOLOGY"},
		Limit:       5,
	})

	if err == nil {
		fmt.Printf("Found %d entities in knowledge graph\n", len(graphResult.Entities))

		for i, entity := range graphResult.Entities {
			if i >= 3 {
				break
			}
			fmt.Printf("  - %s (%s)\n", entity.Name, entity.Type)
		}

		if len(graphResult.Relationships) > 0 {
			fmt.Printf("\nFound %d relationships\n", len(graphResult.Relationships))
			for i, rel := range graphResult.Relationships {
				if i >= 3 {
					break
				}
				fmt.Printf("  - %s -> %s (%s)\n", rel.Source, rel.Target, rel.Type)
			}
		}
	}

	// Demonstrate document operations
	fmt.Println("\n=== Document Operations ===\n")

	// Add a new document
	newDoc := rag.Document{
		ID: "doc_new",
		Content: `Diffusion models are a class of generative models that work by gradually
adding noise to data until it becomes random noise, then learning to reverse this process
to generate new data. They have shown impressive results in image generation, with models
like DALL-E 2, Stable Diffusion, and Midjourney using this approach.`,
		Metadata: map[string]any{
			"source": "diffusion_models.txt",
			"topic":  "Diffusion Models",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	fmt.Println("Adding new document...")
	err = lightrag.AddDocuments(ctx, []rag.Document{newDoc})
	if err != nil {
		log.Printf("Failed to add document: %v\n", err)
	} else {
		fmt.Println("Document added successfully")

		// Query about the new topic
		result, err := lightrag.Query(ctx, "What are diffusion models?")
		if err == nil {
			fmt.Printf("Query about new topic returned %d sources\n", len(result.Sources))
		}
	}

	// Update document
	fmt.Println("\nUpdating document...")
	newDoc.Content = `Diffusion models are a class of generative models that work by
gradually adding noise to data until it becomes random noise, then learning to reverse this
process to generate new data. Models like DALL-E 2, Stable Diffusion, and Midjourney use
this approach. They compete with GANs (Generative Adversarial Networks) in image generation.`
	newDoc.UpdatedAt = time.Now()

	err = lightrag.UpdateDocument(ctx, newDoc)
	if err != nil {
		log.Printf("Failed to update document: %v\n", err)
	} else {
		fmt.Println("Document updated successfully")
	}

	// Performance comparison
	fmt.Println("\n=== Performance Comparison ===\n")

	benchmarkQueries := []string{
		"What is machine learning?",
		"Explain neural networks",
		"How do transformers work?",
		"What are the applications of AI?",
	}

	fmt.Printf("Running %d queries for performance comparison...\n", len(benchmarkQueries))

	for _, m := range modes {
		var totalDuration time.Duration
		successCount := 0

		for _, query := range benchmarkQueries {
			start := time.Now()
			_, err := lightrag.QueryWithConfig(ctx, query, &rag.RetrievalConfig{
				K:          5,
				SearchType: m.mode,
			})
			duration := time.Since(start)

			if err == nil {
				totalDuration += duration
				successCount++
			}
		}

		if successCount > 0 {
			avgDuration := totalDuration / time.Duration(successCount)
			fmt.Printf("%s Mode: Avg %v per query (%d/%d successful)\n",
				m.name, avgDuration, successCount, len(benchmarkQueries))
		}
	}

	// Final metrics
	fmt.Println("\n=== Final Metrics ===\n")

	metrics := lightrag.GetMetrics()
	fmt.Printf("Total Queries: %d\n", metrics.TotalQueries)
	fmt.Printf("Total Documents: %d\n", metrics.TotalDocuments)
	fmt.Printf("Average Latency: %v\n", metrics.AverageLatency)
	fmt.Printf("Min Latency: %v\n", metrics.MinLatency)
	fmt.Printf("Max Latency: %v\n", metrics.MaxLatency)
	fmt.Printf("Indexing Latency: %v\n", metrics.IndexingLatency)

	fmt.Println("\n=== Example Complete ===")
}

func createSampleDocuments() []rag.Document {
	now := time.Now()
	return []rag.Document{
		{
			ID: "doc1",
			Content: `Transformer architecture revolutionized natural language processing by introducing
self-attention mechanisms. Unlike RNNs, transformers can process all tokens in parallel,
making them much faster to train. The transformer architecture consists of an encoder
and a decoder, each containing multiple layers of self-attention and feed-forward networks.
Key components include multi-head attention, positional encoding, and layer normalization.`,
			Metadata: map[string]any{
				"source": "transformers.txt",
				"topic":  "Transformers",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID: "doc2",
			Content: `Neural networks are computing systems inspired by biological neural networks.
They consist of interconnected nodes (neurons) organized in layers. Deep learning uses
neural networks with many layers (deep neural networks) to learn hierarchical representations
of data. Common architectures include Convolutional Neural Networks (CNNs) for images,
Recurrent Neural Networks (RNNs) for sequences, and Transformers for text.`,
			Metadata: map[string]any{
				"source": "neural_networks.txt",
				"topic":  "Neural Networks",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID: "doc3",
			Content: `Large Language Models (LLMs) like GPT-4, Claude, and LLaMA have demonstrated
remarkable capabilities in natural language understanding and generation. They are trained
on vast amounts of text data using transformer architectures. Key techniques include
pre-training on large corpora, fine-tuning for specific tasks, and reinforcement learning
from human feedback (RLHF). Applications range from chatbots to code generation.`,
			Metadata: map[string]any{
				"source": "llms.txt",
				"topic":  "LLMs",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID: "doc4",
			Content: `Machine learning is a subset of artificial intelligence that enables systems
to learn from data without being explicitly programmed. Main paradigms include supervised
learning (learning from labeled data), unsupervised learning (finding patterns in unlabeled
data), and reinforcement learning (learning through trial and error). Common algorithms
include linear regression, decision trees, support vector machines, and neural networks.`,
			Metadata: map[string]any{
				"source": "machine_learning.txt",
				"topic":  "Machine Learning",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID: "doc5",
			Content: `Attention mechanisms allow neural networks to focus on different parts of the
input when producing each part of the output. Self-attention, where each position in a
sequence attends to all other positions, is the key innovation behind transformers.
Multi-head attention allows the model to attend to different representation subspaces
simultaneously. This has become fundamental to modern NLP architectures.`,
			Metadata: map[string]any{
				"source": "attention.txt",
				"topic":  "Attention",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID: "doc6",
			Content: `Retrieval-Augmented Generation (RAG) combines retrieval systems with language models
to improve factual accuracy and reduce hallucinations. In a RAG system, a query is first
used to retrieve relevant documents from a knowledge base, then both the query and retrieved
documents are provided to the language model for generation. LightRAG is an implementation
that uses knowledge graphs for enhanced retrieval.`,
			Metadata: map[string]any{
				"source": "rag.txt",
				"topic":  "RAG",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID: "doc7",
			Content: `Fine-tuning adapts a pre-trained model to a specific task or domain. It involves
further training on a smaller, task-specific dataset. Techniques include full fine-tuning
(updating all parameters), partial fine-tuning (updating some layers), and parameter-efficient
methods like LoRA (Low-Rank Adaptation) and adapters. Fine-tuning can significantly improve
performance on specialized tasks compared to using a general pre-trained model.`,
			Metadata: map[string]any{
				"source": "fine_tuning.txt",
				"topic":  "Fine-tuning",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID: "doc8",
			Content: `Embeddings are dense vector representations of text that capture semantic meaning.
Similar texts have similar embeddings in the vector space. Models like BERT, GPT, and
sentence-transformers can generate embeddings. They are used for semantic search, clustering,
classification, and as input to other models. The dimension of embeddings ranges from
hundreds to thousands of floats.`,
			Metadata: map[string]any{
				"source": "embeddings.txt",
				"topic":  "Embeddings",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
