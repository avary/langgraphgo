# Golog 日志集成

本示例演示了如何将 **golog**（https://github.com/kataras/golog）与 LangGraphGo 的日志系统集成。

## 概述

LangGraphGo 提供了灵活的日志接口，可以配置使用各种日志后端。本示例展示了如何使用流行的 golog 库作为日志后端。

## 什么是 golog？

golog 是一个快速、简单且支持分级的 Go 日志库，提供：
- **分级日志**（DEBUG、INFO、WARN、ERROR、FATAL）
- **可自定义前缀**
- **多种输出目标**
- **线程安全**
- **性能优化**

## 示例特性

示例演示了三种常见的日志场景：

### 1. 使用默认 golog 日志器

```go
// 使用默认的 golog 实例
defaultLogger := golog.Default
logger1 := log.NewGologLogger(defaultLogger)
logger1.Info("使用默认 golog logger")
logger1.SetLevel(log.LogLevelDebug)
logger1.Debug("调试信息")
```

### 2. 创建自定义 golog 日志器

```go
// 创建具有特定配置的自定义日志器
customLogger := golog.New()
customLogger.SetPrefix("[ MyApp ] ")
customLogger.SetOutput(os.Stdout)
logger2 := log.NewGologLogger(customLogger)
logger2.SetLevel(log.LogLevelInfo)
logger2.Info("使用自定义 golog logger")
```

### 3. 仅错误日志器

```go
// 创建只显示错误级别消息的日志器
errorLogger := golog.New()
errorLogger.SetLevel("error")
errorLogger.SetPrefix("[ ERROR ] ")
logger3 := log.NewGologLogger(errorLogger)
logger3.Debug("这条不会显示")  // 被过滤掉
logger3.Error("错误信息会显示")
```

## 运行示例

```bash
cd examples/logger
go run golog_logger_example.go
```

## 预期输出

```
使用默认 golog logger
DEBUG [17:30:45] 调试信息
[ MyApp ] INFO [17:30:45] 使用自定义 golog logger
[ ERROR ] ERROR [17:30:45] 错误信息会显示
使用默认 golog logger
DEBUG [17:30:45] 调试信息
```

## 集成步骤

### 步骤 1：导入必需的包

```go
import (
    "github.com/kataras/golog"
    "github.com/smallnest/langgraphgo/log"
)
```

### 步骤 2：创建 golog 实例

```go
// 使用默认实例
gologLogger := golog.Default

// 或创建自定义实例
gologLogger := golog.New()
gologLogger.SetPrefix("[MyApp] ")
gologLogger.SetOutput(os.Stdout)
```

### 步骤 3：用 LangGraphGo 日志器包装

```go
// 用 LangGraphGo 日志器接口包装 golog
logger := log.NewGologLogger(gologLogger)
```

### 步骤 4：配置日志级别

```go
// 设置适当的日志级别
logger.SetLevel(log.LogLevelInfo)
```

### 步骤 5：设为默认日志器（可选）

```go
// 设为 LangGraphGo 的默认日志器
log.SetDefaultLogger(logger)
```

## 高级配置

### 自定义日志级别

```go
// 将 golog 级别映射到 LangGraphGo 级别
logger := log.NewGologLogger(gologLogger)

// 可用级别：
// - log.LogLevelDebug
// - log.LogLevelInfo
// - log.LogLevelWarn
// - log.LogLevelError
// - log.LogLevelFatal
```

### 格式化日志

```go
// golog 支持格式化消息
logger.Infof("用户 %s 在 %v 登录", username, time.Now())
logger.Errorf("连接失败: %v", err)
```

### 多输出

```go
// 创建多写入器日志器
multiWriter := io.MultiWriter(os.Stdout, logFile)
gologLogger := golog.New()
gologLogger.SetOutput(multiWriter)
logger := log.NewGologLogger(gologLogger)
```

## 在 LangGraphGo 应用中使用

### 图执行日志

```go
func setupLogging() {
    gologLogger := golog.New()
    gologLogger.SetPrefix("[LangGraph] ")
    gologLogger.SetLevel("debug")

    logger := log.NewGologLogger(gologLogger)
    log.SetDefaultLogger(logger)
}

func main() {
    setupLogging()

    // 图执行现在将使用 golog
    g := graph.NewStateGraph[map[string]any]()
    // ... 构建图
}
```

### 节点特定日志

```go
g.AddNode("process", "处理数据", func(ctx context.Context, state any) (any, error) {
    log.Info("开始数据处理")
    log.Debugf("处理 %d 个项目", len(items))

    result, err := processData(state)
    if err != nil {
        log.Errorf("处理失败: %v", err)
        return state, err
    }

    log.Info("处理成功完成")
    return result, nil
})
```

## 使用 golog 的好处

1. **性能**：golog 为高吞吐量日志进行了优化
2. **灵活性**：易于配置不同的日志级别和输出
3. **熟悉的 API**：与其他流行的 Go 日志库类似
4. **零依赖**：轻量级，无外部依赖
5. **线程安全**：安全用于并发使用

## 最佳实践

1. **设置适当的日志级别**：开发环境使用 DEBUG，生产环境使用 INFO
2. **使用有意义的前缀**：帮助识别日志来源
3. **记录结构化数据**：考虑为结构化日志使用 JSON 格式
4. **避免敏感数据**：不要记录密码、令牌或个人身份信息
5. **考虑日志轮转**：对于长时间运行的应用程序

## 与其他日志器的集成

LangGraphGo 支持多种日志后端：

- **默认日志器**：内置的简单日志器
- **golog**：如本例所示
- **logrus**：流行的结构化日志库
- **zap**：高性能结构化日志
- **自定义实现**：实现 Logger 接口

要使用其他日志器，可以创建类似的适配器函数：

```go
func NewLogrusLogger(logrusLogger *logrus.Logger) log.Logger {
    return &logrusAdapter{logger: logrusLogger}
}

func NewZapLogger(zapLogger *zap.Logger) log.Logger {
    return &zapAdapter{logger: zapLogger}
}
```

## 相关示例

- [配置示例](../configuration/) - 演示图配置
- [内存示例](../memory/) - 带日志的内存管理
- [流式管道](../streaming_pipeline/) - 带日志的实时处理

## 了解更多

- [golog 文档](https://github.com/kataras/golog)
- [LangGraphGo 文档](../../README.md)
- [Go 日志最佳实践](https://go.dev/blog/go1.16errors)