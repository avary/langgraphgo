package main

import (
	"context"
	"fmt"
	"log"
	"os"

	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tool"
	"github.com/smallnest/langgraphgo/tooltypes"
)

func main() {
	// Check for API keys
	if os.Getenv("EXA_API_KEY") == "" {
		log.Fatal("Please set EXA_API_KEY environment variable")
	}
	if os.Getenv("OPENAI_API_KEY") == "" && os.Getenv("DEEPSEEK_API_KEY") == "" {
		log.Fatal("Please set OPENAI_API_KEY or DEEPSEEK_API_KEY environment variable")
	}

	ctx := context.Background()

	// 1. Initialize the LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// 2. Initialize the Tool
	exaTool, err := tool.NewExaSearch("",
		tool.WithExaNumResults(5),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Create the ReAct Agent using map state convenience function
	agent, err := prebuilt.CreateAgentMap(llm, []tooltypes.Tool{exaTool}, 20)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// 4. Run the Agent
	query := "Latest news about SpaceX Starship in 2025"
	fmt.Printf("User: %s\n\n", query)

	inputs := map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, query),
		},
	}

	response, err := agent.Invoke(ctx, inputs)
	if err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	// 5. Print the Result
	messages, ok := response["messages"].([]llmtypes.MessageContent)
	if ok {
		lastMsg := messages[len(messages)-1]
		for _, part := range lastMsg.Parts {
			if text, ok := part.(llmtypes.TextContent); ok {
				fmt.Printf("\nAgent: %s\n", text.Text)
			}
		}
	}
}
