package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tool"
)

func main() {
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	// Initialize LLM
	opts := []openai.Option{}
	if base := os.Getenv("OPENAI_API_BASE"); base != "" {
		opts = append(opts, openai.WithBaseURL(base))
	}
	if modelName := os.Getenv("OPENAI_MODEL"); modelName != "" {
		opts = append(opts, openai.WithModel(modelName))
	}

	model, err := openai.New(opts...)
	if err != nil {
		log.Fatal(err)
	}

	// Define custom nodes that can be used in the workflow
	nodes := []graph.TypedNode[map[string]any]{
		{
			Name:        "fetch_data",
			Description: "Fetch user data from the database",
			Function:    fetchDataNode,
		},
		{
			Name:        "validate_data",
			Description: "Validate the integrity and format of the data",
			Function:    validateDataNode,
		},
		{
			Name:        "transform_data",
			Description: "Transform and normalize the data into JSON format",
			Function:    transformDataNode,
		},
		{
			Name:        "analyze_data",
			Description: "Perform statistical analysis on the data",
			Function:    analyzeDataNode,
		},
		{
			Name:        "save_results",
			Description: "Save the processed results to storage",
			Function:    saveResultsNode,
		},
		{
			Name:        "generate_report",
			Description: "Generate a summary report from the analysis",
			Function:    generateReportNode,
		},
	}

	// Create Planning Agent with verbose output
	agent, err := prebuilt.CreatePlanningAgentMap(
		model,
		nodes,
		[]tool.Tool{},
		prebuilt.WithVerbose(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 1: Data processing workflow
	fmt.Println("=== Example 1: Data Processing Workflow ===")
	query1 := "Fetch user data, validate it, transform it to JSON, and save the results"
	runAgent(agent, query1)

	fmt.Println("\n=== Example 2: Data Analysis Workflow ===")
	query2 := "Fetch data, analyze it, and generate a report"
	runAgent(agent, query2)

	fmt.Println("\n=== Example 3: Complete Pipeline ===")
	query3 := "Fetch data, validate and transform it, analyze the results, and generate a comprehensive report"
	runAgent(agent, query3)
}

func runAgent(agent *graph.StateRunnable[map[string]any], query string) {
	fmt.Printf("\nUser Query: %s\n\n", query)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, query),
		},
	}

	res, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	// Print final result
	messages := res["messages"].([]llms.MessageContent)

	fmt.Println("\n--- Execution Result ---")
	for i, msg := range messages {
		if msg.Role == llms.ChatMessageTypeHuman {
			continue // Skip user message
		}
		for _, part := range msg.Parts {
			if textPart, ok := part.(llms.TextContent); ok {
				fmt.Printf("Step %d: %s\n", i, textPart.Text)
			}
		}
	}
	fmt.Println("------------------------")
}

// Node implementations

func fetchDataNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)

	fmt.Println("📥 Fetching data from database...")

	// Simulate data fetching
	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart("Data fetched: 1000 user records retrieved")},
	}

	state["messages"] = append(messages, msg)
	return state, nil
}

func validateDataNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)

	fmt.Println("✅ Validating data...")

	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart("Data validation passed: all records valid")},
	}

	state["messages"] = append(messages, msg)
	return state, nil
}

func transformDataNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)

	fmt.Println("🔄 Transforming data...")

	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart("Data transformed to JSON format successfully")},
	}

	state["messages"] = append(messages, msg)
	return state, nil
}

func analyzeDataNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)

	fmt.Println("📊 Analyzing data...")

	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart("Analysis complete: avg_age=32.5, total_users=1000, active_rate=78%")},
	}

	state["messages"] = append(messages, msg)
	return state, nil
}

func saveResultsNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)

	fmt.Println("💾 Saving results...")

	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart("Results saved to database successfully")},
	}

	state["messages"] = append(messages, msg)
	return state, nil
}

func generateReportNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)

	fmt.Println("📝 Generating report...")

	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart("Report generated: summary.pdf created with all analysis results")},
	}

	state["messages"] = append(messages, msg)
	return state, nil
}
