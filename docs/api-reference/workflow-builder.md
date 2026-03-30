# Workflow Builder API

The `WorkflowBuilder` provides a fluent API for constructing workflows with chained steps, parallel execution, and conditional logic.

## Creating a Builder

### `NewWorkflow`

```go
func NewWorkflow(id, name string) *WorkflowBuilder
```

Creates a new workflow builder with the given ID and name.

```go
wf, err := gorkflow.NewWorkflow("my-workflow", "My Workflow").
    ThenStep(step1).
    ThenStep(step2).
    Build()
```

## Builder Methods

All builder methods return `*WorkflowBuilder` for chaining (except `Build` and `MustBuild`).

### `WithDescription`

```go
func (b *WorkflowBuilder) WithDescription(description string) *WorkflowBuilder
```

Sets a human-readable description for the workflow.

```go
wf, _ := gorkflow.NewWorkflow("etl", "ETL Pipeline").
    WithDescription("Extracts, transforms, and loads user data").
    ThenStep(extractStep).
    Build()
```

### `WithVersion`

```go
func (b *WorkflowBuilder) WithVersion(version string) *WorkflowBuilder
```

Sets the workflow version. This is stored in `WorkflowRun.WorkflowVersion` for observability.

```go
wf, _ := gorkflow.NewWorkflow("etl", "ETL Pipeline").
    WithVersion("2.1.0").
    ThenStep(extractStep).
    Build()
```

### `WithConfig`

```go
func (b *WorkflowBuilder) WithConfig(config ExecutionConfig) *WorkflowBuilder
```

Sets the default execution configuration for all steps in the workflow. Individual step configurations override these defaults.

```go
wf, _ := gorkflow.NewWorkflow("resilient", "Resilient Workflow").
    WithConfig(gorkflow.ExecutionConfig{
        MaxRetries:      5,
        RetryDelayMs:    2000,
        RetryBackoff:    gorkflow.BackoffExponential,
        TimeoutSeconds:  60,
        ContinueOnError: false,
    }).
    ThenStep(step1).
    Build()
```

### `WithTags`

```go
func (b *WorkflowBuilder) WithTags(tags map[string]string) *WorkflowBuilder
```

Sets metadata tags on the workflow. Tags are persisted with each workflow run.

```go
wf, _ := gorkflow.NewWorkflow("process", "Process").
    WithTags(map[string]string{
        "team":        "platform",
        "environment": "production",
    }).
    ThenStep(step1).
    Build()
```

### `WithContext`

```go
func (b *WorkflowBuilder) WithContext(ctx any) *WorkflowBuilder
```

Sets a custom context value accessible in all step handlers via `ctx.CustomContext` or the type-safe `GetContext[T]` helper.

```go
type AppContext struct {
    DB     *sql.DB
    Config *AppConfig
}

appCtx := &AppContext{DB: db, Config: cfg}

wf, _ := gorkflow.NewWorkflow("app", "App Workflow").
    WithContext(appCtx).
    ThenStep(step1).
    Build()
```

## Step Chaining Methods

### `ThenStep`

```go
func (b *WorkflowBuilder) ThenStep(step StepExecutor) *WorkflowBuilder
```

Adds a step that executes sequentially after the previous step(s). The first call to `ThenStep` sets the entry point. Each subsequent call chains the new step after all current "last" steps.

```go
wf, _ := gorkflow.NewWorkflow("pipeline", "Pipeline").
    ThenStep(step1).  // Entry point
    ThenStep(step2).  // Runs after step1
    ThenStep(step3).  // Runs after step2
    Build()
```

### `Parallel`

```go
func (b *WorkflowBuilder) Parallel(steps ...StepExecutor) *WorkflowBuilder
```

Adds multiple steps that execute in parallel after the previous step(s). All parallel steps must complete before the next `ThenStep` can run.

```go
wf, _ := gorkflow.NewWorkflow("fan-out", "Fan Out").
    ThenStep(setupStep).
    Parallel(taskA, taskB, taskC).  // All run after setupStep
    ThenStep(aggregateStep).        // Runs after all parallel steps complete
    Build()
```

### `Sequence`

```go
func (b *WorkflowBuilder) Sequence(steps ...StepExecutor) *WorkflowBuilder
```

Adds multiple steps and chains them together sequentially. This is a convenience method equivalent to calling `ThenStep` for each step.

```go
// These are equivalent:
wf, _ := gorkflow.NewWorkflow("seq", "Sequential").
    Sequence(step1, step2, step3).
    Build()

wf, _ := gorkflow.NewWorkflow("seq", "Sequential").
    ThenStep(step1).
    ThenStep(step2).
    ThenStep(step3).
    Build()
```

### `ThenStepIf`

```go
func (b *WorkflowBuilder) ThenStepIf(step StepExecutor, condition Condition, defaultValue any) *WorkflowBuilder
```

Chains a conditional step that only executes if the condition returns `true` at runtime.

- `condition` — `func(ctx *StepContext) (bool, error)` evaluated before the step runs
- `defaultValue` — output used when the step is skipped; pass `nil` for the zero value

```go
shouldProcess := func(ctx *gorkflow.StepContext) (bool, error) {
    var flag bool
    ctx.State.Get("should_process", &flag)
    return flag, nil
}

wf, _ := gorkflow.NewWorkflow("conditional", "Conditional").
    ThenStep(setupStep).
    ThenStepIf(processStep, shouldProcess, nil).
    ThenStep(finalStep).
    Build()
```

See [Conditional Execution](../advanced-usage/conditional-execution.md) for detailed examples.

### `SetEntryPoint`

```go
func (b *WorkflowBuilder) SetEntryPoint(stepID string) *WorkflowBuilder
```

Explicitly sets which step the workflow starts from. This is rarely needed since the first `ThenStep` call automatically sets the entry point.

```go
wf, _ := gorkflow.NewWorkflow("explicit", "Explicit Entry").
    ThenStep(step1).
    ThenStep(step2).
    SetEntryPoint("step1").
    Build()
```

## Build Methods

### `Build`

```go
func (b *WorkflowBuilder) Build() (*Workflow, error)
```

Finalizes and validates the workflow. Returns an error if:

- The execution graph is invalid (cycles, missing entry point)
- A step referenced in the graph is not registered

During build, workflow-level config is propagated to any step that still uses `DefaultExecutionConfig`.

```go
wf, err := gorkflow.NewWorkflow("my-wf", "My Workflow").
    ThenStep(step1).
    Build()
if err != nil {
    log.Fatalf("invalid workflow: %v", err)
}
```

### `MustBuild`

```go
func (b *WorkflowBuilder) MustBuild() *Workflow
```

Same as `Build` but panics on error. Useful for package-level workflow definitions.

```go
var myWorkflow = gorkflow.NewWorkflow("my-wf", "My Workflow").
    ThenStep(step1).
    ThenStep(step2).
    MustBuild()
```

## Complete Example

```go
package main

import (
    "time"

    "github.com/sicko7947/gorkflow"
)

func main() {
    wf, err := gorkflow.NewWorkflow("order-processing", "Order Processing").
        WithDescription("Processes incoming orders end-to-end").
        WithVersion("1.0.0").
        WithTags(map[string]string{"domain": "orders"}).
        WithConfig(gorkflow.ExecutionConfig{
            MaxRetries:     3,
            RetryDelayMs:   1000,
            RetryBackoff:   gorkflow.BackoffExponential,
            TimeoutSeconds: 60,
        }).
        ThenStep(validateStep).
        Parallel(checkInventory, calculateShipping).
        ThenStep(chargePayment).
        ThenStepIf(sendNotification, shouldNotify, nil).
        Build()
    if err != nil {
        panic(err)
    }

    // Use with engine...
}
```

---

**Next**: Learn about the [Step API](step-api.md) →
