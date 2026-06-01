// Package postgres provides PostgreSQL-backed storage for LangGraph Go checkpoints and state.
//
// This package implements durable checkpoint storage using PostgreSQL, allowing graph
// executions to be persisted and resumed across different runs and processes. It's
// designed for production use with robust error handling, connection pooling, and
// migration support.
//
// # Key Features
//
//   - Persistent storage of graph checkpoints
//   - Thread-safe operations
//   - Connection pooling for performance
//   - Automatic schema initialization and migrations
//   - Support for custom table names
//   - Efficient serialization of complex state objects
//   - Transaction support for data consistency
//   - TTL (time-to-live) support for automatic cleanup
//
// # Basic Usage
//
//	import (
//		"context"
//		"github.com/smallnest/langgraphgo/store/postgres"
//	)
//
//	// Create a PostgreSQL checkpoint store
//	store, err := postgres.NewPostgresCheckpointStore(ctx, postgres.PostgresOptions{
//		ConnString: "postgres://user:password@localhost/langgraph?sslmode=disable",
//		TableName:  "workflow_checkpoints", // Optional, defaults to "checkpoints"
//	})
//	if err != nil {
//		return err
//	}
//	defer store.Close()
//
//	// Initialize the database schema
//	if err := store.InitSchema(ctx); err != nil {
//		return err
//	}
//
//	// Use with a graph
//	g := graph.NewStateGraph[map[string]any]()
//	// ... configure graph ...
//
//	// Enable checkpointing
//	compileConfig := graph.CompileConfig{
//		CheckpointConfig: graph.CheckpointConfig{
//			Store: store,
//		},
//	}
//
//	runnable, err := g.CompileWithOptions(compileConfig)
//
// # Configuration
//
// ## Connection String
//
// The connection string follows PostgreSQL format:
//
//	postgres://[user[:password]@][host][:port][/dbname][?param1=value1&...]
//
// Examples:
//
//	// Local PostgreSQL
//	"postgres://postgres:password@localhost:5432/langgraph?sslmode=disable"
//
//	// With SSL
//	"postgres://user:pass@host:5432/dbname?sslmode=require"
//
//	// Unix socket
//	"postgres:///dbname?host=/var/run/postgresql"
//
// ## Connection Pool
//
// For more control over connection pooling:
//
//	pool, err := postgres.NewConnectionPool(ctx, postgres.PoolConfig{
//		ConnString:            connString,
//		MaxConns:              20,           // Maximum connections
//		MinConns:              5,            // Minimum connections
//		MaxConnLifetime:       time.Hour,    // Connection lifetime
//		MaxConnIdleTime:       30 * time.Minute,
//		HealthCheckPeriod:     time.Minute,
//	})
//
//	store, err := postgres.NewCheckpointStoreFromPool(pool, "checkpoints")
//
// # Advanced Features
//
// ## Custom Table Configuration
//
//	// Configure with custom table options
//	store, err := postgres.NewPostgresCheckpointStore(ctx, postgres.PostgresOptions{
//		ConnString: connString,
//		TableName:  "custom_checkpoints",
//	})
//
//	// The store will create the table with the following schema:
//	// CREATE TABLE IF NOT EXISTS custom_checkpoints (
//	//     id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
//	//     thread_id    VARCHAR(255) NOT NULL,
//	//     checkpoint_id VARCHAR(255) NOT NULL,
//	//     checkpoint_data BYTEA,
//	//     metadata     JSONB,
//	//     created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
//	//     expires_at   TIMESTAMP WITH TIME ZONE,
//	// );
//
// ## TTL Support
//
//	// Create checkpoints that expire after 24 hours
//	checkpoint := &graph.Checkpoint{
//		ID:       "checkpoint-123",
//		ThreadID: "thread-456",
//		State:    state,
//		Metadata: map[string]any{
//			"expires_at": time.Now().Add(24 * time.Hour),
//		},
//	}
//
//	// The store will automatically clean up expired checkpoints
//	if err := store.Put(ctx, checkpoint); err != nil {
//		return err
//	}
//
//	// Manually trigger cleanup
//	deleted, err := store.CleanupExpired(ctx)
//	fmt.Printf("Deleted %d expired checkpoints\n", deleted)
//
// ## Batch Operations
//
//	// Save multiple checkpoints in a transaction
//	checkpoints := []*graph.Checkpoint{checkpoint1, checkpoint2, checkpoint3}
//	if err := store.PutBatch(ctx, checkpoints); err != nil {
//		return err
//	}
//
//	// List checkpoints with pagination
//	listOptions := postgres.ListOptions{
//		ThreadID: "thread-456",
//		Limit:    10,
//		Offset:   20,
//		OrderBy:  "created_at DESC",
//	}
//
//	checkpoints, total, err := store.ListWithOptions(ctx, listOptions)
//
// # Migration and Schema Management
//
// ## Automatic Migration
//
//	// The store can automatically handle schema migrations
//	store, err := postgres.NewPostgresCheckpointStore(ctx, postgres.PostgresOptions{
//		ConnString: connString,
//		AutoMigrate: true, // Automatically create/update schema
//	})
//
// ## Manual Schema Control
//
//	// Get current schema version
//	version, err := store.GetSchemaVersion(ctx)
//	if err != nil {
//		return err
//	}
//
//	// Run migrations manually
//	if err := store.MigrateTo(ctx, targetVersion); err != nil {
//		return err
//	}
//
// # Performance Optimization
//
// ## Indexing Strategy
//
//	// Create additional indexes for better performance
//	_, err := pool.Exec(ctx, `
//		CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_checkpoints_thread_created
//		ON checkpoints (thread_id, created_at DESC);
//
//		CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_checkpoints_expires
//		ON checkpoints (expires_at) WHERE expires_at IS NOT NULL;
//	`)
//
// ## Connection Tuning
//
//	// Optimize for high-throughput scenarios
//	config := &pgxpool.Config{
//		ConnString: connString,
//		MaxConns:    50,                // More connections
//		MinConns:    10,                // Maintain minimum
//		MaxConnLifetime: 30 * time.Minute,
//		MaxConnIdleTime: 5 * time.Minute,
//		HealthCheckPeriod: 30 * time.Second,
//		// Additional performance tuning
//		BeforeAcquire: func(ctx context.Context, conn *pgx.Conn) bool {
//			// Validate connection before use
//			return conn.Ping(ctx) == nil
//		},
//	}
//
// # Monitoring and Metrics
//
//	// Get connection pool statistics
//	stats := store.Pool().Stat()
//	fmt.Printf("Total connections: %d\n", stats.TotalConns())
//	fmt.Printf("Idle connections: %d\n", stats.IdleConns())
//	fmt.Printf("Acquired connections: %d\n", stats.AcquiredConns())
//
//	// Monitor checkpoint operations
//	metrics := store.GetMetrics()
//	fmt.Printf("Puts: %d, Gets: %d, Lists: %d\n",
//		metrics.Puts, metrics.Gets, metrics.Lists)
//
// # Error Handling
//
//	// Handle specific PostgreSQL errors
//	if err := store.Put(ctx, checkpoint); err != nil {
//		var pgErr *pgconn.PgError
//		if errors.As(err, &pgErr) {
//			switch pgErr.Code {
//			case "23505": // Unique violation
//				// Handle duplicate checkpoint
//			case "23503": // Foreign key violation
//				// Handle missing reference
//			case "23514": // Check constraint violation
//				// Handle constraint failure
//			default:
//				// Handle other PostgreSQL errors
//			}
//		}
//	}
//
// # Integration Examples
//
// ## With Supervisor Agent
//
//	store, _ := postgres.NewPostgresCheckpointStore(ctx, postgres.PostgresOptions{
//		ConnString: "postgres://...",
//	})
//
//	members := map[string]*graph.StateRunnableUntyped
//		"worker1": worker1,
//		"worker2": worker2,
//	}
//
//	supervisor, _ := prebuilt.CreateSupervisor(
//		llm,
//		members,
//		"router",
//		graph.WithCheckpointing(graph.CheckpointConfig{
//			Store: store,
//		}),
//	)
//
//	// Execute with automatic checkpointing
//	result, _ := supervisor.Invoke(ctx, input,
//		graph.WithExecutionID("supervisor-run-123"))
//
//	// Resume from checkpoint
//	resumed, _ := supervisor.Resume(ctx, "supervisor-run-123", "checkpoint-456")
//
// ## With Streaming Execution
//
//	// Store streaming checkpoints
//	streaming := graph.NewStreamingStateGraph(g, graph.StreamConfig{
//		BufferSize: 100,
//	})
//
//	streaming.WithCheckpointing(graph.CheckpointConfig{
//		Store:     store,
//		CheckpointInterval: 5 * time.Second, // Checkpoint every 5 seconds
//	})
//
// # Best Practices
//
//  1. Use connection pooling for production
//  2. Set appropriate timeouts on operations
//  3. Handle transient errors with retries
//  4. Use transactions for multi-step operations
//  5. Monitor connection pool health
//  6. Implement proper cleanup for expired checkpoints
//  7. Use SSL/TLS for connections in production
//  8. Create indexes based on query patterns
//  9. Set up proper backup strategies
//  10. Test schema migrations in staging
//
// # Security Considerations
//
//   - Use environment variables for credentials
//   - Enable SSL/TLS for all connections
//   - Implement proper user permissions
//   - Use connection limiting
//   - Audit checkpoint access
//   - Encrypt sensitive data before storage
//   - Use prepared statements to prevent SQL injection
//
// # Docker Integration
//
// Use with Docker Compose:
//
// ```yaml
// version: '3.8'
// services:
//
//	langgraph:
//	  image: your-app
//	  environment:
//	    - DB_URL=postgres://postgres:password@postgres:5432/langgraph
//	  depends_on:
//	    - postgres
//
//	postgres:
//	  image: postgres:15
//	  environment:
//	    - POSTGRES_DB=langgraph
//	    - POSTGRES_USER=postgres
//	    - POSTGRES_PASSWORD=password
//	  volumes:
//	    - postgres_data:/var/lib/postgresql/data
//	  ports:
//	    - "5432:5432"
//
// volumes:
//
//	postgres_data:
//
// ```
package postgres
