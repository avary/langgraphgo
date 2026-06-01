// Package graph provides the core graph construction and execution engine for LangGraph Go.
//
// This package implements the fundamental building blocks for creating stateful, multi-agent applications
// using directed graphs. It offers both untyped and typed interfaces for building workflows,
// with support for parallel execution, checkpointing, streaming, and comprehensive event handling.
//
// # Core Concepts
//
// StateGraph
// The primary component for building graphs is StateGraph, which maintains state as it flows
// through nodes. Each node can process and transform the state before passing it to the next node
// based on defined edges.
//
// Nodes and Edges
// Nodes represent processing units (functions, agents, tools) that transform state.
// Edges define the flow between nodes, supporting conditional routing based on state content.
//
// Typed Support
// For type safety, the package provides StateGraph[S] which uses Go generics to enforce
// state types at compile time, reducing runtime errors and improving code maintainability.
//
// # Key Features
//
//   - Parallel node execution with coordination
//   - Checkpointing for durable execution with resume capability
//   - Streaming for real-time event monitoring
//   - Comprehensive listener system for observability
//   - Built-in retry mechanisms with configurable policies
//   - Subgraph composition for modular design
//   - Graph visualization (Mermaid, ASCII, DOT)
//   - Interrupt support for human-in-the-loop workflows
//
// # Example Usage
//
// Basic State Graph
//
//	g := graph.NewStateGraph[map[string]any]()
//
//	// Add nodes
//	g.AddNode("process", "Process node", func(ctx context.Context, state any) (any, error) {
//		// Process the state
//		s := state.(map[string]any)
//		s["processed"] = true
//		return s, nil
//	})
//
//	g.AddNode("validate", "Validate node", func(ctx context.Context, state any) (any, error) {
//		// Validate the processed state
//		s := state.(map[string]any)
//		if s["processed"].(bool) {
//			s["valid"] = true
//		}
//		return s, nil
//	})
//
//	// Set entry point and edges
//	g.SetEntryPoint("process")
//	g.AddEdge("process", "validate")
//	g.AddEdge("validate", graph.END)
//
//	// Compile and run
//	runnable := g.Compile()
//	result, err := runnable.Invoke(context.Background(), map[string]any{
//		"data": "example",
//	})
//
// Typed State Graph
//
//	type WorkflowState struct {
//		Input    string `json:"input"`
//		Output   string `json:"output"`
//		Complete bool   `json:"complete"`
//	}
//
//	g := graph.NewStateGraph[WorkflowState]()
//
//	g.AddNode("process", "Process the input", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
//		state.Output = strings.ToUpper(state.Input)
//		state.Complete = true
//		return state, nil
//	})
//
//	// Add validate node
//	g.AddNode("validate", "Validate the output", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
//		return state, nil
//	})
//	g.AddNode("retry", "Retry processing", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
//		return state, nil
//	})
//
//	// Conditional routing
//	g.AddConditionalEdge("process", func(ctx context.Context, state WorkflowState) string {
//		if state.Complete {
//			return "validate"
//		}
//		return "retry"
//	})
//	g.AddEdge("validate", graph.END)
//	g.AddEdge("retry", "process")
//
// Parallel Execution
//
//	// Add parallel nodes
//	g.AddParallelNodes("parallel_tasks", map[string]func(context.Context, any) (any, error){
//		"task1": func(ctx context.Context, state any) (any, error) {
//			// First task logic
//			return state, nil
//		},
//		"task2": func(ctx context.Context, state any) (any, error) {
//			// Second task logic
//			return state, nil
//		},
//	})
//
// Checkpointing
//
//	// Note: Checkpointing is handled at the runnable level
//	// See store package examples for checkpointing implementation
//
//	runnable := g.Compile()
//
//	// Execute with context
//	result, err := runnable.Invoke(context.Background(), initialState)
//
// Streaming
//
//	// Create listenable graph for streaming
//	g := graph.NewListenableStateGraph()
//	g.AddNode("process", "Process node", func(ctx context.Context, state map[string]any) (map[string]any, error) {
//		state["processed"] = true
//		return state, nil
//	})
//	g.SetEntryPoint("process")
//	g.AddEdge("process", graph.END)
//
//	// Compile to listenable runnable
//	runnable, _ := g.CompileListenable()
//
//	// Create streaming runnable
//	streaming := graph.NewStreamingRunnableWithDefaults(runnable)
//
//	// Stream execution
//	result := streaming.Stream(context.Background(), initialState)
//
//	// Process events
//	for event := range result.Events {
//		fmt.Printf("Event: %v\n", event)
//	}
//
// # Listener System
//
// The package provides a powerful listener system for monitoring and reacting to graph events:
//
//   - ProgressListener: Track execution progress
//   - LoggingListener: Structured logging of events
//   - MetricsListener: Collect performance metrics
//   - ChatListener: Chat-style output formatting
//   - Custom listeners: Implement NodeListener interface
//
// Error Handling
//
//   - Built-in retry policies with exponential backoff
//   - Custom error filtering for selective retries
//   - Interrupt handling for pausing execution
//   - Comprehensive error context in events
//
// # Visualization
//
// Export graphs for documentation and debugging:
//
//	exporter := graph.NewExporter(g)
//
//	// Mermaid diagram
//	mermaid := exporter.DrawMermaid()
//
//	// Mermaid with options
//	mermaidWithOptions := exporter.DrawMermaidWithOptions(graph.MermaidOptions{
//		Direction: "LR", // Left to right
//	})
//
// # Thread Safety
//
// All graph structures are thread-safe for read operations. Write operations (adding nodes,
// edges, or listeners) should be performed before compilation or protected by external synchronization.
//
// Best Practices
//
//  1. Use typed graphs when possible for better type safety
//  2. Set appropriate buffer sizes for streaming to balance memory and performance
//  3. Implement proper error handling in node functions
//  4. Use checkpoints for long-running or critical workflows
//  5. Add listeners for debugging and monitoring
//  6. Keep node functions pure and stateless when possible
//  7. Use conditional edges for complex routing logic
//  8. Leverage parallel execution for independent tasks
package graph
