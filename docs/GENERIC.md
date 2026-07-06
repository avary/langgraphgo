# LangGraphGo 泛型使用指南

本文档介绍 LangGraphGo 项目中的泛型实现，包括类型安全的状态图、预构建代理的使用，以及从非泛型版本迁移到泛型版本的完整指南。

## 目录

- [概述](#概述)
- [核心泛型类型](#核心泛型类型)
- [泛型与非泛型对比](#泛型与非泛型对比)
- [快速开始](#快速开始)
- [迁移指南](#迁移指南)
- [Prebuilt 代理泛型支持](#prebuilt-代理泛型支持)
- [最佳实践](#最佳实践)

---

## 概述

LangGraphGo 提供了完整的泛型支持，允许在编译时获得类型安全的状态管理。泛型版本通过类型参数 `[S any]` 来指定状态类型，消除了运行时类型断言的需要。

### 主要优势

| 特性 | 非泛型版本 | 泛型版本 |
|------|-----------|---------|
| **类型安全** | 运行时断言，可能 panic | 编译时检查，零运行时错误 |
| **代码可读性** | 需要 `state.(Type)` 断言 | 直接访问 `state.Field` |
| **IDE 支持** | 弱类型推断 | 完整的代码补全和重构 |
| **性能** | 反射开销 | 零额外开销 |
| **维护性** | 隐式状态结构 | 明确的状态结构定义 |

---

## 核心泛型类型

### StateGraph[S] - 状态图

`StateGraph[S any]` 是泛型状态图的核心类型，`S` 是状态类型的类型参数。

```go
package graph

// StateGraph 表示一个泛型状态图
type StateGraph[S any] struct {
    nodes            map[string]TypedNode[S]
    edges            []Edge
    conditionalEdges map[string]func(ctx context.Context, state S) string
    entryPoint       string
    retryPolicy      *RetryPolicy
    stateMerger      TypedStateMerger[S]
    Schema           StateSchema[S]
}

// 创建新的泛型状态图
func NewStateGraph[S any]() *StateGraph[S]
```

### TypedNode[S] - 类型化节点

```go
// TypedNode 表示一个泛型节点
type TypedNode[S any] struct {
    Name        string
    Description string
    Function    func(ctx context.Context, state S) (S, error)
}
```

### StateRunnable[S] - 可执行图

编译后的可执行图，类型安全地调用图执行：

```go
// StateRunnable 表示编译后的泛型状态图
type StateRunnable[S any] struct {
    graph      *StateGraph[S]
    tracer     *Tracer
    nodeRunner func(ctx context.Context, nodeName string, state S) (S, error)
}

// Invoke 执行图
func (r *StateRunnable[S]) Invoke(ctx context.Context, initialState S) (S, error)

// Stream 流式执行图
func (r *StateRunnable[S]) Stream(ctx context.Context, initialState S) (<-chan StreamEvent[S], error)
```

### ListenableStateGraph[S] - 可监听图

```go
// ListenableStateGraph 扩展了 StateGraph，支持节点事件监听
type ListenableStateGraph[S any] struct {
    *StateGraph[S]
    listenableNodes map[string]*ListenableNode[S]
}

// NodeListener 定义节点事件监听器接口
type NodeListener[S any] interface {
    OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state S, err error)
}

// NodeListenerFunc 是函数适配器
type NodeListenerFunc[S any] func(ctx context.Context, event NodeEvent, nodeName string, state S, err error)

func (f NodeListenerFunc[S]) OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state S, err error) {
    f(ctx, event, nodeName, state, err)
}
```

### StateSchema[S] - 状态模式

定义状态的结构和更新逻辑：

```go
// StateSchema 定义状态结构和更新逻辑
type StateSchema[S any] interface {
    Init() S
    Update(current, new S) (S, error)
}

// StructSchema 实现了基于结构体的状态模式
type StructSchema[S any] struct {
    InitialValue S
    MergeFunc    func(current, new S) (S, error)
}

func NewStructSchema[S any](initial S, merge func(S, S) (S, error)) *StructSchema[S]
```

### 并行执行支持

```go
// ParallelNode 表示一组可并行执行的节点
type ParallelNode[S any] struct {
    nodes []TypedNode[S]
    name  string
}

// MapReduceNode 执行 map-reduce 模式
type MapReduceNode[S any] struct {
    name     string
    mapNodes []TypedNode[S]
    reducer  func([]S) (S, error)
}

// 添加并行节点
func (g *StateGraph[S]) AddParallelNodes(
    groupName string,
    nodes map[string]func(context.Context, S) (S, error),
    merger func([]S) S,
)

// 添加 map-reduce 节点
func (g *StateGraph[S]) AddMapReduceNode(
    name string,
    mapFunctions map[string]func(context.Context, S) (S, error),
    reducer func([]S) (S, error),
)
```

---

## 泛型与非泛型对比

### 类型别名兼容性

为了向后兼容，项目提供了类型别名：

```go
// 非泛型类型实际上是泛型类型的别名
type Runnable = StateRunnable[map[string]any]
type StateGraphMap = StateGraph[map[string]any]
type ListenableStateGraphMap = ListenableStateGraph[map[string]any]
```

### 代码对比

#### 1. 创建图

```go
// 非泛型版本
g := graph.NewStateGraph[map[string]any]()

// 泛型版本
g := graph.NewStateGraph[MyState]()
```

#### 2. 定义节点函数

```go
// 非泛型版本 - 需要类型断言
g.AddNode("process", "Process data", func(ctx context.Context, state any) (any, error) {
    s := state.(MyState)  // 运行时断言，可能 panic
    s.Count++
    return s, nil
})

// 泛型版本 - 类型安全
g.AddNode("process", "Process data", func(ctx context.Context, state MyState) (MyState, error) {
    state.Count++  // 直接访问，编译时检查
    return state, nil
})
```

#### 3. 访问状态字段

```go
// 非泛型版本
s := state.(map[string]any)
count := s["count"].(int)  // 多层断言

// 泛型版本
count := state.Count  // 简洁、安全
```

#### 4. 条件边

```go
// 非泛型版本
g.AddConditionalEdge("check", func(ctx context.Context, state any) string {
    s := state.(map[string]any)
    if s["is_adult"].(bool) {
        return "adult_path"
    }
    return "minor_path"
})

// 泛型版本
g.AddConditionalEdge("check", func(ctx context.Context, state MyState) string {
    if state.IsAdult {
        return "adult_path"
    }
    return "minor_path"
})
```

#### 5. 编译和执行

```go
// 非泛型版本
app, err := g.Compile()
result, err := app.Invoke(ctx, map[string]any{"count": 0})

// 泛型版本
app, err := g.Compile()
result, err := app.Invoke(ctx, MyState{Count: 0})
```

---

## 快速开始

### 示例 1: 基础泛型状态图

```go
package main

import (
    "context"
    "fmt"

    "github.com/smallnest/langgraphgo/graph"
)

// 定义状态结构体
type WorkflowState struct {
    Request       UserRequest
    IsAdult       bool
    IsEligible    bool
    Notifications []string
    Result        string
}

type UserRequest struct {
    Name string
    Age  int
}

func main() {
    // 创建泛型状态图
    g := graph.NewStateGraph[WorkflowState]()

    // 添加节点 - 类型安全
    g.AddNode("check_age", "Check if user is adult", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
        state.IsAdult = state.Request.Age >= 18
        state.Notifications = append(state.Notifications,
            fmt.Sprintf("Age check: %s is adult=%v", state.Request.Name, state.IsAdult))
        return state, nil
    })

    g.AddNode("check_eligibility", "Check service eligibility", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
        state.IsEligible = state.IsAdult && state.Request.Age < 65
        state.Notifications = append(state.Notifications,
            fmt.Sprintf("Eligibility check: %v", state.IsEligible))
        return state, nil
    })

    g.AddNode("approve", "Approve request", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
        state.Result = "Approved"
        return state, nil
    })

    g.AddNode("reject", "Reject request", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
        state.Result = "Rejected"
        return state, nil
    })

    // 添加边
    g.SetEntryPoint("check_age")
    g.AddEdge("check_age", "check_eligibility")

    // 添加条件边
    g.AddConditionalEdge("check_eligibility", func(ctx context.Context, state WorkflowState) string {
        if state.IsEligible {
            return "approve"
        }
        return "reject"
    })

    g.AddEdge("approve", graph.END)
    g.AddEdge("reject", graph.END)

    // 编译并执行
    app, err := g.Compile()
    if err != nil {
        panic(err)
    }

    initialState := WorkflowState{
        Request: UserRequest{Name: "Alice", Age: 25},
    }

    finalState, err := app.Invoke(context.Background(), initialState)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Result: %s\n", finalState.Result)
    fmt.Printf("Notifications: %v\n", finalState.Notifications)
}
```

### 示例 2: 使用 Schema 的状态管理

```go
package main

import (
    "context"
    "fmt"

    "github.com/smallnest/langgraphgo/graph"
)

type ProcessState struct {
    Items      []string
    Count      int
    MaxCount   int
    Processing bool
}

func main() {
    g := graph.NewStateGraph[ProcessState]()

    // 创建 Schema，定义合并逻辑
    schema := graph.NewStructSchema(
        ProcessState{MaxCount: 5, Processing: true},
        func(current, new ProcessState) (ProcessState, error) {
            // 自定义合并逻辑
            current.Items = append(current.Items, new.Items...)
            current.Count += new.Count
            current.Processing = new.Processing
            return current, nil
        },
    )
    g.SetSchema(schema)

    // 添加处理节点
    g.AddNode("process", "Process items", func(ctx context.Context, state ProcessState) (ProcessState, error) {
        if state.Count >= state.MaxCount {
            return ProcessState{Processing: false}, nil
        }

        item := fmt.Sprintf("item_%d", state.Count+1)
        return ProcessState{
            Items:      []string{item},
            Count:      state.Count + 1,
            MaxCount:   state.MaxCount,
            Processing: true,
        }, nil
    })

    g.AddNode("validate", "Validate results", func(ctx context.Context, state ProcessState) (ProcessState, error) {
        if !state.Processing {
            return state, nil
        }
        // 继续处理
        return state, nil
    })

    g.SetEntryPoint("process")

    // 条件边：根据 Processing 状态决定是否继续
    g.AddConditionalEdge("process", func(ctx context.Context, state ProcessState) string {
        if state.Processing && state.Count < state.MaxCount {
            return "process"  // 循环继续处理
        }
        return "validate"
    })

    g.AddEdge("validate", graph.END)

    app, _ := g.Compile()
    result, _ := app.Invoke(context.Background(), ProcessState{})

    fmt.Printf("Processed %d items: %v\n", result.Count, result.Items)
}
```

### 示例 3: 可监听的泛型图

```go
package main

import (
    "context"
    "fmt"

    "github.com/smallnest/langgraphgo/graph"
)

type TypedState struct {
    Count int
    Log   []string
}

func main() {
    // 创建可监听的泛型状态图
    g := graph.NewListenableStateGraph[TypedState]()

    // 添加节点
    node := g.AddNode("increment", "Increment counter", func(ctx context.Context, state TypedState) (TypedState, error) {
        state.Count++
        state.Log = append(state.Log, "Incremented")
        return state, nil
    })

    // 添加类型化的监听器
    listener := graph.NodeListenerFunc[TypedState](
        func(ctx context.Context, event graph.NodeEvent, nodeName string, state TypedState, err error) {
            fmt.Printf("[Listener] Event: %s, Node: %s, Count: %d\n", event, nodeName, state.Count)
        },
    )

    node.AddListener(listener)

    g.SetEntryPoint("increment")
    g.AddEdge("increment", graph.END)

    // 编译为可监听图
    runnable, err := g.CompileListenable()
    if err != nil {
        panic(err)
    }

    result, err := runnable.Invoke(context.Background(), TypedState{Count: 0})
    if err != nil {
        panic(err)
    }

    fmt.Printf("Final count: %d\n", result.Count)
}
```

### 示例 4: 并行执行

```go
package main

import (
    "context"
    "fmt"

    "github.com/smallnest/langgraphgo/graph"
)

type ParallelState struct {
    Results    map[string]string
    Combined   string
    Processors []string
}

func main() {
    g := graph.NewStateGraph[ParallelState]()

    // 添加并行节点组
    g.AddParallelNodes("parallel_processing",
        map[string]func(context.Context, ParallelState) (ParallelState, error){
            "processor_a": func(ctx context.Context, state ParallelState) (ParallelState, error) {
                if state.Results == nil {
                    state.Results = make(map[string]string)
                }
                state.Results["a"] = "Result from A"
                return state, nil
            },
            "processor_b": func(ctx context.Context, state ParallelState) (ParallelState, error) {
                if state.Results == nil {
                    state.Results = make(map[string]string)
                }
                state.Results["b"] = "Result from B"
                return state, nil
            },
            "processor_c": func(ctx context.Context, state ParallelState) (ParallelState, error) {
                if state.Results == nil {
                    state.Results = make(map[string]string)
                }
                state.Results["c"] = "Result from C"
                return state, nil
            },
        },
        // 合并函数
        func(states []ParallelState) ParallelState {
            merged := ParallelState{
                Results: make(map[string]string),
            }
            for _, s := range states {
                for k, v := range s.Results {
                    merged.Results[k] = v
                }
            }
            return merged
        },
    )

    // 添加聚合节点
    g.AddNode("aggregate", "Aggregate results", func(ctx context.Context, state ParallelState) (ParallelState, error) {
        combined := ""
        for k, v := range state.Results {
            combined += fmt.Sprintf("[%s: %s] ", k, v)
        }
        state.Combined = combined
        return state, nil
    })

    g.SetEntryPoint("parallel_processing")
    g.AddEdge("parallel_processing", "aggregate")
    g.AddEdge("aggregate", graph.END)

    app, _ := g.Compile()
    result, _ := app.Invoke(context.Background(), ParallelState{})

    fmt.Printf("Combined results: %s\n", result.Combined)
}
```

---

## 迁移指南

### 步骤 1: 定义状态结构体

将原来使用 `map[string]any` 的状态转换为明确的结构体：

```go
// 之前：使用 map[string]any
state := map[string]any{
    "messages": []llms.MessageContent{...},
    "count": 0,
    "active": true,
}

// 之后：定义结构体
type MyState struct {
    Messages []llms.MessageContent
    Count    int
    Active   bool
}
```

### 步骤 2: 更新图创建

```go
// 之前
g := graph.NewStateGraph[map[string]any]()

// 之后
g := graph.NewStateGraph[MyState]()
```

### 步骤 3: 更新节点函数签名

```go
// 之前
g.AddNode("process", "Process", func(ctx context.Context, state any) (any, error) {
    s := state.(map[string]any)
    s["count"] = s["count"].(int) + 1
    return s, nil
})

// 之后
g.AddNode("process", "Process", func(ctx context.Context, state MyState) (MyState, error) {
    state.Count++
    return state, nil
})
```

### 步骤 4: 更新条件边函数

```go
// 之前
g.AddConditionalEdge("check", func(ctx context.Context, state any) string {
    s := state.(map[string]any)
    if s["active"].(bool) {
        return "continue"
    }
    return "stop"
})

// 之后
g.AddConditionalEdge("check", func(ctx context.Context, state MyState) string {
    if state.Active {
        return "continue"
    }
    return "stop"
})
```

### 步骤 5: 更新 Invoke 调用

```go
// 之前
app, _ := g.Compile()
result, _ := app.Invoke(ctx, map[string]any{"count": 0})

// 之后
app, _ := g.Compile()
result, _ := app.Invoke(ctx, MyState{Count: 0})
```

### 完整迁移示例

```go
// ====== 迁移前：非泛型版本 ======
func oldVersion() error {
    g := graph.NewStateGraph[map[string]any]()

    g.AddNode("increment", "Increment", func(ctx context.Context, state any) (any, error) {
        s := state.(map[string]any)
        count := s["count"].(int)
        s["count"] = count + 1
        return s, nil
    })

    g.AddNode("decrement", "Decrement", func(ctx context.Context, state any) (any, error) {
        s := state.(map[string]any)
        count := s["count"].(int)
        s["count"] = count - 1
        return s, nil
    })

    g.AddConditionalEdge("start", func(ctx context.Context, state any) string {
        s := state.(map[string]any)
        if s["direction"].(string) == "up" {
            return "increment"
        }
        return "decrement"
    })

    g.AddEdge("increment", graph.END)
    g.AddEdge("decrement", graph.END)

    app, _ := g.Compile()
    result, _ := app.Invoke(context.Background(), map[string]any{
        "count":     0,
        "direction": "up",
    })

    fmt.Printf("Result: %v\n", result)
    return nil
}

// ====== 迁移后：泛型版本 ======
type CounterState struct {
    Count     int
    Direction string
}

func newVersion() error {
    g := graph.NewStateGraph[CounterState]()

    g.AddNode("increment", "Increment", func(ctx context.Context, state CounterState) (CounterState, error) {
        state.Count++
        return state, nil
    })

    g.AddNode("decrement", "Decrement", func(ctx context.Context, state CounterState) (CounterState, error) {
        state.Count--
        return state, nil
    })

    g.AddConditionalEdge("start", func(ctx context.Context, state CounterState) string {
        if state.Direction == "up" {
            return "increment"
        }
        return "decrement"
    })

    g.AddEdge("increment", graph.END)
    g.AddEdge("decrement", graph.END)

    app, _ := g.Compile()
    result, _ := app.Invoke(context.Background(), CounterState{
        Count:     0,
        Direction: "up",
    })

    fmt.Printf("Result: %+v\n", result)
    return nil
}
```

---

## Prebuilt 代理泛型支持

LangGraphGo 的 prebuilt 包提供了多种预构建代理，均支持泛型版本。每个代理都有两个版本：
- `*Map` 版本：使用 `map[string]any` 状态
- 泛型版本：使用自定义状态类型 `S`

### 预定义状态类型

所有预定义的状态类型位于 `prebuilt/agent_states.go`：

```go
// AgentState - 通用代理状态
type AgentState struct {
    Messages   []llms.MessageContent
    ExtraTools []tools.Tool
}

// ReactAgentState - ReAct 代理状态
type ReactAgentState struct {
    Messages       []llms.MessageContent
    IterationCount int
}

// PlanningAgentState - 规划代理状态
type PlanningAgentState struct {
    Messages     []llms.MessageContent
    WorkflowPlan *WorkflowPlan
}

// ReflectionAgentState - 反思代理状态
type ReflectionAgentState struct {
    Messages   []llms.MessageContent
    Iteration  int
    Reflection string
    Draft      string
}

// PEVAgentState - Plan-Execute-Verify 代理状态
type PEVAgentState struct {
    Messages           []llms.MessageContent
    Plan               []string
    CurrentStep        int
    LastToolResult     string
    IntermediateSteps  []string
    Retries            int
    VerificationResult string
    FinalAnswer        string
}

// TreeOfThoughtsState - 树思维代理状态
type TreeOfThoughtsState struct {
    ActivePaths   map[string]*SearchPath
    Solution      string
    VisitedStates map[string]bool
    Iteration     int
}

// ChatAgentState - 聊天代理状态
type ChatAgentState struct {
    Messages     []llms.MessageContent
    SystemPrompt string
    ExtraTools   []tools.Tool
}

// SupervisorState - 监督器状态
type SupervisorState struct {
    Messages []llms.MessageContent
    Next     string
}
```

### CreateAgent - 通用代理

```go
package prebuilt

// Map 版本
func CreateAgentMap(
    model llms.Model,
    inputTools []tools.Tool,
    opts ...CreateAgentOption,
) (*graph.StateRunnable[map[string]any], error)

// 泛型版本
func CreateAgent[S any](
    model llms.Model,
    inputTools []tools.Tool,
    getMessages func(S) []llms.MessageContent,
    setMessages func(S, []llms.MessageContent) S,
    getExtraTools func(S) []tools.Tool,
    setExtraTools func(S, []tools.Tool) S,
    opts ...CreateAgentOption,
) (*graph.StateRunnable[S], error)
```

#### 使用示例

```go
// 定义自定义状态
type MyAgentState struct {
    Messages []llms.MessageContent
    Tools    []tools.Tool
    UserID   string  // 自定义字段
}

func main() {
    model := llms.NewModel(...)
    tools := []tools.Tool{...}

    // 使用泛型版本创建代理
    agent, err := prebuilt.CreateAgent[MyAgentState](
        model,
        tools,
        // getter: 获取消息
        func(s MyAgentState) []llms.MessageContent { return s.Messages },
        // setter: 设置消息
        func(s MyAgentState, msgs []llms.MessageContent) MyAgentState {
            s.Messages = msgs
            return s
        },
        // getter: 获取工具
        func(s MyAgentState) []tools.Tool { return s.Tools },
        // setter: 设置工具
        func(s MyAgentState, ts []tools.Tool) MyAgentState {
            s.Tools = ts
            return s
        },
    )

    // 执行
    result, err := agent.Invoke(ctx, MyAgentState{
        Messages: []llms.MessageContent{...},
        Tools:    tools,
        UserID:   "user123",
    })
}
```

### CreateReactAgent - ReAct 代理

```go
// Map 版本
func CreateReactAgentMap(
    model llms.Model,
    inputTools []tools.Tool,
    maxIterations int,
) (*graph.StateRunnable[map[string]any], error)

// 泛型版本
func CreateReactAgent[S any](
    model llms.Model,
    inputTools []tools.Tool,
    getMessages func(S) []llms.MessageContent,
    setMessages func(S, []llms.MessageContent) S,
    getIterationCount func(S) int,
    setIterationCount func(S, int) S,
    maxIterations int,
) (*graph.StateRunnable[S], error)
```

#### 使用示例

```go
type CustomReactState struct {
    Messages       []llms.MessageContent
    IterationCount int
    Context        string  // 自定义上下文
}

agent, err := prebuilt.CreateReactAgent[CustomReactState](
    model,
    tools,
    func(s CustomReactState) []llms.MessageContent { return s.Messages },
    func(s CustomReactState, msgs []llms.MessageContent) CustomReactState {
        s.Messages = msgs
        return s
    },
    func(s CustomReactState) int { return s.IterationCount },
    func(s CustomReactState, count int) CustomReactState {
        s.IterationCount = count
        return s
    },
    10, // maxIterations
)
```

### CreatePlanningAgent - 规划代理

```go
// Map 版本
func CreatePlanningAgentMap(
    model llms.Model,
    availableNodes []graph.TypedNode[map[string]any],
    inputTools []tools.Tool,
    opts ...CreateAgentOption,
) (*graph.StateRunnable[map[string]any], error)

// 泛型版本
func CreatePlanningAgent[S any](
    model llms.Model,
    availableNodes []graph.TypedNode[S],
    getMessages func(S) []llms.MessageContent,
    setMessages func(S, []llms.MessageContent) S,
    getPlan func(S) *WorkflowPlan,
    setPlan func(S, *WorkflowPlan) S,
    opts ...CreateAgentOption,
) (*graph.StateRunnable[S], error)
```

### CreateReflectionAgent - 反思代理

```go
// Map 版本
func CreateReflectionAgentMap(
    config ReflectionAgentConfig,
) (*graph.StateRunnable[map[string]any], error)

// 泛型版本
func CreateReflectionAgent[S any](
    config ReflectionAgentConfig,
    getMessages func(S) []llms.MessageContent,
    setMessages func(S, []llms.MessageContent) S,
    getDraft func(S) string,
    setDraft func(S, string) S,
    getIteration func(S) int,
    setIteration func(S, int) S,
    getReflection func(S) string,
    setReflection func(S, string) S,
) (*graph.StateRunnable[S], error)
```

### CreatePEVAgent - Plan-Execute-Verify 代理

```go
// Map 版本
func CreatePEVAgentMap(
    config PEVAgentConfig,
) (*graph.StateRunnable[map[string]any], error)

// 泛型版本
func CreatePEVAgent[S any](
    config PEVAgentConfig,
    getMessages func(S) []llms.MessageContent,
    setMessages func(S, []llms.MessageContent) S,
    getPlan func(S) []string,
    setPlan func(S, []string) S,
    getCurrentStep func(S) int,
    setCurrentStep func(S, int) S,
    getLastToolResult func(S) string,
    setLastToolResult func(S, string) S,
    getIntermediateSteps func(S) []string,
    setIntermediateSteps func(S, []string) S,
    getRetries func(S) int,
    setRetries func(S, int) S,
    getVerificationResult func(S) string,
    setVerificationResult func(S, string) S,
    getFinalAnswer func(S) string,
    setFinalAnswer func(S, string) S,
) (*graph.StateRunnable[S], error)
```

### CreateTreeOfThoughtsAgent - 树思维代理

```go
// Map 版本
func CreateTreeOfThoughtsAgentMap(
    config TreeOfThoughtsConfig,
) (*graph.StateRunnable[map[string]any], error)

// 泛型版本
func CreateTreeOfThoughtsAgent[S any](
    config TreeOfThoughtsConfig,
    getActivePaths func(S) map[string]*SearchPath,
    setActivePaths func(S, map[string]*SearchPath) S,
    getSolution func(S) string,
    setSolution func(S, string) S,
    getVisitedStates func(S) map[string]bool,
    setVisitedStates func(S, map[string]bool) S,
    getIteration func(S) int,
    setIteration func(S, int) S,
) (*graph.StateRunnable[S], error)
```

### CreateSupervisor - 监督器

```go
// Map 版本
func CreateSupervisorMap(
    model llms.Model,
    members map[string]*graph.StateRunnable[map[string]any],
) (*graph.StateRunnable[map[string]any], error)

// 泛型版本
func CreateSupervisor[S any](
    model llms.Model,
    members map[string]*graph.StateRunnable[S],
    getMessages func(S) []llms.MessageContent,
    getNext func(S) string,
    setNext func(S, string) S,
) (*graph.StateRunnable[S], error)
```

### Prebuilt 迁移示例

```go
// ====== 迁移前：使用 Map 版本 ======
func oldReactAgent() {
    agent, err := prebuilt.CreateReactAgentMap(
        model,
        tools,
        10, // maxIterations
    )

    result, err := agent.Invoke(ctx, map[string]any{
        "messages": []llms.MessageContent{...},
        "iteration_count": 0,
    })
}

// ====== 迁移后：使用泛型版本 ======
type MyReactState struct {
    Messages       []llms.MessageContent
    IterationCount int
    SessionID      string  // 自定义字段
    StartTime      time.Time
}

func newReactAgent() {
    agent, err := prebuilt.CreateReactAgent[MyReactState](
        model,
        tools,
        func(s MyReactState) []llms.MessageContent { return s.Messages },
        func(s MyReactState, msgs []llms.MessageContent) MyReactState {
            s.Messages = msgs
            return s
        },
        func(s MyReactState) int { return s.IterationCount },
        func(s MyReactState, count int) MyReactState {
            s.IterationCount = count
            return s
        },
        10,
    )

    result, err := agent.Invoke(ctx, MyReactState{
        Messages:   []llms.MessageContent{...},
        SessionID:  "session123",
        StartTime:  time.Now(),
    })
}
```

---

## 最佳实践

### 1. 状态结构设计

```go
// ✅ 推荐：扁平化状态结构
type GoodState struct {
    Messages   []llms.MessageContent
    Count      int
    Active     bool
    Metadata   map[string]string
}

// ❌ 避免：过度嵌套
type BadState struct {
    Data struct {
        Messages []llms.MessageContent
        Counters struct {
            Main  int
            Aux   int
        }
    }
}
```

### 2. 状态更新模式

```go
// ✅ 推荐：返回更新后的状态（值语义）
g.AddNode("process", "Process", func(ctx context.Context, state MyState) (MyState, error) {
    state.Count++  // 修改副本
    return state, nil  // 返回新状态
})

// ❌ 避免：使用指针（除非必要）
g.AddNode("process", "Process", func(ctx context.Context, state *MyState) (*MyState, error) {
    state.Count++  // 修改原始状态
    return state, nil
})
```

### 3. 使用 Schema 管理复杂状态

```go
type ComplexState struct {
    Items    []string
    Count    int
    MaxCount int
}

func main() {
    g := graph.NewStateGraph[ComplexState]()

    schema := graph.NewStructSchema(
        ComplexState{MaxCount: 10},
        func(current, new ComplexState) (ComplexState, error) {
            // 自定义合并逻辑
            current.Items = append(current.Items, new.Items...)
            current.Count += new.Count
            if current.Count > current.MaxCount {
                current.Count = current.MaxCount
            }
            return current, nil
        },
    )
    g.SetSchema(schema)
}
```

### 4. Getter/Setter 函数命名

```go
// ✅ 推荐：一致的命名模式
agent, err := prebuilt.CreateReactAgent[MyState](
    model,
    tools,
    getMessages,      // getter: getXxx
    setMessages,      // setter: setXxx
    getIterationCount,
    setIterationCount,
    maxIterations,
)

// Getter 实现
func getMessages(s MyState) []llms.MessageContent {
    return s.Messages
}

// Setter 实现：返回新状态
func setMessages(s MyState, msgs []llms.MessageContent) MyState {
    s.Messages = msgs
    return s
}
```

### 5. 错误处理

```go
g.AddNode("risky", "Risky operation", func(ctx context.Context, state MyState) (MyState, error) {
    result, err := someOperation()
    if err != nil {
        // 返回原始状态和错误，图可以选择重试或终止
        return state, fmt.Errorf("operation failed: %w", err)
    }
    state.Result = result
    return state, nil
})
```

### 6. 并行安全的状态合并

```go
g.AddParallelNodes("parallel",
    map[string]func(context.Context, SharedState) (SharedState, error){
        "worker_a": func(ctx context.Context, state SharedState) (SharedState, error) {
            // 每个并行节点返回独立的部分状态
            return SharedState{PartialResults: []string{"A"}}, nil
        },
        "worker_b": func(ctx context.Context, state SharedState) (SharedState, error) {
            return SharedState{PartialResults: []string{"B"}}, nil
        },
    },
    // 合并函数：线程安全地合并所有并行节点的结果
    func(states []SharedState) SharedState {
        merged := SharedState{
            PartialResults: make([]string, 0),
        }
        for _, s := range states {
            merged.PartialResults = append(merged.PartialResults, s.PartialResults...)
        }
        return merged
    },
)
```

### 7. 监听器使用

```go
type MonitorState struct {
    Count   int
    Latency time.Duration
}

func main() {
    g := graph.NewListenableStateGraph[MonitorState]()

    node := g.AddNode("work", "Do work", func(ctx context.Context, state MonitorState) (MonitorState, error) {
        start := time.Now()
        // ... 执行工作 ...
        state.Latency = time.Since(start)
        state.Count++
        return state, nil
    })

    // 添加性能监控监听器
    node.AddListener(graph.NodeListenerFunc[MonitorState](
        func(ctx context.Context, event graph.NodeEvent, nodeName string, state MonitorState, err error) {
            if event == graph.NodeEventAfter {
                fmt.Printf("Node %s completed in %v, count: %d\n",
                    nodeName, state.Latency, state.Count)
            }
        },
    ))
}
```

### 8. 类型安全的状态转换

```go
// 使用状态枚举确保类型安全
type WorkflowState string

const (
    StateInitial   WorkflowState = "initial"
    StateProcessing WorkflowState = "processing"
    StateCompleted WorkflowState = "completed"
    StateFailed    WorkflowState = "failed"
)

type ProcessState struct {
    CurrentState WorkflowState
    Data         string
    Error        error
}

g.AddConditionalEdge("check", func(ctx context.Context, state ProcessState) string {
    switch state.CurrentState {
    case StateProcessing:
        return "continue"
    case StateCompleted, StateFailed:
        return "end"
    default:
        return "error"
    }
})
```

### 9. 泛型约束和接口

```go
// 为需要特定行为的状态定义接口
type MessageContainer interface {
    GetMessages() []llms.MessageContent
    SetMessages([]llms.MessageContent)
}

// 在泛型函数中使用约束
func ProcessMessages[T MessageContainer](state T) T {
    msgs := state.GetMessages()
    // ... 处理消息 ...
    state.SetMessages(msgs)
    return state
}
```

### 10. 测试泛型代码

```go
func TestGenericGraph(t *testing.T) {
    type TestState struct {
        Count   int
        History []string
    }

    g := graph.NewStateGraph[TestState]()

    g.AddNode("increment", "Increment", func(ctx context.Context, state TestState) (TestState, error) {
        state.Count++
        state.History = append(state.History, "incremented")
        return state, nil
    })

    g.SetEntryPoint("increment")
    g.AddEdge("increment", graph.END)

    app, err := g.Compile()
    require.NoError(t, err)

    result, err := app.Invoke(context.Background(), TestState{Count: 0})
    require.NoError(t, err)
    assert.Equal(t, 1, result.Count)
    assert.Equal(t, []string{"incremented"}, result.History)
}
```

---

## 类型映射表

| 非泛型类型 | 泛型类型 | 描述 |
|-----------|---------|------|
| `StateGraph` | `StateGraph[S any]` | 状态图 |
| `StateRunnable` | `StateRunnable[S any]` | 编译后可执行图 |
| `Node` | `TypedNode[S any]` | 图节点 |
| `StateSchema` | `StateSchema[S any]` | 状态模式接口 |
| `NodeListener` | `NodeListener[S any]` | 节点监听器 |
| `ListenableStateGraph` | `ListenableStateGraph[S any]` | 可监听状态图 |
| `StreamingStateGraph` | `StreamingStateGraph[S any]` | 流式状态图 |
| `StreamEvent` | `StreamEvent[S any]` | 流式事件 |
| `ParallelNode` | `ParallelNode[S any]` | 并行节点 |
| `MapReduceNode` | `MapReduceNode[S any]` | Map-Reduce 节点 |

## 相关文档

- [RFC: Generic StateGraph](docs/GENERIC/RFC_GENERIC_STATEGRAPH.md) - 泛型设计的 RFC 文档
- [examples/generic_state_graph](examples/generic_state_graph/) - 基础泛型图示例
- [examples/generic_state_graph_listenable](examples/generic_state_graph_listenable/) - 可监听泛型图示例
- [examples/generic_state_graph_react_agent](examples/generic_state_graph_react_agent/) - ReAct 代理示例
