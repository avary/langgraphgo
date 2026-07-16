package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tool"
)

// CalculatorTool (same as in react_agent example)
type CalculatorTool struct{}

func (t CalculatorTool) Name() string {
	return "calculator"
}

func (t CalculatorTool) Description() string {
	return "Useful for performing basic arithmetic operations. Input should be a string like '2 + 2' or '5 * 10'."
}

func (t CalculatorTool) Call(ctx context.Context, input string) (string, error) {
	parts := strings.Fields(input)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid input format")
	}
	a, _ := strconv.ParseFloat(parts[0], 64)
	b, _ := strconv.ParseFloat(parts[2], 64)
	op := parts[1]
	var result float64
	switch op {
	case "+":
		result = a + b
	case "-":
		result = a - b
	case "*":
		result = a * b
	case "/":
		result = a / b
	}
	return fmt.Sprintf("%f", result), nil
}

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

	// 1. Create Math Agent
	mathAgent, err := prebuilt.CreateAgentMap(model, []tool.Tool{CalculatorTool{}}, 20)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Create General Agent
	generalAgent, err := prebuilt.CreateAgentMap(model, []tool.Tool{}, 20)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Create Supervisor
	g := graph.NewStateGraph[map[string]any]()

	agentNode := func(agent *graph.StateRunnable[map[string]any]) func(context.Context, map[string]any) (map[string]any, error) {
		return func(ctx context.Context, state map[string]any) (map[string]any, error) {
			return agent.Invoke(ctx, state)
		}
	}

	g.AddNode("MathExpert", "Math Agent", agentNode(mathAgent))
	g.AddNode("GeneralAssistant", "General Agent", agentNode(generalAgent))

	supervisorNode := func(ctx context.Context, state map[string]any) (map[string]any, error) {
		messages := state["messages"].([]llms.MessageContent)

		systemPrompt := `You are a supervisor tasked with managing a conversation between the following workers:
- MathExpert
- GeneralAssistant

Given the conversation, decide who should act next.
If the task is finished, return FINISH.

Return only the name of the next actor or FINISH.`

		// Combine system prompt with history
		msgs := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		}
		msgs = append(msgs, messages...)

		resp, err := model.GenerateContent(ctx, msgs)
		if err != nil {
			return nil, err
		}

		choice := strings.TrimSpace(resp.Choices[0].Content)
		fmt.Printf("DEBUG: Supervisor choice raw: %q\n", choice)

		// Robust parsing
		choiceLower := strings.ToLower(choice)
		if strings.Contains(choiceLower, "mathexpert") {
			choice = "MathExpert"
		} else if strings.Contains(choiceLower, "generalassistant") {
			choice = "GeneralAssistant"
		} else if strings.Contains(choiceLower, "finish") {
			choice = "FINISH"
		} else {
			fmt.Printf("WARNING: Supervisor returned unknown choice: %s. Defaulting to FINISH.\n", choice)
			choice = "FINISH"
		}

		state["next"] = choice
		return state, nil
	}

	g.AddNode("Supervisor", "Supervisor", supervisorNode)

	g.SetEntryPoint("Supervisor")

	g.AddConditionalEdge("Supervisor", func(ctx context.Context, state map[string]any) string {
		next, _ := state["next"].(string)
		if next == "FINISH" {
			return graph.END
		}
		return next
	})

	g.AddEdge("MathExpert", "Supervisor")
	g.AddEdge("GeneralAssistant", "Supervisor")

	supervisor, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Execute
	query := "Calculate 10 * 5 and then tell me a joke about the result."
	fmt.Printf("User: %s\n", query)

	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, query),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := supervisor.Invoke(ctx, initialState)
	if err != nil {
		log.Fatal(err)
	}

	// Print Result
	messages := res["messages"].([]llms.MessageContent)

	fmt.Println("\n=== Conversation History ===")
	for _, msg := range messages {
		role := msg.Role
		var content string
		if len(msg.Parts) > 0 {
			if textPart, ok := msg.Parts[0].(llms.TextContent); ok {
				content = textPart.Text
			} else if _, ok := msg.Parts[0].(llms.ToolCall); ok {
				content = "[Tool Call]"
			} else if _, ok := msg.Parts[0].(llms.ToolCallResponse); ok {
				content = "[Tool Response]"
			}
		}
		fmt.Printf("%s: %s\n", role, content)
	}
}
