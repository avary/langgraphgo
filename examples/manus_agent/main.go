package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tool"
)

func main() {
	model, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Create work directory
	workDir := "./manus_work"
	os.MkdirAll(workDir, 0755)
	// Note: Work directory is preserved after execution for review

	// Define nodes for research workflow
	nodes := []graph.TypedNode[map[string]any]{
		{
			Name:        "research",
			Description: "Search for and gather information from external sources",
			Function:    researchNode,
		},
		{
			Name:        "compile",
			Description: "Compile findings into organized notes",
			Function:    compileNode,
		},
		{
			Name:        "write",
			Description: "Write final deliverable based on research",
			Function:    writeNode,
		},
		{
			Name:        "review",
			Description: "Review and validate the output",
			Function:    reviewNode,
		},
	}

	// Configure Manus Agent
	config := prebuilt.ManusConfig{
		WorkDir:    workDir,
		PlanPath:   filepath.Join(workDir, "task_plan.md"),
		NotesPath:  filepath.Join(workDir, "notes.md"),
		OutputPath: filepath.Join(workDir, "output.md"),
		AutoSave:   true,
		Verbose:    true,
	}

	// Create Manus Agent
	agent, err := prebuilt.CreateManusAgent(
		model,
		nodes,
		[]tool.Tool{},
		config,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example: Research TypeScript benefits
	fmt.Println("🚀 Manus Agent Example")
	fmt.Println("=====================")
	fmt.Println()
	fmt.Println("Task: Research TypeScript benefits and write a summary")
	fmt.Println()

	// Prepare initial state
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			{
				Role: llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{
					llms.TextPart("Research TypeScript benefits and write a comprehensive summary"),
				},
			},
		},
		"goal":   "Research and document the benefits of TypeScript for development teams",
		"errors": []string{},
	}

	// Execute the agent
	fmt.Println("⏳ Executing Manus Agent...")
	fmt.Println()

	startTime := time.Now()
	result, err := agent.Invoke(context.Background(), initialState)
	if err != nil {
		log.Printf("Agent execution failed: %v", err)
	}

	executionTime := time.Since(startTime)

	fmt.Println()
	fmt.Println("✅ Execution completed!")
	fmt.Printf("⏱️  Total time: %v\n", executionTime)
	fmt.Println()

	// Display results
	displayResults(workDir, result)
}

// researchNode simulates a research phase
func researchNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)

	fmt.Println("🔍 Phase: Research")
	fmt.Println("   - Searching for TypeScript documentation")
	fmt.Println("   - Analyzing community feedback")
	fmt.Println("   - Gathering statistical data")
	fmt.Println()

	// Simulate research delay
	time.Sleep(500 * time.Millisecond)

	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart("Research complete: Found 15 relevant sources\n\nKey findings:\n- Type safety prevents runtime errors\n- Better IDE support with autocomplete\n- Easier refactoring with type checking\n- Improved code documentation\n- Better team collaboration")},
	}

	state["messages"] = append(messages, msg)
	return state, nil
}

// compileNode compiles research findings
func compileNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)

	fmt.Println("📝 Phase: Compile Findings")
	fmt.Println("   - Organizing research data")
	fmt.Println("   - Extracting key points")
	fmt.Println("   - Creating structured notes")
	fmt.Println()

	// Simulate compilation delay
	time.Sleep(300 * time.Millisecond)

	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart("Findings compiled: Organized into 5 key benefit categories\n\n1. Type Safety\n2. Developer Experience\n3. Code Quality\n4. Team Productivity\n5. Long-term Maintainability")},
	}

	state["messages"] = append(messages, msg)
	return state, nil
}

// writeNode writes the final deliverable
func writeNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)

	fmt.Println("✍️  Phase: Write Summary")
	fmt.Println("   - Drafting introduction")
	fmt.Println("   - Writing body sections")
	fmt.Println("   - Creating conclusion")
	fmt.Println()

	// Simulate writing delay
	time.Sleep(700 * time.Millisecond)

	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart("Summary written: 2000 word comprehensive TypeScript benefits document\n\nStructure:\n- Introduction to TypeScript\n- Detailed Benefits (5 sections)\n- Real-world Examples\n- Conclusion")},
	}

	state["messages"] = append(messages, msg)
	return state, nil
}

// reviewNode validates the output
func reviewNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	messages := state["messages"].([]llms.MessageContent)

	fmt.Println("✅ Phase: Review")
	fmt.Println("   - Checking factual accuracy")
	fmt.Println("   - Validating structure")
	fmt.Println("   - Quality assessment")
	fmt.Println()

	// Simulate review delay
	time.Sleep(200 * time.Millisecond)

	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart("Review complete: Output validated successfully\n\nQuality Score: 9.5/10\n- All claims verified\n- Structure is logical\n- Examples are clear")},
	}

	state["messages"] = append(messages, msg)
	return state, nil
}

// displayResults shows the generated files
func displayResults(workDir string, result map[string]any) {
	fmt.Println("📁 Generated Files:")
	fmt.Println("==================")
	fmt.Println()

	// Display task_plan.md
	planPath := filepath.Join(workDir, "task_plan.md")
	if planContent, err := os.ReadFile(planPath); err == nil {
		fmt.Println("📄 task_plan.md:")
		fmt.Println("---------------")
		fmt.Println(string(planContent))
		fmt.Println()
	}

	// Display notes.md
	notesPath := filepath.Join(workDir, "notes.md")
	if notesContent, err := os.ReadFile(notesPath); err == nil {
		if len(notesContent) > 0 {
			fmt.Println("📝 notes.md:")
			fmt.Println("-----------")
			fmt.Println(string(notesContent))
			fmt.Println()
		}
	}

	// Display output.md
	outputPath := filepath.Join(workDir, "output.md")
	if outputContent, err := os.ReadFile(outputPath); err == nil {
		fmt.Println("📋 output.md:")
		fmt.Println("-----------")
		fmt.Println(string(outputContent))
		fmt.Println()
	}

	// Display execution status
	if status, ok := result["status"].(string); ok {
		fmt.Printf("📊 Final Status: %s\n", status)
	}

	if output, ok := result["output"].(string); ok && output != "" {
		fmt.Printf("📦 Output size: %d bytes\n", len(output))
	}

	fmt.Println()
	fmt.Println("💡 Next Steps:")
	fmt.Println("   - Review files in ./manus_work/")
	fmt.Println("   - Edit task_plan.md to adjust the workflow")
	fmt.Println("   - Run again to continue from updated plan")
	fmt.Println("   - Clean up with: rm -rf ./manus_work")
}
