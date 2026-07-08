package prebuilt

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
)

// TestCreateManusAgent tests the basic functionality of the Manus agent
func TestCreateManusAgent(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Define test nodes
	nodes := []graph.TypedNode[map[string]any]{
		{
			Name:        "test_node",
			Description: "A test node",
			Function: func(ctx context.Context, state map[string]any) (map[string]any, error) {
				messages := state["messages"].([]llmtypes.MessageContent)
				msg := llmtypes.MessageContent{
					Role:  llmtypes.ChatMessageTypeAI,
					Parts: []llmtypes.ContentPart{llmtypes.TextPart("Test executed")},
				}
				return map[string]any{
					"messages": append(messages, msg),
				}, nil
			},
		},
	}

	config := ManusConfig{
		WorkDir:    tempDir,
		PlanPath:   filepath.Join(tempDir, "task_plan.md"),
		NotesPath:  filepath.Join(tempDir, "notes.md"),
		OutputPath: filepath.Join(tempDir, "output.md"),
		AutoSave:   true,
		Verbose:    false,
	}

	t.Run("creates agent without error", func(t *testing.T) {
		// Note: This test doesn't use a real LLM, so it will fail during execution
		// We're just testing that the agent can be created
		_, err := CreateManusAgent(nil, nodes, nil, config)
		if err != nil {
			t.Errorf("CreateManusAgent() error = %v", err)
		}
	})
}

// TestParsePhasesFromPlan tests the phase parsing logic
func TestParsePhasesFromPlan(t *testing.T) {
	tests := []struct {
		name     string
		planText string
		wantLen  int
	}{
		{
			name: "single phase",
			planText: `%% Goal
Test goal

%% Phases
- [ ] Phase 1: Research
  Description: Research phase
  Node: research
`,
			wantLen: 1,
		},
		{
			name: "multiple phases",
			planText: `%% Goal
Test goal

%% Phases
- [ ] Phase 1: Research
  Description: Research phase
  Node: research

- [ ] Phase 2: Write
  Description: Write phase
  Node: write
`,
			wantLen: 2,
		},
		{
			name: "complete phase",
			planText: `%% Goal
Test goal

%% Phases
- [x] Phase 1: Research
  Description: Research phase
  Node: research
`,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phases := parsePhasesFromPlan(tt.planText)
			if len(phases) != tt.wantLen {
				t.Errorf("parsePhasesFromPlan() returned %d phases, want %d", len(phases), tt.wantLen)
			}
		})
	}
}

// TestGeneratePlanMarkdown tests the plan generation logic
func TestGeneratePlanMarkdown(t *testing.T) {
	phases := []Phase{
		{
			Name:        "Research",
			Description: "Research phase",
			NodeName:    "research",
			Complete:    true,
		},
		{
			Name:        "Write",
			Description: "Write phase",
			NodeName:    "write",
			Complete:    false,
		},
	}

	state := map[string]any{
		"goal": "Test goal",
	}

	markdown := generatePlanMarkdown(phases, state)

	// Check that it contains the goal
	if !contains(markdown, "Test goal") {
		t.Error("generatePlanMarkdown() missing goal")
	}

	// Check that it marks the first phase as complete
	if !contains(markdown, "- [x] Phase 1: Research") {
		t.Error("generatePlanMarkdown() first phase should be marked complete")
	}

	// Check that it marks the second phase as incomplete
	if !contains(markdown, "- [ ] Phase 2: Write") {
		t.Error("generatePlanMarkdown() second phase should be marked incomplete")
	}
}

// TestSaveErrorToNotes tests error logging
func TestSaveErrorToNotes(t *testing.T) {
	tempDir := t.TempDir()
	notesPath := filepath.Join(tempDir, "notes.md")

	state := map[string]any{}

	errMsg := "Test error message"

	err := saveErrorToNotes(notesPath, errMsg, state)
	if err != nil {
		t.Errorf("saveErrorToNotes() error = %v", err)
	}

	// Check that file was created
	if _, err := os.Stat(notesPath); os.IsNotExist(err) {
		t.Error("saveErrorToNotes() didn't create notes file")
	}

	// Check content
	content, err := os.ReadFile(notesPath)
	if err != nil {
		t.Errorf("Failed to read notes file: %v", err)
	}

	if !contains(string(content), "Test error message") {
		t.Error("saveErrorToNotes() didn't write error message")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
