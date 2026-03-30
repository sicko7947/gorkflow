# FAQ

Frequently asked questions about Gorkflow.

## Steps and Execution

### Why is my step not retrying?

Check the step's `MaxRetries` configuration. The default is `3`, but if you set `WithRetries(0)`, the step won't retry at all.

```go
// This step will NOT retry
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithRetries(0),
)

// This step retries up to 5 times
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithRetries(5),
)
```

Also check if the step is succeeding but returning unexpected results — retries only happen when the handler returns an error.

### How do I pass data between steps?

There are two approaches:

**1. Step chaining (automatic):** Each step's output is automatically passed as input to the next step via JSON serialization. Ensure the output type of one step is JSON-compatible with the input type of the next.

```go
// Step 1 outputs UserData
step1 := gorkflow.NewStep("fetch", "Fetch", fetchHandler) // returns UserData

// Step 2 takes UserData as input
step2 := gorkflow.NewStep("process", "Process", processHandler) // accepts UserData

wf, _ := gorkflow.NewWorkflow("wf", "WF").
    ThenStep(step1).
    ThenStep(step2). // Automatically gets step1's output
    Build()
```

**2. Workflow state:** Use `ctx.State` for sharing data between non-adjacent steps or for accumulating values.

```go
// In step 1
ctx.State.Set("user_count", 42)

// In step 3 (not directly connected to step 1)
count, _ := gorkflow.GetTyped[int](ctx.State, "user_count")
```

**3. GetOutput helper:** Access any previous step's output by ID:

```go
userData, err := gorkflow.GetOutput[UserData](ctx, "fetch-step")
```

### Why does my workflow appear stuck?

Common causes:

1. **Async execution (default):** The workflow runs in a background goroutine. If you're checking status immediately after `StartWorkflow`, the workflow may still be running. Use `WithSynchronousExecution()` to wait for completion.

2. **Long timeouts:** Default step timeout is 30 seconds, with 3 retries. A failing step could take 2+ minutes before the workflow fails.

3. **External calls hanging:** If a step makes HTTP or database calls without using `ctx.Context`, the timeout won't cancel the operation.

4. **Process exited:** For async workflows, if the process exits before completion, the run stays in `RUNNING` state in the store with no goroutine to finish it.

### How do I make a step non-retriable?

Set `WithRetries(0)`:

```go
step := gorkflow.NewStep("validate", "Validate", handler,
    gorkflow.WithRetries(0),
)
```

### Can I access the workflow input from any step?

The first step receives the workflow input directly. For later steps, use `GetInput` to access any step's input:

```go
originalInput, err := gorkflow.GetInput[WorkflowInput](ctx, "first-step-id")
```

Or store it in state in the first step:

```go
// First step
ctx.State.Set("original_input", input)

// Any later step
originalInput, _ := gorkflow.GetTyped[WorkflowInput](ctx.State, "original_input")
```

## Configuration

### What are the default settings?

```go
// Per-step defaults
MaxRetries:      3
RetryDelayMs:    1000    // 1 second
RetryBackoff:    "LINEAR"
TimeoutSeconds:  30
MaxConcurrency:  1
ContinueOnError: false

// Engine defaults
MaxConcurrentWorkflows: 10
DefaultTimeout:         5 minutes
```

### How does config propagation work?

Workflow-level config is applied to steps that use `DefaultExecutionConfig`. Steps with explicitly set options keep their own config:

```go
wf, _ := gorkflow.NewWorkflow("wf", "WF").
    WithConfig(gorkflow.ExecutionConfig{MaxRetries: 10}).
    ThenStep(stepA).  // Gets MaxRetries=10 from workflow
    ThenStep(stepB).  // stepB has WithRetries(1), keeps MaxRetries=1
    Build()
```

## State and Data

### What's the difference between State and Step Data?

| | State (`ctx.State`) | Step Data (`ctx.Data`) |
|-|---------------------|------------------------|
| **Purpose** | Shared key-value store | Access step inputs/outputs |
| **Access** | Read/write from any step | Read-only |
| **Scope** | Arbitrary keys | By step ID |
| **Use case** | Flags, counters, shared config | Structured data flow |

### Are state operations persisted immediately?

Yes. Every `Set`, `Delete`, and `GetAll` call persists to the configured store. The accessor also maintains an in-memory cache for faster reads within the same run.

## Storage

### Which store should I use?

| Store | Use Case |
|-------|----------|
| `MemoryStore` | Testing, development, prototyping |
| `LibSQLStore` | Production (local SQLite or remote Turso) |
| Custom | Any other backend (PostgreSQL, Redis, etc.) |

### Can I switch stores without changing my workflow code?

Yes. The store is passed to the engine, not to workflows or steps. Swap the store implementation and everything else stays the same:

```go
// Development
eng := engine.NewEngine(store.NewMemoryStore())

// Production
libsqlStore, _ := store.NewLibSQLStore("file:./production.db")
eng := engine.NewEngine(libsqlStore)
```

## Error Handling

### How do I let a workflow continue after a step fails?

Use `WithContinueOnError`:

```go
step := gorkflow.NewStep("optional", "Optional", handler,
    gorkflow.WithContinueOnError(true),
)
```

Or set it at the workflow level:

```go
wf, _ := gorkflow.NewWorkflow("wf", "WF").
    WithConfig(gorkflow.ExecutionConfig{ContinueOnError: true}).
    ThenStep(step1).
    Build()
```

### What happens if a condition function returns an error?

The workflow fails immediately. Condition errors are not retried.

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    return false, fmt.Errorf("something went wrong")
    // → Workflow fails with "condition evaluation failed: something went wrong"
}
```

### What happens if a step panics?

Panics are recovered by the engine and treated as errors. The step enters the normal retry flow:

```go
// This panic will be caught
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    panic("unexpected error")
    // → Treated as error "step panicked: unexpected error"
    // → Will be retried according to step config
}
```

## Validation

### How do I disable validation?

```go
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithoutValidation(),
)
```

### Can I use a custom validator?

Yes, pass a custom `go-playground/validator` instance:

```go
v := validator.New()
// Register custom validations...

step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithCustomValidator(v),
)
```

---

**See also**: [Common Issues](common-issues.md) | [Debugging](debugging.md)
