# Quick Start

Build and run your first Gorkflow workflow in under 5 minutes!

## Overview

In this quick start, you'll create a simple calculation workflow that:

1. Adds two numbers
2. Multiplies the result
3. Formats the output

## Complete Example

Create a file `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sicko7947/gorkflow"
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
)

// Define your data types
type CalculationInput struct {
    A int `json:"a"`
    B int `json:"b"`
}

type SumOutput struct {
    Sum int `json:"sum"`
}

type ProductOutput struct {
    Product int `json:"product"`
}

type ResultOutput struct {
    Result  int    `json:"result"`
    Message string `json:"message"`
}

func main() {
    // Step 1: Create steps
    addStep := gorkflow.NewStep(
        "add",
        "Add Numbers",
        func(ctx *gorkflow.StepContext, input CalculationInput) (SumOutput, error) {
            sum := input.A + input.B
            ctx.Logger.Info().Int("sum", sum).Msg("Addition completed")
            return SumOutput{Sum: sum}, nil
        },
    )

    multiplyStep := gorkflow.NewStep(
        "multiply",
        "Multiply by 2",
        func(ctx *gorkflow.StepContext, input SumOutput) (ProductOutput, error) {
            product := input.Sum * 2
            ctx.Logger.Info().Int("product", product).Msg("Multiplication completed")
            return ProductOutput{Product: product}, nil
        },
    )

    formatStep := gorkflow.NewStep(
        "format",
        "Format Result",
        func(ctx *gorkflow.StepContext, input ProductOutput) (ResultOutput, error) {
            message := fmt.Sprintf("Final result: %d", input.Product)
            return ResultOutput{
                Result:  input.Product,
                Message: message,
            }, nil
        },
    )

    // Step 2: Build the workflow
    wf, err := gorkflow.NewWorkflow("calculation", "Calculation Workflow").
        WithDescription("A simple calculation workflow").
        WithVersion("1.0").
        Sequence(addStep, multiplyStep, formatStep).
        Build()

    if err != nil {
        log.Fatal("Failed to build workflow:", err)
    }

    // Step 3: Create engine and storage
    store := store.NewMemoryStore()
    eng := engine.NewEngine(store)

    // Step 4: Execute the workflow
    ctx := context.Background()
    runID, err := eng.StartWorkflow(
        ctx,
        wf,
        CalculationInput{A: 10, B: 5},
    )

    if err != nil {
        log.Fatal("Failed to start workflow:", err)
    }

    fmt.Printf("Workflow started with ID: %s\n", runID)

    // Step 5: Get the result
    run, err := eng.GetRun(ctx, runID)
    if err != nil {
        log.Fatal("Failed to get workflow status:", err)
    }

    fmt.Printf("Status: %s\n", run.Status)
    fmt.Printf("Progress: %.0f%%\n", run.Progress*100)

    if run.Output != nil {
        fmt.Printf("Output: %s\n", string(run.Output))
    }
}
```

## Run It

```bash
go run main.go
```

**Expected Output:**

```
Workflow started with ID: 550e8400-e29b-41d4-a716-446655440000
Status: completed
Progress: 100%
Output: {"result":30,"message":"Final result: 30"}
```

## Breaking It Down

### 1. Define Data Types

```go
type CalculationInput struct {
    A int `json:"a"`
    B int `json:"b"`
}
```

Each step needs typed input and output structs. These provide type safety and enable automatic marshaling.

### 2. Create Steps

```go
addStep := gorkflow.NewStep(
    "add",                    // Step ID (must be unique)
    "Add Numbers",            // Step name (human-readable)
    func(ctx *gorkflow.StepContext, input CalculationInput) (SumOutput, error) {
        // Your business logic here
        return SumOutput{Sum: input.A + input.B}, nil
    },
)
```

Steps are the building blocks of workflows. They're type-safe using Go generics.

### 3. Build the Workflow

```go
wf, _ := gorkflow.NewWorkflow("calculation", "Calculation Workflow").
    Sequence(addStep, multiplyStep, formatStep).
    Build()
```

The fluent builder API makes it easy to compose workflows. `Sequence()` chains steps together.

### 4. Create Engine and Storage

```go
store := store.NewMemoryStore()
eng := engine.NewEngine(store)
```

The engine orchestrates workflow execution. Storage backends persist workflow state.

### 5. Execute the Workflow

```go
runID, err := eng.StartWorkflow(ctx, wf, CalculationInput{A: 10, B: 5})
```

Start the workflow with your input data. The engine returns a unique run ID.

## What's Happening?

1. **Type Safety**: Go generics ensure your step inputs and outputs match at compile time
2. **Automatic Marshaling**: Data is automatically serialized between steps
3. **Progress Tracking**: The engine tracks execution progress
4. **State Persistence**: All state is stored (even with MemoryStore)
5. **Logging**: Each step logs its execution

## Next Steps

This was a basic sequential workflow. Now learn about:

- **[First Workflow Tutorial](first-workflow.md)** - Detailed walkthrough with explanations
- **[Parallel Execution](../advanced-usage/parallel-execution.md)** - Run steps concurrently
- **[Conditional Execution](../advanced-usage/conditional-execution.md)** - Dynamic workflows
- **[Validation](../core-concepts/validation.md)** - Add input/output validation
- **[Storage Backends](../storage/overview.md)** - Use persistent storage (LibSQL)

## Common Variations

### Add Retries and Timeouts

```go
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithRetries(3),
    gorkflow.WithTimeout(30 * time.Second),
)
```

### Use Custom Logger

```go
logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
eng := engine.NewEngine(store, engine.WithLogger(logger))
```

### Add Tags

```go
runID, _ := eng.StartWorkflow(ctx, wf, input,
    gorkflow.WithTags(map[string]string{
        "environment": "production",
        "team": "backend",
    }),
)
```

---

**Ready for a deeper dive?** Continue to the [First Workflow Tutorial](first-workflow.md) →
