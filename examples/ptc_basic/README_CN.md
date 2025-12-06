# 程序化工具调用 (PTC) 示例

本示例演示了**程序化工具调用 (Programmatic Tool Calling, PTC)**，这是一种先进的技术，允许 LLM 编写代码来程序化地调用工具，而不是为每次工具调用都需要多次 API 往返。

## 什么是 PTC？

**传统的工具调用**:
```
用户查询 → LLM → 工具调用 → 工具结果 → LLM → 工具调用 → ... → 最终答案
```
每次工具调用都需要一次到 LLM 的往返，这增加了延迟和 token 成本。

**程序化工具调用 (PTC)**:
```
用户查询 → LLM → 生成的代码（包含工具调用）→ 执行代码 → LLM → 最终答案
```
LLM 生成代码（Python 或 Go）来程序化地调用多个工具，在单次执行中完成。

## PTC 的优势

1. **降低延迟**: 多次工具调用在一个步骤中执行，而不是多次 API 往返
2. **Token 效率**: 代码可以在返回结果之前过滤和聚合数据
3. **程序化控制**: 在代码中使用循环、条件和数据处理
4. **成本节省**: 更少的 API 调用和减少的 token 使用

## 示例用例

本示例要求 LLM 完成以下任务：
1. 获取旧金山和纽约的天气（2 次工具调用）
2. 计算平均温度（数据处理）
3. 将结果乘以 2（计算）

使用传统的工具调用，这需要 4+ 次 API 往返。使用 PTC，LLM 生成一次性执行所有操作的代码。

## 代码结构

### 定义的工具

1. **CalculatorTool**: 执行算术操作（加、减、乘、除、幂、平方根）
2. **WeatherTool**: 返回城市的模拟天气数据
3. **DataProcessorTool**: 处理数组（求和、平均值、最大值、最小值）

### PTC 配置

```go
agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
    Model:         model,           // 任何实现 llms.Model 的 LLM
    Tools:         toolList,        // 可用的工具
    Language:      ptc.LanguagePython, // Python 或 Go
    MaxIterations: 5,               // 最大迭代次数
})
```

## 工作原理

1. **创建 Agent**: 创建包含 agent 和代码执行节点的 StateGraph
2. **处理查询**: 用户查询发送到 LLM
3. **代码生成**: LLM 生成 Python/Go 代码来调用工具
4. **代码执行**: 代码在沙箱中运行，通过 HTTP 访问工具
5. **结果处理**: 执行结果发送回 LLM
6. **最终答案**: LLM 基于结果提供最终答案

## 运行示例

```bash
# 设置你的 OpenAI API 密钥
export OPENAI_API_KEY=your-api-key

# 运行示例
cd examples/ptc_basic
go run main.go
```

## 预期输出

示例将会：
1. 显示用户查询
2. 展示 LLM 生成调用工具的代码
3. 执行生成的代码
4. 显示所有消息（包括生成的代码和工具结果）
5. 显示最终答案

## 演示的关键特性

- ✅ 在单次执行中调用多个工具
- ✅ 程序化数据处理
- ✅ 减少 API 延迟
- ✅ 透明执行（查看生成的代码）
- ✅ 支持任何 LLM（OpenAI、Google、Anthropic 等）

## 配置选项

### 语言选择

```go
ptc.LanguagePython  // 推荐：LLM 支持更好
ptc.LanguageGo      // 备选：性能更好
```

**推荐**: 使用 Python，因为 LLM 拥有更多 Python 代码生成的训练数据。

### 自定义系统提示

```go
ptc.PTCAgentConfig{
    SystemPrompt: "你是一个数据分析助手...",
    // ...
}
```

### 迭代限制

```go
ptc.PTCAgentConfig{
    MaxIterations: 10,  // 防止无限循环
    // ...
}
```

## 高级示例

查看更复杂的 PTC 示例：
- [PTC Simple](../ptc_simple/): 基础 PTC 用法
- [PTC Expense Analysis](../ptc_expense_analysis/): 使用大型数据集的真实费用处理
- [PTC 文档](../../ptc/README_CN.md): 完整的 API 参考和架构

## 与传统方法的对比

| 方面 | 传统方式 | PTC |
|------|---------|-----|
| API 调用 | 4+ | 1-2 |
| 延迟 | 高 | 低 |
| Token 使用 | 所有工具结果 | 仅过滤后的结果 |
| 数据处理 | 模型推理 | 基于代码 |
| 复杂逻辑 | 困难 | 简单（循环、条件） |
| 大型数据集 | Token 消耗大 | 高效过滤 |

## 了解更多

- [PTC 包文档](../../ptc/README_CN.md)
- [PTC 包 README (English)](../../ptc/README.md)
- [Anthropic PTC Cookbook](https://github.com/anthropics/claude-cookbooks/blob/main/tool_use/programmatic_tool_calling_ptc.ipynb)
