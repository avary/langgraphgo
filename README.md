# LangGraphGo
![](https://github.com/smallnest/lango-website/blob/master/images/logo/lango5.png)

> Abbreviated as `lango`, 中文: `懒狗`

[![License](https://img.shields.io/:license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/smallnest/langgraphgo) [![github actions](https://github.com/smallnest/langgraphgo/actions/workflows/go.yaml/badge.svg)](https://github.com/smallnest/langgraphgo/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/smallnest/langgraphgo)](https://goreportcard.com/report/github.com/smallnest/langgraphgo) [![Coverage Status](https://coveralls.io/repos/github/smallnest/langgraphgo/badge.svg?branch=master)](https://coveralls.io/github/smallnest/langgraphgo?branch=master)

[English](./README.md) | [简体中文](./README_CN.md)

Website: [http://lango.rpcx.io](http://lango.rpcx.io)


> 🔀 **Forked from [paulnegz/langgraphgo](https://github.com/paulnegz/langgraphgo)** - Enhanced with streaming, visualization, observability, and production-ready features.
>
> This fork aims for **feature parity with the Python LangGraph library**, adding support for parallel execution, persistence, advanced state management, pre-built agents, and human-in-the-loop workflows.

## Test coverage

![](coverage.svg)

## 🌐 Websites Built with LangGraphGo

Real-world applications built with LangGraphGo:

| [Insight](https://insight.rpcx.io) | [NoteX](https://notex.rpcx.io) |
| :--------------------------------: | :----------------------------: |
|       ![](docs/insight.png)        |      ![](docs/notex.png)       |

**Insight** - An AI-powered knowledge management and insight generation platform that uses LangGraphGo to build intelligent analysis workflows, helping users extract key insights from massive amounts of information.

**NoteX** - An intelligent note-taking and knowledge organization tool that leverages AI for automatic categorization, tag extraction, and content association, making knowledge management more efficient.

## 📦 Installation

```bash
go get github.com/smallnest/langgraphgo
```

## 🚀 Features

- **Core Runtime**:
    - **Parallel Execution**: Concurrent node execution (fan-out) with thread-safe state merging.
    - **Runtime Configuration**: Propagate callbacks, tags, and metadata via `RunnableConfig`.
    - **Generic Types**: Type-safe state management with generic StateGraph implementations.
    - **LangChain Compatible**: Works seamlessly with `langchaingo`.

- **Persistence & Reliability**:
    - **Checkpointers**: Redis, Postgres, SQLite, and File implementations for durable state.
    - **File Checkpointing**: Lightweight file-based checkpointing without external dependencies.
    - **State Recovery**: Pause and resume execution from checkpoints.

- **Advanced Capabilities**:
    - **State Schema**: Granular state updates with custom reducers (e.g., `AppendReducer`).
    - **Smart Messages**: Intelligent message merging with ID-based upserts (`AddMessages`).
    - **Command API**: Dynamic control flow and state updates directly from nodes.
    - **Ephemeral Channels**: Temporary state values that clear automatically after each step.
    - **Subgraphs**: Compose complex agents by nesting graphs within graphs.
    - **Enhanced Streaming**: Real-time event streaming with multiple modes (`updates`, `values`, `messages`).
    - **Pre-built Agents**: Ready-to-use `ReAct`, `CreateAgent`, and `Supervisor` agent factories.
    - **Programmatic Tool Calling (PTC)**: LLM generates code that calls tools programmatically, reducing latency and token usage by 10x.

- **Developer Experience**:
    - **Visualization**: Export graphs to Mermaid, DOT, and ASCII with conditional edge support.
    - **Human-in-the-loop (HITL)**: Interrupt execution, inspect state, edit history (`UpdateState`), and resume.
    - **Observability**: Built-in tracing and metrics support.
    - **Tools**: Integrated `Tavily` and `Exa` search tools.

## 🎯 Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	ctx := context.Background()
	model, _ := openai.New()

	// 1. Create Graph
	g := graph.NewStateGraph[map[string]any]()

	// 2. Add Nodes
	g.AddNode("generate", "generate", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		input, ok := state["input"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid input")
		}

		response, err := model.Call(ctx, input)
		if err != nil {
			return nil, err
		}

		state["output"] = response
		return state, nil
	})

	// 3. Define Edges
	g.AddEdge("generate", graph.END)
	g.SetEntryPoint("generate")

	// 4. Compile
	runnable, _ := g.Compile()

	// 5. Invoke
	initialState := map[string]any{
		"input": "Hello, LangGraphGo!",
	}
	result, _ := runnable.Invoke(ctx, initialState)

	fmt.Println(result)
}
```

## 📚 Examples

This project includes **90+ comprehensive examples** organized into categories:

### Featured Examples

- **[ReAct Agent](./examples/react_agent/)** - Reason and Action agent using tools
- **[RAG Pipeline](./examples/rag_pipeline/)** - Complete retrieval-augmented generation
- **[Chat Agent](./examples/chat_agent/)** - Multi-turn conversation with session management
- **[Supervisor](./examples/supervisor/)** - Multi-agent orchestration
- **[Tree of Thoughts](./examples/tree_of_thoughts/)** - Search-based reasoning with multiple solution paths
- **[Planning Agent](./examples/planning_agent/)** - Dynamic workflow plan creation
- **[PEV Agent](./examples/pev_agent/)** - Plan-Execute-Verify with self-correction
- **[Reflection Agent](./examples/reflection_agent/)** - Iterative improvement through self-reflection
- **[Mental Loop](./examples/mental_loop/)** - Simulator-in-the-loop for safe action testing
- **[Reflexive Metacognitive Agent](./examples/reflexive_metacognitive/)** - Self-aware agent with explicit capabilities model

### Example Categories

- **[Basic Concepts](./examples/README.md#basic-concepts)** - Simple LLM integration, LangChain compatibility
- **[State Management](./examples/README.md#state-management)** - State schema, custom reducers, smart messages
- **[Graph Structure](./examples/README.md#graph-structure--routing)** - Conditional routing, subgraphs, generics
- **[Parallel Execution](./examples/README.md#parallel-execution)** - Fan-out/fan-in with state merging
- **[Streaming & Events](./examples/README.md#streaming--events)** - Real-time updates, listeners, logging
- **[Persistence](./examples/README.md#persistence-checkpointing)** - Checkpointing with file, memory, databases
- **[Human-in-the-Loop](./examples/README.md#human-in-the-loop)** - Interrupts, approval, time travel
- **[Pre-built Agents](./examples/README.md#pre-built-agents)** - ReAct, Supervisor, Chat, Planning agents
- **[Programmatic Tool Calling](./examples/README.md#programmatic-tool-calling-ptc)** - PTC for 10x latency reduction
- **[Memory](./examples/README.md#memory)** - Buffer, sliding window, summarization strategies
- **[RAG](./examples/README.md#rag-retrieval-augmented-generation)** - Vector stores, GraphRAG with FalkorDB
- **[Tools & Integrations](./examples/README.md#tools--integrations)** - Search tools, GoSkills, MCP

**[View All 90+ Examples →](./examples/README.md)**

## 🔧 Key Concepts

### Parallel Execution
LangGraphGo automatically executes nodes in parallel when they share the same starting node. Results are merged using the graph's state merger or schema.

```go
g.AddEdge("start", "branch_a")
g.AddEdge("start", "branch_b")
// branch_a and branch_b run concurrently
```

### Human-in-the-loop (HITL)
Pause execution to allow for human approval or input.

```go
config := &graph.Config{
    InterruptBefore: []string{"human_review"},
}

// Execution stops before "human_review" node
state, err := runnable.InvokeWithConfig(ctx, input, config)

// Resume execution
resumeConfig := &graph.Config{
    ResumeFrom: []string{"human_review"},
}
runnable.InvokeWithConfig(ctx, state, resumeConfig)
```

### Pre-built Agents
Quickly create complex agents using factory functions.

```go
// Create a ReAct agent
agent, err := prebuilt.CreateReactAgent(model, tools)

// Create an agent with options
agent, err := prebuilt.CreateAgent(model, tools, prebuilt.WithSystemMessage("System prompt"))

// Create a Supervisor agent
supervisor, err := prebuilt.CreateSupervisor(model, agents)
```

### Programmatic Tool Calling (PTC)
Generate code that calls tools directly, reducing API round-trips and token usage.

```go
// Create a PTC agent
agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
    Model:         model,
    Tools:         toolList,
    Language:      ptc.LanguagePython, // or ptc.LanguageGo
    ExecutionMode: ptc.ModeDirect,     // Subprocess (default) or ModeServer
    MaxIterations: 10,
})

// LLM generates code that calls tools programmatically
result, err := agent.Invoke(ctx, initialState)
```

See the [PTC README](./ptc/README.md) for detailed documentation.

## 🎨 Graph Visualization

```go
exporter := runnable.GetGraph()
fmt.Println(exporter.DrawMermaid()) // Generates Mermaid flowchart
```

## 📈 Performance

- **Graph Operations**: ~14-94μs depending on format
- **Tracing Overhead**: ~4μs per execution
- **Event Processing**: 1000+ events/second
- **Streaming Latency**: <100ms

## 🧪 Testing

```bash
go test ./... -v
```

# Contributors

This project is open for contributions! if you are interested in being a contributor please create feature issues first, then submit PRs..	


## 📄 License

MIT License - see original repository for details.