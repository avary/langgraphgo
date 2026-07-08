# LangChain Memory 集成

LangGraphGo 提供了与 [LangChainGo memory 包](https://github.com/tmc/langchaingo/tree/main/memory) 的无缝集成，使您能够轻松管理基于图的应用程序中的对话历史和上下文。

## 概述

Memory（记忆）对于对话式 AI 应用至关重要。它允许系统：
- 记住之前的交互
- 在多轮对话中维护上下文
- 基于历史记录提供个性化响应

我们提供了一个 `LangChainMemory` 适配器，它封装了 LangChainGo 的 memory 实现，使其与 LangGraphGo 的架构兼容。该适配器位于独立子包中，因此核心 `memory` 包不再依赖 `langchaingo`：

```go
import (
    langchainadapter "github.com/smallnest/langgraphgo/memory/langchainadapter"
    langchainmemory "github.com/tmc/langchaingo/memory"
)
```

## 可用的 Memory 类型

该集成支持所有标准的 LangChainGo memory 类型：

### 1. ConversationBuffer (对话缓冲)
存储完整的对话历史记录。当您希望模型能够访问所有已说内容时，这非常有用。

```go
mem := langchainadapter.NewConversationBufferMemory(
    langchainmemory.WithReturnMessages(true),
)
```

### 2. ConversationWindowBuffer (对话窗口缓冲)
保留最近 N 轮对话的滑动窗口。这对于在保持上下文大小可控的同时保留最近的上下文非常有用。

```go
// 仅保留最后 5 轮
mem := langchainadapter.NewConversationWindowBufferMemory(5,
    langchainmemory.WithReturnMessages(true),
)
```

### 3. ChatMessageHistory (聊天消息历史)
提供对底层消息历史存储的直接访问，允许您手动添加或检索消息。

```go
history := langchainadapter.NewChatMessageHistory()
history.AddUserMessage(ctx, "你好")
history.AddAIMessage(ctx, "你好！")
```

## 与 LangGraph 集成

将 memory 集成到 LangGraph 节点中通常涉及三个步骤：

1.  **加载 (Load)**: 从 memory 中检索对话历史。
2.  **处理 (Process)**: 使用历史记录（结合新输入）生成响应。
3.  **保存 (Save)**: 将新的输入和输出存回 memory。

### 示例模式

```go
// 1. 定义图节点
func chatNode(ctx context.Context, state *State) (*State, error) {
    // --- 加载 ---
    // 加载 memory 变量（例如：history）
    memVars, _ := mem.LoadMemoryVariables(ctx, map[string]any{})
    
    // 从 "history" 键中提取消息
    var historyMessages []llms.ChatMessage
    if history, ok := memVars["history"]; ok {
        if msgs, ok := history.([]llms.ChatMessage); ok {
            historyMessages = msgs
        }
    }
    
    // --- 处理 ---
    // 将历史记录与当前用户输入组合
    allMessages := append(historyMessages, llms.HumanChatMessage{
        Content: state.Input,
    })
    
    // 调用 LLM
    response, _ := llm.GenerateContent(ctx, allMessages)
    aiResponse := response.Choices[0].Content
    
    // --- 保存 ---
    // 将本轮对话保存到 memory
    mem.SaveContext(ctx, map[string]any{
        "input": state.Input,
    }, map[string]any{
        "output": aiResponse,
    })
    
    // 更新状态
    state.Output = aiResponse
    return state, nil
}
```

## 自定义

### 自定义键 (Custom Keys)
如果您的应用程序需要特定的命名约定，您可以自定义用于输入、输出和 memory 变量的键。

```go
mem := langchainadapter.NewConversationBufferMemory(
    langchainmemory.WithInputKey("user_query"),      // 默认: "input"
    langchainmemory.WithOutputKey("bot_response"),   // 默认: "output"
    langchainmemory.WithMemoryKey("chat_log"),       // 默认: "history"
    langchainmemory.WithReturnMessages(true),
)
```

### Human/AI 前缀
当消息作为字符串返回时（即 `WithReturnMessages(false)`），您还可以自定义用于消息的前缀。

```go
mem := langchainadapter.NewConversationBufferMemory(
    langchainmemory.WithHumanPrefix("用户"),
    langchainmemory.WithAIPrefix("助手"),
)
```

## 示例

我们提供了完整的示例来演示这些概念：

- **[Memory 基础示例](../../examples/memory_basic/)**: 展示如何使用不同的 memory 类型和配置。
- **[带 Memory 的聊天机器人](../../examples/memory_chatbot/)**: 一个完整的聊天机器人模拟，演示状态管理和 memory 持久化。
