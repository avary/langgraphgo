// Mental Loop Trading Agent
//
// This example implements the "Mental Loop" (Simulator-in-the-Loop) architecture
// from the Agentic Architectures series by Fareed Khan.
//
// Architecture Overview:
//
// 1. OBSERVE: The agent observes the current state of the environment
// 2. PROPOSE: Based on goals and current state, propose a high-level action/strategy
// 3. SIMULATE: Fork the environment state and run the proposed action forward
//    to observe potential outcomes in a sandboxed simulation
// 4. ASSESS & REFINE: Analyze simulation results to evaluate risks and rewards,
//    refining the initial proposal into a final, concrete action
// 5. EXECUTE: Execute the final, refined action in the real environment
// 6. REPEAT: Begin again from the new state
//
// This "think before you act" approach allows agents to:
// - Perform what-if analysis
// - Anticipate consequences
// - Refine plans for safety and effectiveness
//
// Use cases: Robotics (simulating movements), High-stakes decisions (finance,
// healthcare), Complex game AI, and any domain where mistakes have real consequences.
//
// Reference: https://github.com/FareedKhan-dev/all-agentic-architectures/blob/main/10_mental_loop.ipynb

package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
)

// ==================== Data Models ====================

// Portfolio represents the agent's trading portfolio
type Portfolio struct {
	Cash   float64
	Shares int
}

// Value returns the total portfolio value at the current price
func (p *Portfolio) Value(currentPrice float64) float64 {
	return p.Cash + float64(p.Shares)*currentPrice
}

// MarketSimulator simulates a stock market environment
// This serves as both the "real world" and the sandbox for simulations
type MarketSimulator struct {
	Day        int
	Price      float64
	Volatility float64 // Standard deviation for price changes
	Drift      float64 // General trend (daily return)
	MarketNews string
	Portfolio  Portfolio
}

// ProposedAction represents the high-level strategy proposed by the analyst
type ProposedAction struct {
	Strategy  string // e.g., "buy aggressively", "sell cautiously", "hold"
	Reasoning string
}

// FinalDecision represents the final, concrete action to be executed
type FinalDecision struct {
	Action    string // "buy", "sell", or "hold"
	Amount    float64
	Reasoning string
}

// SimulationResult stores the outcome of one simulation run
type SimulationResult struct {
	SimNum       int
	InitialValue float64
	FinalValue   float64
	ReturnPct    float64
}

// ==================== Agent State ====================

// AgentState represents the state passed between nodes in the graph
type AgentState struct {
	RealMarket        *MarketSimulator
	ProposedAction    *ProposedAction
	SimulationResults []SimulationResult
	FinalDecision     *FinalDecision
}

// ==================== Market Simulator Methods ====================

// Step advances the simulation by one day, executing a trade first
func (m *MarketSimulator) Step(action string, amount float64) {
	// 1. Execute trade
	switch action {
	case "buy": // amount is number of shares
		sharesToBuy := int(amount)
		cost := float64(sharesToBuy) * m.Price
		if m.Portfolio.Cash >= cost {
			m.Portfolio.Shares += sharesToBuy
			m.Portfolio.Cash -= cost
		}
	case "sell": // amount is number of shares
		sharesToSell := int(amount)
		if m.Portfolio.Shares >= sharesToSell {
			m.Portfolio.Shares -= sharesToSell
			m.Portfolio.Cash += float64(sharesToSell) * m.Price
		}
	}

	// 2. Update market price using Geometric Brownian Motion
	// daily_return = normal(drift, volatility)
	dailyReturn := rand.NormFloat64()*m.Volatility + m.Drift
	m.Price *= (1 + dailyReturn)

	// 3. Advance time
	m.Day++

	// 4. Potentially update news
	if rand.Float64() < 0.1 { // 10% chance of new news
		newsOptions := []string{
			"Positive earnings report expected.",
			"New competitor enters the market.",
			"Macroeconomic outlook is strong.",
			"Regulatory concerns are growing.",
		}
		m.MarketNews = newsOptions[rand.Intn(len(newsOptions))]

		// News affects drift
		if strings.Contains(m.MarketNews, "Positive") || strings.Contains(m.MarketNews, "strong") {
			m.Drift = 0.05
		} else {
			m.Drift = -0.05
		}
	} else {
		m.Drift = 0.01 // Revert to normal drift
	}
}

// GetStateString returns a human-readable description of the market state
func (m *MarketSimulator) GetStateString() string {
	return fmt.Sprintf("Day %d: Price=$%.2f, News: %s\nPortfolio: $%.2f (%d shares, $%.2f cash)",
		m.Day, m.Price, m.MarketNews,
		m.Portfolio.Value(m.Price), m.Portfolio.Shares, m.Portfolio.Cash)
}

// Copy creates a deep copy for simulation (sandboxing)
func (m *MarketSimulator) Copy() *MarketSimulator {
	return &MarketSimulator{
		Day:        m.Day,
		Price:      m.Price,
		Volatility: m.Volatility,
		Drift:      m.Drift,
		MarketNews: m.MarketNews,
		Portfolio: Portfolio{
			Cash:   m.Portfolio.Cash,
			Shares: m.Portfolio.Shares,
		},
	}
}

// ==================== Graph Nodes ====================

// ProposeActionNode observes the market and proposes a high-level strategy
// This is the "OBSERVE -> PROPOSE" step of the mental loop
func ProposeActionNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	agentState := state["agent_state"].(*AgentState)

	fmt.Println("\n--- 🧐 Analyst Proposing Action ---")

	// Create prompt for the analyst
	prompt := fmt.Sprintf(`You are a sharp financial analyst. Based on the current market state, propose a trading strategy.

Market State:
%s

Respond in the following format (keep reasoning concise):
STRATEGY: [buy aggressively|buy cautiously|sell aggressively|sell cautiously|hold]
REASONING: [brief reasoning for the proposed strategy]`,
		agentState.RealMarket.GetStateString())

	// Call LLM
	llm := state["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("analyst LLM call failed: %w", err)
	}

	// Parse the response
	proposal := parseProposedAction(resp)
	agentState.ProposedAction = proposal

	fmt.Printf("Proposal: %s. Reason: %s\n",
		proposal.Strategy, proposal.Reasoning)

	return state, nil
}

// RunSimulationNode runs the proposed strategy in a sandboxed simulation
// This is the "SIMULATE" step of the mental loop
func RunSimulationNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	agentState := state["agent_state"].(*AgentState)

	fmt.Println("\n--- 🤖 Running Simulations ---")

	strategy := agentState.ProposedAction.Strategy
	numSimulations := 5
	simulationHorizon := 10 // days
	results := make([]SimulationResult, numSimulations)

	for i := 0; i < numSimulations; i++ {
		// IMPORTANT: Create a deep copy to not affect the real market state
		simulatedMarket := agentState.RealMarket.Copy()
		initialValue := simulatedMarket.Portfolio.Value(simulatedMarket.Price)

		// Translate strategy to a concrete action for the simulation
		var action string
		var amount float64

		if strings.Contains(strategy, "buy") {
			action = "buy"
			// Aggressively = 25% of cash, Cautiously = 10%
			cashRatio := 0.25
			if strings.Contains(strategy, "cautiously") {
				cashRatio = 0.1
			}
			amount = math.Floor((simulatedMarket.Portfolio.Cash * cashRatio) / simulatedMarket.Price)
		} else if strings.Contains(strategy, "sell") {
			action = "sell"
			// Aggressively = 25% of shares, Cautiously = 10%
			sharesRatio := 0.25
			if strings.Contains(strategy, "cautiously") {
				sharesRatio = 0.1
			}
			amount = math.Floor(float64(simulatedMarket.Portfolio.Shares) * sharesRatio)
		} else {
			action = "hold"
			amount = 0
		}

		// Run the simulation forward
		simulatedMarket.Step(action, amount)
		for j := 0; j < simulationHorizon-1; j++ {
			simulatedMarket.Step("hold", 0) // Just hold after the initial action
		}

		finalValue := simulatedMarket.Portfolio.Value(simulatedMarket.Price)
		returnPct := (finalValue - initialValue) / initialValue * 100

		results[i] = SimulationResult{
			SimNum:       i + 1,
			InitialValue: initialValue,
			FinalValue:   finalValue,
			ReturnPct:    returnPct,
		}
	}

	agentState.SimulationResults = results
	fmt.Println("Simulation complete. Results will be passed to the risk manager.")

	return state, nil
}

// RefineAndDecideNode analyzes simulation results and makes a final decision
// This is the "ASSESS & REFINE" step of the mental loop
func RefineAndDecideNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	agentState := state["agent_state"].(*AgentState)

	fmt.Println("\n--- 🧠 Risk Manager Refining Decision ---")

	// Format simulation results
	var resultsSummary strings.Builder
	for _, r := range agentState.SimulationResults {
		resultsSummary.WriteString(fmt.Sprintf("Sim %d: Initial=$%.2f, Final=$%.2f, Return=%.2f%%\n",
			r.SimNum, r.InitialValue, r.FinalValue, r.ReturnPct))
	}

	// Calculate statistics
	var avgReturn, minReturn, maxReturn, positiveCount float64
	minReturn = math.Inf(1)
	maxReturn = math.Inf(-1)
	for _, r := range agentState.SimulationResults {
		avgReturn += r.ReturnPct
		if r.ReturnPct < minReturn {
			minReturn = r.ReturnPct
		}
		if r.ReturnPct > maxReturn {
			maxReturn = r.ReturnPct
		}
		if r.ReturnPct > 0 {
			positiveCount++
		}
	}
	avgReturn /= float64(len(agentState.SimulationResults))

	// Create prompt for the risk manager
	prompt := fmt.Sprintf(`You are a cautious risk manager. Your analyst proposed a strategy. You have run simulations to test it.

Based on the potential outcomes, make a final, concrete decision.

If results are highly variable or negative, reduce risk (e.g., buy/sell fewer shares, or hold).

Initial Proposal: %s

Simulation Results:
%s

Real Market State:
%s

Simulation Statistics:
- Average Return: %.2f%%
- Best Case: %.2f%%
- Worst Case: %.2f%%
- Positive Outcomes: %d/%d

Respond in the following format:
DECISION: [buy|sell|hold]
AMOUNT: [number of shares, 0 if holding]
REASONING: [final reasoning, referencing simulation results]`,
		agentState.ProposedAction.Strategy,
		resultsSummary.String(),
		agentState.RealMarket.GetStateString(),
		avgReturn, maxReturn, minReturn,
		int(positiveCount), len(agentState.SimulationResults))

	// Call LLM
	llm := state["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("risk manager LLM call failed: %w", err)
	}

	// Parse the response

	decision := parseFinalDecision(resp)
	agentState.FinalDecision = decision

	fmt.Printf("Final Decision: %s %.0f shares. Reason: %s\n",
		decision.Action, decision.Amount, decision.Reasoning)

	return state, nil
}

// ExecuteInRealWorldNode executes the final decision in the real market
// This is the "EXECUTE" step of the mental loop
func ExecuteInRealWorldNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	agentState := state["agent_state"].(*AgentState)

	fmt.Println("\n--- 🚀 Executing in Real World ---")

	decision := agentState.FinalDecision
	realMarket := agentState.RealMarket

	fmt.Printf("Before: %s\n", realMarket.GetStateString())
	realMarket.Step(decision.Action, decision.Amount)
	fmt.Printf("After: %s\n", realMarket.GetStateString())

	return state, nil
}

// ==================== Parsing Helpers ====================

func parseProposedAction(response string) *ProposedAction {
	proposal := &ProposedAction{
		Strategy:  "hold",
		Reasoning: response,
	}

	lines := strings.Split(response, "\n")
	inReasoning := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		upperLine := strings.ToUpper(line)

		if strings.HasPrefix(upperLine, "STRATEGY:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				proposal.Strategy = strings.TrimSpace(parts[1])
				// Remove markdown formatting like **STRATEGY:**
				proposal.Strategy = strings.ReplaceAll(proposal.Strategy, "**", "")
				proposal.Strategy = strings.ToLower(proposal.Strategy)
			}
			inReasoning = false
		} else if strings.HasPrefix(upperLine, "REASONING:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				proposal.Reasoning = strings.TrimSpace(parts[1])
				// Remove markdown formatting
				proposal.Reasoning = strings.ReplaceAll(proposal.Reasoning, "**", "")
			}
			inReasoning = true
		} else if inReasoning && line != "" {
			// Continue collecting reasoning
			if proposal.Reasoning != "" && proposal.Reasoning != response {
				proposal.Reasoning += " " + line
			}
		}
	}

	// If no explicit reasoning field was found, use the whole response
	// but try to extract just the reasoning part
	if proposal.Reasoning == response {
		// Try to find reasoning after STRATEGY line
		lines := strings.Split(response, "\n")
		for i, line := range lines {
			if strings.Contains(strings.ToUpper(line), "STRATEGY:") {
				if i+1 < len(lines) {
					reasoningLines := []string{}
					for j := i + 1; j < len(lines); j++ {
						nextLine := strings.TrimSpace(lines[j])
						if nextLine != "" && !strings.HasPrefix(strings.ToUpper(nextLine), "STRATEGY:") {
							reasoningLines = append(reasoningLines, nextLine)
						}
					}
					if len(reasoningLines) > 0 {
						proposal.Reasoning = strings.Join(reasoningLines, " ")
					}
				}
				break
			}
		}
	}

	return proposal
}

func parseFinalDecision(response string) *FinalDecision {
	decision := &FinalDecision{
		Action:    "hold",
		Amount:    0,
		Reasoning: response,
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		upperLine := strings.ToUpper(line)

		if strings.HasPrefix(upperLine, "DECISION:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				decision.Action = strings.TrimSpace(parts[1])
				decision.Action = strings.ToLower(decision.Action)
				// Extract just the action word
				words := strings.Fields(decision.Action)
				if len(words) > 0 {
					decision.Action = words[0]
				}
			}
		} else if strings.HasPrefix(upperLine, "AMOUNT:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				var amount float64
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &amount)
				decision.Amount = amount
			}
		} else if strings.HasPrefix(upperLine, "REASONING:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				decision.Reasoning = strings.TrimSpace(parts[1])
			}
		}
	}

	return decision
}

// ==================== Main Function ====================

func main() {
	// Check for API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	fmt.Println("=== 📘 Mental Loop (Simulator-in-the-Loop) Architecture ===")
	fmt.Println()
	fmt.Println("This demo implements a trading agent that uses an internal simulator")
	fmt.Println("to test proposed actions before executing them in the real market.")
	fmt.Println()
	fmt.Println("Architecture: OBSERVE -> PROPOSE -> SIMULATE -> REFINE -> EXECUTE")
	fmt.Println()

	// Create LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Create the mental loop graph with map state
	workflow := graph.NewStateGraph[map[string]any]()

	// Add nodes
	workflow.AddNode("propose", "Observe and propose action", ProposeActionNode)
	workflow.AddNode("simulate", "Run simulations", RunSimulationNode)
	workflow.AddNode("refine", "Refine decision", RefineAndDecideNode)
	workflow.AddNode("execute", "Execute in real world", ExecuteInRealWorldNode)

	// Define edges: propose -> simulate -> refine -> execute
	workflow.AddEdge("propose", "simulate")
	workflow.AddEdge("simulate", "refine")
	workflow.AddEdge("refine", "execute")
	workflow.AddEdge("execute", graph.END)

	// Set entry point
	workflow.SetEntryPoint("propose")

	// Compile the graph
	app, err := workflow.Compile()
	if err != nil {
		log.Fatalf("Failed to compile graph: %v", err)
	}

	ctx := context.Background()

	// Create initial market state
	realMarket := &MarketSimulator{
		Day:        0,
		Price:      100.0,
		Volatility: 0.1,  // Standard deviation for price changes
		Drift:      0.01, // General trend
		MarketNews: "Market is stable.",
		Portfolio: Portfolio{
			Cash:   10000.0,
			Shares: 0,
		},
	}

	fmt.Println("--- Initial Market State ---")
	fmt.Println(realMarket.GetStateString())

	// Day 1: Good News
	fmt.Println("\n--- Day 1: Good News Hits! ---")
	realMarket.MarketNews = "Positive earnings report expected."
	realMarket.Drift = 0.05

	agentState := &AgentState{RealMarket: realMarket}
	input := map[string]any{
		"llm":         llm,
		"agent_state": agentState,
	}

	result, err := app.Invoke(ctx, input)
	if err != nil {
		log.Fatalf("Mental loop execution failed: %v", err)
	}
	agentState = result["agent_state"].(*AgentState)

	// Day 2: Bad News
	fmt.Println("\n--- Day 2: Bad News Hits! ---")
	agentState.RealMarket.MarketNews = "New competitor enters the market."
	agentState.RealMarket.Drift = -0.05

	input = map[string]any{
		"llm":         llm,
		"agent_state": agentState,
	}

	result, err = app.Invoke(ctx, input)
	if err != nil {
		log.Fatalf("Mental loop execution failed: %v", err)
	}

	// Print final summary
	fmt.Println("\n=== 📊 Final Results ===")
	finalState := result["agent_state"].(*AgentState)
	fmt.Printf("Final Market State: %s\n", finalState.RealMarket.GetStateString())

	initialValue := 10000.0
	finalValue := finalState.RealMarket.Portfolio.Value(finalState.RealMarket.Price)
	totalReturn := finalValue - initialValue
	returnPct := (totalReturn / initialValue) * 100
	fmt.Printf("\nTotal Return: $%.2f (%.2f%%)\n", totalReturn, returnPct)

	fmt.Println("\n=== 🎯 Key Takeaways ===")
	fmt.Println("The Mental Loop architecture enables agents to:")
	fmt.Println("1. Think before acting by simulating outcomes")
	fmt.Println("2. Assess risks before committing to real-world actions")
	fmt.Println("3. Refine strategies based on what-if analysis")
	fmt.Println("4. Make more nuanced, safer decisions in dynamic environments")
}
