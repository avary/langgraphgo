# Generic StateGraph Example

This example demonstrates the **type-safe generic StateGraph** implementation in LangGraphGo.

## Overview

Generic StateGraph provides compile-time type safety for your state management, eliminating the need for type assertions and reducing runtime errors.

## Key Benefits

✅ **Compile-Time Type Safety** - Catch errors before runtime
✅ **No Type Assertions** - Direct field access on state
✅ **Better IDE Support** - Full autocomplete and refactoring
✅ **Cleaner Code** - Less boilerplate, more readable
✅ **Zero Runtime Overhead** - Generics are compile-time only

## Examples Included

### Example 1: Simple Type-Safe Graph

Demonstrates basic usage with a workflow that checks user eligibility:
- Type-safe node functions
- No type assertions needed
- Direct access to state fields

```go
g := graph.NewStateGraph[WorkflowState]()

g.AddNode("check_age", "Check age", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
    state.IsAdult = state.Request.Age >= 18  // Type-safe!
    return state, nil
})
```

### Example 2: Conditional Routing

Shows type-safe conditional edges:
- Conditional routing based on state
- Multiple execution paths
- Type safety in condition functions

```go
g.AddConditionalEdge("check_age", func(ctx context.Context, state WorkflowState) string {
    if state.IsAdult {  // No type assertion needed!
        return "adult_path"
    }
    return "minor_path"
})
```

### Example 3: Schema-Based State Merging

Demonstrates advanced state management with custom merge logic:
- Initialize state with default values
- Custom merge functions for complex state updates
- Partial state updates that get merged automatically

```go
schema := graph.NewStructSchema(
    ProcessState{MaxCount: 5},
    func(current, new ProcessState) (ProcessState, error) {
        // Custom merge logic
        current.Items = append(current.Items, new.Items...)
        current.Count += new.Count
        return current, nil
    },
)
g.SetSchema(schema)
```

## Running the Example

```bash
cd examples/generic_state_graph
go run main.go
```

## Expected Output

```
=== Generic StateGraph Example ===

Example 1: Simple Type-Safe Graph
-----------------------------------
Checking age for Alice (25 years old)
Checking eligibility for Alice
Final result: ✅ Alice is eligible!

Final State:
  Result: ✅ Alice is eligible!
  Notifications: 2 messages

==================================================

Example 2: Conditional Routing
-------------------------------
Checking age for Bob (30 years old)
  → Taking adult path
Result: Bob (adult) - Full access granted

Checking age for Charlie (15 years old)
  → Taking minor path
Result: Charlie (minor) - Limited access

==================================================

Example 3: Using Schema for State Merging
-----------------------------------------
Processing: item_1 (count: 1/5)
Processing: item_2 (count: 2/5)
Processing: item_3 (count: 3/5)
Processing: item_4 (count: 4/5)
Processing: item_5 (count: 5/5)

Final State:
  Items processed: [item_1 item_2 item_3 item_4 item_5]
  Total count: 5
  Max count: 5
  Processing: true
```

## Comparison: Generic vs Non-Generic

### Non-Generic (Old Way)

```go
g := graph.NewStateGraph[map[string]any]()

g.AddNode("process", "desc", func(ctx context.Context, state any) (any, error) {
    s := state.(WorkflowState)  // Type assertion required ❌
    s.Count++
    return s, nil
})

result, _ := app.Invoke(ctx, initialState)
finalState := result.(WorkflowState)  // Another type assertion ❌
```

### Generic (New Way)

```go
g := graph.NewStateGraph[WorkflowState]()

g.AddNode("process", "desc", func(ctx context.Context, state WorkflowState) (WorkflowState, error) {
    state.Count++  // Direct access ✅
    return state, nil
})

finalState, _ := app.Invoke(ctx, initialState)  // Type-safe result ✅
```

## State Type Definition

```go
type WorkflowState struct {
    Request       UserRequest
    IsAdult       bool
    IsEligible    bool
    Notifications []string
    Result        string
}
```

## API Reference

### Creating a Generic Graph

```go
g := graph.NewStateGraph[YourStateType]()
```

### Adding Typed Nodes

```go
g.AddNode(name string, description string,
    fn func(ctx context.Context, state YourStateType) (YourStateType, error))
```

### Adding Typed Conditional Edges

```go
g.AddConditionalEdge(from string,
    condition func(ctx context.Context, state YourStateType) string)
```

### Compiling and Invoking

```go
app, err := g.Compile()
result, err := app.Invoke(ctx, initialState)  // Returns YourStateType
```

### Using Schema

```go
schema := graph.NewStructSchema(
    initialValue YourStateType,
    mergeFunc func(current, new YourStateType) (YourStateType, error),
)
g.SetSchema(schema)
```

## When to Use Generic StateGraph

**Use Generic StateGraph when:**
- ✅ You have a well-defined state struct
- ✅ Type safety is important
- ✅ Building a new project
- ✅ You want better IDE support

**Use Non-Generic StateGraph when:**
- ✅ You need maximum flexibility
- ✅ State structure is dynamic
- ✅ Using `map[string]any` with complex reducers
- ✅ Migrating from Python LangGraph

## Learn More

- [RFC: Generic StateGraph Design](../../docs/RFC_GENERIC_STATEGRAPH.md)
- [LangGraphGo Documentation](../../README.md)
- [State Schema Documentation](../state_schema/README.md)

## Migration from Non-Generic

Migrating is straightforward:

1. **Change constructor:**
   ```go
   // Before
   g := graph.NewStateGraph[map[string]any]()

   // After
   g := graph.NewStateGraph[MyState]()
   ```

2. **Update node functions:**
   ```go
   // Before
   func(ctx context.Context, state any) (any, error) {
       s := state.(MyState)
       // ...
   }

   // After
   func(ctx context.Context, state MyState) (MyState, error) {
       // Direct access to state fields
   }
   ```

3. **No changes to edges:**
   ```go
   g.AddEdge("from", "to")  // Same as before
   g.SetEntryPoint("start")  // Same as before
   ```

4. **Update invocation:**
   ```go
   // Before
   result, _ := app.Invoke(ctx, initialState)
   finalState := result.(MyState)

   // After
   finalState, _ := app.Invoke(ctx, initialState)
   ```

That's it! Your graph is now type-safe.
