# Configuration

Gorkflow provides two levels of configuration: step-level (`ExecutionConfig`) and engine-level (`EngineConfig`).

## ExecutionConfig

Controls per-step execution behavior including retries, timeouts, and failure handling.

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

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `MaxRetries` | `int` | `3` | Maximum number of retry attempts after the initial execution |
| `RetryDelayMs` | `int` | `1000` | Base delay between retries in milliseconds |
| `RetryBackoff` | `BackoffStrategy` | `BackoffLinear` | Backoff strategy for retry delays |
| `TimeoutSeconds` | `int` | `30` | Per-attempt timeout in seconds |
| `MaxConcurrency` | `int` | `1` | Reserved — not currently used by the engine. Parallel steps within a level all run concurrently. |
| `ContinueOnError` | `bool` | `false` | If `true`, workflow continues even if this step fails |

### DefaultExecutionConfig

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

### Config Propagation

When `Build()` is called, the workflow-level config is propagated to any step that still uses `DefaultExecutionConfig`. Steps with explicitly set configurations are not overridden.

```go
wf, _ := gorkflow.NewWorkflow("wf", "WF").
    WithConfig(gorkflow.ExecutionConfig{
        MaxRetries:   5,
        RetryDelayMs: 2000,
    }).
    ThenStep(step1).  // Gets workflow config (MaxRetries=5)
    ThenStep(step2).  // step2 has WithRetries(1), keeps its config
    Build()
```

## BackoffStrategy

```go
type BackoffStrategy string

const (
    BackoffLinear      BackoffStrategy = "LINEAR"
    BackoffExponential BackoffStrategy = "EXPONENTIAL"
    BackoffNone        BackoffStrategy = "NONE"
)
```

| Strategy | Formula | Example (base=1s) |
|----------|---------|-------------------|
| `BackoffLinear` | `base * attempt` | 1s, 2s, 3s, 4s |
| `BackoffExponential` | `base * 2^(attempt-1)` | 1s, 2s, 4s, 8s |
| `BackoffNone` | `0` | 0, 0, 0, 0 |

See [Retry Strategies](../advanced-usage/retry-strategies.md) for detailed behavior.

## EngineConfig

Controls engine-level behavior.

```go
type EngineConfig struct {
    MaxConcurrentWorkflows int           `json:"max_concurrent_workflows"`
    DefaultTimeout         time.Duration `json:"default_timeout"`
}
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `MaxConcurrentWorkflows` | `int` | `10` | Maximum number of workflows that can run concurrently |
| `DefaultTimeout` | `time.Duration` | `5 * time.Minute` | Default engine-level timeout |

### DefaultEngineConfig

```go
var DefaultEngineConfig = EngineConfig{
    MaxConcurrentWorkflows: 10,
    DefaultTimeout:         5 * time.Minute,
}
```

## Step Option Functions

Functional options for configuring individual steps.

### `WithRetries`

```go
func WithRetries(max int) StepOption
```

Sets `MaxRetries`. Pass `0` to disable retries.

### `WithTimeout`

```go
func WithTimeout(d time.Duration) StepOption
```

Sets `TimeoutSeconds` (converted from `time.Duration`).

### `WithBackoff`

```go
func WithBackoff(strategy BackoffStrategy) StepOption
```

Sets `RetryBackoff`.

### `WithRetryDelay`

```go
func WithRetryDelay(d time.Duration) StepOption
```

Sets `RetryDelayMs` (converted from `time.Duration`).

### `WithContinueOnError`

```go
func WithContinueOnError(continueOnError bool) StepOption
```

Sets `ContinueOnError`.

### `WithoutValidation`

```go
func WithoutValidation() StepOption
```

Disables input/output validation for the step. Validation is enabled by default.

### `WithCustomValidator`

```go
func WithCustomValidator(v *validator.Validate) StepOption
```

Sets a custom `go-playground/validator` instance for the step.

## Engine Option Functions

Functional options for configuring the engine.

### `engine.WithLogger`

```go
func WithLogger(logger zerolog.Logger) EngineOption
```

Sets a custom `zerolog.Logger`. Default is a pretty console logger at `Info` level.

### `engine.WithConfig`

```go
func WithConfig(config gorkflow.EngineConfig) EngineOption
```

Sets a custom `EngineConfig`.

## Start Option Functions

Functional options for configuring workflow execution at start time.

### `WithSynchronousExecution`

```go
func WithSynchronousExecution() StartOption
```

Runs the workflow synchronously. `StartWorkflow` blocks until completion.

### `WithResourceID`

```go
func WithResourceID(id string) StartOption
```

Associates the run with a resource ID for grouping and concurrency control.

### `WithConcurrencyCheck`

```go
func WithConcurrencyCheck(check bool) StartOption
```

Enables concurrency checking for the resource ID.

### `WithTags`

```go
func WithTags(tags map[string]string) StartOption
```

Sets metadata tags on the workflow run at start time.

## Complete Example

```go
// Step with full configuration
step := gorkflow.NewStep("api-call", "External API Call", handler,
    gorkflow.WithRetries(5),
    gorkflow.WithRetryDelay(500 * time.Millisecond),
    gorkflow.WithBackoff(gorkflow.BackoffExponential),
    gorkflow.WithTimeout(10 * time.Second),
    gorkflow.WithContinueOnError(false),
)

// Workflow with config defaults
wf, _ := gorkflow.NewWorkflow("pipeline", "Pipeline").
    WithConfig(gorkflow.ExecutionConfig{
        MaxRetries:   3,
        RetryDelayMs: 1000,
        RetryBackoff: gorkflow.BackoffLinear,
    }).
    ThenStep(step).
    Build()

// Engine with config
eng := engine.NewEngine(store,
    engine.WithLogger(logger),
    engine.WithConfig(gorkflow.EngineConfig{
        MaxConcurrentWorkflows: 20,
        DefaultTimeout:         10 * time.Minute,
    }),
)

// Start with options
runID, err := eng.StartWorkflow(ctx, wf, input,
    gorkflow.WithSynchronousExecution(),
    gorkflow.WithResourceID("user-123"),
    gorkflow.WithTags(map[string]string{"source": "api"}),
)
```

---

**See also**: [Step API](step-api.md) | [Engine API](engine-api.md)
