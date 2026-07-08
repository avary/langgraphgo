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
	"github.com/smallnest/langgraphgo/rag/store"
)

func main() {
	ctx := context.Background()

	// Initialize LLM
	ollm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Create adapter for our LLM interface
	llm := adapter.NewOpenAIAdapter(ollm)

	// Create FalkorDB knowledge graph
	fmt.Println("Initializing FalkorDB knowledge graph...")
	falkorDBConnStr := "falkordb://localhost:6379/fast_rag_graph"
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

	fmt.Println("Fast RAG with FalkorDB Knowledge Graph")
	fmt.Println("=====================================\n")

	// 示例1：手动添加预定义实体和关系（快速方式）
	fmt.Println("1. Adding predefined entities and relationships...")
	startTime := time.Now()

	// 手动定义实体（避免LLM调用）
	entities := []*rag.Entity{
		{
			ID:   "apple_inc",
			Name: "Apple Inc.",
			Type: "ORGANIZATION",
			Properties: map[string]any{
				"industry": "Technology",
				"founded":  "1976",
				"location": "Cupertino, California",
			},
		},
		{
			ID:   "steve_jobs",
			Name: "Steve Jobs",
			Type: "PERSON",
			Properties: map[string]any{
				"role":    "Co-founder",
				"company": "Apple Inc.",
			},
		},
		{
			ID:   "iphone",
			Name: "iPhone",
			Type: "PRODUCT",
			Properties: map[string]any{
				"category": "Smartphone",
				"company":  "Apple Inc.",
			},
		},
		{
			ID:   "microsoft",
			Name: "Microsoft",
			Type: "ORGANIZATION",
			Properties: map[string]any{
				"industry": "Technology",
				"founded":  "1975",
				"location": "Redmond, Washington",
			},
		},
		{
			ID:   "bill_gates",
			Name: "Bill Gates",
			Type: "PERSON",
			Properties: map[string]any{
				"role":    "Co-founder",
				"company": "Microsoft",
			},
		},
		{
			ID:   "windows",
			Name: "Windows",
			Type: "PRODUCT",
			Properties: map[string]any{
				"category": "Operating System",
				"company":  "Microsoft",
			},
		},
		{
			ID:   "machine_learning",
			Name: "Machine Learning",
			Type: "CONCEPT",
			Properties: map[string]any{
				"category":    "Artificial Intelligence",
				"description": "Subset of AI that enables computers to learn from data",
			},
		},
	}

	// 添加实体到知识图谱
	for _, entity := range entities {
		err := kg.AddEntity(ctx, entity)
		if err != nil {
			log.Printf("Failed to add entity %s: %v", entity.ID, err)
		}
	}

	// 手动定义关系
	relationships := []*rag.Relationship{
		{
			ID:     "steve_founded_apple",
			Source: "steve_jobs",
			Target: "apple_inc",
			Type:   "FOUNDED",
		},
		{
			ID:     "apple_makes_iphone",
			Source: "apple_inc",
			Target: "iphone",
			Type:   "PRODUCES",
		},
		{
			ID:     "bill_founded_microsoft",
			Source: "bill_gates",
			Target: "microsoft",
			Type:   "FOUNDED",
		},
		{
			ID:     "microsoft_makes_windows",
			Source: "microsoft",
			Target: "windows",
			Type:   "PRODUCES",
		},
		{
			ID:     "apple_vs_microsoft",
			Source: "apple_inc",
			Target: "microsoft",
			Type:   "COMPETES_WITH",
		},
		{
			ID:     "jobs_vs_gates",
			Source: "steve_jobs",
			Target: "bill_gates",
			Type:   "RIVALRY",
		},
		{
			ID:     "ml_used_by_tech",
			Source: "machine_learning",
			Target: "apple_inc",
			Type:   "USED_BY",
		},
		{
			ID:     "ml_used_by_microsoft",
			Source: "machine_learning",
			Target: "microsoft",
			Type:   "USED_BY",
		},
	}

	// 添加关系到知识图谱
	for _, rel := range relationships {
		err := kg.AddRelationship(ctx, rel)
		if err != nil {
			log.Printf("Failed to add relationship %s: %v", rel.ID, err)
		}
	}

	entityAddTime := time.Since(startTime)
	fmt.Printf("Added %d entities and %d relationships in %v\n\n", len(entities), len(relationships), entityAddTime)

	// 示例2：快速查询示例
	fmt.Println("2. Fast Query Examples")
	fmt.Println("=====================")

	queries := []struct {
		description string
		query       *rag.GraphQuery
	}{
		{
			description: "Find all organizations",
			query: &rag.GraphQuery{
				EntityTypes: []string{"ORGANIZATION"},
				Limit:       10,
			},
		},
		{
			description: "Find all people",
			query: &rag.GraphQuery{
				EntityTypes: []string{"PERSON"},
				Limit:       10,
			},
		},
		{
			description: "Find all products",
			query: &rag.GraphQuery{
				EntityTypes: []string{"PRODUCT"},
				Limit:       10,
			},
		},
		{
			description: "Find all entities (limit 5)",
			query: &rag.GraphQuery{
				Limit: 5,
			},
		},
	}

	for i, testQuery := range queries {
		fmt.Printf("Query %d: %s\n", i+1, testQuery.description)

		queryStart := time.Now()
		result, err := kg.Query(ctx, testQuery.query)
		queryTime := time.Since(queryStart)

		if err != nil {
			log.Printf("Query failed: %v", err)
			continue
		}

		fmt.Printf("  Found %d entities and %d relationships in %v\n",
			len(result.Entities), len(result.Relationships), queryTime)

		for j, entity := range result.Entities {
			fmt.Printf("    [%d] %s (%s)\n", j+1, entity.Name, entity.Type)
			if entity.Properties != nil {
				if industry, ok := entity.Properties["industry"]; ok {
					fmt.Printf("        Industry: %v\n", industry)
				}
				if role, ok := entity.Properties["role"]; ok {
					fmt.Printf("        Role: %v\n", role)
				}
			}
		}
		fmt.Println(strings.Repeat("-", 50))
	}

	// 示例3：关系遍历
	fmt.Println("\n3. Relationship Traversal Examples")
	fmt.Println("===================================")

	// 查找与Apple相关的实体
	fmt.Println("Entities related to Apple Inc.:")
	startTime = time.Now()
	relatedEntities, err := kg.GetRelatedEntities(ctx, "apple_inc", 2)
	traversalTime := time.Since(startTime)

	if err != nil {
		log.Printf("Failed to get related entities: %v", err)
	} else {
		fmt.Printf("Found %d related entities in %v:\n", len(relatedEntities), traversalTime)
		for i, entity := range relatedEntities {
			if entity.ID != "apple_inc" { // 排除Apple自己
				fmt.Printf("  [%d] %s (%s)\n", i, entity.Name, entity.Type)
			}
		}
	}

	// 示例4：简单的基于知识的问答
	fmt.Println("\n4. Knowledge-Based Q&A")
	fmt.Println("======================")

	questions := []string{
		"Who founded Apple Inc.?",
		"What does Microsoft produce?",
		"Which people are related to these companies?",
		"How is machine learning used in technology?",
	}

	for i, question := range questions {
		fmt.Printf("\nQuestion %d: %s\n", i+1, question)

		// 简单的基于图的关键词匹配
		answer := generateAnswerFromKnowledgeGraph(ctx, kg, question, llm)
		fmt.Printf("Answer: %s\n", answer)
	}

	fmt.Println("\n=== Fast RAG Demo Complete ===")
	fmt.Println("Performance summary:")
	fmt.Printf("- Entity addition: %v\n", entityAddTime)
	fmt.Println("- Queries completed quickly (no LLM calls for extraction)")
	fmt.Println("- Knowledge graph ready for RAG applications")
}

// generateAnswerFromKnowledgeGraph generates answers using knowledge graph without full RAG pipeline
func generateAnswerFromKnowledgeGraph(ctx context.Context, kg rag.KnowledgeGraph, question string, llm rag.LLMInterface) string {
	// 简单的关键词匹配来查找相关实体
	questionLower := strings.ToLower(question)

	var relevantEntities []*rag.Entity
	var relevantRelationships []*rag.Relationship

	// 查找相关的实体（简化版本）
	allEntitiesQuery := &rag.GraphQuery{
		Limit: 20,
	}

	result, err := kg.Query(ctx, allEntitiesQuery)
	if err != nil {
		return "I couldn't access the knowledge graph to answer your question."
	}

	// 简单的关键词匹配
	for _, entity := range result.Entities {
		if strings.Contains(questionLower, strings.ToLower(entity.Name)) ||
			strings.Contains(questionLower, strings.ToLower(entity.Type)) {
			relevantEntities = append(relevantEntities, entity)
		}
	}

	// 查找相关关系
	for _, rel := range result.Relationships {
		relLower := strings.ToLower(rel.Type)
		if strings.Contains(questionLower, relLower) {
			relevantRelationships = append(relevantRelationships, rel)
		}
	}

	// 构建上下文
	var context strings.Builder

	if len(relevantEntities) > 0 {
		context.WriteString("Relevant entities:\n")
		for _, entity := range relevantEntities {
			context.WriteString(fmt.Sprintf("- %s (%s): ", entity.Name, entity.Type))
			if entity.Properties != nil {
				for k, v := range entity.Properties {
					context.WriteString(fmt.Sprintf("%s=%v ", k, v))
				}
			}
			context.WriteString("\n")
		}
	}

	if len(relevantRelationships) > 0 {
		context.WriteString("\nRelevant relationships:\n")
		for _, rel := range relevantRelationships {
			context.WriteString(fmt.Sprintf("- %s %s %s\n", rel.Source, rel.Type, rel.Target))
		}
	}

	if context.Len() == 0 {
		// 如果没有找到直接相关信息，返回一般性回答
		if strings.Contains(questionLower, "apple") {
			return "Apple Inc. is a technology company founded by Steve Jobs. It produces products like the iPhone and competes with Microsoft."
		} else if strings.Contains(questionLower, "microsoft") {
			return "Microsoft is a technology company founded by Bill Gates. It produces the Windows operating system and competes with Apple."
		} else if strings.Contains(questionLower, "machine learning") {
			return "Machine learning is a concept used by technology companies like Apple and Microsoft for various applications."
		} else {
			return "I found some information in the knowledge graph, but need more specific details to answer your question accurately."
		}
	}

	// 使用LLM生成回答
	prompt := fmt.Sprintf(`Based on the following knowledge graph information, answer the question briefly and accurately.

Question: %s

Knowledge Graph Context:
%s

Answer:`, question, context.String())

	answer, err := llm.Generate(ctx, prompt)
	if err != nil {
		return "I encountered an error while generating an answer based on the knowledge graph."
	}

	return answer
}
