# Cancellation

Gorkflow supports cancelling running workflows through the engine API and Go context propagation.

## Cancelling a Workflow

### Using `Engine.Cancel`

```go
err := eng.Cancel(ctx, runID)
if err != nil {
    log.Printf("Could not cancel: %v", err)
}
```

`Cancel` retrieves the run, checks that it is not already in a terminal state (`COMPLETED`, `FAILED`, or `CANCELLED`), and marks it as `CANCELLED`.

If the workflow is already completed, failed, or cancelled, `Cancel` returns an error:

```go
// Returns: "cannot cancel workflow in COMPLETED state"
```

### Using Context Cancellation (Async Workflows)

For async workflows (the default), the engine launches execution in a background goroutine with `context.Background()`. This means cancelling the context passed to `StartWorkflow` does **not** cancel the async execution.

Use `Engine.Cancel` to cancel async workflows.

### Using Context Cancellation (Sync Workflows)

For synchronous workflows (`WithSynchronousExecution()`), the engine uses the provided context. Cancelling that context will stop the workflow between steps:

```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    time.Sleep(5 * time.Second)
    cancel() // Cancel after 5 seconds
}()

runID, err := eng.StartWorkflow(ctx, wf, input,
    gorkflow.WithSynchronousExecution(),
)
// err will reflect the cancellation if it was triggered
```

## What Happens on Cancellation

When a workflow is cancelled:

1. The run status is set to `CANCELLED`
2. `CompletedAt` is set to the current time
3. The run is persisted to the store

### Between Steps

The engine checks for context cancellation before each step:

```go
select {
case <-ctx.Done():
    // Cancel the workflow
default:
    // Execute the next step
}
```

If cancellation is detected between steps, the workflow is cancelled cleanly without starting the next step.

### During Step Execution

If a step is currently executing when cancellation occurs:

- The step's context (`ctx.Context`) carries the cancellation signal
- Operations using `ctx.Context` (HTTP requests, DB queries, etc.) will receive the cancellation
- The step handler should check `ctx.Context.Done()` for long-running operations
- The current step will complete (or fail) before the workflow transitions to `CANCELLED`

### Handling Cancellation in Step Handlers

```go
step := gorkflow.NewStep("long-task", "Long Task",
    func(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
        for i := 0; i < len(input.Items); i++ {
            select {
            case <-ctx.Context.Done():
                return MyOutput{}, ctx.Context.Err()
            default:
                processItem(input.Items[i])
            }
        }
        return MyOutput{Processed: len(input.Items)}, nil
    },
)
```

## Run Status After Cancellation

```go
run, _ := eng.GetRun(ctx, runID)

if run.Status == gorkflow.RunStatusCancelled {
    fmt.Println("Workflow was cancelled")
    fmt.Printf("Cancelled at: %v\n", run.CompletedAt)
}

// Check with IsTerminal
if run.Status.IsTerminal() {
    fmt.Println("Workflow is done (completed, failed, or cancelled)")
}
```

## Example: Timeout-Based Cancellation

```go
// Cancel workflow if it takes longer than 10 minutes
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()

runID, err := eng.StartWorkflow(ctx, wf, input,
    gorkflow.WithSynchronousExecution(),
)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        fmt.Println("Workflow timed out")
    }
}
```

---

**Next**: Learn about [Tags and Metadata](tags-and-metadata.md) →
