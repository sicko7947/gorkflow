# Workflows

Understanding workflow definitions and how they orchestrate step execution.

## What is a Workflow?

A workflow is a directed acyclic graph (DAG) of steps that execute in a specific order. Each workflow:

- Has a unique ID and name
- Contains one or more steps
- Defines the execution order and dependencies
- Can execute steps sequentially, in parallel, or conditionally
- Maintains state throughout execution

## Creating a Workflow

Use the fluent `WorkflowBuilder` API:

```go
wf, err := gorkflow.NewWorkflow("my-workflow", "My Workflow").
    WithDescription("Process user data").
    WithVersion("1.0").
    Sequence(step1, step2, step3).
    Build()
```

## Workflow Properties

### Required Properties

```go
gorkflow.NewWorkflow(
    "workflow-id",    // Unique identifier
    "Workflow Name",  // Human-readable name
)
```

### Optional Properties

```go
.WithDescription("Detailed description of what this workflow does")
.WithVersion("1.0.0")
.WithTags(map[string]string{
    "team":        "backend",
    "environment": "production",
})
.WithConfig(gorkflow.ExecutionConfig{
    MaxRetries:     3,
    RetryDelayMs:   1000,
    TimeoutSeconds: 300,
})
.WithContext(customContextStruct)
```

## Building Workflows

### Sequential Execution

Steps execute one after another:

```go
wf, _ := gorkflow.NewWorkflow("seq", "Sequential").
    Sequence(
        step1,  // Runs first
        step2,  // Runs after step1
        step3,  // Runs after step2
    ).
    Build()
```

**Execution Order**: step1 → step2 → step3

### Chain Execution

Alternative syntax using `ThenStep`:

```go
wf, _ := gorkflow.NewWorkflow("chain", "Chain").
    ThenStep(step1).
    ThenStep(step2).
    ThenStep(step3).
    Build()
```

**Execution Order**: step1 → step2 → step3

### Parallel Execution

Independent steps run concurrently:

```go
wf, _ := gorkflow.NewWorkflow("parallel", "Parallel").
    ThenStep(step1).
    Parallel(
        step2,  // Runs in parallel after step1
        step3,  // Runs in parallel after step1
        step4,  // Runs in parallel after step1
    ).
    ThenStep(step5).  // Runs after all parallel steps complete
    Build()
```

**Execution Order**:

```
step1
  ├─→ step2 ─┐
  ├─→ step3 ─┼─→ step5
  └─→ step4 ─┘
```

### Conditional Execution

Steps execute only if a condition is met:

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var shouldProcess bool
    ctx.State.Get("should_process", &shouldProcess)
    return shouldProcess, nil
}

wf, _ := gorkflow.NewWorkflow("conditional", "Conditional").
    ThenStep(setupStep).
    ThenStepIf(processStep, condition, nil).  // Conditional
    ThenStep(finalStep).
    Build()
```

See [Conditional Execution](../advanced-usage/conditional-execution.md) for details.

## Workflow Configuration

### Default Configuration

Workflows have default execution configuration:

```go
gorkflow.ExecutionConfig{
    MaxRetries:      3,
    RetryDelayMs:    1000,
    RetryBackoff:    gorkflow.BackoffLinear,
    TimeoutSeconds:  300,
    ContinueOnError: false,
}
```

### Custom Configuration

Override defaults for the entire workflow:

```go
wf, _ := gorkflow.NewWorkflow("my-wf", "My Workflow").
    WithConfig(gorkflow.ExecutionConfig{
        MaxRetries:      5,
        RetryDelayMs:    2000,
        RetryBackoff:    gorkflow.BackoffExponential,
        TimeoutSeconds:  600,
        ContinueOnError: true,
    }).
    Sequence(step1, step2).
    Build()
```

Individual steps can override workflow configuration:

```go
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithRetries(10),  // Overrides workflow MaxRetries
)
```

## Workflow Lifecycle

```
┌─────────────┐
│   Created   │ ← Build() called
└──────┬──────┘
       │
       ↓
┌─────────────┐
│   Pending   │ ← StartWorkflow() called
└──────┬──────┘
       │
       ↓
┌─────────────┐
│   Running   │ ← Steps executing
└──────┬──────┘
       │
       ├────────→ ┌────────────┐
       │          │  Cancelled │
       │          └────────────┘
       │
       ├────────→ ┌────────────┐
       │          │   Failed   │
       │          └────────────┘
       │
       ↓
┌─────────────┐
│  Completed  │
└─────────────┘
```

## Workflow Metadata

### Tags

Add metadata for filtering and organizing:

```go
wf, _ := gorkflow.NewWorkflow("my-wf", "My Workflow").
    WithTags(map[string]string{
        "team":        "platform",
        "environment": "production",
        "version":     "2.0",
    }).
    Build()
```

Use tags when starting workflows:

```go
runID, _ := eng.StartWorkflow(ctx, wf, input,
    gorkflow.WithTags(map[string]string{
        "user_id": "12345",
        "request_id": "abc-def",
    }),
)
```

### Custom Context

Pass custom data accessible in all steps:

```go
type MyContext struct {
    UserID  string
    TraceID string
    Config  AppConfig
}

wf, _ := gorkflow.NewWorkflow("my-wf", "My Workflow").
    WithContext(MyContext{
        UserID:  "user123",
        TraceID: "trace456",
    }).
    Build()
```

Access in steps:

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    myCtx, err := gorkflow.GetContext[MyContext](ctx)
    if err != nil {
        return MyOutput{}, err
    }
    ctx.Logger.Info().Str("user_id", myCtx.UserID).Msg("Processing")
    // ...
}
```

See [Context](context.md) for details.

## Best Practices

### 1. Use Descriptive IDs and Names

✅ **Good**:

```go
gorkflow.NewWorkflow("user-registration-v2", "User Registration Workflow")
```

❌ **Bad**:

```go
gorkflow.NewWorkflow("wf1", "Workflow")
```

### 2. Set Appropriate Timeouts

```go
.WithConfig(gorkflow.ExecutionConfig{
    TimeoutSeconds: 300,  // 5 minutes for the entire workflow
})
```

### 3. Use Workflow-Level Defaults

Set defaults at the workflow level, override in specific steps:

```go
wf, _ := gorkflow.NewWorkflow("my-wf", "My Workflow").
    WithConfig(gorkflow.ExecutionConfig{
        MaxRetries: 3,  // Default for all steps
    }).
    Sequence(
        step1,  // Uses 3 retries
        gorkflow.NewStep("critical", "Critical", handler,
            gorkflow.WithRetries(10),  // Override to 10 retries
        ),
    ).
    Build()
```

### 4. Version Your Workflows

```go
.WithVersion("2.1.0")
```

Track changes and maintain backward compatibility.

### 5. Add Tags for Organization

```go
.WithTags(map[string]string{
    "team": "payments",
    "sla": "high",
})
```

## Advanced Patterns

### Fan-Out/Fan-In

Distribute work across parallel steps, then aggregate:

```go
wf, _ := gorkflow.NewWorkflow("fan-out-in", "Fan Out/In").
    ThenStep(prepareStep).
    Parallel(
        processA,
        processB,
        processC,
    ).
    ThenStep(aggregateStep).  // Aggregates all parallel results
    Build()
```

### Conditional Branching

Different paths based on runtime data:

```go
wf, _ := gorkflow.NewWorkflow("branching", "Branching").
    ThenStep(checkStep).
    ThenStepIf(pathA, conditionA, nil).
    ThenStepIf(pathB, conditionB, nil).
    ThenStep(mergeStep).
    Build()
```

### Multi-Stage Processing

Group steps into logical stages:

```go
wf, _ := gorkflow.NewWorkflow("multi-stage", "Multi-Stage").
    // Stage 1: Validation
    Sequence(validateInput, checkPermissions).
    // Stage 2: Processing (parallel)
    Parallel(processData, generateReport, sendNotification).
    // Stage 3: Finalization
    Sequence(saveResults, cleanup).
    Build()
```

## Validation

Workflows are validated during `Build()`:

```go
wf, err := builder.Build()
if err != nil {
    // Validation errors:
    // - Cycle detection (no DAG cycles allowed)
    // - Missing steps
    // - Invalid entry points
    log.Fatal(err)
}
```

## Error Handling

### Workflow-Level Error Handling

```go
.WithConfig(gorkflow.ExecutionConfig{
    ContinueOnError: true,  // Continue even if steps fail
})
```

### Check Workflow Status

```go
run, _ := eng.GetRun(ctx, runID)

switch run.Status {
case gorkflow.RunStatusCompleted:
    // Success
case gorkflow.RunStatusFailed:
    fmt.Println("Error:", run.Error.Message)
case gorkflow.RunStatusCancelled:
    // Cancelled by user
}
```

---

**Next**: Learn about [Steps](steps.md), the building blocks of workflows →
