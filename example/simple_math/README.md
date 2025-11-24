# Simple Math Workflow Example

A basic workflow demonstration showing sequential step execution with data passing between steps.

## Overview

This example demonstrates:
- Sequential workflow execution
- Type-safe input/output between steps
- Step configuration (retries, timeouts, backoff)
- Fluent workflow builder API

## Workflow Steps

1. **Add** - Adds two numbers
2. **Multiply** - Multiplies the result by a factor
3. **Format** - Formats the final output as a message

## Usage

```go
import (
    "context"
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
    "github.com/sicko7947/gorkflow/example/simple_math"
)

func main() {
    // Create engine with default logger and config
    eng := engine.NewEngine(store.NewMemoryStore())

    // Create workflow
    wf, _ := simple_math.NewSimpleMathWorkflow()

    // Execute
    input := simple_math.WorkflowInput{
        Val1: 10,
        Val2: 5,
        Mult: 2,
    }

    runID, _ := eng.StartWorkflow(context.Background(), wf, input)

    // Result: (10 + 5) * 2 = 30
}
```

## Using the Orchestrator

For a more structured approach, use the included orchestrator:

```go
import (
    "context"
    "github.com/rs/zerolog"
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
    "github.com/sicko7947/gorkflow/example/simple_math"
)

func main() {
    logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
    store := store.NewMemoryStore()

    // Create orchestrator (encapsulates workflow + engine)
    orch, _ := simple_math.NewOrchestrator(
        store,
        logger,
        engine.DefaultEngineConfig,
    )

    // Start workflow
    input := simple_math.WorkflowInput{Val1: 10, Val2: 5, Mult: 2}
    runID, _ := orch.StartWorkflow(context.Background(), input)

    // Get status
    status, _ := orch.GetWorkflowStatus(context.Background(), runID)
    fmt.Printf("Status: %s, Output: %+v\n", status.Status, status.Output)

    // Cancel if needed
    _ = orch.CancelWorkflow(context.Background(), runID)
}
```

## Features Demonstrated

- **Type Safety**: Each step has strongly-typed inputs and outputs
- **Data Flow**: Output from one step becomes input for the next
- **Configuration**: The multiply step shows custom retry and timeout settings
- **Builder Pattern**: Clean, fluent API for constructing workflows
- **Orchestrator Pattern**: Encapsulates engine and workflow for cleaner API
