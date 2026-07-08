// Package memory provides various memory management strategies for conversational AI applications.
//
// This package implements multiple approaches to managing conversation history and context,
// from simple buffers to sophisticated OS-inspired memory management with paging and eviction.
// It's designed to help maintain relevant context within token limits while preserving
// important information from long conversations.
//
// # Core Interface
//
// The Memory interface defines the contract that all memory strategies must implement:
//
//   - AddMessage: Add a new message to memory
//   - GetContext: Retrieve relevant context for the current query
//   - Clear: Remove all messages from memory
//   - GetStats: Get statistics about memory usage
//
// # Available Memory Strategies
//
// ## Buffer Memory
// Simple first-in-first-out buffer with configurable size:
//
//	buffer := memory.NewBufferMemory(100) // Keep last 100 messages
//	buffer.AddMessage(ctx, message)
//	context, _ := buffer.GetContext(ctx, "current query")
//
// ## Sliding Window Memory
// Maintains a sliding window of recent messages with overlap:
//
//	window := memory.NewSlidingWindowMemory(50, 5) // 50 messages with 5 overlap
//
// ## Summarization Memory
// Automatically summarizes older messages to save tokens:
//
//	summ := memory.NewSummarizationMemory(llmClient, 1000) // 1000 token limit
//
// ## Hierarchical Memory
// Multi-level memory with different retention policies:
//
//	hierarchical := memory.NewHierarchicalMemory(
//		&memory.Config{
//			WorkingMemorySize: 50,
//			LongTermSize:      1000,
//			ArchiveSize:       10000,
//		},
//	)
//
// ## OS-Inspired Memory
// Sophisticated memory management with active, cached, and archived pages:
//
//	osMemory := memory.NewOSLikeMemory(&memory.OSLikeConfig{
//		ActiveLimit:  100,
//		CacheLimit:   500,
//		AccessWindow: time.Hour,
//	})
//
// ## Graph-Based Memory
// Organizes messages as a graph for better context retrieval:
//
//	graphMemory := memory.NewGraphBasedMemory(
//		embeddingModel,
//		&memory.GraphConfig{
//			MaxNodes:      1000,
//			SimilarityThreshold: 0.7,
//		},
//	)
//
// # Message Structure
//
// Each message contains:
//
//	type Message struct {
//		ID         string         // Unique identifier
//		Role       string         // "user", "assistant", "system"
//		Content    string         // Message content
//		Timestamp  time.Time      // When created
//		Metadata   map[string]any // Additional metadata
//		TokenCount int            // Approximate token count
//	}
//
// # Example Usage
//
// ## Basic Buffer Memory
//
//	import (
//		"context"
//		"time"
//
//		"github.com/smallnest/langgraphgo/memory"
//	)
//
//	ctx := context.Background()
//	mem := memory.NewBufferMemory(50)
//
//	// Add messages
//	mem.AddMessage(ctx, memory.NewMessage("user", "Hello!"))
//	mem.AddMessage(ctx, memory.NewMessage("assistant", "Hi there!"))
//
//	// Get context for next query
//	context, err := mem.GetContext(ctx, "How are you?")
//	if err != nil {
//		return err
//	}
//
//	// Use context in LLM prompt
//	for _, msg := range context {
//		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
//	}
//
// ## Summarization Memory
//
//	// Requires an LLM client that implements the Summarizer interface
//	type MyLLM struct{}
//	func (m *MyLLM) Summarize(ctx context.Context, messages []*memory.Message) (string, error) {
//		// Implementation for summarizing messages
//		return "", nil
//	}
//
//	llm := &MyLLM{}
//	mem := memory.NewSummarizationMemory(llm, 2000) // 2000 token limit
//
//	// Add many messages - older ones will be summarized
//	for i := 0; i < 100; i++ {
//		mem.AddMessage(ctx, &memory.Message{
//			Role:    "user",
//			Content: fmt.Sprintf("Message %d", i),
//		})
//	}
//
//	// Context will include recent messages + summaries of older ones
//	context, _ := mem.GetContext(ctx, "latest query")
//
// ## Hierarchical Memory
//
//	config := &memory.HierarchicalConfig{
//		WorkingMemorySize: 20,  // Recent messages
//		LongTermSize:      200, // Important messages
//		ArchiveSize:       2000, // All other messages
//		ImportanceThreshold: 0.5,
//	}
//
//	mem := memory.NewHierarchicalMemory(config)
//
//	// Messages with metadata can be marked as important
//	mem.AddMessage(ctx, &memory.Message{
//		Role:    "user",
//		Content: "Critical information",
//		Metadata: map[string]any{"importance": 0.9},
//	})
//
// # Memory Statistics
//
// All implementations provide statistics:
//
//	stats, _ := mem.GetStats(ctx)
//	fmt.Printf("Total messages: %d\n", stats.TotalMessages)
//	fmt.Printf("Total tokens: %d\n", stats.TotalTokens)
//	fmt.Printf("Active tokens: %d\n", stats.ActiveTokens)
//	fmt.Printf("Compression rate: %.2f\n", stats.CompressionRate)
//
// # Integration with LangChain
//
// LangChain-compatible memory adapters live in the memory/langchainadapter
// subpackage, which is the only place that imports github.com/tmc/langchaingo:
//
//	import "github.com/smallnest/langgraphgo/memory/langchainadapter"
//
//	mem := langchainadapter.NewConversationBufferMemory()
//
// # Compression Strategies
//
// For long conversations, the package provides compression:
//
//	compressor := memory.NewSemanticCompressor(embeddings, 0.3)
//	compressed := compressor.Compress(messages)
//
// # Retrieval-Augmented Memory
//
// Combine with vector storage for semantic retrieval:
//
//	retriever := memory.NewRetrievalMemory(
//		vectorStore,
//		embeddingModel,
//		&memory.RetrievalConfig{
//			TopK:              5,
//			MinSimilarity:     0.7,
//			ContextWindow:    4000,
//		},
//	)
//
// # Choosing a Strategy
//
//   - Buffer: Simple conversations, fixed context size
//   - Sliding Window: Need some context continuity
//   - Summarization: Long conversations, need to preserve all information
//   - Hierarchical: Complex applications with different retention needs
//   - OS-Inspired: Performance-critical applications with access patterns
//   - Graph-Based: Semantic relationships between messages matter
//   - Retrieval: Need to find relevant messages based on content similarity
//
// # Thread Safety
//
// All memory implementations are thread-safe and can be used concurrently from multiple
// goroutines. They use internal mutexes or atomic operations for synchronization.
//
// # Custom Memory Strategies
//
// Implement the Memory interface for custom strategies:
//
//	type CustomMemory struct {
//		// Custom fields
//	}
//
//	func (m *CustomMemory) AddMessage(ctx context.Context, msg *memory.Message) error {
//		// Custom implementation
//		return nil
//	}
//
//	func (m *CustomMemory) GetContext(ctx context.Context, query string) ([]*memory.Message, error) {
//		// Custom retrieval logic
//		return nil, nil
//	}
//
//	func (m *CustomMemory) Clear(ctx context.Context) error {
//		// Clear memory
//		return nil
//	}
//
//	func (m *CustomMemory) GetStats(ctx context.Context) (*memory.Stats, error) {
//		// Return statistics
//		return nil, nil
//	}
//
// # Best Practices
//
//  1. Choose appropriate strategy based on your use case
//  2. Monitor memory usage with GetStats()
//  3. Set reasonable limits to prevent memory bloat
//  4. Use metadata to mark important messages
//  5. Consider token costs when using LLM-based summarization
//  6. Test with realistic conversation lengths
//  7. Clear memory between unrelated conversations
package memory
