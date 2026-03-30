# Debugging Workflows

This guide covers techniques for diagnosing issues in Gorkflow workflows.

## Inspecting Run Status

### Check Workflow Run

```go
run, err := eng.GetRun(ctx, runID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Status: %s\n", run.Status)
fmt.Printf("Progress: %.0f%%\n", run.Progress*100)
fmt.Printf("Created: %s\n", run.CreatedAt)

if run.StartedAt != nil {
    fmt.Printf("Started: %s\n", *run.StartedAt)
}
if run.CompletedAt != nil {
    fmt.Printf("Completed: %s\n", *run.CompletedAt)
}
if run.Error != nil {
    fmt.Printf("Error: [%s] %s\n", run.Error.Code, run.Error.Message)
    if run.Error.Step != "" {
        fmt.Printf("Failed Step: %s\n", run.Error.Step)
    }
}
```

### Check Step Executions

```go
executions, err := eng.GetStepExecutions(ctx, runID)
if err != nil {
    log.Fatal(err)
}

for _, exec := range executions {
    fmt.Printf("Step: %s | Status: %s | Attempt: %d | Duration: %dms\n",
        exec.StepID, exec.Status, exec.Attempt, exec.DurationMs)

    if exec.Error != nil {
        fmt.Printf("  Error: [%s] %s\n", exec.Error.Code, exec.Error.Message)
    }
}
```

### List Runs by Filter

```go
failedStatus := gorkflow.RunStatusFailed
runs, err := eng.ListRuns(ctx, gorkflow.RunFilter{
    WorkflowID: "my-workflow",
    Status:     &failedStatus,
    Limit:      10,
})

for _, run := range runs {
    fmt.Printf("Run %s: %s at %s\n", run.RunID, run.Status, run.CreatedAt)
}
```

## Log Levels

Gorkflow uses `zerolog` with structured events. Adjust the log level for different levels of detail.

### Setting Log Level

```go
import "github.com/rs/zerolog"

// Debug level — shows execution order, progress updates, all events
logger := zerolog.New(os.Stdout).With().Timestamp().Logger().Level(zerolog.DebugLevel)

eng := engine.NewEngine(store, engine.WithLogger(logger))
```

### Log Events by Level

**Info** (default):
| Event | Description |
|-------|-------------|
| `workflow_created` | New workflow run created |
| `workflow_started` | Workflow execution began |
| `workflow_completed` | Workflow finished successfully |
| `step_started` | Step execution began (includes step number and total) |
| `step_completed` | Step finished (includes duration and attempt count) |
| `step_skipped` | Conditional step was skipped |

**Warn**:
| Event | Description |
|-------|-------------|
| `step_retrying` | Step failed and is being retried (includes attempt and delay) |
| `workflow_cancelled` | Workflow was cancelled |

**Error**:
| Event | Description |
|-------|-------------|
| `step_failed` | Step failed (includes error, attempt, duration) |
| `workflow_failed` | Workflow failed |
| `persistence_error` | Store operation failed |

**Debug**:
| Event | Description |
|-------|-------------|
| `workflow_progress` | Progress percentage updated |
| Execution order | Topological sort result logged at workflow start |

### Structured Log Fields

All log events include contextual fields:

```json
{
    "event": "step_completed",
    "run_id": "abc-123",
    "step_id": "process-data",
    "step_name": "Process Data",
    "duration_ms": 1523,
    "attempts": 2
}
```

### In-Handler Logging

Use `ctx.Logger` within step handlers. It's pre-configured with the run ID, step ID, and step name:

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    ctx.Logger.Debug().
        Str("input_id", input.ID).
        Msg("Processing started")

    // ... do work ...

    ctx.Logger.Info().
        Int("items_processed", count).
        Msg("Processing complete")

    return output, nil
}
```

## Common Debugging Scenarios

### Why did my workflow fail?

1. Get the run and check the error:
   ```go
   run, _ := eng.GetRun(ctx, runID)
   fmt.Println(run.Error.Code, run.Error.Message)
   ```

2. Check which step failed:
   ```go
   execs, _ := eng.GetStepExecutions(ctx, runID)
   for _, e := range execs {
       if e.Status == gorkflow.StepStatusFailed {
           fmt.Printf("Step %s failed: %s\n", e.StepID, e.Error.Message)
       }
   }
   ```

### Why is my step retrying?

Enable debug logging and look for `step_retrying` events. Each retry logs the attempt number and delay:

```
WARN step_retrying run_id=abc step_id=api-call attempt=2 delay=2s
```

The step's error from each attempt is logged as `step_failed`.

### Why was my step skipped?

A step is skipped when its condition evaluates to `false`. Look for `step_skipped` events:

```
INFO step_skipped run_id=abc step_id=optional-step reason=condition_not_met
```

### Why is my workflow stuck at PENDING/RUNNING?

For **async workflows** (default), the workflow runs in a background goroutine. If the process exits before the workflow completes, the run stays in `RUNNING`. Use `WithSynchronousExecution()` for testing.

For **sync workflows**, check for:
- Steps with very long timeouts
- External calls that are hanging
- Missing context cancellation

### Persistence errors

Persistence errors are logged but don't always fail the workflow (progress updates, for example). Look for `persistence_error` events:

```
ERROR persistence_error run_id=abc operation=update_run_progress error="connection refused"
```

## Debugging Tips

1. **Use sync execution for testing** — `WithSynchronousExecution()` blocks until the workflow finishes, making errors immediately visible.

2. **Use MemoryStore for tests** — eliminates database-related issues.

3. **Enable debug logging** — shows execution order, progress, and all events.

4. **Check step outputs** — use `GetStepExecutions` to see the input/output of each step.

5. **Validate early** — use struct validation tags to catch input issues before handler logic runs.

---

**See also**: [Common Issues](common-issues.md) | [FAQ](faq.md)
