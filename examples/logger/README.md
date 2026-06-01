# Golog Logger Integration

This example demonstrates how to integrate **golog** (https://github.com/kataras/golog) with LangGraphGo's logging system.

## Overview

LangGraphGo provides a flexible logging interface that can be configured to use various logging backends. This example shows how to use the popular golog library as the logging backend.

## What is golog?

golog is a fast, simple, and leveled logging library for Go that provides:
- **Leveled logging** (DEBUG, INFO, WARN, ERROR, FATAL)
- **Customizable prefixes**
- **Multiple output destinations**
- **Thread-safe logging**
- **Performance optimizations**

## Example Features

The example demonstrates three common logging scenarios:

### 1. Using Default golog Logger

```go
// Use the default golog instance
defaultLogger := golog.Default
logger1 := log.NewGologLogger(defaultLogger)
logger1.Info("Using default golog logger")
logger1.SetLevel(log.LogLevelDebug)
logger1.Debug("Debug information")
```

### 2. Creating Custom golog Logger

```go
// Create a custom logger with specific configuration
customLogger := golog.New()
customLogger.SetPrefix("[ MyApp ] ")
customLogger.SetOutput(os.Stdout)
logger2 := log.NewGologLogger(customLogger)
logger2.SetLevel(log.LogLevelInfo)
logger2.Info("Using custom golog logger")
```

### 3. Error-Only Logger

```go
// Create a logger that only shows error-level messages
errorLogger := golog.New()
errorLogger.SetLevel("error")
errorLogger.SetPrefix("[ ERROR ] ")
logger3 := log.NewGologLogger(errorLogger)
logger3.Debug("This won't be displayed")  // Filtered out
logger3.Error("Error messages will be displayed")
```

## Running the Example

```bash
cd examples/logger
go run golog_logger_example.go
```

## Expected Output

```
使用默认 golog logger
DEBUG [17:30:45] 调试信息
[ MyApp ] INFO [17:30:45] 使用自定义 golog logger
[ ERROR ] ERROR [17:30:45] 错误信息会显示
使用默认 golog logger
DEBUG [17:30:45] 调试信息
```

## Integration Steps

### Step 1: Import Required Packages

```go
import (
    "github.com/kataras/golog"
    "github.com/smallnest/langgraphgo/log"
)
```

### Step 2: Create golog Instance

```go
// Use default instance
gologLogger := golog.Default

// OR create custom instance
gologLogger := golog.New()
gologLogger.SetPrefix("[MyApp] ")
gologLogger.SetOutput(os.Stdout)
```

### Step 3: Wrap with LangGraphGo Logger

```go
// Wrap golog with LangGraphGo logger interface
logger := log.NewGologLogger(gologLogger)
```

### Step 4: Configure Log Level

```go
// Set appropriate log level
logger.SetLevel(log.LogLevelInfo)
```

### Step 5: Use as Default Logger (Optional)

```go
// Set as the default logger for LangGraphGo
log.SetDefaultLogger(logger)
```

## Advanced Configuration

### Custom Log Levels

```go
// Map golog levels to LangGraphGo levels
logger := log.NewGologLogger(gologLogger)

// Available levels:
// - log.LogLevelDebug
// - log.LogLevelInfo
// - log.LogLevelWarn
// - log.LogLevelError
// - log.LogLevelFatal
```

### Formatted Logging

```go
// golog supports formatted messages
logger.Infof("User %s logged in at %v", username, time.Now())
logger.Errorf("Failed to connect: %v", err)
```

### Multiple Outputs

```go
// Create multi-writer logger
multiWriter := io.MultiWriter(os.Stdout, logFile)
gologLogger := golog.New()
gologLogger.SetOutput(multiWriter)
logger := log.NewGologLogger(gologLogger)
```

## Using in LangGraphGo Applications

### Graph Execution Logging

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

    // Graph execution will now use golog
    g := graph.NewStateGraph[map[string]any]()
    // ... build graph
}
```

### Node-Specific Logging

```go
g.AddNode("process", "Process data", func(ctx context.Context, state any) (any, error) {
    log.Info("Starting data processing")
    log.Debugf("Processing %d items", len(items))

    result, err := processData(state)
    if err != nil {
        log.Errorf("Processing failed: %v", err)
        return state, err
    }

    log.Info("Processing completed successfully")
    return result, nil
})
```

## Benefits of Using golog

1. **Performance**: golog is optimized for high-throughput logging
2. **Flexibility**: Easy to configure different log levels and outputs
3. **Familiar API**: Similar to other popular Go logging libraries
4. **Zero Dependencies**: Lightweight with no external dependencies
5. **Thread Safety**: Safe for concurrent use

## Best Practices

1. **Set Appropriate Log Levels**: Use DEBUG for development, INFO for production
2. **Use Meaningful Prefixes**: Help identify log sources
3. **Log Structured Data**: Consider JSON formatting for structured logs
4. **Avoid Sensitive Data**: Don't log passwords, tokens, or PII
5. **Consider Log Rotation**: For long-running applications

## Integration with Other Loggers

LangGraphGo supports various logging backends:

- **Default logger**: Built-in simple logger
- **golog**: As shown in this example
- **logrus**: Popular structured logging library
- **zap**: High-performance structured logging
- **Custom implementations**: Implement the Logger interface

To use other loggers, you would create similar adapter functions:

```go
func NewLogrusLogger(logrusLogger *logrus.Logger) log.Logger {
    return &logrusAdapter{logger: logrusLogger}
}

func NewZapLogger(zapLogger *zap.Logger) log.Logger {
    return &zapAdapter{logger: zapLogger}
}
```

## Related Examples

- [Configuration Example](../configuration/) - Demonstrates graph configuration
- [Memory Examples](../memory/) - Shows memory management with logging
- [Streaming Pipeline](../streaming_pipeline/) - Real-time processing with logs

## Learn More

- [golog Documentation](https://github.com/kataras/golog)
- [LangGraphGo Documentation](../../README.md)
- [Go Best Practices for Logging](https://go.dev/blog/go1.16errors)