// LangGraph Go - Building Stateful, Multi-Agent Applications in Go
//
// LangGraph Go is a Go implementation of LangChain's LangGraph framework for building
// stateful, multi-agent applications with LLMs. It provides a powerful graph-based
// approach to constructing complex AI workflows with support for cycles, checkpoints,
// and human-in-the-loop interactions.
//
// # Quick Start
//
// Install the package:
//
//	go get github.com/smallnest/langgraphgo
//
// Basic example:
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//
//		"github.com/smallnest/langgraphgo/graph"
//		"github.com/smallnest/langgraphgo/prebuilt"
//		"github.com/tmc/langchaingo/llms/openai"
//		"github.com/tmc/langchaingo/tools"
//	)
//
//	func main() {
//		// Initialize LLM
//		llm, _ := openai.New()
//
//		// Create a simple ReAct agent
//		agent, _ := prebuilt.CreateReactAgent(
//			llm,
//			[]tools.Tool{&tools.CalculatorTool{}},
//			10, // max iterations
//		)
//
//		// Execute the agent
//		ctx := context.Background()
//		result, _ := agent.Invoke(ctx, map[string]any{
//			"messages": []llms.MessageContent{
//				{
//					Role: llms.ChatMessageTypeHuman,
//					Parts: []llms.ContentPart{
//						llms.TextPart("What is 123 * 456?"),
//					},
//				},
//			},
//		})
//
//		fmt.Println(result)
//	}
//
// # Key Features
//
//   - Stateful Graphs: Define complex workflows with state persistence
//   - Agent Orchestration: Build multi-agent systems with specialized roles
//   - Checkpointing: Save and resume execution state
//   - Streaming: Real-time event streaming during execution
//   - Memory Management: Various strategies for conversation memory
//   - Tool Integration: Extensive ecosystem of built-in tools
//   - Type Safety: Generic-based typed graphs for compile-time safety
//   - Visualization: Graph visualization and debugging tools
//
// # Core Concepts
//
// # Graph Structure
//
// LangGraph Go uses a directed graph structure where:
//   - Nodes represent processing units (agents, tools, functions)
//   - Edges define the flow of execution
//   - State flows through the graph and evolves at each node
//
// # State Management
//
// State can be managed in different ways:
//
//   - Untyped: Using map[string]any for flexibility
//   - Typed: Using Go generics for type safety
//   - Structured: Using predefined schemas
//
// # Package Structure
//
// # Core Packages
//
// graph/
// The core graph construction and execution engine
//
//	// Create a state graph
//	g := graph.NewStateGraph[map[string]any]()
//
//	// Add nodes
//	g.AddNode("process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
//		state["processed"] = true
//		return state, nil
//	})
//
//	// Define execution flow
//	g.SetEntry("process")
//	g.AddEdge("process", graph.END)
//
//	// Compile and run
//	runnable, _ := g.Compile()
//	result, _ := runnable.Invoke(ctx, initialState)
//
// prebuilt/
// Ready-to-use agent implementations
//
// Types of agents:
//   - ReAct Agent: Reason and act pattern
//   - Supervisor Agent: Orchestrates multiple agents
//   - Planning Agent: Creates and executes plans
//   - Reflection Agent: Self-correcting agent
//   - Tree of Thoughts: Multi-path reasoning
//
// Example:
//
//	// Create a supervisor with multiple agents
//	members := map[string]*graph.StateRunnableUntyped
//		"analyst":   analystAgent,
//		"coder":     coderAgent,
//		"reviewer":  reviewerAgent,
//	}
//
//	supervisor, _ := prebuilt.CreateSupervisor(llm, members, "router")
//
// ### memory/
// Various memory management strategies
//
// Types:
//   - Buffer: Simple FIFO buffer
//   - Sliding Window: Maintains recent context with overlap
//   - Summarization: Compresses older conversations
//   - Hierarchical: Multi-level memory with importance scoring
//   - OS-like: Sophisticated paging and eviction
//
// Example:
//
//	// Use summarization memory
//	memory := memory.NewSummarizationMemory(llm, 2000)
//	agent, _ := prebuilt.CreateChatAgent(llm, "", memory)
//
// ### tool/
// Collection of useful tools
//
// Categories:
//   - Web Search: Tavily, Brave, EXA, Bocha
//   - File Operations: Read, write, list files
//   - Code Execution: Shell, Python
//   - Web APIs: HTTP requests
//
// Example:
//
//	// Use web search tool
//	searchTool, _ := tool.NewTavilySearchTool(apiKey)
//	agent, _ := prebuilt.CreateReactAgent(llm, []tools.Tool{searchTool}, 10)
//
// # Storage Packages
//
// store/
// Checkpoint persistence implementations
//
// Options:
//   - SQLite: Lightweight, file-based storage
//   - PostgreSQL: Scalable relational database
//   - Redis: High-performance in-memory storage
//
// Example:
//
//	// PostgreSQL checkpoint store
//	store, _ := postgres.NewPostgresCheckpointStore(ctx, postgres.PostgresOptions{
//		ConnString: "postgres://user:pass@localhost/langgraph",
//	})
//
//	g.WithCheckpointing(graph.CheckpointConfig{Store: store})
//
// # Adapter Packages
//
// adapter/
// Integration adapters for external systems
//
// Adapters:
//   - GoSkills: Custom Go-based skills
//   - MCP: Model Context Protocol tools
//
// Example:
//
//	// Load GoSkills
//	skills, _ := goskills.LoadSkillsFromDir("./skills")
//	tools, _ := goskills.ConvertToLangChainTools(skills)
//
// # Specialized Packages
//
// ptc/
// Programmatic Tool Calling - agents generate code to use tools
//
//	agent, _ := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
//		Model:    llm,
//		Tools:    tools,
//		Language: ptc.LanguagePython,
//	})
//
// log/
// Simple logging utilities
//
//	logger := log.NewDefaultLogger(log.LogLevelInfo)
//	listener := graph.NewLoggingListener(logger, log.LogLevelInfo, false)
//
// # RAG (Retrieval-Augmented Generation) Package
//
// The rag package provides comprehensive RAG capabilities for LangGraph applications:
//
//	// Basic Vector RAG
//	llm, _ := openai.New()
//	embedder, _ := openai.NewEmbedder()
//	vectorStore, _ := pgvector.New(ctx, pgvector.WithEmbedder(embedder))
//
//	vectorRAG := rag.NewVectorRAG(llm, embedder, vectorStore, 5)
//	result, _ := vectorRAG.Query(ctx, "What is quantum computing?")
//
//	// GraphRAG with Knowledge Graph
//	graphRAG, _ := rag.NewGraphRAGEngine(rag.GraphRAGConfig{
//		DatabaseURL:     "falkordb://localhost:6379",
//		ModelProvider:   "openai",
//		EntityTypes:     []string{"PERSON", "ORGANIZATION", "LOCATION"},
//		EnableReasoning: true,
//	}, llm, embedder)
//
//	// Hybrid RAG combining vector and graph approaches
//	vectorRetriever := retrievers.NewVectorRetriever(vectorStore, embedder, config)
//	graphRetriever := retrievers.NewGraphRetriever(knowledgeGraph, embedder, config)
//	hybridRetriever := retrievers.NewHybridRetriever(
//		[]rag.Retriever{vectorRetriever, graphRetriever},
//		[]float64{0.6, 0.4}, config)
//
// # RAG Features
//
//   - Multiple Retrieval Strategies: Vector similarity, graph-based, hybrid
//   - Knowledge Graph Integration: Automatic entity and relationship extraction
//   - Document Processing: Various loaders and text splitters
//   - Flexible Storage: Support for multiple vector stores and graph databases
//   - LangGraph Integration: Seamless integration with agents and workflows
//
// # Advanced Examples
//
// 1. Multi-Agent System with Supervisor
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//
//		"github.com/smallnest/langgraphgo/prebuilt"
//		"github.com/tmc/langchaingo/llms/openai"
//		"github.com/tmc/langchaingo/tools"
//	)
//
//	func main() {
//		llm, _ := openai.New()
//
//		// Create specialized agents
//		researcher, _ := prebuilt.CreateReactAgent(llm, researchTools, 10)
//		writer, _ := prebuilt.CreateReactAgent(llm, writingTools, 10)
//		critic, _ := prebuilt.CreateReactAgent(llm, criticTools, 5)
//
//		// Create supervisor
//		members := map[string]*graph.StateRunnableUntyped
//			"researcher": researcher,
//			"writer":    writer,
//			"critic":    critic,
//		}
//
//		supervisor, _ := prebuilt.CreateSupervisor(llm, members, "router")
//
//		// Execute workflow
//		result, _ := supervisor.Invoke(ctx, map[string]any{
//			"messages": []llms.MessageContent{
//				{
//					Role: llms.ChatMessageTypeHuman,
//					Parts: []llms.ContentPart{
//						llms.TextPart("Write a research paper on quantum computing"),
//					},
//				},
//			},
//		})
//
//		fmt.Println(result)
//	}
//
// 2. RAG (Retrieval-Augmented Generation) System
//
//	package main
//
//	import (
//		"context"
//
//		"github.com/smallnest/langgraphgo/prebuilt"
//		"github.com/smallnest/langgraphgo/memory"
//		"github.com/smallnest/langgraphgo/store/postgres"
//		"github.com/tmc/langchaingo/embeddings/openai"
//		"github.com/tmc/langchaingo/llms"
//		"github.com/tmc/langchaingo/vectorstores/pgvector"
//	)
//
//	func main() {
//		// Initialize components
//		llm, _ := openai.NewChat(openai.GPT4)
//		embedder, _ := openai.NewEmbedder()
//
//		// Setup vector store
//		store, _ := pgvector.New(
//			ctx,
//			pgvector.WithConnString("postgres://localhost/postgres"),
//			pgvector.WithEmbedder(embedder),
//		)
//
//		// Create RAG agent
//		rag, _ := prebuilt.CreateRAGAgent(
//			llm,
//			documentLoader,
//			textSplitter,
//			embedder,
//			store,
//			5, // retrieve 5 documents
//		)
//
//		// Add memory for conversation
//		mem := memory.NewBufferMemory(100)
//		rag.WithMemory(mem)
//
//		// Enable checkpointing
//		checkpointStore, _ := postgres.NewPostgresCheckpointStore(ctx, postgres.PostgresOptions{
//			ConnString: "postgres://localhost/langgraph",
//		})
//		rag.WithCheckpointing(graph.CheckpointConfig{
//			Store: checkpointStore,
//		})
//
//		// Query the system
//		result, _ := rag.Invoke(ctx, map[string]any{
//			"messages": []llms.MessageContent{
//				{
//					Role: llms.ChatMessageTypeHuman,
//					Parts: []llms.ContentPart{
//						llms.TextPart("What are the latest developments in AI?"),
//					},
//				},
//			},
//		})
//	}
//
// 3. Typed Graph with Custom State
//
//	package main
//
//	import (
//		"context"
//		"fmt"
//
//		"github.com/smallnest/langgraphgo/graph"
//		"github.com/smallnest/langgraphgo/prebuilt"
//	)
//
//	type WorkflowState struct {
//		Input     string `json:"input"`
//	Processed  string `json:"processed"`
//	Validated  bool   `json:"validated"`
//	Output     string `json:"output"`
//	StepCount  int    `json:"step_count"`
//	}
//
//	func main() {
//		// Create typed graph
//		g := graph.NewStateGraph[WorkflowState]()
//
//		// Add typed nodes
//		g.AddNode("process", "Process the input", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
//			state.Processed = strings.ToUpper(state.Input)
//			state.StepCount++
//			return state, nil
//		})
//
//		g.AddNode("validate", "Validate the output", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
//			state.Validated = len(state.Processed) > 0
//			state.StepCount++
//			return state, nil
//		})
//
//		g.AddNode("output", "Generate final output", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
//			state.Output = fmt.Sprintf("Processed: %s, Validated: %v", state.Processed, state.Validated)
//			state.StepCount++
//			return state, nil
//		})
//
//		// Define flow
//		g.SetEntryPoint("process")
//		g.AddEdge("process", "validate")
//		g.AddConditionalEdge("validate", func(ctx context.Context, state WorkflowState) string {
//			if state.Validated {
//				return "output"
//			}
//			return "process" // Retry
//		})
//		g.AddEdge("output", graph.END)
//
//		// Compile and run
//		runnable, _ := g.Compile()
//
//		result, _ := runnable.Invoke(ctx, WorkflowState{
//			Input: "hello world",
//		})
//
//		fmt.Printf("Result: %+v\n", result)
//		fmt.Printf("Steps: %d\n", result.StepCount)
//	}
//
// # Best Practices
//
//  1. Choose the right agent type for your use case
//     - ReAct for general tasks
//     - Supervisor for multi-agent workflows
//     - PTC for complex tool interactions
//
//  2. Use typed graphs when possible for better type safety
//
//  3. Implement proper error handling in all node functions
//
//  4. Add checkpoints for long-running or critical workflows
//
//  5. Use appropriate memory strategy for conversations
//
//  6. Monitor execution with listeners and logging
//
//  7. Test thoroughly with various input scenarios
//
// # Configuration
//
// The library supports configuration through environment variables:
//
//   - OPENAI_API_KEY: OpenAI API key for LLM access
//   - LANGGRAPH_LOG_LEVEL: Logging level (debug, info, warn, error)
//   - LANGGRAPH_CHECKPOINT_DIR: Default directory for checkpoints
//   - LANGGRAPH_MAX_ITERATIONS: Default max iterations for agents
//
// # Community and Support
//
//   - GitHub: https://github.com/smallnest/langgraphgo
//   - Documentation: https://pkg.go.dev/github.com/smallnest/langgraphgo
//   - Examples: ./examples directory
//   - Issues: Report bugs and request features on GitHub
//
// # Contributing
//
// We welcome contributions! Please see:
//   - CONTRIBUTING.md for guidelines
//   - CODE_OF_CONDUCT.md for community standards
//   - Examples in ./examples for reference implementations
//
// # License
//
// This project is licensed under the MIT License - see the LICENSE file for details.
package langgraphgo // import "github.com/smallnest/langgraphgo"
