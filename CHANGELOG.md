# Changelog

## [Unreleased]

### LangChainGo Decoupling
- **Native LLM Providers**: Added langchaingo-free LLM providers under `llms/` (#94, #96)
  - `nativeopenai`: OpenAI Chat Completions provider with no langchaingo dependency
  - `openairesponses`: native provider for the OpenAI Responses API
  - `doubao` and `qwen` providers dropped their langchaingo dependency
- **Framework-Owned Types**: The framework now owns its LLM and tool vocabulary
  - `llms` package defines the message, tool, and model types, with a thin
    langchaingo adapter layer (`Wrap`/`ToLangchain`) for interop
  - `tool.Tool` is the framework's own tool interface, no longer aliasing
    `langchaingo/tools`
  - `memory/langchainadapter` isolates the langchaingo memory shim so the core
    `memory` package stays dependency-free (#100)
- **Examples Decoupled**: Examples now build against the framework's own `llms`
  and `tool` packages instead of langchaingo

### Refactoring
- **Package Consolidation**: Merged the `tooltypes` and `llmtypes` packages into
  `tool` and `llms` respectively, so consumers reference `tool.Tool` and `llms.X`
  directly
- **Planning Agents**: Unified the plan-execute agent option surface (#101)
- **RAG Slimming**: Applied a RAG module audit — dropped the dead reranker,
  removed duplication, and deprecated an unused wrapper (#103)
- **Go 1.25 Modernization**: Modernized Go idioms for Go 1.25

### Tooling
- Migrated `.golangci.yaml` to the golangci-lint v2 configuration format

### Removed
- Removed the `docs/`, `showcases/`, and `weekly/` directories (including the
  `showcases` git submodule)

## [0.8.0] - 2026-01-25

### RAG Enhancements
- **BM25 Support**: Added BM25 retrieval algorithm implementation (#86)
  - Classic TF-IDF based retrieval with document length normalization
  - Configurable parameters (k1, b) for fine-tuning
  - Pure Go implementation with no external dependencies
  - Custom tokenizer support with stopword filtering
  - High performance for keyword-based retrieval scenarios
- **Hybrid RAG**: Implemented hybrid retrieval combining vector and BM25 (#86)
  - RRF (Reciprocal Rank Fusion) algorithm for result merging
  - Configurable weights for vector and BM25 retrieval
  - Automatic deduplication and re-ranking
  - Optimal for knowledge-intensive Q&A and document retrieval
- **BM25 Examples**: Added comprehensive BM25 and hybrid RAG examples
  - `bm25_demo`: Complete BM25 retrieval demonstration
  - `hybrid_rag_demo`: Hybrid retrieval with multiple strategies
  - Bilingual documentation (English & Chinese)

### GoSkills Enhancements
- **Skill Framework**: Enhanced GoSkills skill system with improved type safety
  - Better error handling and validation
  - Improved skill loading and execution
  - Enhanced TypeScript integration
- **New Skills**:
  - **baoyu-image-gen**: Image generation skill supporting OpenAI and Google APIs
    - 611 lines of TypeScript implementation
    - Flexible configuration for image generation
    - Support for multiple image sizes and quality settings
  - **baoyu-comic**: Comic generation skill with multiple layouts and styles
    - 553 lines of comic generation script
    - Support for 6 layouts (cinematic, dense, mixed, splash, standard, webtoon)
    - Support for 10 styles (chalkboard, classic, dramatic, ohmsha, realistic, sepia, shoujo, vibrant, warm, wuxia)
    - PDF export functionality (131 lines)
- **Comic Skill Example**: Added comprehensive comic generation example
  - 684 lines of example code and documentation
  - Demonstrates skill system capabilities

### Multi-Agent Improvements
- **Dexter Implementation**: Implemented virattt/dexter multi-agent system (#89)
  - Refactored swarm example to use typed State struct
  - Demonstrates best practices for type-safe multi-agent systems
  - Supervisor pattern with dynamic task routing
  - In-place state updates following Go idioms
  - Reduced code from 79 to 57 lines while improving clarity

### Bug Fixes
- **Struct Merging**: Fixed struct value handling in DefaultStructMerge and FieldMerger (#87)
  - Proper handling of struct values in state merging
  - Improved state accumulation logic
- **Durable Execution**: Fixed accumulate logic in durable_execution examples
  - Correct state accumulation for parallel execution scenarios
  - Fixed checkpoint state persistence
- **Checkpoint Optimization**: Improved checkpoint saving logic with thread_id
  - Distinguished checkpoint saving logic for threaded vs non-threaded scenarios
  - Better data isolation for concurrent executions

### Documentation
- Added BM25 integration guide (366 lines)
- Added BM25 summary documentation (173 lines)
- Added hybrid RAG documentation (690 lines)
- Updated GoSkills skill documentation
- Improved state management examples

## [0.7.0] - 2026-01-17

### Vector Stores & RAG Enhancements
- **Milvus Support**: Added Milvus vector store integration (#59)
  - High-performance vector database support for RAG applications
  - Scalable solution for large-scale vector similarity search
- **Chroma v2 Support**: Updated Chroma integration to v2 (#59)
  - Latest Chroma client with improved performance and features
  - Enhanced vector store capabilities
- **Chromem-go Support**: Added chromem-go embedding store (#59)
  - Lightweight, in-memory vector store option
  - Fast and efficient for small to medium datasets
- **Redis Vector Support**: Added Redis-based vector storage (#59)
  - Leverage Redis for both checkpointing and vector storage
  - Unified infrastructure for state and embeddings

### RAG Features
- **Qwen Ranker**: Added Qwen-based reranking for improved retrieval quality (#83)
  - Intelligent reranking of retrieved documents
  - Enhanced relevance scoring for RAG pipelines
- **LightRAG Examples**: Added LightRAG integration examples
  - `lightrag_simple`: Basic LightRAG usage
  - `lightrag_advanced`: Advanced LightRAG patterns
- **RAG with Memory**: Added memory-enhanced RAG example
  - Combines conversation memory with retrieval-augmented generation
  - Maintains context across multi-turn conversations

### Streaming Enhancements
- **StreamingLLM**: Added StreamingLLM support (#82)
  - Efficient streaming for long-context conversations
  - Reduced memory footprint for streaming applications
- **Streaming Examples**: Added multiple streaming examples
  - `llm_streaming`: Basic LLM streaming with callbacks
  - `langchaingo_streaming`: LangChain Go integration with streaming
  - `adapter_streaming`: Custom adapter streaming implementation

### New Agents
- **Manus Agent**: Added Manus agent pattern (#80)
  - Advanced agent for document processing workflows
  - Intelligent task orchestration for complex documents

### Human-in-the-Loop Enhancements
- **API Interrupt Demo**: Added API-based interrupt demonstration
  - Shows how to implement approval workflows via API
  - Demonstrates programmatic control of HITL patterns
- **Payment Interrupt**: Enhanced payment approval workflow
  - Real-world payment processing with human approval
  - Secure transaction handling with interrupt points

### Bug Fixes & Improvements
- Fixed command_api example (#84)
- Optimized checkpoint data isolation (#72)
- Removed sqlite-vec dependency in favor of more robust solutions (#59)
- Improved error handling and recovery mechanisms

### Documentation
- Updated examples README with new categorization
- Enhanced documentation for vector store integrations
- Added Chinese README_CN.md for new examples

## [0.7.0] - 2025-12-21

### Core Features
- **Graph Database**: Support for FalkorDB (#53)
- **Generic Types**: Major milestone for generic type support (#48)
  - Added generic StateGraph implementations for type-safe state management
  - Replaced `interface{}` with `any` throughout the codebase
  - Enhanced ListenableRunnable with generic support

### Checkpointing
- **File Checkpoint Store**: Implemented file-based checkpointing (#46)
  - Added `FileCheckpointStore` for persistent state storage on local filesystem
  - Support for crash recovery and execution resumption from files
  - Simple and lightweight checkpointing solution without external dependencies

### Code Quality & Testing
- **Improved Test Coverage**: Significantly increased unit test coverage across modules:
  - RAG LangChain adapter
  - Supervisor typed implementations
  - PostgreSQL checkpointing
  - ReAct agent
- **Logger Implementation**: Added golog logger implementation for better debugging and monitoring
- **Documentation**: Added comprehensive `doc.go` files for better package documentation
- **Lint Fixes**: Resolved various linting issues for cleaner code

### Examples & Patterns
- **[Tree of Thoughts](./examples/tree_of_thoughts/)**: Advanced reasoning framework using search tree exploration
  - Implements systematic multi-path problem-solving approach
  - Five key phases: Decomposition, Thought Generation, State Evaluation, Pruning & Expansion, Solution
  - Configurable search strategies with generator and evaluator interfaces
  - Visual search tree representation with path tracking
  - Comprehensive documentation explaining ToT architecture and use cases

### Pre-built Agents
- **Chat Agent**: Enhanced chat capabilities and themes (#55)
- **[PEV Agent](./examples/pev_agent/)**: Problem-Evidence-Verification agent (#38)
  - Structured problem-solving with evidence gathering
  - Verification mechanism for solution validation
  - Support showcase for https://profile.rpcx.io project profile generation

### Refactoring
- **RAG Refactoring**: Refactored RAG module (#54)
- **MessageGraph Removal**: Removed MessageGraph as a special type (#43, #44)
  - Merged MessageGraph features into StateGraph for better consistency
  - Simplified API by removing redundant `NewMessagesStateGraph` method
  - Updated ListenableStateGraph structure for better maintainability

### Bug Fixes & Improvements
- Fixed GoSkills integration issues
- Fixed race conditions in concurrent execution scenarios
- Fixed duration_execution bug in parallel execution scenarios
- Enhanced GitHub Actions CI/CD with updated golangci-lint versions
- Improved unit test reliability
- Added CONTRIBUTING.md for development guidelines

## [0.6.0] - 2025-12-08

### Examples & Patterns
- **[Complex Parallel Execution](./examples/complex_parallel_execution/)**: Advanced parallel execution pattern (#36)
  - Demonstrates fan-out/fan-in with branches of varying lengths
  - Three implementation versions: basic, smart aggregator, and synchronized
  - Comprehensive Mermaid flow diagrams for all three approaches
  - Detailed comparison documentation (COMPARISON.md)
  - Real-world use cases: multi-source data processing, parallel analysis pipelines

### Pre-built Agents
- **[Chat Agent](./examples/chat_agent/)**: Multi-turn conversation agent with session management (#34)
  - Automatic conversation history tracking
  - Session-based memory management
  - Support for multiple concurrent conversations
- **[Chat Agent Async](./examples/chat_agent_async/)**: Asynchronous streaming chat agent
  - Real-time LLM response streaming
  - Non-blocking execution for better performance
- **[Chat Agent Dynamic Tools](./examples/chat_agent_dynamic_tools/)**: Chat agent with runtime tool management
  - Add/remove tools during conversation
  - Dynamic capability adjustment

### Documentation & CI/CD
- **DeerFlow**: Added simple documentation for DeerFlow showcase
- **GitHub Actions**: Improved CI/CD pipeline with golangci-lint integration
- **Examples README**: Updated with new chat agent and parallel execution examples

## [0.5.0] - 2025-12-06

### Programmatic Tool Calling (PTC)
- **PTC Package**: Added new `ptc` package for programmatic tool calling (#31).
  - LLM generates code that calls tools directly instead of requiring API round-trips
  - Supports both Python and Go code execution
  - Two execution modes: `ModeDirect` (subprocess, default) and `ModeServer` (HTTP server, alternative)
  - Reduces latency and token usage by up to 10x for complex tool chains
  - Multi-LLM support (OpenAI, Gemini, Claude, any langchaingo-compatible model)

### PTC Features
- **Code Executor**: Executes LLM-generated Python/Go code with tool access in sandboxed environment
- **Tool Server**: HTTP server exposing tools via REST API for secure code execution
- **Smart Code Generation**: Automatic tool wrapper generation for both Python and Go
- **Error Handling**: Robust error reporting with execution output and debugging information
- **Documentation**: Complete bilingual documentation (English & Chinese) with Mermaid flow diagrams

### PTC Examples
- **[PTC Basic](./examples/ptc_basic/)**: Introduction to PTC with calculator, weather, and data processing tools
- **[PTC Simple](./examples/ptc_simple/)**: Simple calculator example demonstrating basic PTC usage
- **[PTC Expense Analysis](./examples/ptc_expense_analysis/)**: Complex scenario based on Anthropic PTC Cookbook, showing data filtering and aggregation

### Design Patterns
- **Planning Pattern**: Added planning mode for task decomposition and execution planning (#24)
- **Reflection Agent**: Implemented reflection-action loop pattern for self-assessment and quality improvement (#32)

### Showcases & Documentation
- **GPT Researcher**: Complete replication of assafelovic/gpt-researcher (#34)
  - Automated research and report generation
  - Multi-source information integration
- **Trading Agents**: Merged documentation files for comprehensive README (#39)
  - Integrated PROJECT_SUMMARY.md and USAGE.md into README.md
  - Added detailed usage guide, verbose mode examples, and API reference
- **Open Deep Research**: Merged WORKFLOW.md into README files (#38)
  - Added 5 detailed Mermaid workflow diagrams
  - Included key concepts: state accumulation, message sequence, parallel execution
- **Health Insights Agent**: Merged PROJECT_SUMMARY_CN.md into README_CN.md (#37)
  - Added technical architecture, performance metrics, and security considerations
- **DeepAgents**: Added comprehensive documentation (#36)
  - Complete tool reference and best practices guide
- **DeerFlow & BettaFish**: Updated documentation for both showcases (#35)

### Agent Documentation
- **CreateAgent & CreateReactAgent**: Added comprehensive comparison documentation (#33)
  - Detailed API reference and usage examples
  - Best practices and use case guidance

### Website & Knowledge Base
- **Official Website**: http://lango.rpcx.io (source: https://github.com/smallnest/lango-website)
  - 233 HTML pages with bilingual support
  - 16+ detailed guides (Getting Started, Advanced Features, State Management, etc.)
  - Showcase gallery with 6 complete projects
  - Examples page with 20+ code examples
- **Wiki Knowledge Base**: 193 Markdown documents covering:
  - Advanced features (HITL, Visualization, Subgraphs, Parallel Execution)
  - Checkpoint storage (SQLite, Redis, PostgreSQL)
  - Tool integration guides
  - Pre-built components and RAG guides

### Documentation Consolidation
- Simplified documentation structure with clearer naming conventions
- Merged scattered documentation into comprehensive README files
- Improved navigation and discoverability across all showcases

## [0.4.0] - 2025-12-04

### Core & Agents
- **MCP Support**: Added support for Model Context Protocol (MCP) (#21).
- **Skills Integration**:
  - Added support for **Claude Skills** (#20).
  - Updated `CreateAgent` to support dynamic skill loading (#20).
- **LLM Providers**: Added support for alternative OpenAI-compatible LLM providers in BettaFish showcase.

### Tools
- **Search Tools**:
  - Added **Brave Search** API support.
  - Added **Bocha Search** tool (#22).

### Showcases
- **DeerFlow**: Updated DeerFlow showcase.
- **BettaFish**: Added new showcase replicating BettaFish (https://github.com/666ghj/BettaFish) (#19).

### Documentation & Website
- **Website**: Moved website content to https://github.com/smallnest/lango-website.
- **DIFF.md**: Added DIFF.md for showcases (#19).

## [0.3.0] - 2025-12-01

### Core Runtime
- **Parallel Execution**: Implemented fan-out/fan-in execution model with thread-safe state merging.
- **Runtime Configuration**: Added `RunnableConfig` to propagate configuration (like thread IDs, user IDs) through the graph execution context.
- **Command API**: Introduced `Command` struct for dynamic flow control (`Goto`) and state updates (`Update`) directly from nodes.
- **Subgraphs**: Added native support for composing graphs by using compiled graphs as nodes (`AddSubgraph`).

### Persistence & Checkpointing
- **Checkpoint Interface**: Refined `CheckpointSaver` interface for state persistence.
- **Implementations**: Added full support for **Redis**, **PostgreSQL**, and **SQLite** checkpoint stores.

### Advanced State & Streaming
- **State Management**: Introduced `Schema` interface and `Annotated` style reducers (e.g., `AppendMessages`) for complex state updates.
- **Smart Messages**: Implemented `AddMessages` reducer for ID-based message upserts and deduplication.
- **Enhanced Streaming**: Added typed `StreamEvent`s and `CallbackHandler` interface. Implemented multiple streaming modes: `updates`, `values`, `messages`, and `debug`.

### Pre-built Agents
- **ToolExecutor**: Added a dedicated node for executing tools.
- **ReAct Agent**: Implemented a factory for creating ReAct-style agents.
- **Create Agent**: Added `CreateAgent` factory with functional options for flexible agent creation.
- **Supervisor**: Added support for Supervisor agent patterns for multi-agent orchestration.

### Human-in-the-loop (HITL)
- **Interrupts**: Implemented `InterruptBefore` and `InterruptAfter` mechanisms to pause graph execution.
- **Resume & Command**: Added support for resuming execution and updating state via commands.
- **Time Travel**: Implemented `GetState` and `UpdateState` APIs to inspect/modify past checkpoints and fork execution history.

### Visualization
- **Mermaid Export**: Improved graph visualization with better rendering of conditional edges and styling options.

### Experimental & Research
- **Swarm Patterns**: Added prototypes for multi-agent collaboration using subgraphs (`examples/swarm`).
- **Channels RFC**: Added `RFC_CHANNELS.md` proposing a channel-based architecture for future improvements.

### LangChain Integration
- **VectorStore Adapter**: Added `LangChainVectorStore` adapter to integrate any langchaingo vectorstore implementation.
- **Supported Backends**: Full support for Chroma, Weaviate, Pinecone, Qdrant, Milvus, PGVector, and any other langchaingo vectorstore.
- **Unified Interface**: Seamless integration with RAG pipelines through standard `AddDocuments`, `SimilaritySearch`, and `SimilaritySearchWithScore` methods.
- **Complete Adapters**: Now includes adapters for DocumentLoaders, TextSplitters, Embedders, and VectorStores from langchaingo.

### Tools & Integrations
- **Tool Package**: Added a new `tool` package for easy integration of external tools.
- **Search Tools**: Implemented `TavilySearch` and `ExaSearch` tools compatible with `langchaingo` interfaces.
- **Agent Integration**: Updated `ReAct` agent to support tool parameter schema generation and argument parsing for OpenAI-compatible APIs.
- **GoSkills Adapter**: Added `adapter/goskills` to integrate [GoSkills](github.com/smallnest/goskills) as tools.

### Examples
- Added comprehensive examples for:
  - Checkpointing (Postgres, SQLite, Redis)
  - Human-in-the-loop workflows
  - Swarm multi-agent patterns
  - Subgraphs
  - **Smart Messages** (new)
  - **Command API** (new)
  - **Ephemeral Channels** (new)
  - **Streaming Modes** (new)
  - **Time Travel / HITL** (new)
  - **LangChain VectorStore integration** (new)
  - **Chroma vector database integration** (new)
  - **Tavily Search Tool** (new)
  - **Exa Search Tool** (new)
  - **Create Agent** (new)
  - **Dynamic Skill Agent** (new)
  - **Durable Execution** (new)
  - **GoSkills Integration** (new)
- **General**: Improved reliability and correctness of all examples.

## [0.1.0] - 2025-01-02

### Added
- Generic state management - works with any type, not just MessageContent
- Performance optimizations for production use
- Support for any LLM client (removed hard dependency on LangChain)

### Changed
- Simplified API for building graphs
- Updated examples to show generic usage

### Fixed
- CI/CD pipeline issues from original repository
- Build errors with recent Go versions

### Removed
- Hard dependency on LangChain - now works with any LLM library