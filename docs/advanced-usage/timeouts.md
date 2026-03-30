# Timeouts

Gorkflow supports per-step timeouts to prevent steps from running indefinitely.

## Overview

Each step has a configurable timeout. When a step exceeds its timeout, the context is cancelled and the step fails with a timeout error. The step may then be retried according to its retry configuration.

## Per-Step Timeout

### Setting a Timeout

Use `WithTimeout` when creating a step:

```go
step := gorkflow.NewStep("long-task", "Long Task", handler,
    gorkflow.WithTimeout(30 * time.Second),
)
```

The default timeout is **30 seconds** (from `DefaultExecutionConfig.TimeoutSeconds`).

### How It Works

Before each execution attempt, the engine wraps the step's context with `context.WithTimeout`:

```go
execCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds) * time.Second)
```

This timeout context is set on `StepContext.Context`, which means:

1. The step handler receives a context that will be cancelled after the timeout
2. Any operations using `ctx.Context` (HTTP requests, database queries, etc.) will respect the deadline
3. If the timeout fires, the step fails and may be retried

### Checking for Timeout in Handlers

Use `ctx.Context` to propagate the timeout to downstream calls:

```go
step := gorkflow.NewStep("api-call", "API Call",
    func(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
        // The timeout is already on ctx.Context
        req, _ := http.NewRequestWithContext(ctx.Context, "GET", input.URL, nil)
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return MyOutput{}, err // Includes timeout errors
        }
        defer resp.Body.Close()

        // Process response...
        return MyOutput{Data: data}, nil
    },
    gorkflow.WithTimeout(10 * time.Second),
)
```

You can also check for cancellation manually:

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    for i := 0; i < 1000; i++ {
        select {
        case <-ctx.Context.Done():
            return MyOutput{}, ctx.Context.Err()
        default:
            // Continue processing
        }
        processItem(i)
    }
    return MyOutput{Done: true}, nil
}
```

## Timeouts and Retries

Timeouts interact with the retry system:

1. A timeout causes the step to fail
2. The engine checks `context.DeadlineExceeded` and wraps the error with timeout information
3. If retries remain, the step is retried with a **fresh** timeout context
4. The backoff delay occurs between the timeout and the next attempt

```go
step := gorkflow.NewStep("flaky-service", "Flaky Service", handler,
    gorkflow.WithTimeout(5 * time.Second),          // 5s per attempt
    gorkflow.WithRetries(3),                         // Up to 3 retries
    gorkflow.WithBackoff(gorkflow.BackoffExponential),
)
```

In this example, the worst-case timeline is:
```
Attempt 0: runs up to 5s, times out
  delay: 1s
Attempt 1: runs up to 5s, times out
  delay: 2s
Attempt 2: runs up to 5s, times out
  delay: 4s
Attempt 3: runs up to 5s, times out → step FAILED
```

## Engine-Level Default Timeout

The `EngineConfig.DefaultTimeout` sets an overall timeout for the engine, separate from per-step timeouts.

```go
eng := engine.NewEngine(memStore, engine.WithConfig(gorkflow.EngineConfig{
    DefaultTimeout: 10 * time.Minute,
}))
```

Default: `5 * time.Minute`.

## Timeout Error Detection

Use `IsTimeoutError` to check if an error is timeout-related:

```go
if gorkflow.IsTimeoutError(err) {
    log.Println("Step timed out")
}
```

This checks for:
- `StepError` or `WorkflowError` with code `ErrCodeTimeout`
- Error messages containing "timeout" or "context deadline exceeded"

## Configuration Reference

| Setting | Default | Description |
|---------|---------|-------------|
| `ExecutionConfig.TimeoutSeconds` | `30` | Per-step timeout in seconds |
| `EngineConfig.DefaultTimeout` | `5m` | Engine-level default timeout |

## Examples

### Quick Validation Step

```go
validateStep := gorkflow.NewStep("validate", "Validate Input",
    func(ctx *gorkflow.StepContext, input Order) (ValidatedOrder, error) {
        // Quick operation — short timeout
        return validate(input)
    },
    gorkflow.WithTimeout(5 * time.Second),
    gorkflow.WithRetries(0), // Don't retry validation
)
```

### Long-Running Processing

```go
processStep := gorkflow.NewStep("process", "Process Data",
    func(ctx *gorkflow.StepContext, input DataBatch) (ProcessResult, error) {
        for _, item := range input.Items {
            select {
            case <-ctx.Context.Done():
                return ProcessResult{}, ctx.Context.Err()
            default:
                processItem(item)
            }
        }
        return ProcessResult{Processed: len(input.Items)}, nil
    },
    gorkflow.WithTimeout(5 * time.Minute),
    gorkflow.WithRetries(1),
)
```

### Database with Timeout Propagation

```go
dbStep := gorkflow.NewStep("query", "Database Query",
    func(ctx *gorkflow.StepContext, input QueryInput) (QueryResult, error) {
        // ctx.Context carries the step timeout — the DB query will be cancelled if it exceeds it
        rows, err := db.QueryContext(ctx.Context, input.SQL, input.Args...)
        if err != nil {
            return QueryResult{}, err
        }
        defer rows.Close()
        // Process rows...
        return QueryResult{Rows: results}, nil
    },
    gorkflow.WithTimeout(15 * time.Second),
)
```

---

**Next**: Learn about [Error Handling](error-handling.md) →
