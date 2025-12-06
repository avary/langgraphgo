package memory

import (
	"context"
	"fmt"
	"sync"
)

// GraphNode represents a node in the conversation graph
type GraphNode struct {
	Message     *Message
	Connections []string // IDs of connected messages
	Weight      float64  // Importance/relevance weight
}

// GraphBasedMemory models conversations as knowledge graphs
// Pros: Captures relationships between topics, better context understanding
// Cons: More complex, requires relationship tracking
type GraphBasedMemory struct {
	nodes     map[string]*GraphNode // Message ID -> Node
	topK      int                   // Number of messages to retrieve
	mu        sync.RWMutex
	relations map[string][]string // Topic/entity -> related message IDs

	// RelationExtractor identifies relationships between messages
	// In production, this could use NER or topic modeling
	RelationExtractor func(msg *Message) []string
}

// GraphConfig holds configuration for graph-based memory
type GraphConfig struct {
	TopK              int                          // Number of messages to retrieve
	RelationExtractor func(msg *Message) []string // Custom relation extractor
}

// NewGraphBasedMemory creates a new graph-based memory strategy
func NewGraphBasedMemory(config *GraphConfig) *GraphBasedMemory {
	if config == nil {
		config = &GraphConfig{
			TopK: 10,
		}
	}

	if config.TopK <= 0 {
		config.TopK = 10
	}

	extractor := config.RelationExtractor
	if extractor == nil {
		extractor = defaultRelationExtractor
	}

	return &GraphBasedMemory{
		nodes:             make(map[string]*GraphNode),
		topK:              config.TopK,
		relations:         make(map[string][]string),
		RelationExtractor: extractor,
	}
}

// AddMessage adds a message to the graph and establishes connections
func (g *GraphBasedMemory) AddMessage(ctx context.Context, msg *Message) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Create node
	node := &GraphNode{
		Message:     msg,
		Connections: make([]string, 0),
		Weight:      1.0,
	}

	// Extract entities/topics from message
	topics := g.RelationExtractor(msg)

	// Store node
	g.nodes[msg.ID] = node

	// Build connections based on shared topics
	for _, topic := range topics {
		// Link to existing messages with same topic
		if relatedIDs, exists := g.relations[topic]; exists {
			for _, relatedID := range relatedIDs {
				// Create bidirectional connection
				node.Connections = append(node.Connections, relatedID)
				if relatedNode, ok := g.nodes[relatedID]; ok {
					relatedNode.Connections = append(relatedNode.Connections, msg.ID)
				}
			}
		}

		// Add this message to topic index
		g.relations[topic] = append(g.relations[topic], msg.ID)
	}

	return nil
}

// GetContext retrieves messages based on graph traversal
// Uses breadth-first search starting from most recent messages
func (g *GraphBasedMemory) GetContext(ctx context.Context, query string) ([]*Message, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.nodes) == 0 {
		return []*Message{}, nil
	}

	// Extract topics from query
	queryTopics := g.RelationExtractor(&Message{Content: query})

	// Find seed messages (related to query topics)
	seedIDs := make(map[string]bool)
	for _, topic := range queryTopics {
		if relatedIDs, exists := g.relations[topic]; exists {
			for _, id := range relatedIDs {
				seedIDs[id] = true
			}
		}
	}

	// If no seed messages, use most recent messages
	if len(seedIDs) == 0 {
		// Get most recent messages as seeds
		count := 0
		for id := range g.nodes {
			seedIDs[id] = true
			count++
			if count >= 3 {
				break
			}
		}
	}

	// BFS traversal to collect connected messages
	visited := make(map[string]bool)
	queue := make([]string, 0)
	result := make([]*Message, 0)

	// Add seeds to queue
	for id := range seedIDs {
		queue = append(queue, id)
	}

	// Traverse graph
	for len(queue) > 0 && len(result) < g.topK {
		currentID := queue[0]
		queue = queue[1:]

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		// Add message to result
		if node, ok := g.nodes[currentID]; ok {
			result = append(result, node.Message)

			// Add connected nodes to queue
			for _, connID := range node.Connections {
				if !visited[connID] {
					queue = append(queue, connID)
				}
			}
		}

		if len(result) >= g.topK {
			break
		}
	}

	return result, nil
}

// Clear removes all nodes and relationships
func (g *GraphBasedMemory) Clear(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodes = make(map[string]*GraphNode)
	g.relations = make(map[string][]string)
	return nil
}

// GetStats returns statistics about the graph
func (g *GraphBasedMemory) GetStats(ctx context.Context) (*Stats, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	totalTokens := 0

	for _, node := range g.nodes {
		totalTokens += node.Message.TokenCount
	}

	activeTokens := 0
	compressionRate := 0.0

	if len(g.nodes) > 0 {
		activeTokens = totalTokens / len(g.nodes) * g.topK
		compressionRate = float64(g.topK) / float64(len(g.nodes))
	}

	return &Stats{
		TotalMessages:   len(g.nodes),
		TotalTokens:     totalTokens,
		ActiveMessages:  g.topK,
		ActiveTokens:    activeTokens,
		CompressionRate: compressionRate,
	}, nil
}

// GetRelationships returns all topics and their associated message counts
func (g *GraphBasedMemory) GetRelationships() map[string]int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[string]int)
	for topic, ids := range g.relations {
		result[topic] = len(ids)
	}
	return result
}

// defaultRelationExtractor extracts simple keywords as topics
// In production, use NER, topic modeling, or entity extraction
func defaultRelationExtractor(msg *Message) []string {
	// Simple keyword extraction (for demonstration)
	content := msg.Content

	// Common topics/keywords (very basic implementation)
	keywords := []string{"price", "feature", "bug", "question", "help", "error"}
	topics := make([]string, 0)

	for _, keyword := range keywords {
		if contains(content, keyword) {
			topics = append(topics, keyword)
		}
	}

	// If no keywords found, use role as topic
	if len(topics) == 0 {
		topics = append(topics, fmt.Sprintf("role:%s", msg.Role))
	}

	return topics
}

// contains checks if text contains substring (case-insensitive)
func contains(text, substr string) bool {
	// Simple case-insensitive check
	textLower := text
	substrLower := substr
	for i := 0; i < len(textLower); i++ {
		if textLower[i] >= 'A' && textLower[i] <= 'Z' {
			textLower = textLower[:i] + string(textLower[i]+32) + textLower[i+1:]
		}
	}
	for i := 0; i < len(substrLower); i++ {
		if substrLower[i] >= 'A' && substrLower[i] <= 'Z' {
			substrLower = substrLower[:i] + string(substrLower[i]+32) + substrLower[i+1:]
		}
	}

	// Check if substring exists
	for i := 0; i <= len(textLower)-len(substrLower); i++ {
		if textLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}
