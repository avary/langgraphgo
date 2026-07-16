# 更新日志

## [未发布]

### 解耦 LangChainGo
- **原生 LLM Provider**: 在 `llms/` 下新增了不依赖 langchaingo 的 LLM Provider (#94, #96)
  - `nativeopenai`: 不依赖 langchaingo 的 OpenAI Chat Completions Provider
  - `openairesponses`: OpenAI Responses API 的原生 Provider
  - `doubao` 和 `qwen` Provider 移除了对 langchaingo 的依赖
- **框架自有类型**: 框架现在拥有自己的 LLM 和工具类型体系
  - `llms` 包定义了消息、工具和模型类型，并通过一个轻量的 langchaingo 适配层
    (`Wrap`/`ToLangchain`) 实现互操作
  - `tool.Tool` 成为框架自己的工具接口，不再是 `langchaingo/tools` 的别名
  - `memory/langchainadapter` 隔离了 langchaingo 记忆适配层，使核心 `memory`
    包保持零依赖 (#100)
- **示例解耦**: 示例现在基于框架自有的 `llms` 和 `tool` 包构建，不再依赖 langchaingo

### 重构
- **包合并**: 将 `tooltypes` 和 `llmtypes` 包分别合并进 `tool` 和 `llms`，
  使用方直接引用 `tool.Tool` 和 `llms.X`
- **规划 Agent**: 统一了 plan-execute agent 的 option 接口 (#101)
- **RAG 精简**: 基于 RAG 模块审计，移除了无用的 reranker、去除重复代码，
  并废弃了一个未使用的 wrapper (#103)
- **Go 1.25 现代化**: 将代码更新为 Go 1.25 的惯用写法

### 工具链
- 将 `.golangci.yaml` 迁移到 golangci-lint v2 配置格式

### 移除
- 移除了 `docs/`、`showcases/` 和 `weekly/` 目录（含 `showcases` git submodule）

## [0.8.0] - 2025-01-17

### 向量存储与 RAG 增强
- **Milvus 支持**: 新增 Milvus 向量存储集成 (#59)
  - 为 RAG 应用提供高性能向量数据库支持
  - 可扩展的大规模向量相似性搜索解决方案
- **Chroma v2 支持**: 将 Chroma 集成更新到 v2 (#59)
  - 最新的 Chroma 客户端，性能和功能更优
  - 增强的向量存储能力
- **Chromem-go 支持**: 新增 chromem-go 嵌入存储 (#59)
  - 轻量级内存向量存储选项
  - 针对中小型数据集快速高效
- **Redis 向量支持**: 新增基于 Redis 的向量存储 (#59)
  - 利用 Redis 同时支持检查点和向量存储
  - 统一的状态和嵌入基础设施

### RAG 功能
- **Qwen 重排序器**: 新增基于 Qwen 的重排序功能，提升检索质量 (#83)
  - 智能重排序检索到的文档
  - 增强 RAG 流程的相关性评分
- **LightRAG 示例**: 新增 LightRAG 集成示例
  - `lightrag_simple`: 基础 LightRAG 用法
  - `lightrag_advanced`: 高级 LightRAG 模式
- **RAG 与记忆**: 新增记忆增强的 RAG 示例
  - 结合对话记忆与检索增强生成
  - 在多轮对话中保持上下文

### 流式增强
- **StreamingLLM**: 新增 StreamingLLM 支持 (#82)
  - 长上下文对话的高效流式处理
  - 减少流式应用的内存占用
- **流式示例**: 新增多个流式示例
  - `llm_streaming`: 带回调的基础 LLM 流式处理
  - `langchaingo_streaming`: LangChain Go 流式集成
  - `adapter_streaming`: 自定义适配器流式实现

### 新增 Agent
- **Manus Agent**: 新增 Manus agent 模式 (#80)
  - 用于文档处理流程的高级 agent
  - 复杂文档的智能任务编排

### 人在回路增强
- **API 中断演示**: 新增基于 API 的中断演示
  - 展示如何通过 API 实现审批工作流
  - 演示 HITL 模式的程序化控制
- **支付中断**: 增强支付审批工作流
  - 带人工批准的真实支付处理
  - 带中断点的安全交易处理

### 错误修复与改进
- 修复 command_api 示例 (#84)
- 优化检查点数据隔离 (#72)
- 移除 sqlite-vec 依赖，改用更稳健的解决方案 (#59)
- 改进错误处理和恢复机制

### 文档
- 更新示例 README，添加新分类
- 增强向量存储集成文档
- 为新示例添加中文 README_CN.md

## [0.7.0] - 2025-12-21

### 核心特性
- **图数据库**: 支持 FalkorDB (#53)
- **泛型类型 (Generic Types)**: 泛型支持的重要里程碑 (#48)
  - 添加泛型 StateGraph 实现，支持类型安全的状态管理
  - 将代码库中的 `interface{}` 替换为 `any`
  - 增强 ListenableRunnable 的泛型支持

### 检查点
- **文件检查点存储 (File Checkpoint Store)**: 实现基于文件的检查点 (#46)
  - 添加 `FileCheckpointStore` 用于本地文件系统的持久化状态存储
  - 支持从文件崩溃恢复和执行恢复
  - 无外部依赖的轻量级检查点解决方案

### 代码质量与测试
- **提升测试覆盖率**: 大幅增加各模块的单元测试覆盖率：
  - RAG LangChain 适配器
  - Supervisor 类型化实现
  - PostgreSQL 检查点
  - ReAct 代理
- **日志实现**: 添加 golog 日志实现，提供更好的调试和监控
- **文档**: 为更好的包文档添加全面的 `doc.go` 文件
- **Lint 修复**: 解决各种 linting 问题，使代码更整洁

### 示例与模式
- **[思维树 (Tree of Thoughts)](./examples/tree_of_thoughts/)**: 使用搜索树探索的高级推理框架
  - 实现系统的多路径问题求解方法
  - 五个关键阶段：分解、思维生成、状态评估、剪枝与扩展、解决方案
  - 可配置的搜索策略，支持生成器和评估器接口
  - 可视化搜索树表示与路径跟踪
  - 全面的文档，解释 ToT 架构和用例

### 预构建代理
- **Chat Agent**: 增强聊天能力与主题 (#55)
- **[PEV Agent](./examples/pev_agent/)**: 问题-证据-验证代理 (#38)
  - 结构化问题求解与证据收集
  - 解决方案验证机制
  - 支持 https://profile.rpcx.io 项目个人资料生成展示

### 重构
- **RAG 重构**: 重构 RAG 模块 (#54)
- **MessageGraph 移除**: 移除 MessageGraph 特殊类型 (#43, #44)
  - 将 MessageGraph 功能合并到 StateGraph 以获得更好的一致性
  - 移除冗余的 `NewMessagesStateGraph` 方法简化 API
  - 更新 ListenableStateGraph 结构以获得更好的可维护性

### 错误修复与改进
- 修复 GoSkills 集成问题
- 修复并发执行场景中的竞态条件
- 修复并行执行场景中的 duration_execution 错误
- 增强了 GitHub Actions CI/CD，更新了 golangci-lint 版本
- 提高单元测试可靠性
- 添加 CONTRIBUTING.md 开发指南

## [0.6.0] - 2025-12-08

### 示例与模式
- **[复杂并行执行 (Complex Parallel Execution)](./examples/complex_parallel_execution/)**: 高级并行执行模式 (#36)
  - 演示不同长度分支的扇出/扇入模式
  - 三种实现版本：基础版、智能聚合器版、同步版
  - 所有三种方法的完整 Mermaid 流程图
  - 详细对比文档 (COMPARISON.md)
  - 真实场景：多源数据处理、并行分析管道

### 预构建代理
- **[Chat Agent](./examples/chat_agent/)**: 支持会话管理的多轮对话代理 (#34)
  - 自动对话历史跟踪
  - 基于会话的内存管理
  - 支持多个并发对话
- **[Chat Agent Async](./examples/chat_agent_async/)**: 异步流式聊天代理
  - 实时 LLM 响应流式传输
  - 非阻塞执行以获得更好的性能
- **[Chat Agent Dynamic Tools](./examples/chat_agent_dynamic_tools/)**: 支持运行时工具管理的聊天代理
  - 对话期间添加/删除工具
  - 动态能力调整

### 文档与 CI/CD
- **DeerFlow**: 为 DeerFlow 示例添加简单文档
- **GitHub Actions**: 改进 CI/CD 管道，集成 golangci-lint
- **示例 README**: 更新包含新的聊天代理和并行执行示例

## [0.5.0] - 2025-12-06

### 程序化工具调用 (Programmatic Tool Calling, PTC)
- **PTC 包**: 新增 `ptc` 包，支持程序化工具调用 (#31)。
  - LLM 生成代码直接调用工具，无需 API 往返
  - 支持 Python 和 Go 代码执行
  - 两种执行模式：`ModeDirect`（子进程，默认）和 `ModeServer`（HTTP 服务器，备选）
  - 对于复杂工具链，延迟和 token 使用降低高达 10 倍
  - 多 LLM 支持（OpenAI、Gemini、Claude 及任何 langchaingo 兼容模型）

### PTC 特性
- **代码执行器**: 在沙箱环境中执行 LLM 生成的 Python/Go 代码，具有工具访问权限
- **工具服务器**: 通过 REST API 安全暴露工具的 HTTP 服务器
- **智能代码生成**: 自动为 Python 和 Go 生成工具包装器
- **错误处理**: 完善的错误报告，包含执行输出和调试信息
- **文档**: 完整的中英双语文档，附带 Mermaid 流程图

### PTC 示例
- **[PTC Basic](./examples/ptc_basic/)**: PTC 入门，包含计算器、天气和数据处理工具
- **[PTC Simple](./examples/ptc_simple/)**: 简单计算器示例，演示基本 PTC 用法
- **[PTC Expense Analysis](./examples/ptc_expense_analysis/)**: 基于 Anthropic PTC Cookbook 的复杂场景，展示数据过滤和聚合

### 设计模式
- **规划模式 (Planning Pattern)**: 添加规划模式，支持任务分解和执行规划 (#24)
- **反思代理 (Reflection Agent)**: 实现反思-行动循环模式，支持自我评估和质量改进 (#32)

### 示例展示与文档
- **GPT Researcher**: 完整复刻 assafelovic/gpt-researcher (#34)
  - 自动化研究和报告生成
  - 多源信息整合
- **Trading Agents**: 合并文档文件，创建综合 README (#39)
  - 整合 PROJECT_SUMMARY.md 和 USAGE.md 到 README.md
  - 添加详细使用指南、详细模式示例和 API 参考
- **Open Deep Research**: 将 WORKFLOW.md 合并到 README 文件 (#38)
  - 添加 5 个详细的 Mermaid 工作流程图
  - 包含关键概念：状态累积、消息序列、并行执行
- **Health Insights Agent**: 将 PROJECT_SUMMARY_CN.md 合并到 README_CN.md (#37)
  - 添加技术架构、性能指标和安全考虑
- **DeepAgents**: 添加全面文档 (#36)
  - 完整的工具参考和最佳实践指南
- **DeerFlow 和 BettaFish**: 更新两个示例的文档 (#35)

### 代理文档
- **CreateAgent 和 CreateReactAgent**: 添加全面的对比文档 (#33)
  - 详细的 API 参考和使用示例
  - 最佳实践和用例指导

### 网站与知识库
- **官方网站**: http://lango.rpcx.io (源码: https://github.com/smallnest/lango-website)
  - 233 个 HTML 页面，支持中英双语
  - 16+ 个详细指南（快速开始、高级特性、状态管理等）
  - 展示 6 个完整项目的案例画廊
  - 包含 20+ 个代码示例的示例页面
- **Wiki 知识库**: 193 个 Markdown 文档，涵盖：
  - 高级特性（人机协作、可视化、子图、并行执行）
  - 检查点存储（SQLite、Redis、PostgreSQL）
  - 工具集成指南
  - 预构建组件和 RAG 指南

### 文档整合
- 简化文档结构，采用更清晰的命名约定
- 将分散的文档合并到综合 README 文件
- 改进所有示例展示的导航和可发现性

## [0.4.0] - 2025-12-04

### 核心与代理 (Core & Agents)
- **MCP 支持**: 添加了对模型上下文协议 (Model Context Protocol, MCP) 的支持 (#21)。
- **技能集成 (Skills Integration)**:
  - 添加了对 **Claude Skills** 的支持 (#20)。
  - 更新了 `CreateAgent` 以支持动态技能加载 (#20)。
- **LLM 提供商**: 在 BettaFish 示例中添加了对其他 OpenAI 兼容 LLM 提供商的支持。

### 工具 (Tools)
- **搜索工具**:
  - 添加了 **Brave Search** API 支持。
  - 添加了 **Bocha Search** 工具 (#22)。

### 示例展示 (Showcases)
- **DeerFlow**: 更新了 DeerFlow 示例。
- **BettaFish**: 添加了复刻 BettaFish (https://github.com/666ghj/BettaFish) 的新示例 (#19)。

### 文档与网站 (Documentation & Website)
- **网站**: 将网站内容迁移至 https://github.com/smallnest/lango-website。
- **DIFF.md**: 为示例展示添加了 DIFF.md (#19)。

## [0.3.0] - 2025-12-01

### 核心运行时 (Core Runtime)
- **并行执行**: 实现了扇出/扇入 (Fan-out/Fan-in) 执行模型，支持线程安全的状态合并。
- **运行时配置**: 添加了 `RunnableConfig`，用于在图执行上下文中传递配置（如线程 ID、用户 ID 等）。
- **Command API**: 引入了 `Command` 结构体，支持直接从节点进行动态流控制 (`Goto`) 和状态更新 (`Update`)。
- **子图 (Subgraphs)**: 添加了原生支持，通过将编译后的图作为节点来组合图 (`AddSubgraph`)。

### 持久化与检查点 (Persistence & Checkpointing)
- **检查点接口**: 优化了 `CheckpointSaver` 接口以支持状态持久化。
- **实现**: 增加了对 **Redis**、**PostgreSQL** 和 **SQLite** 检查点存储的完整支持。

### 高级状态与流式处理 (Advanced State & Streaming)
- **状态管理**: 引入了 `Schema` 接口和 `Annotated` 风格的归约器 (Reducers)（例如 `AppendMessages`），用于复杂的状态更新。
- **智能消息 (Smart Messages)**: 实现了 `AddMessages` 归约器，用于基于 ID 的消息更新 (Upsert) 和去重。
- **增强流式处理**: 添加了类型化的 `StreamEvent` 和 `CallbackHandler` 接口。实现了多种流式模式：`updates`, `values`, `messages`, 和 `debug`。

### 预构建代理 (Pre-built Agents)
- **ToolExecutor**: 添加了用于执行工具的专用节点。
- **ReAct Agent**: 实现了用于创建 ReAct 风格代理的工厂方法。
- **Create Agent**: 添加了 `CreateAgent` 工厂，支持函数式选项以灵活创建代理。
- **Supervisor**: 添加了对 Supervisor 代理模式的支持，用于多代理编排。

### 人机交互 (Human-in-the-loop, HITL)
- **中断 (Interrupts)**: 实现了 `InterruptBefore` 和 `InterruptAfter` 机制以暂停图的执行。
- **恢复与命令 (Resume & Command)**: 添加了通过命令恢复执行和更新状态的支持。
- **时间旅行 (Time Travel)**: 实现了 `GetState` 和 `UpdateState` API 以检查/修改过去的检查点并分叉执行历史。

### 可视化 (Visualization)
- **Mermaid 导出**: 改进了图的可视化，优化了条件边和样式的渲染。

### 实验性与研究 (Experimental & Research)
- **Swarm 模式**: 使用子图 (`examples/swarm`) 添加了多代理协作的原型。
- **Channels RFC**: 添加了 `RFC_CHANNELS.md`，提议在未来改进中采用基于 Channel 的架构。

### LangChain 集成 (LangChain Integration)
- **VectorStore 适配器**: 添加了 `LangChainVectorStore` 适配器，可集成任何 langchaingo vectorstore 实现。
- **支持的后端**: 完整支持 Chroma、Weaviate、Pinecone、Qdrant、Milvus、PGVector 以及任何其他 langchaingo vectorstore。
- **统一接口**: 通过标准的 `AddDocuments`、`SimilaritySearch` 和 `SimilaritySearchWithScore` 方法与 RAG 管道无缝集成。
- **完整适配器**: 现在包含 langchaingo 的 DocumentLoaders、TextSplitters、Embedders 和 VectorStores 适配器。

### 工具与集成 (Tools & Integrations)
- **Tool 包**: 添加了新的 `tool` 包，便于集成外部工具。
- **搜索工具**: 实现了与 `langchaingo` 接口兼容的 `TavilySearch` 和 `ExaSearch` 工具。
- **Agent 集成**: 更新了 `ReAct` Agent 以支持为 OpenAI 兼容 API 生成工具参数 Schema 和解析参数。
- **GoSkills 适配器**: 添加了 `adapter/goskills` 以集成 [GoSkills](github.com/smallnest/goskills) 作为工具。

### 示例 (Examples)
- 添加了涵盖以下内容的综合示例：
  - 检查点 (Postgres, SQLite, Redis)
  - 人机交互工作流
  - Swarm 多代理模式
  - 子图
  - **智能消息 (Smart Messages)** (新增)
  - **Command API** (新增)
  - **临时通道 (Ephemeral Channels)** (新增)
  - **流式模式 (Streaming Modes)** (新增)
  - **时间旅行 / HITL** (新增)
  - **LangChain VectorStore 集成** (新增)
  - **Chroma 向量数据库集成** (新增)
  - **Tavily 搜索工具** (新增)
  - **Exa 搜索工具** (新增)
  - **Create Agent** (新增)
  - **动态技能代理 (Dynamic Skill Agent)** (新增)
  - **Durable Execution** (新增)
  - **GoSkills 集成** (新增)
- **通用**: 改进了所有示例的可靠性和正确性。

## [0.1.0] - 2025-01-02

### 新增
- 通用状态管理 - 适用于任何类型，不仅仅是 MessageContent
- 针对生产环境的性能优化
- 支持任何 LLM 客户端（移除了对 LangChain 的硬依赖）

### 变更
- 简化了构建图的 API
- 更新了示例以展示通用用法

### 修复
- 原始仓库中的 CI/CD 流水线问题
- 最新 Go 版本的构建错误

### 移除
- 对 LangChain 的硬依赖 - 现在可以与任何 LLM 库一起工作
