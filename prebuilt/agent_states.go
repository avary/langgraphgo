package prebuilt

import (
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/tmc/langchaingo/tools"
)

// AgentState represents the general agent state.
// This is the default state type for generic agents.
type AgentState struct {
	// Messages contains the conversation history
	Messages []llmtypes.MessageContent

	// ExtraTools contains additional tools available to the agent
	ExtraTools []tools.Tool
}

// ReactAgentState represents the default state for a ReAct agent
type ReactAgentState struct {
	// Messages contains the conversation history
	Messages []llmtypes.MessageContent `json:"messages"`
	// IterationCount counts the current iteration number
	IterationCount int `json:"iteration_count"`
}

// PlanningAgentState represents the state for a planning agent.
// The planning agent first generates a workflow plan using LLM,
// then executes according to the generated plan.
type PlanningAgentState struct {
	// Messages contains the conversation history
	Messages []llmtypes.MessageContent

	// WorkflowPlan contains the parsed workflow plan from LLM
	WorkflowPlan *WorkflowPlan
}

// WorkflowPlan represents the parsed workflow plan from LLM
type WorkflowPlan struct {
	Nodes []WorkflowNode `json:"nodes"`
	Edges []WorkflowEdge `json:"edges"`
}

// WorkflowNode represents a node in the workflow plan
type WorkflowNode struct {
	Name string `json:"name"`
	Type string `json:"type"` // "start", "process", "end", "conditional"
}

// WorkflowEdge represents an edge in the workflow plan
type WorkflowEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"` // For conditional edges
}

// ReflectionAgentState represents the state for a reflection agent.
// The reflection agent iteratively improves its response through
// self-reflection and revision.
type ReflectionAgentState struct {
	// Messages contains the conversation history
	Messages []llmtypes.MessageContent

	// Iteration counts the current iteration number
	Iteration int

	// Reflection contains the agent's self-reflection on its draft
	Reflection string

	// Draft contains the current draft response being refined
	Draft string
}

// PEVAgentState represents the state for a Plan-Execute-Verify agent.
// This agent follows a three-step process: plan, execute, and verify.
type PEVAgentState struct {
	// Messages contains the conversation history
	Messages []llmtypes.MessageContent

	// Plan contains the list of steps to execute
	Plan []string

	// CurrentStep is the index of the current step being executed
	CurrentStep int

	// LastToolResult contains the result of the last tool execution
	LastToolResult string

	// IntermediateSteps contains results from intermediate steps
	IntermediateSteps []string

	// Retries counts the number of retries attempted
	Retries int

	// VerificationResult contains the verification status
	VerificationResult string

	// FinalAnswer contains the final answer after verification
	FinalAnswer string
}

// TreeOfThoughtsState represents the state for a tree-of-thoughts agent.
// This agent explores multiple reasoning paths in parallel to find
// the best solution.
type TreeOfThoughtsState struct {
	// ActivePaths contains all active reasoning paths being explored
	ActivePaths map[string]*SearchPath

	// Solution contains the best solution found so far
	Solution string

	// VisitedStates tracks which states have been visited to avoid cycles
	VisitedStates map[string]bool

	// Iteration counts the current iteration number
	Iteration int
}

// ChatAgentState represents the state for a chat agent.
// This is a conversational agent that maintains message history.
type ChatAgentState struct {
	// Messages contains the conversation history
	Messages []llmtypes.MessageContent

	// SystemPrompt is the system prompt for the chat agent
	SystemPrompt string

	// ExtraTools contains additional tools available to the agent
	ExtraTools []tools.Tool
}

// SupervisorState represents the state for a supervisor workflow
type SupervisorState struct {
	// Messages contains the conversation history
	Messages []llmtypes.MessageContent `json:"messages"`
	// Next is the next worker to act
	Next string `json:"next,omitempty"`
}
