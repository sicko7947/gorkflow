# Error Handling

Gorkflow provides structured error types, error codes, and configurable error behavior for workflows and steps.

## Error Types

### `WorkflowError`

Represents an error during workflow execution. Stored on `WorkflowRun.Error` when a workflow fails.

```go
type WorkflowError struct {
    Message   string                 `json:"message"`
    Code      string                 `json:"code"`
    Step      string                 `json:"step,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
    Details   map[string]interface{} `json:"details,omitempty"`
}
```

Create workflow errors:

```go
// Basic error
err := gorkflow.NewWorkflowError(gorkflow.ErrCodeValidation, "invalid input data")

// With step context
err := gorkflow.NewWorkflowErrorWithStep(
    gorkflow.ErrCodeExecutionFailed,
    "step handler returned error",
    "process-step",
)

// With additional details
err := gorkflow.NewWorkflowError(gorkflow.ErrCodeTimeout, "workflow timed out").
    WithDetails(map[string]interface{}{
        "elapsed_seconds": 300,
        "step":            "long-running-task",
    })
```

Error format: `[CODE] message (step: step-id)`

### `StepError`

Represents an error during step execution. Stored on `StepExecution.Error` when a step fails.

```go
type StepError struct {
    Message   string                 `json:"message"`
    Code      string                 `json:"code"`
    Timestamp time.Time              `json:"timestamp"`
    Attempt   int                    `json:"attempt"`
    Details   map[string]interface{} `json:"details,omitempty"`
}
```

Create step errors:

```go
err := gorkflow.NewStepError(gorkflow.ErrCodeExecutionFailed, "database connection lost", 2)

err := gorkflow.NewStepError(gorkflow.ErrCodeTimeout, "step timed out", 0).
    WithDetails(map[string]interface{}{
        "timeout_seconds": 30,
    })
```

Error format: `[CODE] message (attempt: N)`

## Error Codes

| Constant | Value | Description |
|----------|-------|-------------|
| `ErrCodeValidation` | `"VALIDATION_ERROR"` | Input or output validation failed |
| `ErrCodeNotFound` | `"NOT_FOUND"` | Resource not found |
| `ErrCodeTimeout` | `"TIMEOUT"` | Step or workflow timed out |
| `ErrCodeConcurrency` | `"CONCURRENCY_LIMIT"` | Concurrency limit exceeded |
| `ErrCodeExecutionFailed` | `"EXECUTION_FAILED"` | Step handler returned an error |
| `ErrCodeCancelled` | `"CANCELLED"` | Workflow was cancelled |
| `ErrCodePanic` | `"PANIC"` | Step handler panicked |
| `ErrCodeInternalError` | `"INTERNAL_ERROR"` | Internal engine error |

## Sentinel Errors

```go
var (
    ErrStepSkipped           = errors.New("step skipped")
    ErrRunNotFound           = errors.New("workflow run not found")
    ErrStepExecutionNotFound = errors.New("step execution not found")
    ErrStepOutputNotFound    = errors.New("step output not found")
    ErrStateNotFound         = errors.New("state not found")
)
```

### `ErrStepSkipped`

Returned by a conditional step wrapper when the condition evaluates to `false` and the input/output types differ. The engine treats this as a skip (not a failure) and sets the step status to `SKIPPED`.

## Error Helpers

### `IsTimeoutError`

```go
func IsTimeoutError(err error) bool
```

Returns `true` if the error is a timeout error. Checks `StepError`/`WorkflowError` codes and error message content.

```go
if gorkflow.IsTimeoutError(err) {
    log.Println("Operation timed out, consider increasing timeout")
}
```

### `IsConcurrencyError`

```go
func IsConcurrencyError(err error) bool
```

Returns `true` if the error is a concurrency limit error.

## Error Propagation

### Default Behavior

By default, when a step fails (after exhausting all retries), the entire workflow fails:

```
Step fails → retries exhausted → workflow status = FAILED
```

The `WorkflowRun.Error` is set with `ErrCodeExecutionFailed` and the step's error message.

### ContinueOnError

When `ContinueOnError` is enabled for a step, the workflow continues to the next step even if this step fails:

```go
// Step-level
optionalStep := gorkflow.NewStep("optional", "Optional", handler,
    gorkflow.WithContinueOnError(true),
)

// Workflow-level (applies to all steps with default config)
wf, _ := gorkflow.NewWorkflow("resilient", "Resilient").
    WithConfig(gorkflow.ExecutionConfig{
        ContinueOnError: true,
    }).
    ThenStep(step1).
    ThenStep(step2).
    Build()
```

When a step with `ContinueOnError` fails:
- The step status is set to `FAILED`
- The workflow continues to the next step
- Downstream steps receive `null` as input (zero value for their input type)
- The workflow can still complete successfully

### Conditional Step Skip

When a conditional step's condition evaluates to `false`:

1. The step status is set to `SKIPPED`
2. If a default value was provided, it becomes the step's output
3. If input and output types match, the input is passed through
4. Otherwise, a zero value is used and `ErrStepSkipped` is returned
5. The workflow continues to the next step

### Panic Recovery

If a step handler panics, the engine recovers the panic and treats it as an error:

```go
step := gorkflow.NewStep("risky", "Risky", func(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    panic("something went wrong") // Recovered by the engine
})
```

The panic is converted to an error (`"step panicked: something went wrong"`) and enters the normal retry flow.

## Error Handling Patterns

### Wrapping Errors

```go
step := gorkflow.NewStep("fetch", "Fetch Data",
    func(ctx *gorkflow.StepContext, input FetchInput) (FetchOutput, error) {
        data, err := apiClient.Get(input.URL)
        if err != nil {
            return FetchOutput{}, fmt.Errorf("failed to fetch from %s: %w", input.URL, err)
        }
        return FetchOutput{Data: data}, nil
    },
)
```

### Distinguishing Retriable vs Non-Retriable Errors

All errors trigger retries. If you want to fail immediately without retrying, set `WithRetries(0)` or design your handler to return success with an error status in the output:

```go
step := gorkflow.NewStep("validate", "Validate",
    func(ctx *gorkflow.StepContext, input Input) (Output, error) {
        if input.Email == "" {
            // Validation errors shouldn't be retried
            return Output{}, fmt.Errorf("validation failed: email is required")
        }
        return Output{Valid: true}, nil
    },
    gorkflow.WithRetries(0), // Don't retry validation
)
```

### Checking Run Errors

```go
run, err := eng.GetRun(ctx, runID)
if err != nil {
    log.Fatal(err)
}

if run.Status == gorkflow.RunStatusFailed {
    fmt.Printf("Workflow failed: %s (code: %s)\n", run.Error.Message, run.Error.Code)
    if run.Error.Step != "" {
        fmt.Printf("  Failed at step: %s\n", run.Error.Step)
    }
}
```

### Inspecting Step Errors

```go
executions, _ := eng.GetStepExecutions(ctx, runID)
for _, exec := range executions {
    if exec.Status == gorkflow.StepStatusFailed && exec.Error != nil {
        fmt.Printf("Step %s failed on attempt %d: %s\n",
            exec.StepID, exec.Error.Attempt, exec.Error.Message)
    }
}
```

---

**Next**: Learn about [Cancellation](cancellation.md) →
