# Examples修复总结：MessageGraph → StateGraph

## 问题描述

在重构MessageGraph后，`NewMessageGraph()`现在默认包含messages schema，期待状态类型为`map[string]any`并包含"messages"键。但是很多examples使用的是：
- 简单字符串状态
- 自定义结构体（如Task, Document, MarketState等）
- LangChain的`[]llms.MessageContent`类型

这些状态类型与messages schema不兼容，导致运行时错误。

## 修复方案

将所有**不需要messages schema**的examples从`NewMessageGraph()`改为`NewStateGraph()`。

## 修改的文件

### 批次1：使用简单状态的examples（字符串、自定义结构体）

1. **examples/basic_example/main.go**
   - 状态类型：字符串
   - 修改：`NewMessageGraph()` → `NewStateGraph()`
   - 结果：✓ 编译通过，运行正常

2. **examples/dynamic_interrupt/main.go**
   - 状态类型：字符串、nil
   - 修改：`NewMessageGraph()` → `NewStateGraph()`
   - 结果：✓ 编译通过

3. **examples/conditional_routing/main.go**
   - 状态类型：Task结构体
   - 修改：`NewMessageGraph()` → `NewStateGraph()`
   - 结果：✓ 编译通过，运行正常

4. **examples/configuration/main.go**
   - 状态类型：字符串
   - 修改：`NewMessageGraph()` → `NewStateGraph()`
   - 结果：✓ 编译通过，运行正常

5. **examples/visualization/main.go**
   - 状态类型：字符串
   - 修改：`NewMessageGraph()` → `NewStateGraph()`
   - 结果：✓ 编译通过

6. **examples/subgraph/main.go**
   - 状态类型：Document结构体
   - 修改：所有`NewMessageGraph()` → `NewStateGraph()`（包括子图）
   - 结果：✓ 编译通过

7. **examples/subgraphs/main.go**
   - 状态类型：自定义结构体
   - 修改：所有`NewMessageGraph()` → `NewStateGraph()`
   - 结果：✓ 编译通过

8. **examples/mental_loop/main.go**
   - 状态类型：MarketState结构体
   - 修改：`NewMessageGraph()` → `NewStateGraph()`
   - 结果：✓ 编译通过

### 批次2：使用LangChain消息类型的examples

9. **examples/basic_llm/main.go**
   - 状态类型：`[]llms.MessageContent`
   - 修改：`NewMessageGraph()` → `NewStateGraph()`
   - 原因：状态是消息数组，不是map[string]any
   - 结果：✓ 编译通过

10. **examples/langchain_example/main.go**
    - 状态类型：`[]llms.MessageContent`
    - 修改：所有`NewMessageGraph()` → `NewStateGraph()`
    - 结果：✓ 编译通过

11. **examples/conditional_edges_example/main.go**
    - 状态类型：`[]llms.MessageContent`
    - 修改：所有`NewMessageGraph()` → `NewStateGraph()`（3处）
    - 结果：✓ 编译通过

12. **examples/rag_pipeline/main.go**
    - 状态类型：DocumentState结构体
    - 修改：`NewMessageGraph()` → `NewStateGraph()`
    - 结果：✓ 编译通过

## 特殊Graph类型（无需修改）

以下examples使用特殊的Graph类型，它们内部已经使用MessageGraph（现在是StateGraph的别名），无需修改：

- **examples/listeners/main.go** - 使用`NewListenableMessageGraph()`
- **examples/streaming_pipeline/main.go** - 使用`NewStreamingMessageGraph()`
- **examples/streaming_modes/main.go** - 使用`NewStreamingMessageGraph()`
- **examples/checkpointing/main.go** - 使用`NewCheckpointableMessageGraph()`
- **examples/checkpointing/sqlite/main.go** - 使用`NewCheckpointableMessageGraph()`
- **examples/checkpointing/redis/main.go** - 使用`NewCheckpointableMessageGraph()`
- **examples/checkpointing/postgres/main.go** - 使用`NewCheckpointableMessageGraph()`
- **examples/durable_execution/main.go** - 使用`NewCheckpointableMessageGraph()`
- **examples/time_travel/main.go** - 使用`NewCheckpointableMessageGraph()`

这些特殊类型已验证：
- ✓ listeners - 编译通过
- ✓ streaming_modes - 编译通过
- ✓ checkpointing - 编译通过

## 统计

- **修改的文件数量**：12个
- **修改的NewMessageGraph()调用**：约26处
- **保持不变的特殊Graph**：9个examples
- **所有修改后的examples**：✓ 编译通过

## 验证

### 编译验证
所有修改的examples都成功编译：
```bash
✓ basic_example builds successfully
✓ dynamic_interrupt builds successfully
✓ conditional_routing builds successfully
✓ visualization builds successfully
✓ configuration builds successfully
✓ basic_llm builds successfully
✓ rag_pipeline builds successfully
✓ listeners builds successfully
✓ streaming_modes builds successfully
✓ checkpointing builds successfully
```

### 运行验证
测试了关键examples的运行：
- **basic_example**: 所有4个测试正常运行，结果正确
- **conditional_routing**: 正确路由到不同handler
- **configuration**: 正确读取和使用配置

## API使用指南

### 何时使用NewStateGraph()
```go
// 1. 使用简单状态（字符串、数字等）
g := graph.NewStateGraph[map[string]any]()

// 2. 使用自定义结构体
type MyState struct { ... }
g := graph.NewStateGraph[map[string]any]()

// 3. 使用LangChain消息但作为状态本身
g := graph.NewStateGraph[map[string]any]() // 状态是 []llms.MessageContent
```

### 何时使用NewMessageGraph()
```go
// 仅当状态是 map[string]any 并且需要"messages"键时
g := graph.NewMessageGraph()
// 状态示例：
state := map[string]any{
    "messages": []map[string]any{
        {"role": "user", "content": "Hello"},
    },
}
```

### 特殊Graph类型
```go
// 需要监听器功能
g := graph.NewListenableMessageGraph()

// 需要流式输出
g := graph.NewStreamingMessageGraph()

// 需要检查点功能
g := graph.NewCheckpointableMessageGraph()
```

## 总结

成功将12个examples从`NewMessageGraph()`迁移到`NewStateGraph()`，消除了运行时schema类型不匹配的错误。所有examples现在都使用正确的Graph构造函数，根据其实际状态类型需求。

特殊Graph类型（Listenable、Streaming、Checkpointable）继续正常工作，因为MessageGraph现在是StateGraph的别名。
