# Performance

Tips for getting the best performance from Gorkflow workflows.

## Store Selection

The store backend has the biggest impact on performance.

### Use MemoryStore for Tests

`MemoryStore` has zero I/O overhead. Always use it for unit and integration tests:

```go
memStore := store.NewMemoryStore()
eng := engine.NewEngine(memStore)
```

### LibSQL Store Tuning

For production workloads with LibSQL:

```go
store, _ := store.NewLibSQLStoreWithOptions("file:./workflow.db", store.LibSQLStoreOptions{
    MaxOpenConns:    10,
    MaxIdleConns:    5,
    ConnMaxLifetime: time.Hour,
    CacheSize:       -2000, // 2000 pages (~8MB with 4KB pages)
})
```

- **Local file** (`file:./local.db`) is faster than remote Turso for single-process deployments
- **Remote Turso** (`libsql://...`) adds network latency per store operation but enables multi-process access

## Step Design

### Keep Steps Focused

Smaller, focused steps are easier to retry and parallelize:

```go
// Good: focused steps
validateStep := gorkflow.NewStep("validate", "Validate", validateHandler)
processStep := gorkflow.NewStep("process", "Process", processHandler)
saveStep := gorkflow.NewStep("save", "Save", saveHandler)

// Bad: monolithic step
doEverythingStep := gorkflow.NewStep("all", "Do Everything", megaHandler)
```

### Use Parallel Steps

When steps are independent, run them in parallel:

```go
wf, _ := gorkflow.NewWorkflow("fast", "Fast Pipeline").
    ThenStep(setupStep).
    Parallel(fetchUsers, fetchOrders, fetchProducts). // Run concurrently
    ThenStep(aggregateStep).
    Build()
```

### Avoid Heavy Work in Conditions

Condition functions run before the step executes. Keep them lightweight:

```go
// Good: read from state (cached)
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var flag bool
    ctx.State.Get("enabled", &flag)
    return flag, nil
}

// Bad: expensive operation in condition
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    resp, err := http.Get("https://api.example.com/check") // Network call!
    // ...
}
```

## Retry and Timeout Tuning

### Right-Size Timeouts

Overly generous timeouts delay failure detection:

```go
// Good: timeout matches expected duration
apiStep := gorkflow.NewStep("api", "API Call", handler,
    gorkflow.WithTimeout(5 * time.Second), // API should respond in <1s
)

// Bad: excessive timeout
apiStep := gorkflow.NewStep("api", "API Call", handler,
    gorkflow.WithTimeout(5 * time.Minute), // Waits 5 min before failing
)
```

### Use Exponential Backoff for External Services

Linear backoff can overwhelm external services. Use exponential for external calls:

```go
step := gorkflow.NewStep("external", "External API", handler,
    gorkflow.WithRetries(5),
    gorkflow.WithRetryDelay(500 * time.Millisecond),
    gorkflow.WithBackoff(gorkflow.BackoffExponential),
)
// Delays: 0, 500ms, 1s, 2s, 4s
```

### Skip Retries for Non-Transient Errors

Validation and business logic errors won't succeed on retry:

```go
validateStep := gorkflow.NewStep("validate", "Validate", handler,
    gorkflow.WithRetries(0), // Don't retry validation failures
)
```

## Context Propagation

### Use ctx.Context for I/O

Always pass `ctx.Context` to external operations so timeouts propagate:

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    // Good: timeout propagates to HTTP call
    req, _ := http.NewRequestWithContext(ctx.Context, "GET", url, nil)
    resp, err := http.DefaultClient.Do(req)

    // Bad: ignores step timeout
    resp, err := http.Get(url)
}
```

### Check Cancellation in Loops

For long-running computations, check for cancellation periodically:

```go
func handler(ctx *gorkflow.StepContext, input BatchInput) (BatchOutput, error) {
    for i, item := range input.Items {
        select {
        case <-ctx.Context.Done():
            return BatchOutput{}, ctx.Context.Err()
        default:
        }
        process(item)
    }
    return BatchOutput{Processed: len(input.Items)}, nil
}
```

## State Usage

### Prefer Step Chaining Over State

Step outputs are automatically passed to the next step. Use state only when you need to share data across non-adjacent steps:

```go
// Good: data flows through step chain
step1 → output → step2 → output → step3

// Unnecessary: using state for adjacent steps
step1: ctx.State.Set("result", result)
step2: ctx.State.Get("result", &result) // Just use the step input instead
```

### Minimize State Operations

Each `Set` and `Get` (on cache miss) involves a store call. Batch related data into a single key when possible:

```go
// Good: one state operation
ctx.State.Set("user_profile", UserProfile{Name: name, Email: email, Age: age})

// Less efficient: three state operations
ctx.State.Set("user_name", name)
ctx.State.Set("user_email", email)
ctx.State.Set("user_age", age)
```

## Logging

### Use Appropriate Log Levels

Debug logging adds overhead. Use `Info` level for production:

```go
// Production
logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel)

// Development/debugging
logger := zerolog.New(os.Stdout).Level(zerolog.DebugLevel)
```

---

**See also**: [Debugging](debugging.md) | [Common Issues](common-issues.md)
