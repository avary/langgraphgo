package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tool"
)

func main() {
	// Check for API keys
	if os.Getenv("BRAVE_API_KEY") == "" {
		log.Fatal("Please set BRAVE_API_KEY environment variable")
	}
	// We also need an LLM API key (e.g., OPENAI_API_KEY or DEEPSEEK_API_KEY)
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
	braveTool, err := tool.NewBraveSearch("",
		tool.WithBraveCount(5),
		tool.WithBraveCountry("US"),
		tool.WithBraveLang("en"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Create the ReAct Agent using map state convenience function
	agent, err := prebuilt.CreateAgentMap(llm, []tool.Tool{braveTool}, 20)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// 4. Run the Agent
	query := "What are the latest developments in AI technology in 2025?"
	fmt.Printf("User: %s\n\n", query)
	fmt.Println("Agent is thinking and searching...")

	inputs := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, query),
		},
	}

	response, err := agent.Invoke(ctx, inputs)
	if err != nil {
		log.Fatalf("Agent failed: %v", err)
	}

	// 5. Print the Result
	messages, ok := response["messages"].([]llms.MessageContent)
	if ok {
		// The last message should be the AI's final answer
		lastMsg := messages[len(messages)-1]
		for _, part := range lastMsg.Parts {
			if text, ok := part.(llms.TextContent); ok {
				fmt.Printf("\nAgent: %s\n", text.Text)
			}
		}
	}
}
