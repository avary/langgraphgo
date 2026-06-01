// Package store provides storage implementations for persisting LangGraph checkpoints and state.
//
// Store implementations allow graph executions to be persisted across different runs,
// processes, or even different machines. This enables features like resuming
// interrupted workflows, debugging complex executions, and maintaining state
// in distributed systems.
//
// The store package includes implementations for three popular storage backends:
//   - SQLite: Lightweight, serverless file-based storage
//   - PostgreSQL: Robust, scalable relational database
//   - Redis: High-performance in-memory storage
//
// # Core Concepts
//
// ## Checkpointing
//
// Checkpointing captures the state of a graph execution at specific points, including:
//   - The current node being executed
//   - The complete state object
//   - Execution metadata
//   - Timestamp and configuration information
//
// This allows execution to be paused and later resumed from the exact same state.
//
// ## Store Interface
//
// All store implementations follow the same interface defined in the graph package:
//
//	type CheckpointStore interface {
//	    // Save a checkpoint
//	    Put(ctx context.Context, checkpoint *Checkpoint) error
//
//	    // Retrieve a specific checkpoint
//	    Get(ctx context.Context, threadID, checkpointID string) (*Checkpoint, error)
//
//	    // List all checkpoints for a thread
//	    List(ctx context.Context, threadID string) ([]*Checkpoint, error)
//
//	    // Delete a checkpoint
//	    Delete(ctx context.Context, threadID, checkpointID string) error
//
//	    // Clear all checkpoints for a thread
//	    Clear(ctx context.Context, threadID string) error
//	}
//
// # Available Implementations
//
// ## SQLite Store (store/sqlite)
//
// Best for:
//   - Single-process applications
//   - Development and testing
//   - Desktop applications
//   - Scenarios requiring zero configuration
//
// Features:
//   - Serverless, file-based database
//   - ACID transactions
//   - No external dependencies
//   - Built-in full-text search
//
// Example:
//
//	import "github.com/smallnest/langgraphgo/store/sqlite"
//
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//	    Path: "./checkpoints.db",
//	})
//
// ## PostgreSQL Store (store/postgres)
//
// Best for:
//   - Production deployments
//   - High-throughput applications
//   - Complex querying requirements
//   - Distributed systems
//
// Features:
//   - Scalable relational database
//   - Connection pooling
//   - Advanced indexing
//   - JSONB support for metadata
//
// Example:
//
//	import "github.com/smallnest/langgraphgo/store/postgres"
//
//	store, err := postgres.NewPostgresCheckpointStore(ctx, postgres.PostgresOptions{
//	    ConnString: "postgres://user:pass@localhost/langgraph",
//	})
//
// ## Redis Store (store/redis)
//
// Best for:
//   - High-performance requirements
//   - Distributed caching scenarios
//   - Temporary checkpoint storage
//   - Real-time collaboration features
//
// Features:
//   - In-memory storage with optional persistence
//   - Automatic TTL (time-to-live) expiration
//   - Atomic operations
//   - Pub/Sub notifications
//
// Example:
//
//	import "github.com/smallnest/langgraphgo/store/redis"
//
//	store := redis.NewRedisCheckpointStore(redis.RedisOptions{
//	    Addr: "localhost:6379",
//	    TTL:  24 * time.Hour,
//	})
//
// # Usage Patterns
//
// ## Basic Checkpointing
//
//	// Create a graph
//	g := graph.NewStateGraph[map[string]any]()
//	// ... configure graph ...
//
//	// Choose and configure a store
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//	    Path: "./checkpoints.db",
//	})
//	if err != nil {
//	    return err
//	}
//	defer store.Close()
//
//	// Enable checkpointing
//	compileConfig := graph.CompileConfig{
//	    CheckpointConfig: graph.CheckpointConfig{
//	        Store: store,
//	    },
//	}
//
//	runnable, err := g.CompileWithOptions(compileConfig)
//
//	// Execute with automatic checkpointing
//	result, err := runnable.Invoke(ctx, input,
//	    graph.WithExecutionID("unique-execution-id"))
//
//	// Resume from a checkpoint
//	resumed, err := runnable.Resume(ctx,
//	    "unique-execution-id",
//	    "checkpoint-to-resume-from")
//
// ## Custom Checkpointing Strategy
//
//	// Checkpoint at specific intervals
//	type IntervalCheckpointConfig struct {
//	    graph.CheckpointConfig
//	    Interval time.Duration
//	    LastCheckpoint time.Time
//	}
//
//	// Or checkpoint on specific conditions
//	type ConditionalCheckpointConfig struct {
//	    graph.CheckpointConfig
//	    ShouldCheckpoint func(state any) bool
//	}
//
// # Choosing the Right Store
//
// ## Decision Guide
//
// Use SQLite when:
//   - You need a simple, self-contained solution
//   - Your application runs on a single machine
//   - You prefer zero configuration
//   - You need to store checkpoints in files
//
// Use PostgreSQL when:
//   - You need robust persistence and scalability
//   - Your application requires complex queries
//   - You have multiple processes accessing the same data
//   - You need enterprise-grade features (backups, replication)
//
// Use Redis when:
//   - Performance is the primary concern
//   - You need automatic expiration of old checkpoints
//   - You're building a distributed system
//   - You need real-time notifications for checkpoint changes
//
// ## Migration Between Stores
//
// The package provides utilities to migrate between different store implementations:
//
//	// Migrate from SQLite to PostgreSQL
//	migrator := store.NewMigrator(sqliteStore, postgresStore)
//	err := migrator.MigrateAll(ctx)
//
//	// Or migrate specific checkpoints
//	err := migrator.MigrateThread(ctx, "thread-id")
//
// # Performance Considerations
//
// ## Serialization
//
// All stores use JSON serialization for checkpoint data. For optimal performance:
//   - Keep state objects relatively small
//   - Avoid storing large binary data in checkpoints
//   - Consider compression for large state objects
//
// ## Batch Operations
//
// Some stores support batch operations for better performance:
//
//	// Batch save multiple checkpoints
//	checkpoints := []*graph.Checkpoint{cp1, cp2, cp3}
//	err := store.PutBatch(ctx, checkpoints)
//
// # Best Practices
//
//  1. **Choose the right store for your use case**
//     - SQLite for simple applications
//     - PostgreSQL for production systems
//     - Redis for high-performance scenarios
//
//  2. **Handle errors gracefully**
//     - Implement retry logic for transient errors
//     - Provide fallback mechanisms
//     - Log checkpoint failures for debugging
//
//  3. **Manage checkpoint lifecycle**
//     - Clean up old checkpoints regularly
//     - Use TTL for automatic cleanup (Redis)
//     - Implement retention policies
//
//  4. **Secure checkpoint data**
//     - Encrypt sensitive data before storage
//     - Use secure database connections
//     - Implement proper access controls
//
//  5. **Monitor storage usage**
//     - Track checkpoint sizes and counts
//     - Monitor database performance metrics
//     - Set up alerts for storage limits
//
// # Integration with LangGraph
//
// Stores integrate seamlessly with all LangGraph components:
//
//	// With prebuilt agents
//	agent := prebuilt.CreateReactAgent(llm, tools, 10,
//	    prebuilt.WithCheckpointing(graph.CheckpointConfig{
//	        Store: store,
//	    }),
//	)
//
//	// With custom graphs
//	g := graph.NewStateGraph[map[string]any]()
//	g.WithCheckpointing(graph.CheckpointConfig{
//	    Store: store,
//	})
//
//	// With streaming execution
//	streaming := graph.NewStreamingStateGraph(g, graph.StreamConfig{
//	    BufferSize: 100,
//	})
//	streaming.WithCheckpointing(graph.CheckpointConfig{
//	    Store: store,
//	})
//
// # Advanced Features
//
// ## Checkpoint Versioning
//
// Some stores support checkpoint versioning for tracking evolution:
//
//	versionedStore := postgres.NewVersionedCheckpointStore(opts)
//	err := versionedStore.PutVersion(ctx, checkpoint, "v1.0")
//
//	versions, err := versionedStore.ListVersions(ctx, checkpointID)
//
// ## Checkpoint Compression
//
// For large state objects, consider compression:
//
//	compressedStore := store.NewCompressedWrapper(store, gzip.BestCompression)
//	err := compressedStore.Put(ctx, checkpoint)
//
// ## Checkpoint Encryption
//
// Encrypt sensitive checkpoint data:
//
//	encryptedStore := store.NewEncryptedWrapper(store, encryptionKey)
//	err := encryptedStore.Put(ctx, checkpoint)
//
// # Extending the Package
//
// To add a new store implementation:
//
//  1. Implement the CheckpointStore interface
//  2. Add the package to the store directory
//  3. Create comprehensive tests
//  4. Add documentation with examples
//  5. Include migration utilities if needed
//
// Example implementation structure:
//
//	package mystore
//
//	type MyStore struct {
//	    // Implementation details
//	}
//
//	func (s *MyStore) Put(ctx context.Context, cp *graph.Checkpoint) error {
//	    // Implementation
//	}
//
//	// Implement other interface methods...
//
// # Community Contributions
//
// The store package welcomes contributions for additional storage backends:
//   - MongoDB store
//   - DynamoDB store
//   - Cassandra store
//   - S3/object storage store
//   - etcd store
//
// Please follow the established patterns and provide comprehensive tests and documentation.
package store
