# Parallel Execution

Execute multiple independent steps concurrently to improve workflow performance.

## Overview

Parallel execution allows multiple steps to run simultaneously when they don't depend on each other's outputs. This can significantly reduce total workflow execution time.

## Basic Parallel Execution

Use the `Parallel()` builder method:

```go
wf, _ := gorkflow.NewWorkflow("parallel-example", "Parallel Workflow").
    ThenStep(prepareStep).     // Runs first
    Parallel(                   // These run simultaneously
        processA,
        processB,
        processC,
    ).
    ThenStep(aggregateStep).   // Runs after all parallel steps complete
    Build()
```

**Execution Flow:**

```
prepareStep
    ├─→ processA ─┐
    ├─→ processB ─┼─→ aggregateStep
    └─→ processC ─┘
```

## Complete Example

```go
package main

import (
    "fmt"
    "time"
    "github.com/sicko7947/gorkflow"
)

type DataInput struct {
    ID int `json:"id"`
}

type ProcessedA struct {
    ResultA string `json:"resultA"`
}

type ProcessedB struct {
    ResultB string `json:"resultB"`
}

type ProcessedC struct {
    ResultC string `json:"resultC"`
}

type AggregatedOutput struct {
    Combined string `json:"combined"`
}

func main() {
    // Create parallel steps
    processA := gorkflow.NewStep(
        "process-a",
        "Process A",
        func(ctx *gorkflow.StepContext, input DataInput) (ProcessedA, error) {
            time.Sleep(1 * time.Second)  // Simulate work
            return ProcessedA{ResultA: fmt.Sprintf("A-%d", input.ID)}, nil
        },
    )

    processB := gorkflow.NewStep(
        "process-b",
        "Process B",
        func(ctx *gorkflow.StepContext, input DataInput) (ProcessedB, error) {
            time.Sleep(1 * time.Second)  // Simulate work
            return ProcessedB{ResultB: fmt.Sprintf("B-%d", input.ID)}, nil
        },
    )

    processC := gorkflow.NewStep(
        "process-c",
        "Process C",
        func(ctx *gorkflow.StepContext, input DataInput) (ProcessedC, error) {
            time.Sleep(1 * time.Second)  // Simulate work
            return ProcessedC{ResultC: fmt.Sprintf("C-%d", input.ID)}, nil
        },
    )

    // Aggregate results
    aggregate := gorkflow.NewStep(
        "aggregate",
        "Aggregate Results",
        func(ctx *gorkflow.StepContext, input DataInput) (AggregatedOutput, error) {
            // Get outputs from all parallel steps
            var a ProcessedA
            var b ProcessedB
            var c ProcessedC

            ctx.Outputs.Get(ctx.Context, "process-a", &a)
            ctx.Outputs.Get(ctx.Context, "process-b", &b)
            ctx.Outputs.Get(ctx.Context, "process-c", &c)

            combined := fmt.Sprintf("%s|%s|%s", a.ResultA, b.ResultB, c.ResultC)
            return AggregatedOutput{Combined: combined}, nil
        },
    )

    // Build workflow with parallel execution
    wf, _ := gorkflow.NewWorkflow("parallel-wf", "Parallel Workflow").
        Parallel(processA, processB, processC).
        ThenStep(aggregate).
        Build()
}
```

**Performance:**

- **Sequential**: 3 seconds (1 + 1 + 1)
- **Parallel**: ~1 second (max of 1, 1, 1)

## Use Cases

### Fan-Out / Fan-In

Distribute work across multiple processors, then aggregate:

```go
wf, _ := gorkflow.NewWorkflow("fan-out-in", "Fan Out/In").
    ThenStep(prepareData).    // Prepare data
    Parallel(                  // Fan-out: Process in parallel
        processRegionA,
        processRegionB,
        processRegionC,
    ).
    ThenStep(aggregateResults). // Fan-in: Combine results
    Build()
```

### Independent Operations

Execute unrelated operations simultaneously:

```go
wf, _ := gorkflow.NewWorkflow("independent", "Independent Operations").
    ThenStep(validateInput).
    Parallel(
        sendEmail,           // Send notification
        updateDatabase,      // Save to DB
        callWebhook,         // Trigger webhook
        generateReport,      // Create report
    ).
    ThenStep(finalize).
    Build()
```

### Multi-Source Data Fetching

Fetch from multiple sources in parallel:

```go
wf, _ := gorkflow.NewWorkflow("multi-fetch", "Multi-Source Fetch").
    Parallel(
        fetchFromAPI1,
        fetchFromAPI2,
        fetchFromDatabase,
        fetchFromCache,
    ).
    ThenStep(mergeData).
    Build()
```

## Accessing Parallel Step Outputs

In steps that follow parallel execution, access outputs from all parallel steps:

```go
aggregateStep := gorkflow.NewStep(
    "aggregate",
    "Aggregate Results",
    func(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
        // Get outputs from each parallel step
        var stepA StepAOutput
        var stepB StepBOutput
        var stepC StepCOutput

        if err := ctx.Outputs.Get(ctx.Context, "step-a", &stepA); err != nil {
            return MyOutput{}, err
        }
        if err := ctx.Outputs.Get(ctx.Context, "step-b", &stepB); err != nil {
            return MyOutput{}, err
        }
        if err := ctx.Outputs.Get(ctx.Context, "step-c", &stepC); err != nil {
            return MyOutput{}, err
        }

        // Combine results
        return MyOutput{
            Total: stepA.Value + stepB.Value + stepC.Value,
        }, nil
    },
)
```

## Chaining Parallel Blocks

You can have multiple parallel blocks:

```go
wf, _ := gorkflow.NewWorkflow("multi-parallel", "Multi-Parallel").
    ThenStep(prepare).
    Parallel(group1StepA, group1StepB).  // First parallel group
    ThenStep(intermediate).
    Parallel(group2StepA, group2StepB student_OFFSET2StepC).  // Second parallel group
    ThenStep(finalize).
    Build()
```

**Execution Flow:**

```
prepare
  ├─→ group1StepA ─┐
  └─→ group1StepB ─┴─→ intermediate
                          ├─→ group2StepA ─┐
                          ├─→ group2StepB ─┼─→ finalize
                          └─→ group2StepC ─┘
```

## Error Handling in Parallel Steps

### Default Behavior

If any parallel step fails, the entire workflow fails:

```go
wf, _ := gorkflow.NewWorkflow("parallel-wf", "Parallel").
    Parallel(stepA, stepB, stepC).  // If stepB fails, workflow fails
    Build()
```

### Continue on Error

Allow workflow to continue even if some parallel steps fail:

```go
wf, _ := gorkflow.NewWorkflow("parallel-wf", "Parallel").
    WithConfig(gorkflow.ExecutionConfig{
        ContinueOnError: true,  // Continue despite failures
    }).
    Parallel(stepA, stepB, stepC).
    Build()
```

Check which steps succeeded in the aggregation step:

```go
func aggregate(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    var a AOutput
    errA := ctx.Outputs.Get(ctx.Context, "step-a", &a)

    var b BOutput
    errB := ctx.Outputs.Get(ctx.Context, "step-b", &b)

    // Handle partial success
    if errA != nil && errB != nil {
        return MyOutput{}, errors.New("all parallel steps failed")
    }

    // Use available data
    // ...
}
```

## Timeouts for Parallel Steps

Set different timeouts for each parallel step:

```go
quickStep := gorkflow.NewStep("quick", "Quick", handler,
    gorkflow.WithTimeout(5 * time.Second),
)

slowStep := gorkflow.NewStep("slow", "Slow", handler,
    gorkflow.WithTimeout(60 * time.Second),
)

wf, _ := gorkflow.NewWorkflow("parallel-wf", "Parallel").
    Parallel(quickStep, slowStep).  // Different timeouts
    Build()
```

## Performance Considerations

### Concurrency Limits

The engine handles concurrency automatically, but be mindful of:

- **External API rate limits**
- **Database connection pools**
- **Memory usage**
- **CPU cores**

### Balancing Work

Try to balance work across parallel steps:

✅ **Good** - Balanced workload:

```go
.Parallel(
    processChunk1,  // ~10 seconds
    processChunk2,  // ~10 seconds
    processChunk3,  // ~10 seconds
)
// Total: ~10 seconds
```

❌ **Bad** - Unbalanced workload:

```go
.Parallel(
    quickTask,      // ~1 second
    mediumTask,     // ~5 seconds
    longTask,       // ~30 seconds
)
// Total: ~30 seconds (faster steps wait for slowest)
```

## Best Practices

### 1. Use Parallel for Independent Operations

✅ **Good** - Independent operations:

```go
.Parallel(
    sendEmail,        // No dependencies
    updateDatabase,   // No dependencies
    callWebhook,      // No dependencies
)
```

❌ **Bad** - Dependent operations:

```go
.Parallel(
    validateUser,     // Must run first
    createUser,       // Depends on validation
)
```

### 2. Set Appropriate Timeouts

```go
apiCall := gorkflow.NewStep("api", "API Call", handler,
    gorkflow.WithTimeout(30 * time.Second),
    gorkflow.WithRetries(3),
)

dbWrite := gorkflow.NewStep("db", "DB Write", handler,
    gorkflow.WithTimeout(10 * time.Second),
    gorkflow.WithRetries(2),
)

.Parallel(apiCall, dbWrite)
```

### 3. Handle Partial Failures Gracefully

```go
func aggregate(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    results := []string{}

    // Try to get each result, skip failures
    for _, stepID := range []string{"step-a", "step-b", "step-c"} {
        var result StepResult
        if err := ctx.Outputs.Get(ctx.Context, stepID, &result); err == nil {
            results = append(results, result.Data)
        }
    }

    return MyOutput{Results: results}, nil
}
```

### 4. Monitor Parallel Execution

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    start := time.Now()

    // Do work...

    ctx.Logger.Info().
        Dur("duration", time.Since(start)).
        Msg("Parallel step completed")

    return output, nil
}
```

---

**Next**: Learn about [Conditional Execution](conditional-execution.md) →
