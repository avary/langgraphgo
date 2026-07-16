# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**LangGraphGo** (nickname: `lango`) is a Go implementation of LangChain's LangGraph framework for building stateful, multi-agent applications with LLMs. It uses a directed graph structure where nodes are processing units (agents, tools, functions) and edges define execution flow, with state flowing through and evolving at each node.

## Build and Test Commands

```bash
# Build
make build              # Build all packages

# Testing
make test               # Run all tests
go test -v ./path/to/pkg -run TestName   # Run single test
make test-race          # Run with race detector
make test-coverage      # Generate coverage.out, coverage.html, coverage.svg
make test-checkpoint    # Test checkpointing only
make test-graph         # Test graph package only
make test-prebuilt      # Test prebuilt agents only
make test-ptc           # Test PTC only
make test-memory        # Test memory only

# Code Quality (run before committing)
make fmt                # Format code with gofmt
make fmt-check          # Check formatting
make vet                # Run go vet
make lint               # Run golangci-lint
make check              # Run fmt-check, vet, lint
make pre-commit         # Run fmt, vet, lint, test

# Dependencies
make tidy               # Tidy and verify dependencies
make deps               # Download dependencies
make install-tools      # Install golangci-lint

# Other
make examples           # Build all examples
make clean              # Clean artifacts
make info               # Display project info
```

## Architecture

### Core Graph Engine (`graph/`)

The heart of the framework. Key types:
- `StateGraph` - Untyped graph using `map[string]any`
- `StateGraphTyped[S]` - Generic typed version for compile-time safety
- `StateRunnable` - Compiled executable graph
- `MessageGraph` - Specialized for message-passing workflows

**Key concepts:**
- **Nodes** have name, description, and function (signature: `func(ctx context.Context, state S) (S, error)`)
- **Edges** connect nodes: static (`AddEdge`) or conditional (`AddConditionalEdge`)
- **Parallel execution**: When multiple nodes share the same starting point, they execute concurrently with thread-safe state merging
- **Checkpointing**: Save/resume execution state (SQLite, PostgreSQL, Redis, file, memory backends in `store/`)
- **Streaming**: Real-time event streaming with listeners
- **Human-in-the-loop**: `InterruptBefore`/`InterruptAfter` for pauses, `UpdateState` for editing history

### State Management

State can be:
1. **Untyped**: `map[string]any` for flexibility
2. **Typed**: Use `StateGraphTyped[MyStruct]` for compile-time safety
3. **Structured**: `StateSchema` with reducers for granular updates

Reducers control how state updates merge:
- `AppendReducer` - Append slices
- `AddMessages` - Smart message merging with ID-based upserts
- Custom reducers supported

### Prebuilt Agents (`prebuilt/`)

Ready-to-use agent factories:
- `CreateReactAgent` - ReAct (Reasoning + Acting) pattern
- `CreateSupervisor` - Multi-agent orchestration
- `CreatePlanningAgent` - Plan creation and execution
- `CreateReflectionAgent` - Self-correcting
- `CreateTreeOfThoughts` - Multi-path reasoning
- `CreatePEVAgent` - Plan-Execute-Verify
- `CreateChatAgent` - Multi-turn conversations
- `CreateAgent` - Flexible factory with options

### Programmatic Tool Calling (`ptc/`)

Agents generate code to call tools, reducing latency by ~10x. Supports Python, JavaScript, and Shell execution modes.

### Memory (`memory/`)

Conversation memory strategies:
- `Buffer` - Simple FIFO
- `SlidingWindow` - Recent context with overlap
- `Summarization` - Compress older conversations
- `Hierarchical` - Multi-level with importance scoring
- `OSLike` - Paging/eviction
- `GraphBased` - Graph-structured
- `Retrieval` - Retrieval-based

### RAG (`rag/`)

Retrieval-Augmented Generation:
- Vector RAG with various stores
- GraphRAG with FalkorDB knowledge graphs
- Hybrid approaches
- Document loaders, text splitters, retrievers

### Tools (`tool/`)

Web search (Tavily, Brave, Exa, Bocha), file operations, code execution.

### Adapters (`adapter/`)

- `goskills/` - Custom Go-based skills
- `mcp/` - Model Context Protocol tools

## Code Style

From `AGENTS.md`:
- Exported: PascalCase; private: camelCase
- Package-level errors: `var ErrX = errors.New("...")`
- Godoc comments for all exported identifiers
- Table-driven tests with `t.Run()`, `t.Parallel()` where appropriate
- Always check errors

**Clean up generated artifacts**:
- Delete temporary binary files after tasks complete
- Use `go run` instead of `go build && ./binary` for one-off executions
- If you must build binaries, clean them up: `rm -f ./binary_name` or `make clean`

**Logging conventions**:
- Log messages should start with a lowercase letter (not capitalized)
- This is because log packages typically add timestamps/prefixes
- Good: `log.Println("processing request")`  or `log.Printf("user %s not found", userID)`
- Bad:  `log.Println("Processing request")`  or `log.Printf("User %s not found", userID)`
- Exceptions: Proper nouns, acronyms, and names should remain capitalized (e.g., "GitHub", "API", "LangGraph")

## Development Workflow

**MANDATORY: After completing any code changes, always run:**
1. `make fmt` - Format code with gofmt
2. `make vet` - Run go vet
3. `make lint` - Run golangci-lint

Only after these pass, proceed with:
4. `make test` - Run tests
5. Commit

**Quick check command:** `make check` (runs fmt-check, vet, lint in one command)

## Single Test Execution

```bash
go test -v ./path/to/package -run TestSpecificFunction
```

## CI Configuration

- Runs on push/PR to master branch
- Main branch for PRs is `main`
- Uses `go test -race` with coverage

## Key Dependencies

- `github.com/tmc/langchaingo` - LangChain Go integration
- `github.com/sashabaranov/go-openai` - OpenAI client
- `github.com/redis/go-redis/v9` - Redis support
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/mattn/go-sqlite3` - SQLite driver
