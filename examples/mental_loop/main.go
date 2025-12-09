package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// MarketState represents the state of the stock market
type MarketState struct {
	Price       float64
	Trend       string // "bullish", "bearish", "neutral"
	NewsEvent   string
	DayNumber   int
	AvgVolume   int
	Volatility  float64
	Shares      int
	Cash        float64
}

// NewMarketState creates a new market state
func NewMarketState(price float64, trend, news string, day int) *MarketState {
	return &MarketState{
		Price:      price,
		Trend:      trend,
		NewsEvent:  news,
		DayNumber:  day,
		AvgVolume:  1000000,
		Volatility: 0.02,
		Shares:     0,
		Cash:       10000.0,
	}
}

// Copy creates a deep copy of the market state (for simulation)
func (m *MarketState) Copy() *MarketState {
	return &MarketState{
		Price:      m.Price,
		Trend:      m.Trend,
		NewsEvent:  m.NewsEvent,
		DayNumber:  m.DayNumber,
		AvgVolume:  m.AvgVolume,
		Volatility: m.Volatility,
		Shares:     m.Shares,
		Cash:       m.Cash,
	}
}

// Step simulates one day of market activity
func (m *MarketState) Step(action string, amount int) {
	// Execute the action
	switch action {
	case "buy":
		cost := float64(amount) * m.Price
		if cost <= m.Cash {
			m.Shares += amount
			m.Cash -= cost
		}
	case "sell":
		if amount <= m.Shares {
			m.Shares -= amount
			m.Cash += float64(amount) * m.Price
		}
	case "hold":
		// Do nothing
	}

	// Simulate market movement
	priceChange := 0.0
	switch m.Trend {
	case "bullish":
		priceChange = m.Price * (0.01 + rand.Float64()*m.Volatility)
	case "bearish":
		priceChange = -m.Price * (0.01 + rand.Float64()*m.Volatility)
	case "neutral":
		priceChange = m.Price * (rand.Float64()*m.Volatility*2 - m.Volatility)
	}

	m.Price += priceChange
	m.DayNumber++
}

// GetStateString returns a human-readable description
func (m *MarketState) GetStateString() string {
	portfolioValue := m.Cash + float64(m.Shares)*m.Price
	return fmt.Sprintf(
		"Day %d | Price: $%.2f | Trend: %s | News: %s | Shares: %d | Cash: $%.2f | Portfolio: $%.2f",
		m.DayNumber, m.Price, m.Trend, m.NewsEvent, m.Shares, m.Cash, portfolioValue,
	)
}

// AgentState represents the state of the trading agent
type AgentState struct {
	RealMarket        *MarketState
	ProposedAction    string
	ProposedAmount    int
	SimulationResults []SimulationResult
	FinalDecision     string
	FinalAmount       int
	Reasoning         string
}

// SimulationResult stores the outcome of one simulation
type SimulationResult struct {
	Action       string
	Amount       int
	FinalPrice   float64
	FinalValue   float64
	ProfitLoss   float64
}

// AnalystNode observes the market and proposes a trading strategy
func AnalystNode(ctx context.Context, state interface{}) (interface{}, error) {
	stateMap := state.(map[string]interface{})
	agentState := stateMap["agent_state"].(*AgentState)
	market := agentState.RealMarket

	// Create prompt for the analyst
	prompt := fmt.Sprintf(`You are a stock market analyst. Based on the current market conditions, propose a trading strategy.

Current Market State:
%s

You must respond in EXACTLY this format:
ACTION: buy
AMOUNT: 25
REASONING: Your detailed analysis here explaining why you recommend this action.

Valid actions are: buy, sell, or hold
Amount should be between 0 and 50 shares
Provide a clear reasoning based on the market trend and news.`,
		market.GetStateString())

	// Call LLM
	llm := stateMap["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("analyst LLM call failed: %w", err)
	}

	// Parse the response
	action, amount, reasoning := parseAnalystResponse(resp)

	agentState.ProposedAction = action
	agentState.ProposedAmount = amount
	agentState.Reasoning = reasoning

	fmt.Println("\n📊 ANALYST PROPOSAL:")
	fmt.Printf("Action: %s %d shares\n", action, amount)
	fmt.Printf("Reasoning: %s\n", reasoning)

	return stateMap, nil
}

// SimulatorNode runs multiple simulations to test the proposed strategy
func SimulatorNode(ctx context.Context, state interface{}) (interface{}, error) {
	stateMap := state.(map[string]interface{})
	agentState := stateMap["agent_state"].(*AgentState)
	numSimulations := 5
	horizon := 10 // Simulate 10 days ahead

	fmt.Println("\n🔬 RUNNING SIMULATIONS:")

	var results []SimulationResult

	for i := 0; i < numSimulations; i++ {
		// Create a copy of the market for simulation
		simMarket := agentState.RealMarket.Copy()
		initialValue := simMarket.Cash + float64(simMarket.Shares)*simMarket.Price

		// Execute proposed action
		simMarket.Step(agentState.ProposedAction, agentState.ProposedAmount)

		// Run forward for the horizon
		for day := 0; day < horizon; day++ {
			simMarket.Step("hold", 0)
		}

		// Calculate final value
		finalValue := simMarket.Cash + float64(simMarket.Shares)*simMarket.Price
		profitLoss := finalValue - initialValue

		result := SimulationResult{
			Action:     agentState.ProposedAction,
			Amount:     agentState.ProposedAmount,
			FinalPrice: simMarket.Price,
			FinalValue: finalValue,
			ProfitLoss: profitLoss,
		}

		results = append(results, result)

		fmt.Printf("  Sim %d: Final Value: $%.2f (P/L: $%.2f)\n",
			i+1, finalValue, profitLoss)
	}

	agentState.SimulationResults = results

	return stateMap, nil
}

// RiskManagerNode analyzes simulations and makes the final decision
func RiskManagerNode(ctx context.Context, state interface{}) (interface{}, error) {
	stateMap := state.(map[string]interface{})
	agentState := stateMap["agent_state"].(*AgentState)

	// Calculate simulation statistics
	var totalPL, minPL, maxPL float64
	minPL = 1e9
	maxPL = -1e9

	for _, result := range agentState.SimulationResults {
		totalPL += result.ProfitLoss
		if result.ProfitLoss < minPL {
			minPL = result.ProfitLoss
		}
		if result.ProfitLoss > maxPL {
			maxPL = result.ProfitLoss
		}
	}

	avgPL := totalPL / float64(len(agentState.SimulationResults))

	// Create prompt for risk manager
	prompt := fmt.Sprintf(`You are a risk manager evaluating a trading decision.

Original Proposal: %s %d shares
Analyst Reasoning: %s

Simulation Results (%d scenarios, 10-day horizon):
- Average P/L: $%.2f
- Best Case: $%.2f
- Worst Case: $%.2f

Current Market: %s

Based on the simulation results, make your final decision. You can:
1. Approve the proposal as-is
2. Reduce the position size to manage risk (e.g., from 50 to 20 shares)
3. Reject and hold instead if risk is too high

You must respond in EXACTLY this format:
DECISION: buy
AMOUNT: 20
REASONING: Based on simulations showing positive expected value (avg $XXX), I approve the trade but reduce size to 20 shares to manage downside risk.

Valid decisions are: buy, sell, or hold
Amount should reflect your risk-adjusted position size.`,
		agentState.ProposedAction,
		agentState.ProposedAmount,
		agentState.Reasoning,
		len(agentState.SimulationResults),
		avgPL,
		maxPL,
		minPL,
		agentState.RealMarket.GetStateString())

	// Call LLM
	llm := stateMap["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("risk manager LLM call failed: %w", err)
	}

	// Debug: print raw response
	if false { // Set to true for debugging
		fmt.Printf("\n[DEBUG] Risk Manager Raw Response:\n%s\n", resp)
	}

	// Parse the response
	action, amount, reasoning := parseAnalystResponse(resp)

	agentState.FinalDecision = action
	agentState.FinalAmount = amount

	fmt.Println("\n⚖️  RISK MANAGER DECISION:")
	fmt.Printf("Decision: %s %d shares\n", action, amount)
	fmt.Printf("Reasoning: %s\n", reasoning)
	fmt.Printf("\nSimulation Stats - Avg P/L: $%.2f, Range: [$%.2f, $%.2f]\n",
		avgPL, minPL, maxPL)

	return stateMap, nil
}

// ExecuteNode executes the final decision in the real market
func ExecuteNode(ctx context.Context, state interface{}) (interface{}, error) {
	stateMap := state.(map[string]interface{})
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("\n💼 EXECUTING IN REAL MARKET:")
	beforeValue := agentState.RealMarket.Cash + float64(agentState.RealMarket.Shares)*agentState.RealMarket.Price
	fmt.Printf("Before: %s\n", agentState.RealMarket.GetStateString())

	// Execute the decision
	agentState.RealMarket.Step(agentState.FinalDecision, agentState.FinalAmount)

	afterValue := agentState.RealMarket.Cash + float64(agentState.RealMarket.Shares)*agentState.RealMarket.Price
	fmt.Printf("After:  %s\n", agentState.RealMarket.GetStateString())
	fmt.Printf("Immediate Impact: $%.2f\n", afterValue-beforeValue)

	return stateMap, nil
}

// parseAnalystResponse extracts action, amount, and reasoning from LLM response
func parseAnalystResponse(response string) (string, int, string) {
	action := "hold"
	amount := 0
	reasoning := ""

	lines := strings.Split(response, "\n")
	inReasoning := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		upperLine := strings.ToUpper(line)

		// Support both ACTION: and DECISION: keywords
		if strings.HasPrefix(upperLine, "ACTION:") || strings.HasPrefix(upperLine, "DECISION:") {
			// Extract action
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				actionStr := strings.TrimSpace(parts[1])
				actionStr = strings.ToLower(actionStr)
				// Handle cases like "buy" or "buy 20 shares"
				actionWords := strings.Fields(actionStr)
				if len(actionWords) > 0 {
					firstWord := actionWords[0]
					if firstWord == "buy" || firstWord == "sell" || firstWord == "hold" {
						action = firstWord
					}
				}
			}
			inReasoning = false
		} else if strings.HasPrefix(upperLine, "AMOUNT:") {
			// Extract amount
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				amountStr := strings.TrimSpace(parts[1])
				// Extract first number from the string
				var num int
				fmt.Sscanf(amountStr, "%d", &num)
				if num >= 0 && num <= 100 {
					amount = num
				}
			}
			inReasoning = false
		} else if strings.HasPrefix(upperLine, "REASONING:") {
			// Start collecting reasoning
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				reasoning = strings.TrimSpace(parts[1])
			}
			inReasoning = true
		} else if inReasoning && line != "" {
			// Continue collecting reasoning
			reasoning += " " + line
		}
	}

	// Fallback: if no action found, try to find keywords in the response
	if action == "hold" && reasoning == "" {
		lowerResp := strings.ToLower(response)
		if strings.Contains(lowerResp, "buy") && !strings.Contains(lowerResp, "don't buy") {
			action = "buy"
			// Try to extract a number
			words := strings.Fields(response)
			for i, word := range words {
				if strings.ToLower(word) == "buy" && i+1 < len(words) {
					var num int
					fmt.Sscanf(words[i+1], "%d", &num)
					if num > 0 && num <= 100 {
						amount = num
						break
					}
				}
			}
			if amount == 0 {
				amount = 10 // Default amount
			}
			reasoning = response
		} else if strings.Contains(lowerResp, "sell") {
			action = "sell"
			reasoning = response
		}
	}

	// If reasoning is still empty, use the whole response
	if reasoning == "" {
		reasoning = response
	}

	// Trim reasoning to reasonable length
	if len(reasoning) > 200 {
		reasoning = reasoning[:200] + "..."
	}

	return action, amount, reasoning
}

func main() {
	fmt.Println("=== Mental Loop Trading Agent ===")
	fmt.Println()
	fmt.Println("This demo implements the 'Mental Loop' (Simulator-in-the-Loop) architecture.")
	fmt.Println("The agent tests proposed actions in a simulated environment before executing.")
	fmt.Println()

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Create LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Create initial market state
	market := NewMarketState(100.0, "bullish", "Positive earnings report", 1)

	// Create agent state
	agentState := &AgentState{
		RealMarket: market,
	}

	// Create the mental loop graph
	workflow := graph.NewMessageGraph()

	// Add nodes
	workflow.AddNode("analyst", "analyst", AnalystNode)
	workflow.AddNode("simulator", "simulator", SimulatorNode)
	workflow.AddNode("risk_manager", "risk_manager", RiskManagerNode)
	workflow.AddNode("execute", "execute", ExecuteNode)

	// Define edges: analyst -> simulator -> risk_manager -> execute
	workflow.AddEdge("analyst", "simulator")
	workflow.AddEdge("simulator", "risk_manager")
	workflow.AddEdge("risk_manager", "execute")
	workflow.AddEdge("execute", graph.END)

	// Set entry point
	workflow.SetEntryPoint("analyst")

	// Compile the graph
	app, err := workflow.Compile()
	if err != nil {
		log.Fatalf("Failed to compile graph: %v", err)
	}

	ctx := context.Background()

	// Run the mental loop for several days
	numDays := 3

	for day := 1; day <= numDays; day++ {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Printf("DAY %d - MENTAL LOOP CYCLE\n", day)
		fmt.Println(strings.Repeat("=", 80))

		// Prepare input state
		input := map[string]interface{}{
			"llm":         llm,
			"agent_state": agentState,
		}

		// Run the mental loop
		result, err := app.Invoke(ctx, input)
		if err != nil {
			log.Fatalf("Mental loop execution failed: %v", err)
		}

		// Extract updated state
		resultMap := result.(map[string]interface{})
		agentState = resultMap["agent_state"].(*AgentState)

		// Update market conditions for next day (simulate external events)
		if day < numDays {
			updateMarketConditions(agentState.RealMarket, day)
		}

		fmt.Println()
	}

	// Print final summary
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("FINAL RESULTS")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Final Market State: %s\n", agentState.RealMarket.GetStateString())
	initialValue := 10000.0
	finalValue := agentState.RealMarket.Cash + float64(agentState.RealMarket.Shares)*agentState.RealMarket.Price
	totalReturn := finalValue - initialValue
	returnPct := (totalReturn / initialValue) * 100
	fmt.Printf("\nTotal Return: $%.2f (%.2f%%)\n", totalReturn, returnPct)

	fmt.Println("\n=== How Mental Loop Works ===")
	fmt.Println("1. OBSERVE: Analyst observes current market conditions")
	fmt.Println("2. PROPOSE: Analyst proposes a trading strategy")
	fmt.Println("3. SIMULATE: Strategy is tested in multiple simulated scenarios")
	fmt.Println("4. ASSESS: Risk manager evaluates simulation results")
	fmt.Println("5. REFINE: Risk manager adjusts strategy based on risk/reward")
	fmt.Println("6. EXECUTE: Final decision is executed in the real market")
	fmt.Println()
	fmt.Println("This 'think before you act' approach reduces costly mistakes")
	fmt.Println("by allowing the agent to preview outcomes in a safe sandbox.")
}

// updateMarketConditions simulates changing market conditions between days
func updateMarketConditions(market *MarketState, day int) {
	fmt.Println("\n🌍 MARKET UPDATE:")

	// Simulate different market scenarios
	scenarios := []struct {
		trend string
		news  string
	}{
		{"bullish", "Strong sector performance reported"},
		{"bearish", "Major competitor announces new product"},
		{"neutral", "Mixed economic indicators"},
		{"bullish", "Analyst upgrades stock rating"},
		{"bearish", "Regulatory concerns emerge"},
	}

	if day < len(scenarios) {
		market.Trend = scenarios[day].trend
		market.NewsEvent = scenarios[day].news
		fmt.Printf("New Trend: %s | News: %s\n", market.Trend, market.NewsEvent)
	}
}
