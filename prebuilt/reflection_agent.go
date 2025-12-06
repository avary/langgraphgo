package prebuilt

import (
	"context"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
)

// ReflectionAgentConfig configures the reflection agent
type ReflectionAgentConfig struct {
	// Model is the LLM to use for both generation and reflection
	Model llms.Model

	// ReflectionModel is an optional separate model for reflection
	// If nil, uses the same model as generation
	ReflectionModel llms.Model

	// MaxIterations is the maximum number of generation-reflection cycles
	MaxIterations int

	// SystemMessage is the system message for the generation step
	SystemMessage string

	// ReflectionPrompt is the system message for the reflection step
	ReflectionPrompt string

	// Verbose enables detailed logging
	Verbose bool
}

// CreateReflectionAgent creates a new Reflection Agent that iteratively
// improves its responses through self-reflection
//
// The Reflection pattern involves:
// 1. Generate: Create an initial response
// 2. Reflect: Critique the response and suggest improvements
// 3. Revise: Generate an improved version based on reflection
// 4. Repeat until satisfactory or max iterations reached
func CreateReflectionAgent(config ReflectionAgentConfig) (*graph.StateRunnable, error) {
	if config.Model == nil {
		return nil, fmt.Errorf("model is required")
	}

	if config.MaxIterations == 0 {
		config.MaxIterations = 3 // Default to 3 iterations
	}

	// Use same model for reflection if not specified
	reflectionModel := config.ReflectionModel
	if reflectionModel == nil {
		reflectionModel = config.Model
	}

	// Default system messages
	if config.SystemMessage == "" {
		config.SystemMessage = "You are a helpful assistant. Generate a high-quality response to the user's request."
	}

	if config.ReflectionPrompt == "" {
		config.ReflectionPrompt = buildDefaultReflectionPrompt()
	}

	// Create the workflow
	workflow := graph.NewStateGraph()

	// Define state schema
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("messages", graph.AppendReducer)
	agentSchema.RegisterReducer("iteration", graph.OverwriteReducer)
	agentSchema.RegisterReducer("reflection", graph.OverwriteReducer)
	agentSchema.RegisterReducer("draft", graph.OverwriteReducer)
	workflow.SetSchema(agentSchema)

	// Add generation node
	workflow.AddNode("generate", "Generate or revise response based on reflection", func(ctx context.Context, state interface{}) (interface{}, error) {
		return generateNode(ctx, state, config.Model, config.SystemMessage, config.Verbose)
	})

	// Add reflection node
	workflow.AddNode("reflect", "Reflect on the generated response and suggest improvements", func(ctx context.Context, state interface{}) (interface{}, error) {
		return reflectNode(ctx, state, reflectionModel, config.ReflectionPrompt, config.Verbose)
	})

	// Set entry point
	workflow.SetEntryPoint("generate")

	// Add conditional edge from generate
	workflow.AddConditionalEdge("generate", func(ctx context.Context, state interface{}) string {
		return shouldContinueAfterGenerate(state, config.MaxIterations, config.Verbose)
	})

	// Add conditional edge from reflect
	workflow.AddConditionalEdge("reflect", func(ctx context.Context, state interface{}) string {
		return shouldContinueAfterReflect(state, config.Verbose)
	})

	return workflow.Compile()
}

// generateNode generates or revises a response
func generateNode(ctx context.Context, state interface{}, model llms.Model, systemMessage string, verbose bool) (interface{}, error) {
	mState := state.(map[string]interface{})

	// Get current iteration
	iteration := 0
	if iter, ok := mState["iteration"].(int); ok {
		iteration = iter
	}

	// Get messages
	messages, ok := mState["messages"].([]llms.MessageContent)
	if !ok || len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in state")
	}

	// Build prompt based on iteration
	var promptMessages []llms.MessageContent

	if iteration == 0 {
		// First generation
		if verbose {
			fmt.Println("🎨 Generating initial response...")
		}

		promptMessages = []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeSystem,
				Parts: []llms.ContentPart{llms.TextPart(systemMessage)},
			},
		}
		promptMessages = append(promptMessages, messages...)
	} else {
		// Revision based on reflection
		reflection, ok := mState["reflection"].(string)
		if !ok || reflection == "" {
			return nil, fmt.Errorf("no reflection found for revision")
		}

		previousDraft, _ := mState["draft"].(string)

		if verbose {
			fmt.Printf("🔄 Revising response (iteration %d)...\n", iteration)
		}

		// Construct revision prompt
		revisionPrompt := fmt.Sprintf(`You are revising your previous response based on reflection.

Original request:
%s

Previous draft:
%s

Reflection and suggestions for improvement:
%s

Generate an improved response that addresses the issues raised in the reflection.`,
			getOriginalRequest(messages), previousDraft, reflection)

		promptMessages = []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeSystem,
				Parts: []llms.ContentPart{llms.TextPart(systemMessage)},
			},
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{llms.TextPart(revisionPrompt)},
			},
		}
	}

	// Generate response
	resp, err := model.GenerateContent(ctx, promptMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	draft := resp.Choices[0].Content

	if verbose {
		fmt.Printf("📝 Draft generated (%d chars)\n", len(draft))
	}

	// Create AI message
	aiMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart(draft)},
	}

	return map[string]interface{}{
		"messages":  []llms.MessageContent{aiMsg},
		"draft":     draft,
		"iteration": iteration + 1,
	}, nil
}

// reflectNode reflects on the generated response
func reflectNode(ctx context.Context, state interface{}, model llms.Model, reflectionPrompt string, verbose bool) (interface{}, error) {
	mState := state.(map[string]interface{})

	draft, ok := mState["draft"].(string)
	if !ok || draft == "" {
		return nil, fmt.Errorf("no draft found to reflect on")
	}

	messages, ok := mState["messages"].([]llms.MessageContent)
	if !ok || len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in state")
	}

	originalRequest := getOriginalRequest(messages)

	if verbose {
		fmt.Println("🤔 Reflecting on the response...")
	}

	// Build reflection prompt
	reflectionMessages := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart(reflectionPrompt)},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart(fmt.Sprintf(`Original request:
%s

Generated response:
%s

Provide a critical reflection on this response.`, originalRequest, draft)),
			},
		},
	}

	// Generate reflection
	resp, err := model.GenerateContent(ctx, reflectionMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate reflection: %w", err)
	}

	reflection := resp.Choices[0].Content

	if verbose {
		fmt.Printf("💭 Reflection:\n%s\n\n", reflection)
	}

	// Determine if response is satisfactory
	isSatisfactory := isResponseSatisfactory(reflection)

	return map[string]interface{}{
		"reflection":     reflection,
		"is_satisfactory": isSatisfactory,
	}, nil
}

// shouldContinueAfterGenerate decides whether to reflect or end
func shouldContinueAfterGenerate(state interface{}, maxIterations int, verbose bool) string {
	mState := state.(map[string]interface{})

	iteration, _ := mState["iteration"].(int)

	// On first iteration, always reflect
	if iteration == 1 {
		return "reflect"
	}

	// If we've reached max iterations, stop
	if iteration >= maxIterations {
		if verbose {
			fmt.Println("✅ Max iterations reached, finalizing response")
		}
		return graph.END
	}

	// Check if previous reflection was satisfactory
	isSatisfactory, _ := mState["is_satisfactory"].(bool)
	if isSatisfactory {
		if verbose {
			fmt.Println("✅ Response is satisfactory, finalizing")
		}
		return graph.END
	}

	// Continue reflecting
	return "reflect"
}

// shouldContinueAfterReflect decides whether to revise or accept
func shouldContinueAfterReflect(state interface{}, verbose bool) string {
	mState := state.(map[string]interface{})

	isSatisfactory, _ := mState["is_satisfactory"].(bool)

	if isSatisfactory {
		if verbose {
			fmt.Println("✅ Reflection indicates response is satisfactory")
		}
		return graph.END
	}

	// Revise based on reflection
	return "generate"
}

// isResponseSatisfactory analyzes the reflection to determine if response is good
func isResponseSatisfactory(reflection string) bool {
	reflectionLower := strings.ToLower(reflection)

	// Keywords indicating satisfaction
	satisfactoryKeywords := []string{
		"excellent",
		"satisfactory",
		"no major issues",
		"no significant issues",
		"well done",
		"comprehensive",
		"thorough",
		"accurate",
		"no improvements needed",
		"meets all requirements",
	}

	// Keywords indicating issues
	issueKeywords := []string{
		"missing",
		"incomplete",
		"unclear",
		"should include",
		"could be improved",
		"lacks",
		"needs to",
		"issue",
		"problem",
		"incorrect",
		"inaccurate",
	}

	satisfactoryCount := 0
	issueCount := 0

	for _, keyword := range satisfactoryKeywords {
		if strings.Contains(reflectionLower, keyword) {
			satisfactoryCount++
		}
	}

	for _, keyword := range issueKeywords {
		if strings.Contains(reflectionLower, keyword) {
			issueCount++
		}
	}

	// If we found satisfactory indicators and no issues, it's good
	if satisfactoryCount > 0 && issueCount == 0 {
		return true
	}

	// If we found more issues than satisfactory indicators, needs improvement
	if issueCount > satisfactoryCount {
		return false
	}

	// Default: continue improving
	return false
}

// getOriginalRequest extracts the original user request from messages
func getOriginalRequest(messages []llms.MessageContent) string {
	for _, msg := range messages {
		if msg.Role == llms.ChatMessageTypeHuman {
			for _, part := range msg.Parts {
				if textPart, ok := part.(llms.TextContent); ok {
					return textPart.Text
				}
			}
		}
	}
	return ""
}

// buildDefaultReflectionPrompt creates the default reflection prompt
func buildDefaultReflectionPrompt() string {
	return `You are a critical reviewer providing constructive feedback on AI-generated responses.

Your task is to evaluate the response and provide:
1. Strengths: What the response does well
2. Weaknesses: What could be improved
3. Specific suggestions: Concrete ways to enhance the response

Evaluation criteria:
- Accuracy: Is the information correct and factual?
- Completeness: Does it fully address the request?
- Clarity: Is it well-organized and easy to understand?
- Relevance: Does it stay focused on the topic?
- Quality: Is the writing clear and professional?

Format your reflection as:
**Strengths:**
[List strengths]

**Weaknesses:**
[List weaknesses or write "No major issues"]

**Suggestions for improvement:**
[Specific actionable suggestions or write "No improvements needed"]

Be honest but constructive. If the response is excellent, say so clearly.`
}
