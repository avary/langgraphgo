package prebuilt_test

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func ExampleCreateReflectionAgent() {
	// Create LLM
	model, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Configure reflection agent
	config := prebuilt.ReflectionAgentConfig{
		Model:         model,
		MaxIterations: 3,
		Verbose:       true,
		SystemMessage: "You are an expert technical writer. Create clear, accurate, and comprehensive responses.",
	}

	// Create agent
	agent, err := prebuilt.CreateReflectionAgent(config)
	if err != nil {
		log.Fatal(err)
	}

	// Prepare initial state
	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart("Explain the CAP theorem in distributed systems")},
			},
		},
	}

	// Invoke agent
	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}

	// Extract final response
	finalState := result.(map[string]interface{})
	messages := finalState["messages"].([]llms.MessageContent)

	fmt.Println("=== Final Response ===")
	for _, msg := range messages {
		if msg.Role == llms.ChatMessageTypeAI {
			for _, part := range msg.Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					fmt.Println(textPart.Text)
				}
			}
		}
	}
}

func ExampleCreateReflectionAgent_withSeparateReflector() {
	// Create generation model
	generationModel, err := openai.New(openai.WithModel("gpt-4"))
	if err != nil {
		log.Fatal(err)
	}

	// Create separate reflection model (could be a different model)
	reflectionModel, err := openai.New(openai.WithModel("gpt-4"))
	if err != nil {
		log.Fatal(err)
	}

	// Configure with separate models
	config := prebuilt.ReflectionAgentConfig{
		Model:           generationModel,
		ReflectionModel: reflectionModel,
		MaxIterations:   2,
		Verbose:         true,
		SystemMessage:   "You are a helpful assistant providing detailed explanations.",
		ReflectionPrompt: `You are a senior technical reviewer.
Evaluate the response for:
1. Technical accuracy
2. Completeness of explanation
3. Clarity for the target audience
4. Use of examples

Be specific in your feedback.`,
	}

	agent, err := prebuilt.CreateReflectionAgent(config)
	if err != nil {
		log.Fatal(err)
	}

	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart("What is a Merkle tree and how is it used in blockchain?")},
			},
		},
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}

	finalState := result.(map[string]interface{})
	draft := finalState["draft"].(string)
	iteration := finalState["iteration"].(int)

	fmt.Printf("Final draft (after %d iterations):\n%s\n", iteration, draft)
}

func ExampleCreateReflectionAgent_customCriteria() {
	model, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Custom reflection criteria for code quality
	config := prebuilt.ReflectionAgentConfig{
		Model:         model,
		MaxIterations: 2,
		Verbose:       true,
		SystemMessage: "You are a senior software engineer reviewing code.",
		ReflectionPrompt: `Evaluate the code review for:
1. **Security**: Are security issues identified?
2. **Performance**: Are performance concerns addressed?
3. **Maintainability**: Are code quality issues noted?
4. **Best Practices**: Are language/framework best practices mentioned?

Provide specific, actionable feedback.`,
	}

	agent, err := prebuilt.CreateReflectionAgent(config)
	if err != nil {
		log.Fatal(err)
	}

	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart("Review this SQL query function for issues")},
			},
		},
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatal(err)
	}

	finalState := result.(map[string]interface{})
	draft := finalState["draft"].(string)
	fmt.Printf("Code review:\n%s\n", draft)
}
