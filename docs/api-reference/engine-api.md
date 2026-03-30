# Engine API

The `Engine` orchestrates workflow execution, managing step ordering, retries, timeouts, persistence, and cancellation.

## Creating an Engine

### `NewEngine`

```go
func NewEngine(store gorkflow.WorkflowStore, opts ...EngineOption) *Engine
```

Creates a new workflow engine with a store backend and optional configuration.

```go
import (
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
)

memStore := store.NewMemoryStore()
eng := engine.NewEngine(memStore)
```

### Engine Options

#### `WithLogger`

```go
func WithLogger(logger zerolog.Logger) EngineOption
```

Sets a custom `zerolog.Logger`. By default, the engine uses a pretty console logger at `Info` level.

```go
logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
eng := engine.NewEngine(memStore, engine.WithLogger(logger))
```

#### `WithConfig`

```go
func WithConfig(config gorkflow.EngineConfig) EngineOption
```

Sets engine-level configuration.

```go
eng := engine.NewEngine(memStore, engine.WithConfig(gorkflow.EngineConfig{
    MaxConcurrentWorkflows: 20,
    DefaultTimeout:         10 * time.Minute,
}))
```

### EngineConfig

```go
type EngineConfig struct {
    MaxConcurrentWorkflows int           `json:"max_concurrent_workflows"`
    DefaultTimeout         time.Duration `json:"default_timeout"`
}
```

Default values:

| Field | Default |
|-------|---------|
| `MaxConcurrentWorkflows` | `10` |
| `DefaultTimeout` | `5 * time.Minute` |

## Starting Workflows

### `StartWorkflow`

```go
func (e *Engine) StartWorkflow(
    ctx context.Context,
    wf *gorkflow.Workflow,
    input any,
    opts ...gorkflow.StartOption,
) (string, error)
```

Initiates a workflow execution. Returns a `runID` (UUID) that can be used to query status.

**Async (default):** The workflow executes in a background goroutine. `StartWorkflow` returns immediately after creating the run record.

```go
runID, err := eng.StartWorkflow(ctx, wf, MyInput{Value: "hello"})
if err != nil {
    log.Fatal(err)
}
fmt.Println("Started run:", runID)
```

**Sync:** Use `WithSynchronousExecution()` to block until the workflow completes. The returned error reflects the workflow result.

```go
runID, err := eng.StartWorkflow(ctx, wf, input,
    gorkflow.WithSynchronousExecution(),
)
if err != nil {
    log.Fatal("Workflow failed:", err)
}
```

### Start Options

#### `WithSynchronousExecution`

```go
func WithSynchronousExecution() StartOption
```

Runs the workflow synchronously. `StartWorkflow` blocks until completion and returns any execution error.

#### `WithResourceID`

```go
func WithResourceID(id string) StartOption
```

Associates the run with a resource ID. Useful for grouping runs and concurrency control.

```go
runID, err := eng.StartWorkflow(ctx, wf, input,
    gorkflow.WithResourceID("user-123"),
)
```

#### `WithConcurrencyCheck`

```go
func WithConcurrencyCheck(check bool) StartOption
```

Enables concurrency checking for the given resource ID.

#### `WithTags`

```go
func WithTags(tags map[string]string) StartOption
```

Sets custom metadata tags on the workflow run. Note: This is `gorkflow.WithTags` from `workflow.go`, which sets tags at run-start time (distinct from the builder's `WithTags` which sets tags at build time).

```go
runID, err := eng.StartWorkflow(ctx, wf, input,
    gorkflow.WithTags(map[string]string{"triggered_by": "api"}),
)
```

### StartOptions

```go
type StartOptions struct {
    ResourceID       string
    CheckConcurrency bool
    Tags             map[string]string
    Synchronous      bool
}
```

## Querying Runs

### `GetRun`

```go
func (e *Engine) GetRun(ctx context.Context, runID string) (*gorkflow.WorkflowRun, error)
```

Retrieves the current state of a workflow run.

```go
run, err := eng.GetRun(ctx, runID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Status: %s, Progress: %.0f%%\n", run.Status, run.Progress*100)
```

### `GetStepExecutions`

```go
func (e *Engine) GetStepExecutions(ctx context.Context, runID string) ([]*gorkflow.StepExecution, error)
```

Retrieves all step executions for a run, sorted by execution index.

```go
executions, err := eng.GetStepExecutions(ctx, runID)
for _, exec := range executions {
    fmt.Printf("Step %s: %s (attempt %d, %dms)\n",
        exec.StepID, exec.Status, exec.Attempt, exec.DurationMs)
}
```

### `ListRuns`

```go
func (e *Engine) ListRuns(ctx context.Context, filter gorkflow.RunFilter) ([]*gorkflow.WorkflowRun, error)
```

Lists workflow runs matching the given filter criteria.

```go
// List recent failed runs
failedStatus := gorkflow.RunStatusFailed
runs, err := eng.ListRuns(ctx, gorkflow.RunFilter{
    WorkflowID: "order-processing",
    Status:     &failedStatus,
    Limit:      10,
})
```

### RunFilter

```go
type RunFilter struct {
    WorkflowID string
    Status     *RunStatus
    ResourceID string
    Limit      int
}
```

All filter fields are optional. An empty filter returns all runs.

| Field | Description |
|-------|-------------|
| `WorkflowID` | Filter by workflow ID |
| `Status` | Filter by run status (pointer; `nil` means any status) |
| `ResourceID` | Filter by resource ID |
| `Limit` | Maximum number of runs to return |

## Cancellation

### `Cancel`

```go
func (e *Engine) Cancel(ctx context.Context, runID string) error
```

Cancels a running workflow. Returns an error if the workflow is already in a terminal state (`COMPLETED`, `FAILED`, or `CANCELLED`).

```go
err := eng.Cancel(ctx, runID)
if err != nil {
    log.Printf("Could not cancel: %v", err)
}
```

See [Cancellation](../advanced-usage/cancellation.md) for details on how cancellation propagates.

## Run Status Values

```go
const (
    RunStatusPending   RunStatus = "PENDING"
    RunStatusRunning   RunStatus = "RUNNING"
    RunStatusCompleted RunStatus = "COMPLETED"
    RunStatusFailed    RunStatus = "FAILED"
    RunStatusCancelled RunStatus = "CANCELLED"
)
```

Use `status.IsTerminal()` to check if a run is in a final state.

## Step Status Values

```go
const (
    StepStatusPending   StepStatus = "PENDING"
    StepStatusRunning   StepStatus = "RUNNING"
    StepStatusCompleted StepStatus = "COMPLETED"
    StepStatusFailed    StepStatus = "FAILED"
    StepStatusSkipped   StepStatus = "SKIPPED"
    StepStatusRetrying  StepStatus = "RETRYING"
)
```

---

**Next**: Learn about [Retry Strategies](../advanced-usage/retry-strategies.md) →
