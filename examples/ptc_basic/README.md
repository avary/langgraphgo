# Programmatic Tool Calling (PTC) Example

This example demonstrates **Programmatic Tool Calling (PTC)**, an advanced technique that allows LLMs to write code that calls tools programmatically, rather than requiring multiple API round-trips for each tool invocation.

## What is PTC?

**Traditional Tool Calling**:
```
User Query → LLM → Tool Call → Tool Result → LLM → Tool Call → ... → Final Answer
```
Every tool call requires a round-trip to the LLM, which increases latency and token costs.

**Programmatic Tool Calling (PTC)**:
```
User Query → LLM → Generated Code (with tool calls) → Execute Code → LLM → Final Answer
```
The LLM generates code (Python or Go) that calls multiple tools programmatically in a single execution.

## Benefits of PTC

1. **Reduced Latency**: Multiple tool calls execute in one step instead of multiple API round-trips
2. **Token Efficiency**: Code can filter and aggregate data before returning results
3. **Programmatic Control**: Use loops, conditions, and data processing in code
4. **Cost Savings**: Fewer API calls and reduced token usage

## Example Use Case

This example asks the LLM to:
1. Get weather for San Francisco and New York (2 tool calls)
2. Calculate the average temperature (data processing)
3. Multiply the result by 2 (calculation)

With traditional tool calling, this would require 4+ API round-trips. With PTC, the LLM generates code that executes all operations in one step.

## Code Structure

### Tools Defined

1. **CalculatorTool**: Performs arithmetic operations (add, subtract, multiply, divide, power, sqrt)
2. **WeatherTool**: Returns mock weather data for cities
3. **DataProcessorTool**: Processes arrays (sum, average, max, min)

### PTC Configuration

```go
agent, err := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
    Model:         model,           // Any LLM implementing llms.Model
    Tools:         toolList,        // Available tools
    Language:      ptc.LanguagePython, // Python or Go
    MaxIterations: 5,               // Maximum iterations
})
```

## How It Works

1. **Agent Creation**: Creates a StateGraph with agent and code execution nodes
2. **Query Processing**: User query is sent to the LLM
3. **Code Generation**: LLM generates Python/Go code to call tools
4. **Code Execution**: Code runs in a sandbox with HTTP access to tools
5. **Result Processing**: Execution results are sent back to the LLM
6. **Final Answer**: LLM provides the final answer based on results

## Running the Example

```bash
# Set your OpenAI API key
export OPENAI_API_KEY=your-api-key

# Run the example
cd examples/ptc_basic
go run main.go
```

## Expected Output

The example will:
1. Display the user query
2. Show the LLM generating code to call tools
3. Execute the generated code
4. Display all messages (including generated code and tool results)
5. Show the final answer

## Key Features Demonstrated

- ✅ Multiple tool calls in single execution
- ✅ Programmatic data processing
- ✅ Reduced API latency
- ✅ Transparent execution (see generated code)
- ✅ Support for any LLM (OpenAI, Google, Anthropic, etc.)

## Configuration Options

### Language Selection

```go
ptc.LanguagePython  // Recommended: Better LLM support
ptc.LanguageGo      // Alternative: Better performance
```

**Recommendation**: Use Python as LLMs have more training data for Python code generation.

### Custom System Prompt

```go
ptc.PTCAgentConfig{
    SystemPrompt: "You are a data analysis assistant...",
    // ...
}
```

### Iteration Limits

```go
ptc.PTCAgentConfig{
    MaxIterations: 10,  // Prevent infinite loops
    // ...
}
```

## Advanced Examples

For more complex PTC examples, see:
- [PTC Simple](../ptc_simple/): Basic PTC usage
- [PTC Expense Analysis](../ptc_expense_analysis/): Real-world expense processing with large datasets
- [PTC Documentation](../../ptc/README.md): Full API reference and architecture

## Comparison with Traditional Approach

| Aspect | Traditional | PTC |
|--------|------------|-----|
| API Calls | 4+ | 1-2 |
| Latency | High | Low |
| Token Usage | All tool results | Filtered results |
| Data Processing | LLM reasoning | Code-based |
| Complex Logic | Difficult | Easy (loops, conditions) |
| Large Datasets | Token-heavy | Efficient |

## Learn More

- [PTC Package Documentation](../../ptc/README.md)
- [PTC Package README (Chinese)](../../ptc/README_CN.md)
- [Anthropic PTC Cookbook](https://github.com/anthropics/claude-cookbooks/blob/main/tool_use/programmatic_tool_calling_ptc.ipynb)
