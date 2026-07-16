package prebuilt_test

import (
	"context"
	"fmt"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
)

// Example demonstrates how to use CreatePlanningAgentMap to build
// a dynamic workflow based on user requests
func Example_planningAgent() {
	// Step 1: Define your custom nodes that can be used in workflows
	nodes := []graph.TypedNode[map[string]any]{
		{
			Name:        "fetch_data",
			Description: "Fetch data from external API or database",
			Function: func(ctx context.Context, state map[string]any) (map[string]any, error) {
				messages := state["messages"].([]llms.MessageContent)

				// Simulate fetching data
				fmt.Println("Fetching data from API...")

				msg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Data fetched successfully: [item1, item2, item3]")},
				}

				return map[string]any{
					"messages": append(messages, msg),
				}, nil
			},
		},
		{
			Name:        "validate_data",
			Description: "Validate the integrity and format of data",
			Function: func(ctx context.Context, state map[string]any) (map[string]any, error) {
				messages := state["messages"].([]llms.MessageContent)

				// Simulate validation
				fmt.Println("Validating data...")

				msg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Data validation passed")},
				}

				return map[string]any{
					"messages": append(messages, msg),
				}, nil
			},
		},
		{
			Name:        "transform_data",
			Description: "Transform and normalize data into required format",
			Function: func(ctx context.Context, state map[string]any) (map[string]any, error) {
				messages := state["messages"].([]llms.MessageContent)

				// Simulate transformation
				fmt.Println("Transforming data...")

				msg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Data transformed to JSON format")},
				}

				return map[string]any{
					"messages": append(messages, msg),
				}, nil
			},
		},
		{
			Name:        "analyze_data",
			Description: "Perform statistical analysis on the data",
			Function: func(ctx context.Context, state map[string]any) (map[string]any, error) {
				messages := state["messages"].([]llms.MessageContent)

				// Simulate analysis
				fmt.Println("Analyzing data...")

				msg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Analysis complete: mean=42, median=40, std=5.2")},
				}

				return map[string]any{
					"messages": append(messages, msg),
				}, nil
			},
		},
		{
			Name:        "save_results",
			Description: "Save processed results to storage",
			Function: func(ctx context.Context, state map[string]any) (map[string]any, error) {
				messages := state["messages"].([]llms.MessageContent)

				// Simulate saving
				fmt.Println("Saving results...")

				msg := llms.MessageContent{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextPart("Results saved to database")},
				}

				return map[string]any{
					"messages": append(messages, msg),
				}, nil
			},
		},
	}

	// Step 2: Create your LLM model
	// In a real application, you would use an actual LLM like OpenAI, Anthropic, etc.
	// var model llms.Model = openai.New(...)

	// For this example, we'll skip the actual LLM call
	// The LLM would receive the user request and available nodes,
	// then generate a workflow plan like:
	// {
	//   "nodes": [
	//     {"name": "fetch_data", "type": "process"},
	//     {"name": "validate_data", "type": "process"},
	//     {"name": "transform_data", "type": "process"},
	//     {"name": "save_results", "type": "process"}
	//   ],
	//   "edges": [
	//     {"from": "START", "to": "fetch_data"},
	//     {"from": "fetch_data", "to": "validate_data"},
	//     {"from": "validate_data", "to": "transform_data"},
	//     {"from": "transform_data", "to": "save_results"},
	//     {"from": "save_results", "to": "END"}
	//   ]
	// }

	fmt.Println("CreatePlanningAgent example:")
	fmt.Println("This agent dynamically creates workflows based on user requests")
	fmt.Println()
	fmt.Println("Available nodes:")
	for i, node := range nodes {
		fmt.Printf("%d. %s: %s\n", i+1, node.Name, node.Description)
	}
	fmt.Println()
	fmt.Println("User request: 'Fetch data, validate it, transform it, and save the results'")
	fmt.Println()
	fmt.Println("The LLM will:")
	fmt.Println("1. Analyze the user request")
	fmt.Println("2. Select appropriate nodes from available nodes")
	fmt.Println("3. Generate a workflow plan (similar to a mermaid diagram)")
	fmt.Println("4. The agent will execute the planned workflow")
	fmt.Println()
	fmt.Println("Expected workflow:")
	fmt.Println("START -> fetch_data -> validate_data -> transform_data -> save_results -> END")

	// Output:
	// CreatePlanningAgent example:
	// This agent dynamically creates workflows based on user requests
	//
	// Available nodes:
	// 1. fetch_data: Fetch data from external API or database
	// 2. validate_data: Validate the integrity and format of data
	// 3. transform_data: Transform and normalize data into required format
	// 4. analyze_data: Perform statistical analysis on the data
	// 5. save_results: Save processed results to storage
	//
	// User request: 'Fetch data, validate it, transform it, and save the results'
	//
	// The LLM will:
	// 1. Analyze the user request
	// 2. Select appropriate nodes from available nodes
	// 3. Generate a workflow plan (similar to a mermaid diagram)
	// 4. The agent will execute the planned workflow
	//
	// Expected workflow:
	// START -> fetch_data -> validate_data -> transform_data -> save_results -> END
}

// Example showing how to use the planning agent with verbose mode
func Example_planningAgentWithVerbose() {
	// In a real application, you would define nodes and create the agent
	// nodes := []graph.TypedNode[map[string]any]{...}
	// agent, err := prebuilt.CreatePlanningAgentMap(model, nodes, []tools.Tool{}, prebuilt.WithVerbose(true))

	fmt.Println("With verbose mode enabled, you will see:")
	fmt.Println("🤔 Planning workflow...")
	fmt.Println("📋 Generated plan: {...}")
	fmt.Println("🚀 Executing planned workflow...")
	fmt.Println("  ✓ Added node: step1")
	fmt.Println("  ✓ Added node: step2")
	fmt.Println("  ✓ Added edge: step1 -> step2")
	fmt.Println("  ✓ Added edge: step2 -> END")
	fmt.Println("✅ Workflow execution completed")

	// Output:
	// With verbose mode enabled, you will see:
	// 🤔 Planning workflow...
	// 📋 Generated plan: {...}
	// 🚀 Executing planned workflow...
	//   ✓ Added node: step1
	//   ✓ Added node: step2
	//   ✓ Added edge: step1 -> step2
	//   ✓ Added edge: step2 -> END
	// ✅ Workflow execution completed
}

// Example showing real usage pattern
func Example_planningAgentRealUsage() {
	fmt.Println("Real usage pattern:")
	fmt.Println()
	fmt.Println("// 1. Define your nodes")
	fmt.Println("nodes := []graph.TypedNode[map[string]any]{...}")
	fmt.Println()
	fmt.Println("// 2. Initialize your LLM model")
	fmt.Println("model := openai.New()")
	fmt.Println()
	fmt.Println("// 3. Create the planning agent")
	fmt.Println("agent, err := prebuilt.CreatePlanningAgentMap(")
	fmt.Println("    model,")
	fmt.Println("    nodes,")
	fmt.Println("    []tools.Tool{},")
	fmt.Println("    prebuilt.WithVerbose(true),")
	fmt.Println(")")
	fmt.Println()
	fmt.Println("// 4. Prepare initial state with user request")
	fmt.Println("initialState := map[string]any{")
	fmt.Println("    \"messages\": []llms.MessageContent{")
	fmt.Println("        llms.TextParts(llms.ChatMessageTypeHuman,")
	fmt.Println("            \"Fetch, validate, and save the customer data\"),")
	fmt.Println("    },")
	fmt.Println("}")
	fmt.Println()
	fmt.Println("// 5. Execute the agent")
	fmt.Println("result, err := agent.Invoke(context.Background(), initialState)")
	fmt.Println()
	fmt.Println("// 6. Access results")
	fmt.Println("mState := result")
	fmt.Println("messages := mState[\"messages\"].([]llms.MessageContent)")

	// Output:
	// Real usage pattern:
	//
	// // 1. Define your nodes
	// nodes := []graph.TypedNode[map[string]any]{...}
	//
	// // 2. Initialize your LLM model
	// model := openai.New()
	//
	// // 3. Create the planning agent
	// agent, err := prebuilt.CreatePlanningAgentMap(
	//     model,
	//     nodes,
	//     []tools.Tool{},
	//     prebuilt.WithVerbose(true),
	// )
	//
	// // 4. Prepare initial state with user request
	// initialState := map[string]any{
	//     "messages": []llms.MessageContent{
	//         llms.TextParts(llms.ChatMessageTypeHuman,
	//             "Fetch, validate, and save the customer data"),
	//     },
	// }
	//
	// // 5. Execute the agent
	// result, err := agent.Invoke(context.Background(), initialState)
	//
	// // 6. Access results
	// mState := result
	// messages := mState["messages"].([]llms.MessageContent)
}

// Example showing how the LLM generates workflow plans
func Example_workflowPlanFormat() {
	fmt.Println("Workflow Plan JSON Format:")
	fmt.Println()
	fmt.Println("{")
	fmt.Println("  \"nodes\": [")
	fmt.Println("    {\"name\": \"node_name\", \"type\": \"process\"},")
	fmt.Println("    {\"name\": \"another_node\", \"type\": \"process\"}")
	fmt.Println("  ],")
	fmt.Println("  \"edges\": [")
	fmt.Println("    {\"from\": \"START\", \"to\": \"node_name\"},")
	fmt.Println("    {\"from\": \"node_name\", \"to\": \"another_node\"},")
	fmt.Println("    {\"from\": \"another_node\", \"to\": \"END\"}")
	fmt.Println("  ]")
	fmt.Println("}")
	fmt.Println()
	fmt.Println("Rules:")
	fmt.Println("1. Workflow must start with edge from 'START'")
	fmt.Println("2. Workflow must end with edge to 'END'")
	fmt.Println("3. Only use nodes from available nodes list")
	fmt.Println("4. Create logical flow based on user request")

	// Output:
	// Workflow Plan JSON Format:
	//
	// {
	//   "nodes": [
	//     {"name": "node_name", "type": "process"},
	//     {"name": "another_node", "type": "process"}
	//   ],
	//   "edges": [
	//     {"from": "START", "to": "node_name"},
	//     {"from": "node_name", "to": "another_node"},
	//     {"from": "another_node", "to": "END"}
	//   ]
	// }
	//
	// Rules:
	// 1. Workflow must start with edge from 'START'
	// 2. Workflow must end with edge to 'END'
	// 3. Only use nodes from available nodes list
	// 4. Create logical flow based on user request
}
