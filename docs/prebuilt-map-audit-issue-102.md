# prebuilt `Create*Agent` / `*Map` 变体专项评估 (Issue #102)

评估日期: 2026-07-09 · 范围: `prebuilt/` 的 `Create*` 导出函数（46 个导出函数中的成对 typed/Map 入口）
方法: 逐个读取每对 `Create<X>Agent[S]` 与 `Create<X>AgentMap` 的函数体，比对图构建逻辑与 state 语义；全仓库交叉引用统计（examples / 内部 / 测试）。

> 结论先行: **`*Map` 变体不是薄包装**。它们是与泛型版**并行的完整实现**，且额外承载了 `map` 专属的 **schema + reducer** 语义。因此本评估落在 issue 的**选项 3（承载实质差异，保留并文档说明取舍）**，而非选项 2（薄封装、可用单一泛型适配层替代）。不建议合并入口；建议补文档说明取舍，并把"节点逻辑去重"作为可选的、独立的后续重构。

---

## 1. 成对入口清单

| Agent | 泛型入口 `Create<X>Agent[S]` | Map 入口 `Create<X>AgentMap` |
|---|---|---|
| Agent（通用） | `create_agent.go:332` | `create_agent.go:105` |
| ReAct | `react_agent.go:173` | `react_agent.go:17` |
| PEV | `pev_agent.go:202` | `pev_agent.go:65` |
| Planning | `planning_agent.go:134` | `planning_agent.go:16` |
| Reflection | `reflection_agent.go:112` | `reflection_agent.go:23` |
| Supervisor | `supervisor.go:103` | `supervisor.go:14` |
| TreeOfThoughts | `tree_of_thoughts.go:142` | `tree_of_thoughts.go:40` |

---

## 2. 封装厚度：`*Map` 不是薄包装

对每一对，`*Map` 与泛型版都**各自独立地** `graph.NewStateGraph(...)`、`AddNode(...)` 并写入全部节点闭包逻辑。`*Map` 不调用泛型版，泛型版也不调用 `*Map`——两者是逐节点重写的并行实现（每个约 150 行）。

**两者的实质差异（不是可机械消除的重复）:**

| 维度 | `Create<X>AgentMap` | `Create<X>Agent[S]`（泛型） |
|---|---|---|
| 状态类型 | `map[string]any` | 任意 `S` |
| 状态访问 | 节点内直接用字符串 key + 类型断言（`state["messages"].([]llmtypes.MessageContent)`） | 调用方注入 getter/setter 闭包 |
| **Schema / Reducer** | **注册 `MapSchema` + `RegisterReducer`**（如 `messages`→`AppendReducer`、`workflow_plan`→`OverwriteReducer`） | **不注册任何 schema/reducer**；由 setter 闭包直接算出新 state |
| 调用方心智成本 | 低（只传 model/tools/config） | 高（PEV 需注入约 8 组 getter/setter，共 16 个函数参数） |

**关键点:** 只有 `*Map` 版调用 `SetSchema`/`RegisterReducer`（见 `react_agent.go:28-30`、`pev_agent.go:82-85`、`supervisor.go:16-18`、`planning_agent.go:28-31`）。泛型版**完全不注册 reducer**，依赖 setter 返回全量新状态。这意味着并行节点的状态合并语义两者不同——把 `*Map` 改写成"转调泛型版 + 标准 map 访问器"的薄适配层，会**丢失 reducer 合并行为**，属于行为回归风险，而非无损精简。

---

## 3. 交叉引用统计

| 入口 | 总引用（除定义处） | examples | 测试文件 | 判定 |
|---|---|---|---|---|
| `CreateAgentMap` | 24 | 13 | 1 | 重度使用（example 主力 API） |
| `CreateSupervisorMap` | 14 | 0 | 1 | 内部+测试重度使用 |
| `CreateTreeOfThoughtsAgentMap` | 13 | 1 | 1 | 使用中 |
| `CreateReflectionAgentMap` | 8 | 3 | 2 | 使用中 |
| `CreatePlanningAgentMap` | 7 | 1 | 2 | 使用中 |
| `CreatePEVAgentMap` | 4 | 1 | 1 | 使用中 |
| `CreateReactAgentMap` | 3 | 1 | 1 | 使用中 |
| 泛型 `CreateReactAgent` | 26 | 0 | — | 内部/测试为主 |
| 泛型 `CreateTreeOfThoughtsAgent` | 9 | 0 | — | 内部/测试为主 |
| 泛型 `CreatePlanningAgent` | 7 | 0 | — | 内部/测试为主 |
| 泛型 `CreateAgent` | 7 | 0 | — | 内部/测试为主 |
| 泛型其余 | 3–5 | 0 | — | 使用中 |

**观察:** `*Map` 是 examples 面向用户的主力 API（`CreateAgentMap` 单独就有 13 处 example 引用），泛型版几乎只在内部/测试出现。删除或重命名 `*Map` 会**大面积破坏 examples**——issue "收益/风险" 中点名要确认的"无外部强绑定"这一前提**不成立**，反向支持保留。

---

## 4. 对照 issue 的三个选项

- **选项 1（确认封装厚度）** ✅ 已确认：`*Map` 是并行完整实现 + map-schema/reducer 语义，非薄包装。
- **选项 2（薄封装→单一泛型适配层替代）** ❌ 不适用：并非薄封装；机械收敛会丢失 reducer 合并语义并破坏 13+ 处 example。
- **选项 3（承载实质差异→保留并文档说明取舍）** ✅ **采纳**：保留两套入口，在 `doc.go` 说明"何时用 typed、何时用 Map"的取舍。

---

## 5. 建议

1. **保留全部 `*Map` 与泛型入口**——两者语义不同、`*Map` 是 example 主力 API，删除/合并收益不抵回归与破坏成本。
2. **补文档取舍说明（低风险，建议本次落地）**：在 `prebuilt/doc.go` 增加一节 "Typed vs Map constructors"，说明：
   - `*Map`：开箱即用、无需注入访问器、自带 `messages` 等 reducer 合并——适合快速上手与大多数 example 场景。
   - `Create*Agent[S]`：编译期类型安全、状态结构自定义、但需注入 getter/setter 且不自带 reducer——适合已有强类型 state 的集成。
3. **（可选、独立后续）节点逻辑去重**：若要减少两套实现里重复的**节点闭包逻辑**（prompt 构造、工具定义、路由判断等），应抽取与 state 表示无关的 **helper**（如 `buildRouteTool`、`buildPlanningPrompt` 已是此模式），而**不是**合并两个入口。这是一次更大的重构，需单独立 issue 并配回归测试。
4. **不建议**：把 `*Map` 改写为转调泛型版的薄适配层——会丢失 map-schema 的 reducer 合并语义。

> 净精简空间: **导出面无法安全缩减**（两套入口都是真实、有区分度、被广泛引用的 API）。本 issue 的实际产出是"确认 + 文档取舍"，而非删代码——这与 #103 的审计结论一致：`prebuilt`/`rag` 里的成对 API 大多是有实质差异的实现，而非薄变体。
