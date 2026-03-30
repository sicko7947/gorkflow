# Step API

Steps are the fundamental building blocks of workflows. Each step is a generic, type-safe unit of work with configurable retries, timeouts, and validation.

## Creating Steps

### `NewStep`

```go
func NewStep[TIn, TOut any](
    id, name string,
    handler StepHandler[TIn, TOut],
    opts ...StepOption,
) *Step[TIn, TOut]
```

Creates a new type-safe step. Input/output types are inferred from the handler signature. Validation is enabled by default.

```go
step := gorkflow.NewStep(
    "create-user",
    "Create User",
    func(ctx *gorkflow.StepContext, input CreateUserInput) (CreateUserOutput, error) {
        // Business logic
        return CreateUserOutput{UserID: "123"}, nil
    },
    gorkflow.WithRetries(3),
    gorkflow.WithTimeout(30 * time.Second),
)
```

### `NewConditionalStep`

```go
func NewConditionalStep[TIn, TOut any](
    step *Step[TIn, TOut],
    condition Condition,
    defaultValue *TOut,
) *ConditionalStep[TIn, TOut]
```

Wraps an existing step with a condition. The step only executes if the condition returns `true`. When skipped, `defaultValue` is used as output (or the zero value if `nil`).

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var enabled bool
    ctx.State.Get("feature_enabled", &enabled)
    return enabled, nil
}

baseStep := gorkflow.NewStep("process", "Process", handler)
conditionalStep := gorkflow.NewConditionalStep(baseStep, condition, nil)
```

### `WrapStepWithCondition`

```go
func WrapStepWithCondition(step StepExecutor, condition Condition, defaultValue any) StepExecutor
```

Type-erased version of conditional wrapping, used internally by the builder's `ThenStepIf`. Works with any `StepExecutor`, not just `*Step[TIn, TOut]`.

## StepHandler

The handler function signature:

```go
type StepHandler[TIn, TOut any] func(ctx *StepContext, input TIn) (TOut, error)
```

- `ctx` — execution context with logger, state, step data, and metadata
- `input` — typed input (automatically unmarshaled and validated)
- Returns typed output and error

## Step Options

Options are passed to `NewStep` to configure execution behavior.

### `WithRetries`

```go
func WithRetries(max int) StepOption
```

Sets the maximum number of retry attempts on failure. Default: `3`.

```go
step := gorkflow.NewStep("api-call", "API Call", handler,
    gorkflow.WithRetries(5),
)
```

### `WithTimeout`

```go
func WithTimeout(d time.Duration) StepOption
```

Sets the maximum execution time for the step. A `context.WithTimeout` is applied before each attempt. Default: `30s`.

```go
step := gorkflow.NewStep("long-task", "Long Task", handler,
    gorkflow.WithTimeout(2 * time.Minute),
)
```

### `WithBackoff`

```go
func WithBackoff(strategy BackoffStrategy) StepOption
```

Sets the retry backoff strategy. Default: `BackoffLinear`.

Available strategies:
- `BackoffLinear` — delay = baseDelay * attempt
- `BackoffExponential` — delay = baseDelay * 2^(attempt-1)
- `BackoffNone` — no delay between retries

```go
step := gorkflow.NewStep("resilient", "Resilient", handler,
    gorkflow.WithRetries(5),
    gorkflow.WithBackoff(gorkflow.BackoffExponential),
)
```

### `WithRetryDelay`

```go
func WithRetryDelay(d time.Duration) StepOption
```

Sets the base delay between retries. Default: `1000ms`.

```go
step := gorkflow.NewStep("api-call", "API Call", handler,
    gorkflow.WithRetries(3),
    gorkflow.WithRetryDelay(2 * time.Second),
    gorkflow.WithBackoff(gorkflow.BackoffExponential),
)
```

### `WithContinueOnError`

```go
func WithContinueOnError(continueOnError bool) StepOption
```

When `true`, the workflow continues to the next step even if this step fails after exhausting all retries. Default: `false`.

```go
step := gorkflow.NewStep("optional", "Optional Step", handler,
    gorkflow.WithContinueOnError(true),
)
```

## StepExecutor Interface

The engine works with the `StepExecutor` interface. Both `Step[TIn, TOut]` and `ConditionalStep[TIn, TOut]` implement it.

```go
type StepExecutor interface {
    GetID() string
    GetName() string
    GetDescription() string
    GetConfig() ExecutionConfig
    SetConfig(config ExecutionConfig)

    InputType() reflect.Type
    OutputType() reflect.Type

    Execute(ctx *StepContext, input []byte) (output []byte, err error)

    ValidateInput(data []byte) error
    ValidateOutput(data []byte) error
}
```

### Key Methods

| Method | Description |
|--------|-------------|
| `GetID()` | Returns the step's unique identifier |
| `GetName()` | Returns the human-readable name |
| `GetConfig()` | Returns the current `ExecutionConfig` |
| `SetConfig()` | Overrides the execution config (used by the builder to propagate workflow-level config) |
| `Execute()` | Runs the step handler with JSON-serialized input, returns JSON-serialized output |
| `ValidateInput()` | Validates that JSON data can be unmarshaled to the input type and passes struct validation |
| `ValidateOutput()` | Validates that JSON data can be unmarshaled to the output type and passes struct validation |

## Condition Type

```go
type Condition func(ctx *StepContext) (bool, error)
```

A function that determines whether a conditional step should execute. Has access to the full `StepContext` including state and previous step outputs.

## ExecutionConfig

```go
type ExecutionConfig struct {
    MaxRetries      int             `json:"max_retries,omitempty"`
    RetryDelayMs    int             `json:"retry_delay_ms,omitempty"`
    RetryBackoff    BackoffStrategy `json:"retry_backoff,omitempty"`
    TimeoutSeconds  int             `json:"timeout_seconds,omitempty"`
    MaxConcurrency  int             `json:"max_concurrency,omitempty"`
    ContinueOnError bool           `json:"continue_on_error,omitempty"`
}
```

### Default Values

```go
var DefaultExecutionConfig = ExecutionConfig{
    MaxRetries:      3,
    RetryDelayMs:    1000,
    RetryBackoff:    BackoffLinear,
    TimeoutSeconds:  30,
    MaxConcurrency:  1,
    ContinueOnError: false,
}
```

---

**Next**: Learn about the [Engine API](engine-api.md) →
