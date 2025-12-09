# Mental Loop (Simulator-in-the-Loop) Example

This example demonstrates the **Mental Loop** architecture pattern, where an agent "thinks before acting" by testing proposed actions in a safe, simulated environment before executing them in the real world.

## Architecture Pattern

The Mental Loop pattern implements a deliberative decision-making process:

```
┌─────────────────────────────────────────────────────┐
│                   MENTAL LOOP                       │
│                                                     │
│  ┌──────────┐   ┌───────────┐   ┌──────────────┐ │
│  │ OBSERVE  │──▶│  PROPOSE  │──▶│   SIMULATE   │ │
│  │  Market  │   │ Strategy  │   │  Outcomes    │ │
│  └──────────┘   └───────────┘   └──────┬───────┘ │
│                                         │         │
│  ┌──────────┐   ┌───────────┐          │         │
│  │ EXECUTE  │◀──│  REFINE   │◀─────────┘         │
│  │  Action  │   │ Decision  │                     │
│  └──────────┘   └───────────┘                     │
└─────────────────────────────────────────────────────┘
```

## Key Components

### 1. **Analyst Node** (Observer & Proposer)
- Observes current market conditions
- Proposes high-level trading strategy (buy/sell/hold)
- Provides reasoning based on market trends and news

### 2. **Simulator Node** (Sandbox Environment)
- Creates deep copies of the market state
- Runs multiple simulations (Monte Carlo approach)
- Tests proposed strategy across different scenarios
- Generates statistical outcomes (best/worst/average cases)

### 3. **Risk Manager Node** (Assessor & Refiner)
- Analyzes simulation results
- Evaluates risk/reward profile
- Refines or adjusts the proposed strategy
- Makes final decision with risk considerations

### 4. **Execute Node** (Real-world Action)
- Applies the final decision to the actual environment
- Reports immediate impact
- Triggers next observation cycle

## Stock Trading Scenario

This example simulates a stock trading agent that:

1. **Observes** daily market conditions (price, trend, news)
2. **Proposes** trading actions through an AI analyst
3. **Simulates** outcomes over a 10-day horizon (5 scenarios)
4. **Assesses** statistical results (avg/min/max profit/loss)
5. **Refines** strategy through AI risk manager
6. **Executes** final decision in the real market

### Market Simulator

The market simulator provides:
- Price movements based on trends (bullish/bearish/neutral)
- Volatility modeling
- Portfolio tracking (shares + cash)
- Event-driven scenarios (earnings, competition, etc.)

## Workflow

```go
workflow := graph.NewGraph[map[string]interface{}]()

// Add nodes in sequence
workflow.AddNode("analyst", AnalystNode)
workflow.AddNode("simulator", SimulatorNode)
workflow.AddNode("risk_manager", RiskManagerNode)
workflow.AddNode("execute", ExecuteNode)

// Define the mental loop flow
workflow.AddEdge("analyst", "simulator")
workflow.AddEdge("simulator", "risk_manager")
workflow.AddEdge("risk_manager", "execute")
```

## Key Features

### Safe Exploration
Strategies are tested in sandboxed copies of the environment without real-world consequences.

### Statistical Analysis
Multiple simulation runs provide confidence intervals and risk metrics.

### Adaptive Decision-Making
The risk manager can modify proposals based on simulation outcomes (e.g., reduce position size if variance is high).

### Audit Trail
Complete reasoning chain from observation → proposal → simulation → decision.

## Example Output

```
=== DAY 1 - MENTAL LOOP CYCLE ===

📊 ANALYST PROPOSAL:
Action: buy 25 shares
Reasoning: Bullish trend with positive earnings. Strong entry point.

🔬 RUNNING SIMULATIONS:
  Sim 1: Final Value: $10,245.32 (P/L: $245.32)
  Sim 2: Final Value: $10,189.67 (P/L: $189.67)
  Sim 3: Final Value: $10,312.45 (P/L: $312.45)
  Sim 4: Final Value: $10,276.89 (P/L: $276.89)
  Sim 5: Final Value: $10,198.54 (P/L: $198.54)

⚖️  RISK MANAGER DECISION:
Decision: buy 20 shares
Reasoning: Positive expected value but reducing position to 20 shares
to manage downside risk. Conservative entry appropriate.

💼 EXECUTING IN REAL MARKET:
Before: Day 1 | Price: $100.00 | Portfolio: $10,000.00
After:  Day 1 | Price: $100.00 | Shares: 20 | Portfolio: $10,000.00
```

## Advantages

✅ **Risk Mitigation**: Test strategies before committing real resources
✅ **Probabilistic Reasoning**: Understand range of possible outcomes
✅ **Explainable Decisions**: Clear reasoning at each step
✅ **Adaptive Behavior**: Adjust strategy based on simulation results

## Limitations

⚠️ **Simulation Fidelity**: Effectiveness depends on accuracy of the simulator
⚠️ **Computational Cost**: Multiple simulations increase latency
⚠️ **Model Assumptions**: Real world may differ from simulated scenarios

## When to Use

This pattern is ideal when:
- Actions have significant consequences (financial, safety-critical)
- You have a reliable simulation environment
- Decision latency is acceptable (not real-time)
- You need explainable, auditable decisions

## Running the Example

```bash
cd examples/mental_loop
go run main.go
```

Make sure you have set your OpenAI API key:
```bash
export OPENAI_API_KEY="your-api-key"
```

## Comparison with Other Patterns

| Pattern | Description | When to Use |
|---------|-------------|-------------|
| **ReAct** | Reason + Act in real-time | Fast, low-stakes decisions |
| **Planning** | Create plan, then execute | Multi-step tasks |
| **Reflection** | Act, then critique | Quality improvement |
| **Mental Loop** | Simulate before acting | High-stakes decisions |

## Further Reading

- Original concept: [Simulator-based RL](https://arxiv.org/abs/2002.04898)
- Mental models in AI: [World Models](https://arxiv.org/abs/1803.10122)
- Agent architectures: [Agentic AI Patterns](https://github.com/FareedKhan-dev/all-agentic-architectures)

## License

This example is part of the LangGraphGo project.
