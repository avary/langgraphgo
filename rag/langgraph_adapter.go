package rag

import (
	"context"
	"fmt"

	"github.com/smallnest/langgraphgo/tool"
)

// ========================================
// LangGraph Adapters
// ========================================

// NewRetrievalNode creates a LangGraph node function that retrieves documents using the RAG engine.
// It expects the input state to be a map[string]any.
//
// Parameters:
//   - engine: The RAG engine to use for retrieval.
//   - inputKey: The key in the state map where the query string is stored.
//   - outputKey: The key in the returned map where the retrieved context (string) will be stored.
//
// Usage:
//
//	graph.AddNode("retrieve", rag.NewRetrievalNode(myEngine, "question", "context"))
func NewRetrievalNode(engine Engine, inputKey, outputKey string) func(context.Context, any) (any, error) {
	return func(ctx context.Context, state any) (any, error) {
		m, ok := state.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("state is not a map[string]any, got %T", state)
		}

		query, ok := m[inputKey].(string)
		if !ok {
			return nil, fmt.Errorf("input key '%s' not found or not a string", inputKey)
		}

		// Perform the query
		result, err := engine.Query(ctx, query)
		if err != nil {
			return nil, err
		}

		// Return the update
		return map[string]any{
			outputKey: result.Context,
		}, nil
	}
}

// RetrieverTool wraps a RAG Engine as a LangChain Tool, allowing agents to query the knowledge base.
type RetrieverTool struct {
	Engine  Engine
	NameVal string
	DescVal string
}

// NewRetrieverTool creates a new RetrieverTool.
// If name or description are empty, defaults will be used.
func NewRetrieverTool(engine Engine, name, description string) *RetrieverTool {
	if name == "" {
		name = "knowledge_base"
	}
	if description == "" {
		description = "A knowledge base tool. Use this to search for information to answer questions."
	}
	return &RetrieverTool{
		Engine:  engine,
		NameVal: name,
		DescVal: description,
	}
}

var _ tool.Tool = &RetrieverTool{}

// Name returns the name of the tool.
func (t *RetrieverTool) Name() string {
	return t.NameVal
}

// Description returns the description of the tool.
func (t *RetrieverTool) Description() string {
	return t.DescVal
}

// Call executes the retrieval query.
func (t *RetrieverTool) Call(ctx context.Context, input string) (string, error) {
	result, err := t.Engine.Query(ctx, input)
	if err != nil {
		return "", err
	}
	return result.Context, nil
}
