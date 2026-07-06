# RAG 架构设计评估与建议

目前的 `rag` 包虽然功能强大（支持 GraphRAG、向量检索、混合检索），但要在 `LangGraph` 中使用它，开发者确实需要编写不少“胶水代码”。

为了提升开发体验（DX），我们建议实现更高层级的抽象。

建议的路线图是：**`RetrievalNode` (封装成节点)** + **`RetrieverTool` (封装成工具)**。`RagAgent` 可以作为一个预构建的 Graph 模版存在，而不是一个硬编码的 Struct。

以下是详细评估：

### 1. 为什么需要 `RetrievalNode` (推荐)

在 LangGraph 中，最常见的模式是将 RAG 作为一个步骤（Node）。目前用户必须手动编写一个函数来从 State 取 Query，调用 Engine，然后写入 Context。

**痛点：**
目前的代码（伪代码）：
```go
// 用户必须自己写这个
func Retrieve(ctx context.Context, state map[string]any) (map[string]any, error) {
    query := state["input"].(string) // 还需要做类型断言
    res, _ := engine.Query(ctx, query)
    return map[string]any{"context": res.Context}, nil
}
```

**解决方案：`RetrievalNode`**
我们可以提供一个工厂函数，自动处理 State 的映射。

**设计建议：**
```go
// 这是一个通用的 RAG 节点生成器
func NewRetrievalNode(engine rag.Engine, inputKey string, outputKey string) func(context.Context, any) (any, error) {
    return func(ctx context.Context, state any) (any, error) {
        // 1. 自动从 State 获取 Query
        m, _ := state.(map[string]any)
        query := m[inputKey].(string)

        // 2. 调用引擎
        result, err := engine.Query(ctx, query)
        if err != nil {
            return nil, err
        }

        // 3. 返回部分 State 更新
        return map[string]any{
            outputKey: result.Context,
        }, nil
    }
}
```

**使用体验提升：**
```go
// 用户只需要这样写：
workflow.AddNode("retrieve", rag.NewRetrievalNode(myEngine, "question", "context"))
```

---

### 2. 为什么需要 `RetrieverTool` (推荐)

如果我们构建的是一个 **ReAct Agent** 或 **Tool Calling Agent**（例如 `chat_agent_dynamic_tools`），Agent 应该根据需要决定*是否*去查资料。这时我们需要把 RAG 引擎包装成一个 `Tool`。

**设计建议：**
```go
type RetrieverTool struct {
    Engine      rag.Engine
    NameVal     string
    DescVal     string
}

func (t *RetrieverTool) Call(ctx context.Context, input string) (string, error) {
    res, err := t.Engine.Query(ctx, input)
    if err != nil {
        return "", err
    }
    // 返回给 LLM 的是 Context 字符串
    return res.Context, nil
}
```

这样你的 GraphRAG 就可以作为一个普通的工具被 Agent 使用了：
```go
agent := create_agent.NewAgent(llm, []Tool{
    rag.NewRetrieverTool(graphEngine, "knowledge_base", "查询内部知识库"),
    &weather.Tool{},
})
```

---

### 3. 关于 `RagAgent` 的评估 (谨慎推荐)

关于是否要实现一个封装了 "Retrieve -> Generate" 固定流程的 `RagAgent` 结构体：

**评估：**
*   **优点**：对于最简单的 "Q&A" 场景，开箱即用，非常快。
*   **缺点**：LangGraph 的核心优势在于**编排复杂流程**（如：Query Rewrite -> Retrieve -> Grade Documents -> Generate）。如果封装成一个黑盒的 `RagAgent`，用户一旦想加一个“评分”步骤，就必须拆掉这个 Agent 重写。

**替代方案：预构建的 Subgraph (子图)**
不要做一个 struct 叫 `RagAgent`，而是提供一个**构建函数**，返回一个配置好的 `Graph`。

```go
// 这是一个标准的 RAG 流程图
func NewRAGGraph(llm LLM, engine rag.Engine) *graph.StateGraph {
    workflow := graph.NewStateGraph[map[string]any]()
    
    workflow.AddNode("retrieve", rag.NewRetrievalNode(engine, ...))
    workflow.AddNode("generate", NewGenerationNode(llm, ...))
    
    workflow.SetEntryPoint("retrieve")
    workflow.AddEdge("retrieve", "generate")
    
    return workflow
}
```
这样用户可以将这个 Graph 作为 Subgraph 嵌入到更大的系统中，或者直接使用它。

### 总结建议

1.  **优先级高 (Must Have):** 实现 **`RetrieverTool`**。因为 Agentic RAG 是趋势，让 Agent 把知识库当工具用是最灵活的。
2.  **优先级中 (Should Have):** 实现 **`NewRetrievalNode`** 辅助函数。这能极大减少构建固定 RAG 流水线的代码量。
3.  **优先级低 (Nice to Have):** `RagAgent`。建议改为提供 `RagWorkflow` 的示例或构造函数，而不是硬编码的 Agent 类。

这样做既保持了 LangGraph 的灵活性，又解决了目前的 boilerplate code 问题。
