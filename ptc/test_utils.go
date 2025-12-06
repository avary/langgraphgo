package ptc

import (
	"context"
)

// mockTool is a simple mock tool for testing
// Defined as lowercase to make it package-private but accessible to tests
type mockTool struct {
	name        string
	description string
	response    string
}

func (t mockTool) Name() string {
	return t.name
}

func (t mockTool) Description() string {
	return t.description
}

func (t mockTool) Call(ctx context.Context, input string) (string, error) {
	return t.response, nil
}

// newMockTool creates a new mock tool for testing
func newMockTool(name, description, response string) mockTool {
	return mockTool{
		name:        name,
		description: description,
		response:    response,
	}
}
