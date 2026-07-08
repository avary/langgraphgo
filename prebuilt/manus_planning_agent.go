package prebuilt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/tooltypes"
)

// ManusConfig configures a Manus-style planning agent with persistent files
type ManusConfig struct {
	WorkDir    string
	PlanPath   string
	NotesPath  string
	OutputPath string
	AutoSave   bool
	Verbose    bool
}

// CreateManusAgent creates a Manus-style planning agent that:
// 1. Generates and saves plans to task_plan.md
// 2. Stores research findings in notes.md
// 3. Tracks progress with checkboxes
// 4. Supports human-in-the-loop intervention
// 5. Maintains persistent state across sessions
func CreateManusAgent(
	model llmtypes.Model,
	availableNodes []graph.TypedNode[map[string]any],
	inputTools []tooltypes.Tool,
	config ManusConfig,
	opts ...CreateAgentOption,
) (*graph.StateRunnable[map[string]any], error) {
	// Validate config
	if config.WorkDir == "" {
		config.WorkDir = "./work"
	}
	if config.PlanPath == "" {
		config.PlanPath = filepath.Join(config.WorkDir, "task_plan.md")
	}
	if config.NotesPath == "" {
		config.NotesPath = filepath.Join(config.WorkDir, "notes.md")
	}
	if config.OutputPath == "" {
		config.OutputPath = filepath.Join(config.WorkDir, "output.md")
	}

	// Create work directory if not exists
	if err := os.MkdirAll(config.WorkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	// Build node map
	nodeMap := make(map[string]graph.TypedNode[map[string]any])
	for _, node := range availableNodes {
		nodeMap[node.Name] = node
	}

	// Create workflow graph
	workflow := graph.NewStateGraph[map[string]any]()
	agentSchema := graph.NewMapSchema()
	agentSchema.RegisterReducer("messages", graph.AppendReducer)
	agentSchema.RegisterReducer("current_phase", graph.OverwriteReducer)
	agentSchema.RegisterReducer("phases", graph.OverwriteReducer)
	agentSchema.RegisterReducer("errors", graph.AppendReducer)
	workflow.SetSchema(agentSchema)

	// Node 1: Read existing plan (if any)
	workflow.AddNode("read_plan", "Read existing plan and notes", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		messages := state["messages"].([]llmtypes.MessageContent)

		// Read existing plan
		planContent, err := os.ReadFile(config.PlanPath)
		if err == nil {
			// Plan exists, parse it
			phases := parsePhasesFromPlan(string(planContent))
			state["phases"] = phases

			msg := llmtypes.MessageContent{
				Role:  llmtypes.ChatMessageTypeSystem,
				Parts: []llmtypes.ContentPart{llmtypes.TextPart(fmt.Sprintf("Loaded existing plan with %d phases", len(phases)))},
			}
			messages = append(messages, msg)
		} else {
			// No existing plan
			state["phases"] = []Phase{}
		}

		// Read existing notes
		notesContent, err := os.ReadFile(config.NotesPath)
		if err == nil {
			msg := llmtypes.MessageContent{
				Role:  llmtypes.ChatMessageTypeSystem,
				Parts: []llmtypes.ContentPart{llmtypes.TextPart(fmt.Sprintf("Loaded existing notes (%d bytes)", len(notesContent)))},
			}
			messages = append(messages, msg)
			state["notes"] = string(notesContent)
		}

		return map[string]any{
			"messages": messages,
			"phases":   state["phases"],
			"notes":    state["notes"],
		}, nil
	})

	// Node 2: Planner - Create/Update plan in Markdown format
	workflow.AddNode("planner", "Generate or update workflow plan", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		messages := state["messages"].([]llmtypes.MessageContent)

		// Build planning prompt with file context
		nodeDescriptions := buildPlanningNodeDescriptions(availableNodes)
		planningPrompt := buildManusPlanningPrompt(nodeDescriptions, config.PlanPath, config.NotesPath)

		planningMessages := []llmtypes.MessageContent{
			{Role: llmtypes.ChatMessageTypeSystem, Parts: []llmtypes.ContentPart{llmtypes.TextPart(planningPrompt)}},
		}
		planningMessages = append(planningMessages, messages...)

		resp, err := model.GenerateContent(ctx, planningMessages)
		if err != nil {
			return nil, fmt.Errorf("planning failed: %w", err)
		}

		planText := resp.Choices[0].Content

		// Save plan to file
		if config.AutoSave {
			if err := saveManusPlan(config.PlanPath, planText, state); err != nil {
				return nil, fmt.Errorf("failed to save plan: %w", err)
			}
		}

		// Parse phases from plan
		phases := parsePhasesFromPlan(planText)

		aiMsg := llmtypes.MessageContent{
			Role:  llmtypes.ChatMessageTypeAI,
			Parts: []llmtypes.ContentPart{llmtypes.TextPart(fmt.Sprintf("Plan updated with %d phases\n\n%s", len(phases), planText))},
		}

		return map[string]any{
			"messages":      append(messages, aiMsg),
			"phases":        phases,
			"current_phase": 0,
		}, nil
	})

	// Node 3: Executor - Execute current phase
	workflow.AddNode("executor", "Execute current phase of the plan", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		messages := state["messages"].([]llmtypes.MessageContent)
		phases := state["phases"].([]Phase)
		phaseIndex := state["current_phase"].(int)

		if phaseIndex >= len(phases) {
			// All phases complete
			return map[string]any{
				"messages":      messages,
				"phases":        phases,
				"current_phase": phaseIndex,
				"status":        "complete",
			}, nil
		}

		phase := phases[phaseIndex]

		// Find and execute the node for this phase
		node, exists := nodeMap[phase.NodeName]
		if !exists {
			return nil, fmt.Errorf("node %s not found", phase.NodeName)
		}

		// Execute the node
		result, err := node.Function(ctx, state)
		if err != nil {
			// Log error and save to notes
			errMsg := fmt.Sprintf("Error in phase %d (%s): %v", phaseIndex, phase.Name, err)

			if config.AutoSave {
				_ = saveErrorToNotes(config.NotesPath, errMsg, state)
			}

			// Add error to state
			errors := state["errors"].([]string)
			errors = append(errors, errMsg)

			return map[string]any{
				"messages":      messages,
				"phases":        phases,
				"current_phase": phaseIndex,
				"errors":        errors,
				"status":        "error",
			}, nil
		}

		// Mark phase as complete
		phases[phaseIndex].Complete = true
		phases[phaseIndex].CompletedAt = time.Now()

		// Update plan file
		if config.AutoSave {
			planText := generatePlanMarkdown(phases, state)
			_ = saveManusPlan(config.PlanPath, planText, state)
		}

		return map[string]any{
			"messages":      result["messages"].([]llmtypes.MessageContent),
			"phases":        phases,
			"current_phase": phaseIndex + 1,
			"status":        "in_progress",
		}, nil
	})

	// Node 4: Check if done
	workflow.AddNode("check_done", "Check if all phases are complete", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		messages := state["messages"].([]llmtypes.MessageContent)
		phases := state["phases"].([]Phase)
		currentPhase := state["current_phase"].(int)

		allComplete := true
		for _, phase := range phases {
			if !phase.Complete {
				allComplete = false
				break
			}
		}

		if allComplete {
			// Generate final output
			result, err := generateFinalOutput(state, config)
			if err != nil {
				return nil, err
			}
			// Ensure continue is set to false when done
			result["continue"] = false
			return result, nil
		}

		// Continue to next phase - pass through all state
		return map[string]any{
			"messages":      messages,
			"phases":        phases,
			"current_phase": currentPhase,
			"continue":      true,
			"status":        "in_progress",
		}, nil
	})

	// Set up edges
	workflow.SetEntryPoint("read_plan")
	workflow.AddEdge("read_plan", "planner")
	workflow.AddEdge("planner", "executor")
	workflow.AddEdge("executor", "check_done")

	// Conditional edge: continue or finish
	workflow.AddConditionalEdge("check_done", func(ctx context.Context, state map[string]any) string {
		if shouldContinue, ok := state["continue"].(bool); ok && shouldContinue {
			return "executor"
		}
		return "END"
	})

	// Enable human-in-the-loop
	// workflow.InterruptBefore([]string{"planner"})

	return workflow.Compile()
}

// Phase represents a single phase in the Manus plan
type Phase struct {
	Name        string
	Description string
	NodeName    string
	Complete    bool
	CompletedAt time.Time
}

func buildManusPlanningPrompt(nodeDescriptions, planPath, notesPath string) string {
	return fmt.Sprintf(`You are a Manus-style planning assistant. Generate a detailed workflow plan in Markdown format.

%s

## File Structure

Your plan will be saved to: %s
Your research notes will be saved to: %s

## Plan Format

Generate a plan in this Markdown format:

%% Goal
[Describe the overall goal in 1-2 sentences]

%% Phases
- [ ] Phase 1: [Phase Name]
  Description: [What this phase accomplishes]
  Node: [node_name_from_available_nodes]

- [ ] Phase 2: [Phase Name]
  Description: [What this phase accomplishes]
  Node: [node_name_from_available_nodes]

...

## Key Principles

1. **Break down complex tasks** into 3-7 clear phases
2. **Use available nodes** - only reference nodes from the list above
3. **Be specific** - each phase should have a clear deliverable
4. **Think incrementally** - each phase builds on the previous
5. **Plan for errors** - include validation/check phases

## Example

%% Goal
Research and write a summary of TypeScript benefits

%% Phases
- [ ] Phase 1: Research TypeScript Benefits
  Description: Search for and analyze TypeScript documentation
  Node: research

- [ ] Phase 2: Compile Findings
  Description: Organize research findings into notes.md
  Node: compile

- [ ] Phase 3: Write Summary
  Description: Generate final markdown summary
  Node: write

Generate ONLY the plan in the format above, no additional text.
`, nodeDescriptions, planPath, notesPath)
}

func parsePhasesFromPlan(planText string) []Phase {
	phases := []Phase{}
	lines := strings.Split(planText, "\n")
	currentPhase := &Phase{}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse phase
		if strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "- [x]") {
			// Complete previous phase
			if currentPhase.Name != "" {
				phases = append(phases, *currentPhase)
			}

			// Start new phase
			currentPhase = &Phase{}
			currentPhase.Complete = strings.HasPrefix(line, "- [x]")

			// Extract phase name
			phaseLine := strings.TrimPrefix(line, "- [ ]")
			phaseLine = strings.TrimPrefix(phaseLine, "- [x]")
			phaseLine = strings.TrimSpace(phaseLine)

			if strings.HasPrefix(phaseLine, "Phase ") {
				// Extract phase name (e.g., "Phase 1: Research")
				parts := strings.SplitN(phaseLine, ":", 2)
				if len(parts) == 2 {
					currentPhase.Name = strings.TrimSpace(parts[1])
				}
			}
		} else if after, ok := strings.CutPrefix(line, "Description:"); ok {
			currentPhase.Description = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "Node:"); ok {
			currentPhase.NodeName = strings.TrimSpace(after)
		}
	}

	// Add last phase
	if currentPhase.Name != "" {
		phases = append(phases, *currentPhase)
	}

	return phases
}

func generatePlanMarkdown(phases []Phase, state map[string]any) string {
	var sb strings.Builder

	sb.WriteString("%% Goal\n\n")
	if goal, ok := state["goal"].(string); ok {
		sb.WriteString(goal)
		sb.WriteString("\n\n")
	}

	sb.WriteString("%% Phases\n\n")
	for i, phase := range phases {
		if phase.Complete {
			sb.WriteString(fmt.Sprintf("- [x] Phase %d: %s\n", i+1, phase.Name))
		} else {
			sb.WriteString(fmt.Sprintf("- [ ] Phase %d: %s\n", i+1, phase.Name))
		}
		sb.WriteString(fmt.Sprintf("  Description: %s\n", phase.Description))
		if phase.NodeName != "" {
			sb.WriteString(fmt.Sprintf("  Node: %s\n", phase.NodeName))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func saveManusPlan(path, content string, state map[string]any) error {
	return os.WriteFile(path, []byte(content), 0600)
}

func saveErrorToNotes(path, errMsg string, state map[string]any) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "\n## Error [%s]\n%s\n", time.Now().Format("2006-01-02 15:04:05"), errMsg)
	return err
}

func generateFinalOutput(state map[string]any, config ManusConfig) (map[string]any, error) {
	messages := state["messages"].([]llmtypes.MessageContent)

	// Collect only the last few non-planner messages to avoid duplicates
	// Keep the most recent outputs from user nodes
	var output strings.Builder
	output.WriteString("# Final Output\n\n")
	output.WriteString(fmt.Sprintf("Generated at: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Collect messages in reverse, skip duplicates and planner messages
	seen := make(map[string]bool)
	count := 0
	maxMessages := 20 // Only keep last 20 unique messages

	for i := len(messages) - 1; i >= 0 && count < maxMessages; i-- {
		msg := messages[i]
		if msg.Role != llmtypes.ChatMessageTypeAI {
			continue
		}

		for _, part := range msg.Parts {
			if text, ok := part.(llmtypes.TextContent); ok {
				textStr := strings.TrimSpace(text.Text)

				// Skip empty messages
				if textStr == "" {
					continue
				}

				// Skip planner messages
				if strings.Contains(textStr, "% Goal") ||
					strings.Contains(textStr, "% Phases") ||
					strings.Contains(textStr, "Plan updated with") {
					continue
				}

				// Skip duplicates
				if seen[textStr] {
					continue
				}
				seen[textStr] = true
				count++
			}
		}
	}

	// Now collect in forward order (reversing our reverse collection)
	var collected []string
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != llmtypes.ChatMessageTypeAI {
			continue
		}

		for _, part := range msg.Parts {
			if text, ok := part.(llmtypes.TextContent); ok {
				textStr := strings.TrimSpace(text.Text)

				// Skip empty messages
				if textStr == "" {
					continue
				}

				// Skip planner messages
				if strings.Contains(textStr, "% Goal") ||
					strings.Contains(textStr, "% Phases") ||
					strings.Contains(textStr, "Plan updated with") {
					continue
				}

				// Only add if we saw this in our first pass (i.e., it's unique and recent)
				if seen[textStr] {
					collected = append(collected, textStr)
					delete(seen, textStr) // Mark as added
				}
			}
		}
	}

	// Write collected messages
	for _, text := range collected {
		output.WriteString(text)
		output.WriteString("\n\n")
	}

	// Save to output file
	if err := os.WriteFile(config.OutputPath, []byte(output.String()), 0600); err != nil {
		return nil, fmt.Errorf("failed to save output: %w", err)
	}

	return map[string]any{
		"messages": messages,
		"status":   "complete",
		"output":   output.String(),
	}, nil
}
