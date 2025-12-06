package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	// Check for API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create LLM
	model, err := openai.New(openai.WithModel("gpt-4"))
	if err != nil {
		log.Fatalf("Failed to create OpenAI client: %v", err)
	}

	fmt.Println("=== Reflection Agent Example ===\n")

	// Example 1: Basic Reflection
	fmt.Println("--- Example 1: Basic Reflection ---")
	runBasicReflection(model)

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Example 2: Technical Writing with Custom Prompts
	fmt.Println("--- Example 2: Technical Writing with Custom Prompts ---")
	runTechnicalWriting(model)

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Example 3: Code Review Reflection
	fmt.Println("--- Example 3: Code Review Reflection ---")
	runCodeReview(model)
}

func runBasicReflection(model llms.Model) {
	config := prebuilt.ReflectionAgentConfig{
		Model:         model,
		MaxIterations: 3,
		Verbose:       true,
	}

	agent, err := prebuilt.CreateReflectionAgent(config)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	query := "Explain the CAP theorem in distributed systems"
	fmt.Printf("Query: %s\n\n", query)

	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart(query)},
			},
		},
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatalf("Failed to invoke agent: %v", err)
	}

	printResults(result)
}

func runTechnicalWriting(model llms.Model) {
	config := prebuilt.ReflectionAgentConfig{
		Model:         model,
		MaxIterations: 2,
		Verbose:       true,
		SystemMessage: "You are an expert technical writer. Create clear, accurate, and comprehensive documentation.",
		ReflectionPrompt: `You are a senior technical editor reviewing documentation.

Evaluate the documentation for:
1. **Clarity**: Is it easy to understand for the target audience?
2. **Completeness**: Does it cover all necessary aspects?
3. **Examples**: Are there practical examples to illustrate concepts?
4. **Structure**: Is the information well-organized?
5. **Accuracy**: Is the technical information correct?

Provide specific, actionable feedback.`,
	}

	agent, err := prebuilt.CreateReflectionAgent(config)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	query := "Write documentation for a REST API endpoint that creates a new user account"
	fmt.Printf("Query: %s\n\n", query)

	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart(query)},
			},
		},
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatalf("Failed to invoke agent: %v", err)
	}

	printResults(result)
}

func runCodeReview(model llms.Model) {
	config := prebuilt.ReflectionAgentConfig{
		Model:         model,
		MaxIterations: 2,
		Verbose:       true,
		SystemMessage: "You are an experienced software engineer providing code review feedback.",
		ReflectionPrompt: `You are a principal engineer reviewing code review comments.

Evaluate the review for:
1. **Constructiveness**: Is the feedback helpful and actionable?
2. **Completeness**: Are all important issues identified?
3. **Balance**: Does it acknowledge both strengths and weaknesses?
4. **Specificity**: Are suggestions concrete and clear?
5. **Tone**: Is the feedback professional and respectful?

Provide recommendations for improvement.`,
	}

	agent, err := prebuilt.CreateReflectionAgent(config)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	codeSnippet := `
func getUserById(id int) (*User, error) {
    var user User
    err := db.QueryRow("SELECT * FROM users WHERE id = " + strconv.Itoa(id)).Scan(&user)
    return &user, err
}
`

	query := fmt.Sprintf("Review this Go function and provide feedback:\n%s", codeSnippet)
	fmt.Printf("Query: Code review for getUserById function\n\n")

	initialState := map[string]interface{}{
		"messages": []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart(query)},
			},
		},
	}

	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Fatalf("Failed to invoke agent: %v", err)
	}

	printResults(result)
}

func printResults(result interface{}) {
	finalState := result.(map[string]interface{})

	// Print iteration count
	iteration, _ := finalState["iteration"].(int)
	fmt.Printf("\n✅ Completed after %d iteration(s)\n\n", iteration)

	// Print final draft
	if draft, ok := finalState["draft"].(string); ok {
		fmt.Println("=== Final Response ===")
		fmt.Println(draft)
	}

	// Print last reflection (if available)
	if reflection, ok := finalState["reflection"].(string); ok {
		fmt.Println("\n=== Final Reflection ===")
		fmt.Println(reflection)
	}
}
