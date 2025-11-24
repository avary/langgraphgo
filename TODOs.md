# LangGraphGo TODOs

This file lists features from the Python LangGraph library that are not yet implemented in LangGraphGo. The goal is to achieve feature parity where it makes sense in the Go ecosystem.

## 🚧 High-Priority Features

### 1. Persistent Checkpoint Backends
The current checkpointing system only includes an in-memory store. To support durable execution for production applications, we need persistent storage backends.

- [ ] Implement a Redis checkpointer.
- [ ] Implement a Postgres checkpointer.
- [ ] Implement a SQLite checkpointer.
- [ ] Add a clear interface for custom storage backends.

### 2. Pre-built Agentic Components
The Python library provides high-level components that simplify building common agent patterns.

- [ ] Create a standard `ToolExecutor` node helper for easier tool/function calling.
- [ ] Implement a pre-built `ReAct` agent pattern.
- [ ] Implement a pre-built `Supervisor` node for orchestrating and delegating tasks between other agents/subgraphs.

## 🔬 Medium-Priority Features

### 3. Advanced State Management
The current state management relies on `interface{}` or concrete types, which is less flexible than Python's system.

- [ ] Investigate a more robust, type-safe state management system, perhaps using generics.
- [ ] Implement an equivalent of Python's `Annotated` mechanism to define how state properties are updated (e.g., append for message lists vs. overwrite for other fields). This would provide more declarative state updates.

### 4. Human-in-the-loop
The graph should be able to pause and wait for external input.

- [ ] Add a built-in mechanism or a clear, documented pattern for implementing human-in-the-loop workflows.
- [ ] Provide an example of a graph that pauses for user approval before continuing.

## 🏗️ Low-Priority / Architectural Features

### 5. Multi-Agent Collaboration (Swarm)
The Python library has concepts for "swarms" to enable multiple agents to work together.

- [ ] Research and design an API for multi-agent collaboration patterns.

### 6. Core Architectural Parity (Channels)
The Python version is built on a Pregel-inspired computation model that uses "Channels" as a core abstraction for state management.

- [ ] Investigate the "Channels" concept in the Python library.
- [ ] Evaluate if a similar abstraction is needed or beneficial for LangGraphGo to enable more complex and parallelizable state interactions.
