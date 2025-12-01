# LangGraphGo Implementation Roadmap

This document outlines the step-by-step plan to implement the features listed in `TODOs.md`.

## Phase 1: Core Runtime Enhancements
Focus on the fundamental execution model and configuration to support complex graphs.

- [ ] **Parallel Execution (Fan-out / Fan-in)**
    - [ ] Design concurrent node execution model in `graph.go`.
    - [ ] Implement `ExecuteParallel` in `pregel` or `graph` package.
    - [ ] Add synchronization mechanism (WaitGroups/Channels) for fan-in.
    - [ ] Ensure thread-safe state merging.
    - [ ] Add unit tests for parallel execution scenarios.

- [ ] **Runtime Configuration**
    - [ ] Define `RunnableConfig` struct (similar to Python's `configurable`).
    - [ ] Update `Invoke` and `Stream` signatures to accept `RunnableConfig`.
    - [ ] Propagate config to `Node` context.
    - [ ] Add helper function to retrieve config from context.

## Phase 2: Persistence & Reliability
Enable durable execution and state recovery.

- [ ] **Persistent Checkpoint Interface**
    - [ ] Refine `CheckpointSaver` interface in `checkpoint` package.
    - [ ] Define serialization format for State (JSON/Gob).

- [ ] **Redis Checkpointer**
    - [ ] Create `checkpoint/redis` package.
    - [ ] Implement `Put` and `Get` using Redis client.
    - [ ] Add integration tests with Redis (mock or container).

- [ ] **Postgres Checkpointer**
    - [ ] Create `checkpoint/postgres` package.
    - [ ] Design schema for checkpoints.
    - [ ] Implement `Put` and `Get` using `database/sql` or `pgx`.

- [ ] **SQLite Checkpointer**
    - [ ] Create `checkpoint/sqlite` package.
    - [ ] Implement file-based persistence for local development.

## Phase 3: Advanced Features
Enhance the capabilities of the graph for complex agentic behaviors.

- [ ] **Advanced State Management**
    - [ ] Design `Schema` interface for state validation.
    - [ ] Implement `Annotated` style reducers (e.g., `AppendMessages`).
    - [ ] Refactor `MessageGraph` to use the new state system.

- [ ] **Enhanced Streaming**
    - [ ] Define `StreamEvent` types (NodeStart, NodeEnd, Token, etc.).
    - [ ] Update `StreamingListener` to handle custom events.
    - [ ] Add `EmitEvent` method to Node context.

- [ ] **Pre-built Agentic Components**
    - [ ] Implement `ToolExecutor` node.
    - [ ] Build `ReAct` agent factory.
    - [ ] Build `Supervisor` node factory.

## Phase 4: Developer Experience & Human-in-the-loop
Tools for debugging, visualizing, and controlling execution.

- [ ] **Visualization Improvements**
    - [ ] Update `Exporter` to traverse and render conditional edges.
    - [ ] Add styling options to `DrawMermaid`.

- [ ] **Human-in-the-loop (HITL)**
    - [ ] Implement `Interrupt` mechanism in graph execution.
    - [ ] Add `Command` support to resume/update state.
    - [ ] Create an example of a "Human Approval" workflow.

## Phase 5: Future & Research
Long-term architectural improvements.

- [ ] **Multi-Agent Collaboration**
    - [ ] Prototype "Swarm" patterns using `Subgraph`.

- [ ] **Channels Architecture**
    - [ ] Research Python's Channel implementation.
    - [ ] Propose RFC for Go implementation if beneficial.
