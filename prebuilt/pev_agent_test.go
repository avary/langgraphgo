package prebuilt

import (
	"testing"

	"github.com/smallnest/langgraphgo/tooltypes"
)

func TestCreatePEVAgentMap(t *testing.T) {
	mockLLM := &PEVMockLLM{
		responses: []string{
			"1. Step",
			`{"is_successful": true, "reasoning": "Ok"}`,
			"Final",
		},
	}
	config := PEVAgentConfig{
		Model:      mockLLM,
		Tools:      []tooltypes.Tool{PEVMockTool{name: "calculator"}},
		MaxRetries: 3,
	}
	agent, err := CreatePEVAgentMap(config)
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
	if agent == nil {
		t.Fatal("Agent is nil")
	}
}
