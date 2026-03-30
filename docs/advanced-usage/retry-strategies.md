# Retry Strategies

Gorkflow supports automatic retries with configurable backoff strategies for handling transient failures.

## Overview

When a step returns an error, the engine retries it according to the step's `ExecutionConfig`:

1. The step fails
2. The engine calculates a backoff delay based on the strategy
3. After the delay, the step is re-executed with an incremented `Attempt` counter
4. This repeats until the step succeeds or `MaxRetries` is exhausted

## Backoff Strategies

### Linear (`BackoffLinear`)

Delay increases linearly with each attempt: `baseDelay * attempt`.

```
Attempt 0: no delay (first attempt)
Attempt 1: 1000ms
Attempt 2: 2000ms
Attempt 3: 3000ms
```

This is the **default** strategy.

```go
step := gorkflow.NewStep("api-call", "API Call", handler,
    gorkflow.WithRetries(3),
    gorkflow.WithRetryDelay(1 * time.Second),
    gorkflow.WithBackoff(gorkflow.BackoffLinear),
)
```

### Exponential (`BackoffExponential`)

Delay doubles with each attempt: `baseDelay * 2^(attempt-1)`.

```
Attempt 0: no delay (first attempt)
Attempt 1: 1000ms
Attempt 2: 2000ms
Attempt 3: 4000ms
Attempt 4: 8000ms
```

Best for external API calls where you want to back off aggressively to avoid overwhelming the service.

```go
step := gorkflow.NewStep("external-api", "External API", handler,
    gorkflow.WithRetries(5),
    gorkflow.WithRetryDelay(500 * time.Millisecond),
    gorkflow.WithBackoff(gorkflow.BackoffExponential),
)
```

### None (`BackoffNone`)

No delay between retries. The step is retried immediately.

```go
step := gorkflow.NewStep("idempotent-op", "Idempotent Op", handler,
    gorkflow.WithRetries(3),
    gorkflow.WithBackoff(gorkflow.BackoffNone),
)
```

## `CalculateBackoff` Function

The backoff calculation is exposed as a public function:

```go
func CalculateBackoff(baseDelayMs int, attempt int, strategy string) time.Duration
```

- `baseDelayMs` — base delay in milliseconds
- `attempt` — current retry attempt (0-based; attempt 0 always returns 0)
- `strategy` — `"LINEAR"`, `"EXPONENTIAL"`, or `"NONE"`

```go
// Examples:
gorkflow.CalculateBackoff(1000, 0, "LINEAR")       // 0 (first attempt)
gorkflow.CalculateBackoff(1000, 1, "LINEAR")       // 1s
gorkflow.CalculateBackoff(1000, 3, "LINEAR")       // 3s
gorkflow.CalculateBackoff(1000, 3, "EXPONENTIAL")  // 4s (1000 * 2^2)
gorkflow.CalculateBackoff(1000, 3, "NONE")         // 0
```

## Configuration

### Step-Level

```go
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithRetries(5),                        // Max retry attempts
    gorkflow.WithRetryDelay(2 * time.Second),       // Base delay
    gorkflow.WithBackoff(gorkflow.BackoffExponential), // Strategy
)
```

### Workflow-Level Default

Set defaults for all steps in a workflow. Steps with explicit configuration are not overridden.

```go
wf, _ := gorkflow.NewWorkflow("resilient", "Resilient").
    WithConfig(gorkflow.ExecutionConfig{
        MaxRetries:   5,
        RetryDelayMs: 2000,
        RetryBackoff: gorkflow.BackoffExponential,
    }).
    ThenStep(step1).  // Uses workflow config if step has default config
    ThenStep(step2).
    Build()
```

### Default Values

| Field | Default |
|-------|---------|
| `MaxRetries` | `3` |
| `RetryDelayMs` | `1000` (1 second) |
| `RetryBackoff` | `BackoffLinear` |

## Retry Behavior in the Engine

During execution, the engine:

1. Runs the step handler within a timeout context
2. On failure, sets step status to `RETRYING` and persists the state
3. Sleeps for the calculated backoff duration
4. Re-executes the handler with an updated `Attempt` counter on the `StepContext`
5. On final failure, sets step status to `FAILED` with a `StepError`

### Accessing Attempt Number

The current attempt is available in the handler via `ctx.Attempt`:

```go
step := gorkflow.NewStep("retry-aware", "Retry Aware", func(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    if ctx.Attempt > 0 {
        ctx.Logger.Info().Int("attempt", ctx.Attempt).Msg("Retrying...")
    }
    // Business logic
    return MyOutput{}, nil
})
```

### Panic Recovery

If a step handler panics, the panic is recovered and treated as an error, triggering the normal retry flow.

## Examples

### External API with Exponential Backoff

```go
apiStep := gorkflow.NewStep(
    "fetch-data",
    "Fetch Data",
    func(ctx *gorkflow.StepContext, input APIRequest) (APIResponse, error) {
        resp, err := http.Get(input.URL)
        if err != nil {
            return APIResponse{}, err // Will be retried
        }
        defer resp.Body.Close()

        if resp.StatusCode >= 500 {
            return APIResponse{}, fmt.Errorf("server error: %d", resp.StatusCode)
        }

        var result APIResponse
        json.NewDecoder(resp.Body).Decode(&result)
        return result, nil
    },
    gorkflow.WithRetries(5),
    gorkflow.WithRetryDelay(500 * time.Millisecond),
    gorkflow.WithBackoff(gorkflow.BackoffExponential),
    gorkflow.WithTimeout(10 * time.Second),
)
```

### Database Write with Linear Backoff

```go
dbStep := gorkflow.NewStep(
    "save-record",
    "Save Record",
    func(ctx *gorkflow.StepContext, input Record) (SaveResult, error) {
        _, err := db.ExecContext(ctx.Context, "INSERT INTO records (data) VALUES (?)", input.Data)
        if err != nil {
            return SaveResult{}, fmt.Errorf("db insert failed: %w", err)
        }
        return SaveResult{Success: true}, nil
    },
    gorkflow.WithRetries(3),
    gorkflow.WithRetryDelay(1 * time.Second),
    gorkflow.WithBackoff(gorkflow.BackoffLinear),
)
```

### Non-Retriable Step

```go
step := gorkflow.NewStep("validate", "Validate", handler,
    gorkflow.WithRetries(0),  // No retries
)
```

---

**Next**: Learn about [Timeouts](timeouts.md) →
