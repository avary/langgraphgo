package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/adapter"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/engine"
	"github.com/smallnest/langgraphgo/rag/store"
)

func main() {
	ctx := context.Background()

	// Initialize LLM (OpenAI in this example)
	ollm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Create adapter for our LLM interface
	llm := adapter.NewOpenAIAdapter(ollm)

	// Initialize embedder for entity extraction
	embedder := store.NewMockEmbedder(128)

	// FalkorDB connection string
	// Format: falkordb://host:port/graph_name
	// For local FalkorDB: falkordb://localhost:6379/rag_graph
	falkorDBConnStr := "falkordb://localhost:6379/rag_graph"

	// Create FalkorDB knowledge graph
	fmt.Println("Initializing FalkorDB knowledge graph...")
	kg, err := store.NewFalkorDBGraph(falkorDBConnStr)
	if err != nil {
		log.Fatalf("Failed to create FalkorDB knowledge graph: %v", err)
	}
	// Close the connection when done (type assert to access Close method)
	defer func() {
		if falkorDB, ok := kg.(*store.FalkorDBGraph); ok {
			falkorDB.Close()
		}
	}()

	// Configure GraphRAG engine
	graphRAGConfig := rag.GraphRAGConfig{
		ExtractionPrompt: `
Extract entities from the following text. Focus on these entity types: %s.
Return a JSON response with this structure:
{
  "entities": [
    {
      "name": "entity_name",
      "type": "entity_type",
      "description": "brief description",
      "properties": {}
    }
  ]
}

Text: %s`,
		EntityTypes: []string{
			"PERSON",
			"ORGANIZATION",
			"LOCATION",
			"PRODUCT",
			"TECHNOLOGY",
			"CONCEPT",
		},
		MaxDepth: 3,
	}

	// Create GraphRAG engine
	fmt.Println("Creating GraphRAG engine...")
	graphEngine, err := engine.NewGraphRAGEngine(graphRAGConfig, llm, embedder, kg)
	if err != nil {
		log.Fatalf("Failed to create GraphRAG engine: %v", err)
	}

	// Sample documents about technology companies and their products
	documents := []rag.Document{
		{
			ID: "doc1",
			Content: "Apple Inc. is a technology company headquartered in Cupertino, California. " +
				"The company was founded by Steve Jobs, Steve Wozniak, and Ronald Wayne in 1976. " +
				"Apple is known for its consumer electronics products including the iPhone, iPad, and Mac computers. " +
				"The iPhone is a smartphone that runs on iOS, Apple's mobile operating system.",
			Metadata: map[string]any{
				"source": "apple_overview.txt",
				"topic":  "Apple Inc.",
			},
		},
		{
			ID: "doc2",
			Content: "Microsoft Corporation is an American technology company based in Redmond, Washington. " +
				"Founded by Bill Gates and Paul Allen in 1975, Microsoft develops software, hardware, and cloud services. " +
				"The company's flagship products include the Windows operating system and Microsoft Office suite. " +
				"Microsoft Azure is their cloud computing platform that competes with Amazon Web Services.",
			Metadata: map[string]any{
				"source": "microsoft_overview.txt",
				"topic":  "Microsoft Corporation",
			},
		},
		{
			ID: "doc3",
			Content: "Google LLC is an American technology company and subsidiary of Alphabet Inc. " +
				"Founded by Larry Page and Sergey Brin in 1998, Google is headquartered in Mountain View, California. " +
				"The company is known for its search engine, Android mobile operating system, and web services. " +
				"Google Chrome is a popular web browser, and Google Cloud Platform (GCP) is their cloud computing service.",
			Metadata: map[string]any{
				"source": "google_overview.txt",
				"topic":  "Google LLC",
			},
		},
		{
			ID: "doc4",
			Content: "Tesla, Inc. is an American electric vehicle and clean energy company based in Palo Alto, California. " +
				"Founded by Elon Musk, Tesla designs and manufactures electric cars, battery energy storage, and solar products. " +
				"The Tesla Model S is an all-electric sedan, and the Model Y is a compact electric SUV. " +
				"Tesla also operates the Supercharger network for electric vehicle charging.",
			Metadata: map[string]any{
				"source": "tesla_overview.txt",
				"topic":  "Tesla, Inc.",
			},
		},
		{
			ID: "doc5",
			Content: "Amazon.com, Inc. is an American multinational technology company based in Seattle, Washington. " +
				"Founded by Jeff Bezos in 1994, Amazon started as an online bookstore but has expanded to e-commerce, " +
				"digital streaming, and artificial intelligence. Amazon Web Services (AWS) is the market leader in cloud computing. " +
				"The Amazon Kindle is a popular e-reader device.",
			Metadata: map[string]any{
				"source": "amazon_overview.txt",
				"topic":  "Amazon.com, Inc.",
			},
		},
	}

	// Add documents to the knowledge graph
	fmt.Println("Adding documents to knowledge graph...")
	fmt.Println("(This will extract entities and relationships from each document)")

	startTime := time.Now()
	for _, doc := range documents {
		fmt.Printf("Processing document: %s\n", doc.Metadata["topic"])
		err := graphEngine.AddDocuments(ctx, []rag.Document{doc})
		if err != nil {
			log.Printf("Failed to add document %s: %v", doc.ID, err)
			continue
		}
	}
	processingTime := time.Since(startTime)

	fmt.Printf("Knowledge graph construction completed in %v\n\n", processingTime)

	// Test queries to demonstrate graph-based retrieval
	queries := []string{
		"What products does Apple make?",
		"Who founded Microsoft and what are their main products?",
		"Tell me about electric vehicle companies and their founders",
		"What cloud computing services are available?",
		"Which technology companies are based in California?",
	}

	fmt.Println("=== GraphRAG Query Examples ===\n")

	for i, query := range queries {
		fmt.Printf("=== Query %d ===\n", i+1)
		fmt.Printf("Question: %s\n\n", query)

		// Perform GraphRAG query
		result, err := graphEngine.Query(ctx, query)
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		// Display graph context
		fmt.Println("Knowledge Graph Context:")
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println(result.Context)
		fmt.Println(strings.Repeat("-", 50))

		// Display retrieved documents/sources
		fmt.Printf("\nRetrieved %d sources:\n", len(result.Sources))
		for j, source := range result.Sources {
			fmt.Printf("  [%d] Source ID: %s\n", j+1, source.ID)
			if topic, ok := source.Metadata["topic"]; ok {
				fmt.Printf("      Topic: %v\n", topic)
			}
			fmt.Printf("      Content: %s\n\n", truncate(source.Content, 200))
		}

		// Display metadata
		fmt.Printf("Query Metadata:\n")
		fmt.Printf("  - Engine Type: %v\n", result.Metadata["engine_type"])
		fmt.Printf("  - Entities Found: %v\n", result.Metadata["entities_found"])
		fmt.Printf("  - Relationships: %v\n", result.Metadata["relationships"])
		fmt.Printf("  - Confidence: %.2f\n", result.Confidence)
		fmt.Printf("  - Response Time: %v\n", result.ResponseTime)

		fmt.Println("\n" + strings.Repeat("=", 80) + "\n")

		// Small delay between queries
		time.Sleep(500 * time.Millisecond)
	}

	// Demonstrate entity exploration
	fmt.Println("\n=== Entity Exploration Examples ===\n")

	// Explore entities related to Apple
	fmt.Println("1. Exploring entities related to 'Apple Inc.':")
	relatedEntities, err := kg.GetRelatedEntities(ctx, "Apple Inc.", 2)
	if err != nil {
		log.Printf("Failed to get related entities: %v", err)
	} else {
		for _, entity := range relatedEntities {
			fmt.Printf("  - %s (%s)\n", entity.Name, entity.Type)
		}
	}

	// Search for specific entity type
	fmt.Println("\n2. Searching for 'PERSON' entities:")
	graphQuery := &rag.GraphQuery{
		EntityTypes: []string{"PERSON"},
		Limit:       10,
	}
	queryResult, err := kg.Query(ctx, graphQuery)
	if err != nil {
		log.Printf("Failed to query graph: %v", err)
	} else {
		fmt.Printf("  Found %d PERSON entities:\n", len(queryResult.Entities))
		for _, entity := range queryResult.Entities {
			fmt.Printf("  - %s: %v\n", entity.Name, entity.Properties)
		}
	}

	// Display relationships
	fmt.Println("\n3. Relationships found in the graph:")
	graphQuery = &rag.GraphQuery{
		Limit: 20,
	}
	queryResult, err = kg.Query(ctx, graphQuery)
	if err != nil {
		log.Printf("Failed to query graph: %v", err)
	} else {
		fmt.Printf("  Found %d relationships:\n", len(queryResult.Relationships))
		for _, rel := range queryResult.Relationships {
			fmt.Printf("  - %s -> %s (%s)\n", rel.Source, rel.Target, rel.Type)
		}
	}

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("This example demonstrated:")
	fmt.Println("- Creating a knowledge graph with FalkorDB")
	fmt.Println("- Extracting entities and relationships from documents")
	fmt.Println("- Performing graph-based retrieval with GraphRAG")
	fmt.Println("- Exploring entities and relationships in the knowledge graph")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
