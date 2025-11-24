# Conditional Execution Example

Demonstrates how to use the `ThenStepIf` builder API for conditional step execution based on runtime state.

## Overview

This example shows:
- **Builder-level conditional execution** using `ThenStepIf()`
- Condition evaluation from workflow state
- Default values when steps are skipped
- Runtime decision-making in workflows

## Workflow Steps

1. **Setup** - Extracts input flags and stores them in workflow state
2. **Double** (Conditional) - Doubles the value if `EnableDoubling` is true
3. **Format** (Conditional) - Formats the result if `EnableFormatting` is true

## Usage

```go
import (
    "context"
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
    "github.com/sicko7947/gorkflow/example/conditional"
)

func main() {
    // Create engine
    eng := engine.NewEngine(store.NewMemoryStore())

    // Create conditional workflow
    wf, _ := conditional.NewConditionalWorkflow()

    // Example 1: Enable both steps
    input1 := conditional.ConditionalInput{
        Value:            10,
        EnableDoubling:   true,
        EnableFormatting: true,
    }
    runID1, _ := eng.StartWorkflow(context.Background(), wf, input1)
    // Result: Value doubled to 20, formatted output

    // Example 2: Skip doubling
    input2 := conditional.ConditionalInput{
        Value:            10,
        EnableDoubling:   false,  // Step skipped!
        EnableFormatting: true,
    }
    runID2, _ := eng.StartWorkflow(context.Background(), wf, input2)
    // Result: Default output used, formatted

    // Example 3: Skip both conditional steps
    input3 := conditional.ConditionalInput{
        Value:            10,
        EnableDoubling:   false,
        EnableFormatting: false,
    }
    runID3, _ := eng.StartWorkflow(context.Background(), wf, input3)
    // Result: Both steps skipped, default values used
}
```

## How It Works

### 1. State-Based Conditions

The setup step stores flags in workflow state:
```go
ctx.State.Set("enable_doubling", input.EnableDoubling)
ctx.State.Set("enable_formatting", input.EnableFormatting)
```

### 2. Condition Functions

Conditions read from state:
```go
shouldDouble := func(ctx *workflow.StepContext) (bool, error) {
    var enableDoubling bool
    ctx.State.Get("enable_doubling", &enableDoubling)
    return enableDoubling, nil
}
```

### 3. Builder-Level API

Use `ThenStepIf` for clean conditional step addition:
```go
builder.NewWorkflow("example", "Example").
    ThenStep(setupStep).
    ThenStepIf(doubleStep, shouldDouble, defaultValue).  // ← Conditional!
    ThenStepIf(formatStep, shouldFormat, nil).
    Build()
```

## Key Concepts

- **Conditions** evaluate at runtime before step execution
- **Default values** are used when conditions return `false`
- **State access** allows conditions to make decisions based on workflow data
- **Builder API** (`ThenStepIf`) provides clean syntax for conditional steps

## Using the Orchestrator

For a structured approach, use the included orchestrator:

```go
import (
    "context"
    "github.com/rs/zerolog"
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
    "github.com/sicko7947/gorkflow/example/conditional"
)

func main() {
    logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
    store := store.NewMemoryStore()

    // Create orchestrator
    orch, _ := conditional.NewOrchestrator(
        store,
        logger,
        engine.DefaultEngineConfig,
    )

    // Start workflow with conditional flags
    input := conditional.ConditionalInput{
        Value:            10,
        EnableDoubling:   true,
        EnableFormatting: false,
    }
    runID, _ := orch.StartWorkflow(context.Background(), input)

    // Get status
    status, _ := orch.GetWorkflowStatus(context.Background(), runID)
    fmt.Printf("Status: %s, Output: %+v\n", status.Status, status.Output)
}
```

## Alternative Approach

For type-safe default values, you can also use the step-level API:

```go
baseStep := workflow.NewStep("process", "Process", handler)
defaultOutput := &ProcessOutput{Status: "skipped"}
conditionalStep := workflow.NewConditionalStep(baseStep, condition, defaultOutput)

builder.ThenStep(conditionalStep)
```

Both approaches are equivalent - choose based on your needs!
