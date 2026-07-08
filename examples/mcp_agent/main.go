package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/smallnest/langgraphgo/adapter/mcp"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/prebuilt"
)

func main() {
	ctx := context.Background()

	// 1. Create MCP client from Claude's config file
	projectRoot := ".."
	configPath := filepath.Join(projectRoot, "testdata", "mcp", "mcp.json")

	mcpClient, err := mcp.NewClientFromConfig(ctx, configPath)
	if err != nil {
		log.Fatalf("Failed to create MCP client: %v\n", err)
	}
	defer mcpClient.Close()

	// 2. Convert MCP tools to langchaingo tools

	tools, err := mcp.MCPToTools(ctx, mcpClient)
	if err != nil {
		log.Fatalf("Failed to get MCP tools: %v\n", err)
	}

	fmt.Printf("Loaded %d MCP tools:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name(), tool.Description())
	}

	// 3. Create OpenAI LLM
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v\n", err)
	}

	// 4. Create agent with MCP tools using CreateAgentMap
	agent, err := prebuilt.CreateAgentMap(
		llm,
		tools,
		0,
		prebuilt.WithSystemMessage("You are a helpful assistant with access to various tools through MCP. Use them to help answer questions."),
	)
	if err != nil {
		log.Fatalf("Failed to create agent: %v\n", err)
	}

	// 5. Test the agent with a query
	query := "What files are in the current directory?"
	fmt.Printf("\nQuery: %s\n", query)

	// Prepare initial state with messages
	initialState := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, query),
		},
	}

	result, err := agent.Invoke(ctx, initialState)
	if err != nil {
		log.Fatalf("Failed to invoke agent: %v\n", err)
	}

	// 6. Print the result
	// Result is map[string]any
	if messages, ok := result["messages"]; ok {
		fmt.Printf("\nAgent messages:\n%+v\n", messages)
	} else {
		fmt.Printf("\nAgent result:\n%+v\n", result)
	}
}
