// Package sqlite provides SQLite-backed storage for LangGraph Go checkpoints and state.
//
// This package implements file-based checkpoint storage using SQLite, perfect for
// applications requiring a lightweight, serverless database solution with ACID
// compliance and zero external dependencies.
//
// # Key Features
//
//   - Serverless, file-based database
//   - ACID transaction support
//   - Zero configuration needed
//   - Cross-platform compatibility
//   - Embedded database (no separate server process)
//   - Full-text search capabilities
//   - Thread-safe operations
//   - Support for custom table schemas
//   - Backup and restore functionality
//   - WAL mode for concurrent access
//
// # Basic Usage
//
//	import (
//		"context"
//		"github.com/smallnest/langgraphgo/store/sqlite"
//	)
//
//	// Create a SQLite checkpoint store
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//		Path:      "./checkpoints.db", // Database file path
//		TableName: "checkpoints",     // Optional table name
//	})
//	if err != nil {
//		return err
//	}
//	defer store.Close()
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
// ## Database File Options
//
//	// In-memory database (volatile)
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//		Path: ":memory:",
//	})
//
//	// Temporary file database
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//		Path: "",
//	}) // Creates temporary file
//
//	// Persistent file database
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//		Path: "./data/langgraph.db",
//	})
//
//	// With custom URI options
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//		Path: "file:./checkpoints.db?cache=shared&mode=rwc",
//	})
//
// ## Connection Pooling
//
//	// Configure connection pool
//	store, err := sqlite.NewSqliteCheckpointStoreWithPool(sqlite.SqlitePoolOptions{
//		Path: "./checkpoints.db",
//		MaxOpenConns: 10,
//		MaxIdleConns: 5,
//		ConnMaxLifetime: time.Hour,
//	})
//
// # Advanced Features
//
// ## WAL Mode for Concurrency
//
//	// Enable Write-Ahead Logging for better concurrent access
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//		Path: "./checkpoints.db",
//	})
//
//	// Enable WAL mode
//	if err := store.EnableWAL(context.Background()); err != nil {
//		return err
//	}
//
//	// Configure WAL checkpointing
//	if err := store.SetWALCheckpointMode(
//		context.Background(),
//		sqlite.WALCheckpointPassive,
//		1000, // WAL size threshold
//	); err != nil {
//		return err
//	}
//
// ## Custom Schema
//
//	// Initialize with custom table schema
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//		Path:      "./checkpoints.db",
//		TableName: "custom_checkpoints",
//	})
//
//	// Add custom indexes
//	_, err = store.Exec(context.Background(), `
//		CREATE INDEX IF NOT EXISTS idx_custom_checkpoints_thread_id
//		ON custom_checkpoints (thread_id);
//
//		CREATE INDEX IF NOT EXISTS idx_custom_checkpoints_created_at
//		ON custom_checkpoints (created_at DESC);
//	`)
//
//	// Add full-text search
//	_, err = store.Exec(context.Background(), `
//		CREATE VIRTUAL TABLE IF NOT EXISTS checkpoint_search
//		USING fts5(checkpoint_id, thread_id, content);
//	`)
//
// ## Backup and Restore
//
//	// Create backup
//	err := store.Backup(context.Background(), "./backup/checkpoints_backup.db")
//	if err != nil {
//		return err
//	}
//
//	// Restore from backup
//	err := store.Restore(context.Background(), "./backup/checkpoints_backup.db")
//	if err != nil {
//		return err
//	}
//
//	// Incremental backup
//	err := store.IncrementalBackup(context.Background(), "./backup/incremental.db")
//
// # Querying and Analytics
//
// ## Advanced Queries
//
//	// Find checkpoints by thread with pagination
//	checkpoints, err := store.QueryWithPagination(context.Background(), `
//		SELECT * FROM checkpoints
//		WHERE thread_id = ?
//		ORDER BY created_at DESC
//		LIMIT ? OFFSET ?
//	`, "thread-123", 10, 0)
//
//	// Search checkpoints by content
//	results, err := store.SearchCheckpoints(context.Background(), `
//		SELECT checkpoint_id, thread_id, created_at
//		FROM checkpoint_search
//		WHERE checkpoint_search MATCH ?
//		ORDER BY rank
//	`, "error OR exception")
//
//	// Get checkpoint statistics
//	stats, err := store.GetStatistics(context.Background())
//	fmt.Printf("Total checkpoints: %d\n", stats.Total)
//	fmt.Printf("Threads: %d\n", stats.Threads)
//	fmt.Printf("Average size: %.2f KB\n", stats.AverageSize)
//
//	type CheckpointStats struct {
//		Total       int64
//		Threads     int64
//		AverageSize float64
//		Oldest      time.Time
//		Newest      time.Time
//	}
//
// # Performance Optimization
//
// ## Pragmas and Settings
//
//	// Optimize for performance
//	pragmas := map[string]string{
//		"journal_mode":      "WAL",
//		"synchronous":       "NORMAL",
//		"cache_size":        "10000",
//		"temp_store":        "MEMORY",
//		"mmap_size":         "268435456", // 256MB
//		"wal_autocheckpoint": "1000",
//	}
//
//	for key, value := range pragmas {
//		_, err := store.Exec(context.Background(),
//			fmt.Sprintf("PRAGMA %s = %s", key, value))
//		if err != nil {
//			return err
//		}
//	}
//
// ## Prepared Statements
//
//	// Use prepared statements for frequent operations
//	insertStmt, err := store.Prepare(context.Background(), `
//		INSERT OR REPLACE INTO checkpoints
//		(id, thread_id, checkpoint_id, checkpoint_data, metadata, created_at)
//		VALUES (?, ?, ?, ?, ?, ?)
//	`)
//	if err != nil {
//		return err
//	}
//	defer insertStmt.Close()
//
//	// Use the prepared statement
//	_, err = insertStmt.ExecContext(context.Background(),
//		checkpoint.ID,
//		checkpoint.ThreadID,
//		checkpoint.CheckpointID,
//		checkpointData,
//		metadataJSON,
//		time.Now(),
//	)
//
// # Transactions
//
//	// Atomic operations with transactions
//	err := store.Transaction(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
//		// Checkpoint current state
//		if err := store.PutTx(ctx, tx, checkpoint1); err != nil {
//			return err
//		}
//
//		// Save related data
//		if err := saveRelatedDataTx(ctx, tx, relatedData); err != nil {
//			return err
//		}
//
//		// Update metadata
//		if err := updateMetadataTx(ctx, tx, metadata); err != nil {
//			return err
//		}
//
//		return nil
//	})
//
// # Monitoring and Maintenance
//
//	// Analyze database
//	_, err := store.Exec(context.Background(), "ANALYZE")
//
//	// Vacuum to reclaim space
//	_, err = store.Exec(context.Background(), "VACUUM")
//
//	// Check integrity
//	result, err := store.QueryRow(context.Background(), "PRAGMA integrity_check").Scan(&result)
//	if result != "ok" {
//		return fmt.Errorf("database integrity check failed: %s", result)
//	}
//
//	// Get database info
//	info := &DatabaseInfo{}
//	store.QueryRow(context.Background(), "PRAGMA page_size").Scan(&info.PageSize)
//	store.QueryRow(context.Background(), "PRAGMA page_count").Scan(&info.PageCount)
//	info.DatabaseSize = int64(info.PageSize) * int64(info.PageCount)
//
//	type DatabaseInfo struct {
//		PageSize    int
//		PageCount   int
//		DatabaseSize int64
//	}
//
// # Error Handling
//
//	// Handle SQLite-specific errors
//	if err := store.Put(ctx, checkpoint); err != nil {
//		if sqlite.IsConstraint(err) {
//			// Handle constraint violation
//			switch sqlite.ConstraintType(err) {
//			case sqlite.ConstraintPrimaryKey:
//				// Duplicate primary key
//			case sqlite.ConstraintUnique:
//				// Unique constraint violation
//			case sqlite.ConstraintForeignKey:
//				// Foreign key violation
//			}
//		} else if sqlite.IsBusy(err) {
//			// Database is locked
//			time.Sleep(time.Second)
//			// Retry...
//		} else if sqlite.IsLocked(err) {
//			// Table is locked
//		}
//	}
//
// # Integration Examples
//
// ## With Desktop Application
//
//	// Local file database for desktop app
//	appDataDir, _ := os.UserConfigDir()
//	dbPath := filepath.Join(appDataDir, "myapp", "checkpoints.db")
//
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//		Path: dbPath,
//	})
//
//	// Ensure directory exists
//	os.MkdirAll(filepath.Dir(dbPath), 0755)
//
// ## With Web Application
//
//	// Per-user SQLite databases
//	func getUserStore(userID string) (graph.CheckpointStore, error) {
//		userDir := filepath.Join("./data", "users", userID)
//		os.MkdirAll(userDir, 0755)
//
//		dbPath := filepath.Join(userDir, "checkpoints.db")
//		return sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//			Path: dbPath,
//		})
//	}
//
// ## Development and Testing
//
//	// In-memory database for tests
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//		Path: ":memory:",
//	})
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	// Or use shared in-memory for multiple connections
//	store, err := sqlite.NewSqliteCheckpointStore(sqlite.SqliteOptions{
//		Path: "file::memory:?cache=shared",
//	})
//
// # Best Practices
//
//  1. Use WAL mode for concurrent access
//  2. Set appropriate timeouts for operations
//  3. Use transactions for multi-step operations
//  4. Close connections when done
//  5. Regularly run VACUUM and ANALYZE
//  6. Monitor database size and performance
//  7. Use prepared statements for frequent queries
//  8. Implement proper backup strategies
//  9. Handle database locking gracefully
//  10. Consider connection pooling for web applications
//
// # Security Considerations
//
//   - Set appropriate file permissions
//   - Use directory isolation for multi-tenant apps
//   - Validate inputs to prevent SQL injection
//   - Encrypt sensitive data before storage
//   - Use file system permissions for access control
//   - Consider full disk encryption for sensitive data
//   - Implement proper backup encryption
//   - Audit database access
//
// # Docker Integration
//
// Use with Docker volumes:
//
// ```yaml
// version: '3.8'
// services:
//
//	langgraph:
//	  image: your-app
//	  volumes:
//	    - ./data:/app/data
//	  environment:
//	    - SQLITE_PATH=/app/data/checkpoints.db
//
//	backup:
//	  image: your-backup-app
//	  volumes:
//	    - ./data:/app/data:ro
//	    - ./backups:/app/backups
//
// ```
//
// # Comparison with Other Stores
//
// | Feature              | SQLite Store | Redis Store | PostgreSQL Store |
// |---------------------|--------------|-------------|------------------|
// | Performance          | Medium       | Very High   | High             |
// | Memory Usage         | Low          | High        | Low              |
// | Concurrency          | Limited      | High        | High             |
// | Persistence          | Yes          | Optional    | Yes              |
// | Scaling              | Single       | Cluster     | Cluster          |
// | Query Capabilities   | SQL          | Basic       | Advanced SQL     |
// | Setup Complexity     | None         | Low         | Medium           |
// | Best For           | Small/Medium | High-speed  | Enterprise       |
// | File Size           | Up to TB     | RAM limited  | Unlimited        |
// | Backup              | Simple copy  | Export/Import| pg_dump          |
package sqlite
