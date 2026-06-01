// Package log provides a simple, leveled logging interface for LangGraph Go applications.
//
// This package implements a lightweight logging system with support for different log levels
// and customizable output destinations. It's designed to integrate seamlessly with the
// LangGraph execution engine, particularly for PTC (Problem-Tactic-Criticality) workflows.
//
// # Log Levels
//
// The package supports five log levels, in order of increasing severity:
//
//   - LogLevelDebug: Detailed debugging information for development
//   - LogLevelInfo: General informational messages about normal operation
//   - LogLevelWarn: Warning messages for potentially problematic situations
//   - LogLevelError: Error messages for failures that need attention
//   - LogLevelNone: Disables all logging output
//
// # Logger Interface
//
// The Logger interface provides four main logging methods:
//
//   - Debug: For detailed troubleshooting information
//   - Info: For general application flow information
//   - Warn: For issues that don't stop execution but need attention
//   - Error: For failures and exceptions
//
// # Example Usage
//
// ## Basic Logging
//
//	// Create a logger with INFO level
//	logger := log.NewDefaultLogger(log.LogLevelInfo)
//
//	// Log messages at different levels
//	logger.Info("Application starting")
//	logger.Debug("Processing request: %v", request)
//	logger.Warn("Rate limit approaching: %d requests", count)
//	logger.Error("Failed to process: %v", err)
//
// ## Custom Output
//
//	// Create a logger that writes to a file
//	file, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer file.Close()
//
//	logger := log.NewCustomLogger(file, log.LogLevelDebug)
//	logger.Debug("This will go to the file")
//
// ## Multi-Writer Logging
//
//	// Create a logger that writes to both console and file
//	multiWriter := io.MultiWriter(os.Stdout, file)
//	logger := log.NewCustomWriterLogger(multiWriter, log.LogLevelInfo)
//
// ## Filtering by Level
//
//	// Create a debug logger for development
//	debugLogger := log.NewDefaultLogger(log.LogLevelDebug)
//
//	// Create a production logger that only shows warnings and errors
//	prodLogger := log.NewDefaultLogger(log.LogLevelWarn)
//
//	// Messages below the set level are filtered out
//	debugLogger.Debug("Visible in debug mode")
//	prodLogger.Debug("Not visible in production")
//
// # Integration with LangGraph
//
// The logger is designed to work with the graph package's listener system:
//
//	import (
//		"github.com/smallnest/langgraphgo/graph"
//		"github.com/smallnest/langgraphgo/log"
//	)
//
//	logger := log.NewDefaultLogger(log.LogLevelInfo)
//
//	g := graph.NewStateGraph[map[string]any]()
//	// ... configure graph ...
//
//	// Add a logging listener
//	listener := graph.NewLoggingListener(logger, log.LogLevelInfo, false)
//	g.AddListener(listener)
//
// # Performance Considerations
//
//   - Log messages are formatted using fmt.Sprintf() - avoid complex formatting in hot paths
//   - Consider setting LogLevelError or LogLevelNone in production for better performance
//   - Buffer file writes for high-volume logging scenarios
//
// # Thread Safety
//
// The DefaultLogger implementation is thread-safe and can be used concurrently from
// multiple goroutines. The underlying log.Logger from Go's standard library handles
// synchronization internally.
//
// # Available Implementations
//
// ## Standard Library Logger
//
// The package provides a DefaultLogger implementation using Go's standard log package.
//
// ## golog Integration
//
// For users who prefer the `github.com/kataras/golog` library, we provide a minimal wrapper:
//
//	import "github.com/kataras/golog"
//
//	// Create a golog logger
//	glogger := golog.New()
//	glogger.SetPrefix("[MyApp] ")
//
//	// Wrap it with LangGraph's Logger interface
//	logger := log.NewGologLogger(glogger)
//
//	// Use like any other LangGraph logger
//	logger.Info("Application started")
//	logger.SetLevel(log.LogLevelDebug)
//	logger.Debug("Debug information")
//
// Key points:
//   - `NewGologLogger()` requires an existing golog.Logger instance
//   - Implements the same Logger interface as other loggers
//   - Respects LangGraph log levels while using golog's formatting
//   - Minimal wrapper - just forwards calls to the underlying golog logger
//
// # Custom Loggers
//
// You can implement the Logger interface for custom logging solutions:
//
//	type CustomLogger struct {
//		// Custom fields
//	}
//
//	func (l *CustomLogger) Debug(format string, v ...any) {
//		// Custom debug implementation
//	}
//
//	func (l *CustomLogger) Info(format string, v ...any) {
//		// Custom info implementation
//	}
//
//	func (l *CustomLogger) Warn(format string, v ...any) {
//		// Custom warn implementation
//	}
//
//	func (l *CustomLogger) Error(format string, v ...any) {
//		// Custom error implementation
//	}
//
// # Best Practices
//
//  1. Use appropriate log levels - Debug for development, Info for operation flow,
//     Warn for recoverable issues, Error for failures
//
//  2. Include context in log messages but avoid sensitive data
//
//  3. Consider structured logging formats for easier parsing in production
//
//  4. Rotate log files for long-running applications
//
//  5. Use conditional logging to avoid unnecessary string formatting:
//
//     if logger.LevelEnabled(log.LogLevelDebug) {
//     logger.Debug("Complex data: %+v", complexStruct)
//     }
package log
