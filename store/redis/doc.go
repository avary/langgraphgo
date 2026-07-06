// Package redis provides Redis-backed storage for LangGraph Go checkpoints and state.
//
// This package implements fast, in-memory checkpoint storage using Redis, ideal for
// scenarios requiring low-latency access to checkpoints and supporting distributed
// graph executions across multiple processes or servers.
//
// # Key Features
//
//   - High-performance checkpoint storage with Redis
//   - Support for TTL (time-to-live) automatic expiration
//   - Atomic operations for consistency
//   - Distributed locking support
//   - Configurable key prefixes for multi-tenancy
//   - JSON serialization of complex state objects
//   - Connection pooling and clustering support
//   - Pub/Sub notifications for checkpoint changes
//
// # Basic Usage
//
//	import (
//		"context"
//		"github.com/smallnest/langgraphgo/store/redis"
//	)
//
//	// Create a Redis checkpoint store
//	store := redis.NewRedisCheckpointStore(redis.RedisOptions{
//		Addr:     "localhost:6379",
//		Password: "yourpassword",
//		DB:       0,                    // Redis database number
//		Prefix:   "langgraph:",         // Optional key prefix
//		TTL:      24 * time.Hour,       // Optional TTL for checkpoints
//	})
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
// ## Connection Options
//
//	// Single Redis instance
//	store := redis.NewRedisCheckpointStore(redis.RedisOptions{
//		Addr:     "localhost:6379",
//		Password: "",
//		DB:       0,
//	})
//
//	// With authentication
//	store := redis.NewRedisCheckpointStore(redis.RedisOptions{
//		Addr:     "redis.example.com:6379",
//		Password: "your-redis-password",
//		DB:       1,
//	})
//
//	// With Unix socket
//	store := redis.NewRedisCheckpointStore(redis.RedisOptions{
//		Addr:     "/var/run/redis/redis.sock",
//		Password: "",
//		DB:       0,
//	})
//
// ## TTL Configuration
//
//	// Set default TTL for all checkpoints (24 hours)
//	store := redis.NewRedisCheckpointStore(redis.RedisOptions{
//		Addr: "localhost:6379",
//		TTL:  24 * time.Hour,
//	})
//
//	// No expiration (persistent)
//	store := redis.NewRedisCheckpointStore(redis.RedisOptions{
//		Addr: "localhost:6379",
//		TTL:  0,
//	})
//
// # Advanced Features
//
// ## Custom Redis Client
//
//	// Use a custom Redis client for more control
//	rdb := redis.NewClient(&redis.Options{
//		Addr:         "localhost:6379",
//		Password:     "",
//		DB:           0,
//		MaxRetries:   3,
//		PoolSize:     10,
//		MinIdleConns: 5,
//		DialTimeout:  5 * time.Second,
//		ReadTimeout:  3 * time.Second,
//		WriteTimeout: 3 * time.Second,
//		PoolTimeout:  4 * time.Second,
//	})
//
//	store := redis.NewCheckpointStoreFromClient(rdb, "langgraph:", time.Hour)
//
// ## Clustering Support
//
//	// Redis Cluster configuration
//	rdb := redis.NewClusterClient(&redis.ClusterOptions{
//		Addrs: []string{
//			"redis-node-1:6379",
//			"redis-node-2:6379",
//			"redis-node-3:6379",
//		},
//		Password: "cluster-password",
//	})
//
//	store := redis.NewCheckpointStoreFromCluster(rdb, "langgraph:", time.Hour)
//
// ## Sentinel Support
//
//	// Redis Sentinel for high availability
//	rdb := redis.NewFailoverClient(&redis.FailoverOptions{
//		MasterName:    "mymaster",
//		SentinelAddrs: []string{
//			"sentinel-1:26379",
//			"sentinel-2:26379",
//			"sentinel-3:26379",
//		},
//		Password: "sentinel-password",
//	})
//
//	store := redis.NewCheckpointStoreFromClient(rdb, "langgraph:", time.Hour)
//
// # Key Management
//
//	// Checkpoints are stored with structured keys
//	// Format: {prefix}checkpoint:{checkpoint_id}
//	// Example: "langgraph:checkpoint:abc123"
//
//	// Thread-specific checkpoints
//	// Format: {prefix}thread:{thread_id}:checkpoint:{checkpoint_id}
//	// Example: "langgraph:thread:xyz789:checkpoint:def456"
//
//	// List all checkpoints for a thread
//	keys, err := store.client.Keys(ctx, "langgraph:thread:xyz789:checkpoint:*")
//
// ## Custom TTL per Checkpoint
//
//	// Override default TTL for specific checkpoint
//	checkpoint := &graph.Checkpoint{
//		ID:       "checkpoint-123",
//		ThreadID: "thread-456",
//		State:    state,
//		Metadata: map[string]any{
//			"ttl": 2 * time.Hour, // Custom TTL
//		},
//	}
//
//	if err := store.PutWithTTL(ctx, checkpoint, 2*time.Hour); err != nil {
//		return err
//	}
//
// # Pub/Sub Notifications
//
//	// Subscribe to checkpoint changes
//	pubsub := store.client.Subscribe(ctx, "langgraph:checkpoint:changes")
//
//	go func() {
//		for msg := range pubsub.Channel() {
//			var event struct {
//				Action     string    `json:"action"`
//				CheckpointID string   `json:"checkpoint_id"`
//				ThreadID   string    `json:"thread_id"`
//				Timestamp  time.Time `json:"timestamp"`
//			}
//			json.Unmarshal([]byte(msg.Payload), &event)
//
//			fmt.Printf("Checkpoint %s: %s\n", event.Action, event.CheckpointID)
//		}
//	}()
//
// # Performance Optimization
//
// ## Pipeline Operations
//
//	// Batch operations with pipelining
//	pipe := store.client.Pipeline()
//
//	checkpoints := []*graph.Checkpoint{cp1, cp2, cp3}
//	for _, cp := range checkpoints {
//		data, _ := json.Marshal(cp)
//		pipe.Set(ctx, store.checkpointKey(cp.ID), data, store.ttl)
//	}
//
//	// Execute all operations atomically
//	_, err := pipe.Exec(ctx)
//
// ## Lua Scripts
//
//	// Atomic update with Lua
//	updateScript := redis.NewScript(`
//		local key = KEYS[1]
//		local checkpoint_id = ARGV[1]
//		local new_data = ARGV[2]
//
//		local old = redis.call('GET', key)
//		if old then
//			redis.call('SET', key, new_data, 'EX', ARGV[3])
//			return old
//		end
//		return nil
//	`)
//
//	result, err := updateScript.Run(ctx, store.client,
//		[]string{store.checkpointKey(checkpointID)},
//		checkpointID, newData, int(ttl.Seconds()),
//	).Result()
//
// # Monitoring and Metrics
//
//	// Get Redis information
//	info, err := store.client.Info(ctx).Result()
//	if err == nil {
//		fmt.Printf("Redis Info: %s\n", info)
//	}
//
//	// Monitor memory usage
//	memStats, err := store.client.MemoryUsage(ctx, store.checkpointKey("*")).Result()
//	if err == nil {
//		fmt.Printf("Memory usage: %v\n", memStats)
//	}
//
//	// Track operations
//	monitor := &RedisMonitor{
//		client: store.client,
//		metrics: make(map[string]int64),
//	}
//
//	store.SetMonitor(monitor)
//
//	type RedisMonitor struct {
//		client  *redis.Client
//		metrics map[string]int64
//		mutex   sync.RWMutex
//	}
//
//	func (m *RedisMonitor) OnOperation(op string, duration time.Duration) {
//		m.mutex.Lock()
//		defer m.mutex.Unlock()
//		m.metrics[op]++
//	}
//
// # Error Handling
//
//	// Handle Redis-specific errors
//	if err := store.Put(ctx, checkpoint); err != nil {
//		if redis.IsNil(err) {
//			// Handle not found
//		} else if redis.IsPoolTimeout(err) {
//			// Handle connection pool timeout
//		} else if redis.IsConnectionError(err) {
//			// Handle connection error
//		}
//	}
//
//	// Retry logic
//	store := redis.NewRedisCheckpointStore(redis.RedisOptions{
//		Addr: "localhost:6379",
//	})
//
//	// With retry policy
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	for i := 0; i < 3; i++ {
//		err := store.Put(ctx, checkpoint)
//		if err == nil {
//			break
//		}
//		if i == 2 {
//			return err
//		}
//		time.Sleep(time.Second * time.Duration(i+1))
//	}
//
// # Integration Examples
//
// ## With Distributed Execution
//
//	// Multiple processes sharing the same Redis store
//	store := redis.NewRedisCheckpointStore(redis.RedisOptions{
//		Addr:   "redis-cluster:6379",
//		Prefix: "distributed-langgraph:",
//		TTL:    6 * time.Hour, // Checkpoints persist for 6 hours
//	})
//
//	// Process 1
//	runnable1, _ := g1.Compile()
//	go func() {
//		result, _ := runnable1.Invoke(ctx, input,
//			graph.WithExecutionID("shared-execution-123"))
//	}()
//
//	// Process 2 can resume from the same execution
//	runnable2, _ := g2.Compile()
//	result, _ := runnable2.Resume(ctx, "shared-execution-123", "checkpoint-456")
//
// ## With Session Affinity
//
//	// Store checkpoints per user session
//	func getUserStore(userID string) graph.CheckpointStore {
//		return redis.NewRedisCheckpointStore(redis.RedisOptions{
//			Addr:   "localhost:6379",
//			Prefix: fmt.Sprintf("user:%s:langgraph:", userID),
//			TTL:    2 * time.Hour, // Session timeout
//		})
//	}
//
//	userStore := getUserStore("user-123")
//
// # Best Practices
//
//  1. Use meaningful key prefixes for organization
//  2. Set appropriate TTL to prevent memory bloat
//  3. Use connection pooling in production
//  4. Implement proper error handling with retries
//  5. Monitor Redis memory usage
//  6. Use Redis Cluster for high availability
//  7. Enable persistence for critical data
//  8. Use pipelining for batch operations
//  9. Consider compression for large state objects
//  10. Test failover scenarios
//
// # Security Considerations
//
//   - Enable Redis AUTH in production
//   - Use TLS/SSL for network connections
//   - Implement proper network isolation
//   - Use Redis ACLs for fine-grained access control
//   - Encrypt sensitive data before storage
//   - Disable dangerous commands (CONFIG, FLUSHDB)
//   - Set up proper firewalls
//   - Monitor for suspicious activity
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
//	    - REDIS_ADDR=redis:6379
//	    - REDIS_PASSWORD=yourpassword
//	  depends_on:
//	    - redis
//
//	redis:
//	  image: redis:7-alpine
//	  command: redis-server --requirepass yourpassword
//	  ports:
//	    - "6379:6379"
//	  volumes:
//	    - redis_data:/data
//
//	redis-commander:
//	  image: rediscommander/redis-commander:latest
//	  environment:
//	    - REDIS_HOSTS=local:redis:6379:0:yourpassword
//	  ports:
//	    - "8081:8081"
//
// volumes:
//
//	redis_data:
//
// ```
//
// # Comparison with Other Stores
//
// | Feature              | Redis Store | PostgreSQL Store | SQLite Store |
// |---------------------|-------------|------------------|-------------|
// | Performance          | Very High   | High             | Medium      |
// | Persistence          | Optional    | Yes              | Yes         |
// | Memory Usage         | High        | Low              | Low         |
// | Scaling              | Horizontal  | Vertical         | Single      |
// | Query Capabilities   | Basic       | Advanced         | Basic       |
// | Transactions        | Limited     | Full             | Full        |
// | TTL Support         | Native      | Manual           | Manual      |
// | Best For           | High-speed  | Complex queries  | Simple apps |
package redis
