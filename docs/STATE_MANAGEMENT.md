# LangGraphGo 状态管理指南

本文档介绍 LangGraphGo 中节点之间如何传递数据，以及在图工作流中管理状态的正确模式。

## 目录

- [概述](#概述)
- [问题所在：返回新 Map 还是修改状态](#问题所在返回新-map-还是修改状态)
- [正确模式：修改并返回状态](#正确模式修改并返回状态)
- [理解状态流转](#理解状态流转)
- [特殊情况](#特殊情况)
- [完整示例](#完整示例)
- [最佳实践](#最佳实践)

---

## 概述

在 LangGraphGo 中，状态会依次流经各个节点。每个节点接收当前状态，可以修改它，并且必须返回（可能已修改的）状态供下一个节点使用。

节点函数的签名如下：

```go
func(ctx context.Context, state S) (S, error)
```

其中 `S` 通常是 `map[string]any`（非类型化图），或自定义结构体（类型化图）。

---

## 问题所在：返回新 Map 还是修改状态

### ❌ 错误模式

一个常见的错误是只创建一个包含新添加字段的新 map：

```go
g.AddNode("process", "process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    // 从 state 读取
    input := state["input"].(string)

    // 处理数据
    result := strings.ToUpper(input)

    // 错误：只返回包含新字段的新 map
    return map[string]any{"output": result}, nil
})

g.AddNode("next_step", "next_step", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    // 错误："input" 不再存在！
    input := state["input"].(string) // panic: interface conversion: nil is not string
    // ...
})
```

**为什么会失败：** `next_step` 节点接收到的 state 只包含 `"output"`。原始的 `"input"` 字段丢失了。

### ✅ 正确模式

修改传入的 state 并返回它：

```go
g.AddNode("process", "process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    // 从 state 读取
    input := state["input"].(string)

    // 处理数据
    result := strings.ToUpper(input)

    // 正确：修改 state 并返回
    state["output"] = result
    return state, nil
})

g.AddNode("next_step", "next_step", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    // 现在可以正常工作！state 同时包含 "input" 和 "output"
    input := state["input"].(string)
    output := state["output"].(string)
    // ...
    return state, nil
})
```

---

## 正确模式：修改并返回状态

### 核心原则

> **状态在流经图的过程中会不断累积。每个节点接收当前的累积状态，可以添加或修改字段，然后必须传递完整的状态。**

### 正确与错误的对比

| 场景 | 错误写法 ❌ | 正确写法 ✅ |
|------|------------|------------|
| 修改现有字段 | `return map[string]any{"field": newValue}, nil` | `state["field"] = newValue; return state, nil` |
| 添加新字段 | `return map[string]any{"new": value}, nil` | `state["new"] = value; return state, nil` |
| 保持现有数据 | `return map[string]any{}, nil` | `return state, nil` |
| 读取后修改 | `return map[string]any{"result": process(state["input"])}, nil` | `state["result"] = process(state["input"]); return state, nil` |

---

## 理解状态流转

考虑这个简单的管道：

```go
g := graph.NewStateGraph[map[string]any]()

// 初始状态: {"value": 5}

g.AddNode("double", "double", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["value"] = state["value"].(int) * 2
    return state, nil
    // 此节点后的状态: {"value": 10}
})

g.AddNode("add_ten", "add_ten", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["value"] = state["value"].(int) + 10
    return state, nil
    // 此节点后的状态: {"value": 20}
})

g.AddNode("square", "square", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["value"] = state["value"].(int) * state["value"].(int)
    return state, nil
    // 此节点后的状态: {"value": 400}
})
```

### 可视化表示

```
初始状态: {"value": 5}
    │
    ▼
┌─────────┐
│ double  │  state["value"] *= 2  →  {"value": 10}
└─────────┘
    │
    ▼
┌──────────┐
│ add_ten  │  state["value"] += 10  →  {"value": 20}
└──────────┘
    │
    ▼
┌─────────┐
│ square  │  state["value"] *= state["value"]  →  {"value": 400}
└─────────┘
    │
    ▼
最终状态: {"value": 400}
```

### 多字段累积示例

```go
// 初始状态: {"user_id": "12345", "context": {}}

// 节点 1
g.AddNode("authenticate", "authenticate", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["authenticated"] = true
    state["user_role"] = "admin"
    return state, nil
    // 状态: {"user_id": "12345", "context": {}, "authenticated": true, "user_role": "admin"}
})

// 节点 2 - 仍然可以访问 user_id
g.AddNode("load_preferences", "load_preferences", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    userID := state["user_id"].(string)  // ✅ 可以访问
    state["preferences"] = map[string]any{"theme": "dark"}
    return state, nil
    // 状态: {"user_id": "12345", "context": {}, "authenticated": true, "user_role": "admin", "preferences": {...}}
})

// 节点 3 - 可以访问所有之前添加的字段
g.AddNode("log_action", "log_action", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    role := state["user_role"].(string)  // ✅ 可以访问
    prefs := state["preferences"].(map[string]any)  // ✅ 可以访问
    state["logged"] = true
    return state, nil
})
```

---

## 特殊情况

### 1. 使用 Reducer（归约器）

Reducer 定义了如何合并状态字段，特别是在并行执行时。LangGraphGo 提供了多种内置 Reducer。

#### 可用的 Reducer 类型

#### A. MapSchema 使用的 Reducer（用于 `map[string]any`）

| Reducer | 功能 | 使用场景 |
|---------|------|----------|
| `OverwriteReducer` | 用新值覆盖旧值 | 默认行为，替换字段值 |
| `AppendReducer` | 将新值追加到切片 | 累积列表、日志、消息等 |

#### B. FieldMerger 使用的字段合并函数（用于结构体）

| 函数 | 功能 | 使用场景 |
|------|------|----------|
| `AppendSliceMerge` | 追加切片到当前切片 | 累积列表数据 |
| `SumIntMerge` | 整数相加 | 计数、总和 |
| `OverwriteMerge` | 用新值覆盖 | 替换字段值 |
| `KeepCurrentMerge` | 保留当前值，忽略新值 | 保护某些字段不被修改 |
| `MaxIntMerge` | 取最大值 | 记录最大值 |
| `MinIntMerge` | 取最小值 | 记录最小值 |

#### 基本用法

```go
schema := graph.NewMapSchema()
schema.RegisterReducer("logs", graph.AppendReducer)      // 日志累积
schema.RegisterReducer("items", graph.AppendReducer)     // 列表累积
schema.RegisterReducer("status", graph.OverwriteReducer) // 覆盖状态（默认）
g.SetSchema(schema)

g.AddNode("add_items", "add_items", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["items"] = []string{"苹果", "香蕉"}
    return state, nil
})

g.AddNode("add_more", "add_more", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["items"] = []string{"樱桃"}
    return state, nil
    // 最终 state["items"] 将是: ["苹果", "香蕉", "樱桃"]
})
```

#### 并行执行中的 Reducer

Reducer 主要用于并行执行时合并多个分支的结果：

```go
package main

import (
    "context"
    "fmt"

    "github.com/smallnest/langgraphgo/graph"
)

func main() {
    g := graph.NewStateGraph[map[string]any]()

    // 配置不同字段的合并器
    schema := graph.NewMapSchema()
    schema.RegisterReducer("logs", graph.AppendReducer)       // 追加日志
    schema.RegisterReducer("tags", graph.AppendReducer)       // 追加标签

    // 自定义 Reducer：累加计数
    schema.RegisterReducer("count", func(current, new any) (any, error) {
        curr := 0
        if current != nil {
            curr = current.(int)
        }
        return curr + new.(int), nil
    })

    // 自定义 Reducer：取最大值
    schema.RegisterReducer("max_value", func(current, new any) (any, error) {
        curr := 0
        if current != nil {
            curr = current.(int)
        }
        newVal := new.(int)
        if newVal > curr {
            return newVal, nil
        }
        return curr, nil
    })

    g.SetSchema(schema)

    // 并行执行分支 A
    g.AddNode("branch_a", "branch_a", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        state["logs"] = []string{"Branch A 执行"}
        state["tags"] = []string{"a"}
        state["count"] = 5
        state["max_value"] = 10
        state["status"] = "A完成"
        return state, nil
    })

    // 并行执行分支 B
    g.AddNode("branch_b", "branch_b", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        state["logs"] = []string{"Branch B 执行"}
        state["tags"] = []string{"b"}
        state["count"] = 3
        state["max_value"] = 20
        state["status"] = "B完成"  // 会覆盖 A 的值（默认 OverwriteReducer）
        return state, nil
    })

    g.SetEntryPoint("branch_a")
    g.AddEdge("branch_a", "branch_b")
    g.AddEdge("branch_b", graph.END)

    runnable, _ := g.Compile()
    result, _ := runnable.Invoke(context.Background(), map[string]any{})

    fmt.Printf("logs: %v\n", result["logs"])           // ["Branch A 执行", "Branch B 执行"]
    fmt.Printf("tags: %v\n", result["tags"])           // ["a", "b"]
    fmt.Printf("count: %v\n", result["count"])         // 8 (5+3)
    fmt.Printf("max_value: %v\n", result["max_value"]) // 20 (max(10, 20))
    fmt.Printf("status: %v\n", result["status"])       // "B完成" (覆盖)
}
```

#### 结构体使用 FieldMerger

```go
type MyState struct {
    Count   int
    Logs    []string
    MaxVal  int
    Status  string
}

merger := graph.NewFieldMerger(MyState{})
merger.RegisterFieldMerge("Count", graph.SumIntMerge)           // 累加
merger.RegisterFieldMerge("Logs", graph.AppendSliceMerge)      // 追加
merger.RegisterFieldMerge("MaxVal", graph.MaxIntMerge)         // 取最大
merger.RegisterFieldMerge("Status", graph.OverwriteMerge)      // 覆盖
```

#### 自定义 Reducer 签名

```go
type Reducer func(current, new any) (any, error)
```

**重要提示：** 即使使用了 Reducer，节点函数仍然需要返回完整的 `state`！Reducer 只是在合并多个节点的结果时才被调用。

### 2. 并行执行

在并行执行中，状态合并器会合并所有分支的结果：

```go
g.AddNode("branch_a", "branch_a", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["result_a"] = "来自 A"
    return state, nil
})

g.AddNode("branch_b", "branch_b", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["result_b"] = "来自 B"
    return state, nil
})

// 两个分支完成后，state 同时包含两个结果
// {"result_a": "来自 A", "result_b": "来自 B"}
```

**重要：** 每个并行分支都接收完整的初始状态，返回的 state 会被框架合并。

### 3. 自定义状态合并器

如果需要自定义合并逻辑，可以设置状态合并器：

```go
g.SetStateMerger(func(ctx context.Context, current any, results []any) (any, error) {
    state := current.(map[string]any)
    if state == nil {
        state = make(map[string]any)
    }
    for _, res := range results {
        resMap := res.(map[string]any)
        for k, v := range resMap {
            state[k] = v  // 这里编写你的自定义合并逻辑
        }
    }
    return state, nil
})
```

**注意：** 设置了自定义合并器后，返回新 map 可能是可以接受的，因为合并器会正确处理。但这仍然不是推荐的做法。

### 4. 使用 Command API

当使用带有 `Update` 字段的 `Command` API 时：

```go
g.AddNode("router", "router", func(ctx context.Context, state any) (any, error) {
    return &graph.Command{
        Goto:   "next_node",
        Update: map[string]any{"path": "high"},  // 这里部分更新是可以的
    }, nil
})
```

`Command.Update` 专为部分更新设计，框架会处理合并。这是唯一推荐的部分更新模式。

---

## 完整示例

### 示例 1：简单数据管道

```go
package main

import (
    "context"
    "fmt"

    "github.com/smallnest/langgraphgo/graph"
)

func main() {
    g := graph.NewStateGraph[map[string]any]()

    // 步骤 1: 获取数据
    g.AddNode("fetch", "fetch", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        state["data"] = "原始数据"
        return state, nil
    })

    // 步骤 2: 转换数据（依赖 fetch 产生的 "data"）
    g.AddNode("transform", "transform", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        data := state["data"].(string)
        state["data"] = fmt.Sprintf("已处理: %s", data)
        state["transformed"] = true
        return state, nil
    })

    // 步骤 3: 保存（依赖 "data" 和 "transformed"）
    g.AddNode("save", "save", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        data := state["data"].(string)
        fmt.Printf("保存: %s\n", data)
        state["saved"] = true
        return state, nil
    })

    g.SetEntryPoint("fetch")
    g.AddEdge("fetch", "transform")
    g.AddEdge("transform", "save")
    g.AddEdge("save", graph.END)

    runnable, _ := g.Compile()
    result, _ := runnable.Invoke(context.Background(), map[string]any{})

    // 最终状态包含所有累积的数据
    // {"data": "已处理: 原始数据", "transformed": true, "saved": true}
    fmt.Printf("最终状态: %v\n", result)
}
```

### 示例 2：多字段累积

```go
package main

import (
    "context"
    "fmt"

    "github.com/smallnest/langgraphgo/graph"
)

func main() {
    g := graph.NewStateGraph[map[string]any]()

    // 初始状态可以有多个字段
    initialState := map[string]any{
        "user_id": "12345",
        "context": map[string]any{},
        "history": []string{},
    }

    g.AddNode("authenticate", "authenticate", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        // 添加新字段，保留现有字段
        state["authenticated"] = true
        state["user_role"] = "admin"
        return state, nil
    })

    g.AddNode("load_preferences", "load_preferences", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        // 仍然可以访问初始状态的 user_id
        userID := state["user_id"].(string)
        fmt.Printf("为用户 %s 加载偏好设置\n", userID)
        state["preferences"] = map[string]any{
            "theme":  "dark",
            "lang":   "zh-CN",
        }
        return state, nil
    })

    g.AddNode("log_action", "log_action", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        // 所有之前的数据都可用
        history := state["history"].([]string)
        state["history"] = append(history, "用户已认证")
        return state, nil
    })

    g.AddNode("build_response", "build_response", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        // 可以访问所有累积的字段
        role := state["user_role"].(string)
        prefs := state["preferences"].(map[string]any)
        theme := prefs["theme"].(string)

        response := map[string]any{
            "welcome": fmt.Sprintf("欢迎回来，%s！", role),
            "theme":   theme,
        }
        state["response"] = response
        return state, nil
    })

    g.SetEntryPoint("authenticate")
    g.AddEdge("authenticate", "load_preferences")
    g.AddEdge("load_preferences", "log_action")
    g.AddEdge("log_action", "build_response")
    g.AddEdge("build_response", graph.END)

    runnable, _ := g.Compile()
    result, _ := runnable.Invoke(context.Background(), initialState)

    fmt.Printf("\n最终状态:\n")
    for k, v := range result {
        fmt.Printf("  %s: %v\n", k, v)
    }
}
```

输出：
```
为用户 12345 加载偏好设置

最终状态:
  user_id: 12345
  context: map[]
  history: [用户已认证]
  authenticated: true
  user_role: admin
  preferences: map[lang:zh-CN theme:dark]
  response: map[theme:dark welcome:欢迎回来，admin！]
```

### 示例 3：条件路由与状态

```go
package main

import (
    "context"
    "fmt"

    "github.com/smallnest/langgraphgo/graph"
)

func main() {
    g := graph.NewStateGraph[map[string]any]()

    g.AddNode("router", "router", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        priority := state["priority"].(string)
        state["routed"] = true
        state["route_path"] = priority
        return state, nil
    })

    g.AddNode("urgent_handler", "urgent_handler", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        state["status"] = "紧急处理完成"
        state["processing_time_ms"] = 50
        return state, nil
    })

    g.AddNode("normal_handler", "normal_handler", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        state["status"] = "常规处理完成"
        state["processing_time_ms"] = 200
        return state, nil
    })

    g.AddNode("batch_handler", "batch_handler", func(ctx context.Context, state map[string]any) (map[string]any, error) {
        state["status"] = "批量处理完成"
        state["processing_time_ms"] = 5000
        return state, nil
    })

    // 设置入口点
    g.SetEntryPoint("router")

    // 添加条件边
    g.AddConditionalEdge("router", func(ctx context.Context, state map[string]any) string {
        // 可以访问之前设置的所有字段
        priority := state["priority"].(string)
        switch priority {
        case "high":
            return "urgent_handler"
        case "low":
            return "batch_handler"
        default:
            return "normal_handler"
        }
    })

    g.AddEdge("urgent_handler", graph.END)
    g.AddEdge("normal_handler", graph.END)
    g.AddEdge("batch_handler", graph.END)

    runnable, _ := g.Compile()

    // 测试高优先级任务
    fmt.Println("=== 高优先级任务 ===")
    result1, _ := runnable.Invoke(context.Background(), map[string]any{
        "priority": "high",
        "task_id":  "A001",
    })
    fmt.Printf("结果: %v\n\n", result1)

    // 测试低优先级任务
    fmt.Println("=== 低优先级任务 ===")
    result2, _ := runnable.Invoke(context.Background(), map[string]any{
        "priority": "low",
        "task_id":  "A002",
    })
    fmt.Printf("结果: %v\n", result2)
}
```

输出：
```
=== 高优先级任务 ===
结果: map[priority:high task_id:A001 routed:true route_path:high status:紧急处理完成 processing_time_ms:50]

=== 低优先级任务 ===
结果: map[priority:low task_id:A002 routed:true route_path:low status:批量处理完成 processing_time_ms:5000]
```

注意最终状态保留了所有字段：初始的 `priority` 和 `task_id`，router 添加的 `routed` 和 `route_path`，以及 handler 添加的 `status` 和 `processing_time_ms`。

---

## 最佳实践

### 1. 始终返回完整状态

```go
// ✅ 正确
state["new_field"] = value
return state, nil

// ❌ 错误
return map[string]any{"new_field": value}, nil
```

### 2. 使用类型化图获得更好的安全性

```go
type MyState struct {
    Input   string
    Output  string
    Count   int
    History []string
}

g := graph.NewStateGraph[MyState]()

g.AddNode("process", "process", func(ctx context.Context, state MyState) (MyState, error) {
    state.Output = strings.ToUpper(state.Input)
    state.Count++
    return state, nil  // 编译器确保所有字段都被保留
})
```

### 3. 为集合使用 Reducer

```go
schema := graph.NewMapSchema()
schema.RegisterReducer("logs", graph.AppendReducer)
schema.RegisterReducer("tags", graph.AppendReducer)
g.SetSchema(schema)
```

### 4. 文档化状态结构

在代码中清楚地记录状态包含哪些字段以及它们在哪里被添加/修改：

```go
// State structure:
// - input (string): Initial user input [entry point]
// - tokens (int): Token count [tokenize node]
// - entities ([]string): Extracted entities [extract node]
// - summary (string): Generated summary [summarize node]
// - confidence (float): Summary confidence score [summarize node]
```

### 5. 测试状态流

验证数据是否正确流经你的图：

```go
func TestStateFlow(t *testing.T) {
    g := setupGraph()
    runnable, _ := g.Compile()

    result, _ := runnable.Invoke(context.Background(), map[string]any{
        "input": "test",
    })

    // 验证所有预期的字段都存在
    assert.Contains(t, result, "input")
    assert.Contains(t, result, "output")
    assert.Contains(t, result, "metadata")
}
```

### 6. 避免状态污染

如果需要修改状态但不希望影响后续节点，先创建副本：

```go
g.AddNode("process", "process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    // 如果需要修改 slice/map 但不影响原始值
    items := state["items"].([]string)
    newItems := make([]string, len(items))
    copy(newItems, items)

    // 修改副本
    newItems = append(newItems, "new_item")
    state["items"] = newItems
    return state, nil
})
```

---

## 常见陷阱

### 陷阱 1：假设状态会自动合并

```go
// ❌ 错误：假设框架会自动合并
g.AddNode("node1", "node1", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    return map[string]any{"field1": "value1"}, nil  // 丢失其他字段！
})

g.AddNode("node2", "node2", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    // state["field1"] 存在，但其他字段都丢失了
    return map[string]any{"field2": "value2"}, nil  // 又丢失了！
})
```

### 陷阱 2：并行分支中过度修改

```go
// ⚠️  可能有问题：并行分支修改相同字段
g.AddNode("branch_a", "branch_a", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["result"] = "A"
    return state, nil
})

g.AddNode("branch_b", "branch_b", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["result"] = "B"  // 会覆盖 A 的结果！
    return state, nil
})
```

**解决方案：** 使用不同的字段名或配置合并器。

### 陷阱 3：忘记返回状态

```go
// ❌ 错误：修改状态但忘记返回
g.AddNode("process", "process", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    state["result"] = "processed"
    return nil, nil  // 返回 nil 会中断执行！
})
```

---

## 快速参考

### 标准模式

```go
// 读取 -> 处理 -> 更新 -> 返回
g.AddNode("node_name", "node_name", func(ctx context.Context, state map[string]any) (map[string]any, error) {
    // 1. 从状态读取
    input := state["input_field"].(string)

    // 2. 处理数据
    result := processData(input)

    // 3. 更新状态
    state["output_field"] = result
    state["processed"] = true

    // 4. 返回完整状态
    return state, nil
})
```

### 检查清单

在编写节点函数时，确保：

- [ ] 修改了 `state` 变量而不是创建新 map
- [ ] 返回了 `state, nil` 而不是 `newMap, nil`
- [ ] 没有意外覆盖重要字段
- [ ] 考虑了并行执行时的字段冲突
- [ ] 添加了适当的错误处理

---

## 总结

LangGraphGo 中的状态管理遵循一个简单的原则：**累积式传递**。

1. **状态只增不减**：每个节点向状态添加新字段或修改现有字段
2. **完整传递**：始终返回完整的 state，而不是部分字段
3. **有序执行**：后续节点可以访问所有前面节点添加的数据
4. **框架合并**：并行分支的结果由框架自动合并

遵循这个模式，你的图工作流将能够可靠地在节点间传递和累积数据。
