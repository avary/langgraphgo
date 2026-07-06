# 泛型 StateGraph 示例

本示例展示了 LangGraphGo 中**类型安全的泛型 StateGraph** 实现。

## 概览

泛型 StateGraph 为状态管理提供编译时类型安全，消除了类型断言的需求，减少运行时错误。

## 主要优势

✅ **编译时类型安全** - 在运行前捕获错误
✅ **无需类型断言** - 直接访问状态字段
✅ **更好的 IDE 支持** - 完整的自动补全和重构
✅ **更清晰的代码** - 更少样板代码，更易读
✅ **零运行时开销** - 泛型仅在编译时存在

## 包含的示例

### 示例 1：简单的类型安全图

展示了检查用户资格的基本用法：

```go
g := graph.NewStateGraph[WorkflowState]()

g.AddNode("check_age", "检查年龄", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
    state.IsAdult = state.Request.Age >= 18  // 类型安全！
    return state, nil
})
```

### 示例 2：条件路由

展示类型安全的条件边：

```go
g.AddConditionalEdge("check_age", func(ctx context.Context, state WorkflowState) string {
    if state.IsAdult {  // 无需类型断言！
        return "adult_path"
    }
    return "minor_path"
})
```

### 示例 3：基于 Schema 的状态合并

展示使用自定义合并逻辑的高级状态管理：

```go
schema := graph.NewStructSchema(
    ProcessState{MaxCount: 5},
    func(current, new ProcessState) (ProcessState, error) {
        // 自定义合并逻辑
        current.Items = append(current.Items, new.Items...)
        current.Count += new.Count
        return current, nil
    },
)
g.SetSchema(schema)
```

## 运行示例

```bash
cd examples/generic_state_graph
go run main.go
```

## 对比：泛型 vs 非泛型

### 非泛型（旧方式）

```go
g := graph.NewStateGraph[map[string]any]()

g.AddNode("process", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(WorkflowState)  // 需要类型断言 ❌
    s.Count++
    return s, nil
})

result, _ := app.Invoke(ctx, initialState)
finalState := result.(WorkflowState)  // 又一次类型断言 ❌
```

### 泛型（新方式）

```go
g := graph.NewStateGraph[WorkflowState]()

g.AddNode("process", "desc", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
    state.Count++  // 直接访问 ✅
    return state, nil
})

finalState, _ := app.Invoke(ctx, initialState)  // 类型安全的结果 ✅
```

## 何时使用泛型 StateGraph

**使用泛型 StateGraph 的场景：**
- ✅ 有明确定义的状态结构体
- ✅ 类型安全很重要
- ✅ 构建新项目
- ✅ 希望获得更好的 IDE 支持

**使用非泛型 StateGraph 的场景：**
- ✅ 需要最大的灵活性
- ✅ 状态结构是动态的
- ✅ 使用 `map[string]any` 配合复杂的 reducer
- ✅ 从 Python LangGraph 迁移

## 从非泛型迁移

迁移非常简单：

1. **更改构造函数：**
   ```go
   // 之前
   g := graph.NewStateGraph[map[string]any]()

   // 之后
   g := graph.NewStateGraph[MyState]()
   ```

2. **更新节点函数：**
   ```go
   // 之前
   func(ctx context.Context, state any) (any, error) {
       s := state.(MyState)
       // ...
   }

   // 之后
   func(ctx context.Context, state MyState) (MyState, error) {
       // 直接访问状态字段
   }
   ```

3. **边定义无需更改：**
   ```go
   g.AddEdge("from", "to")  // 与之前相同
   g.SetEntryPoint("start")  // 与之前相同
   ```

4. **更新调用：**
   ```go
   // 之前
   result, _ := app.Invoke(ctx, initialState)
   finalState := result.(MyState)

   // 之后
   finalState, _ := app.Invoke(ctx, initialState)
   ```

就这样！你的图现在是类型安全的了。

## 了解更多

- [RFC: 泛型 StateGraph 设计](../../docs/RFC_GENERIC_STATEGRAPH.md)
- [LangGraphGo 文档](../../README.md)
- [英文版 README](README.md)
