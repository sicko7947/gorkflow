# Store Interface

The `WorkflowStore` interface defines all persistence operations required by the Gorkflow engine.

## WorkflowStore Interface

```go
type WorkflowStore interface {
    // Workflow runs
    CreateRun(ctx context.Context, run *WorkflowRun) error
    GetRun(ctx context.Context, runID string) (*WorkflowRun, error)
    UpdateRun(ctx context.Context, run *WorkflowRun) error
    UpdateRunStatus(ctx context.Context, runID string, status RunStatus, err *WorkflowError) error
    ListRuns(ctx context.Context, filter RunFilter) ([]*WorkflowRun, error)

    // Step executions
    CreateStepExecution(ctx context.Context, exec *StepExecution) error
    GetStepExecution(ctx context.Context, runID, stepID string) (*StepExecution, error)
    UpdateStepExecution(ctx context.Context, exec *StepExecution) error
    ListStepExecutions(ctx context.Context, runID string) ([]*StepExecution, error)

    // Step outputs (for inter-step communication)
    SaveStepOutput(ctx context.Context, runID, stepID string, output []byte) error
    LoadStepOutput(ctx context.Context, runID, stepID string) ([]byte, error)

    // Workflow state
    SaveState(ctx context.Context, runID, key string, value []byte) error
    LoadState(ctx context.Context, runID, key string) ([]byte, error)
    DeleteState(ctx context.Context, runID, key string) error
    GetAllState(ctx context.Context, runID string) (map[string][]byte, error)

    // Queries
    CountRunsByStatus(ctx context.Context, resourceID string, status RunStatus) (int, error)
}
```

## Method Groups

### Workflow Runs

#### `CreateRun`

```go
CreateRun(ctx context.Context, run *WorkflowRun) error
```

Persists a new workflow run. Should return an error if a run with the same `RunID` already exists.

#### `GetRun`

```go
GetRun(ctx context.Context, runID string) (*WorkflowRun, error)
```

Retrieves a workflow run by ID. Returns `ErrRunNotFound` if not found.

#### `UpdateRun`

```go
UpdateRun(ctx context.Context, run *WorkflowRun) error
```

Updates all fields of an existing workflow run. This is the primary method the engine uses for status transitions, progress updates, and storing output. Returns `ErrRunNotFound` if the run does not exist.

#### `UpdateRunStatus`

```go
UpdateRunStatus(ctx context.Context, runID string, status RunStatus, err *WorkflowError) error
```

Updates only the status and error of a run. Returns `ErrRunNotFound` if not found.

#### `ListRuns`

```go
ListRuns(ctx context.Context, filter RunFilter) ([]*WorkflowRun, error)
```

Lists workflow runs matching the given filter. Results should be sorted by `CreatedAt` descending.

### Step Executions

#### `CreateStepExecution`

```go
CreateStepExecution(ctx context.Context, exec *StepExecution) error
```

Persists a new step execution record. Called before each step starts running.

#### `GetStepExecution`

```go
GetStepExecution(ctx context.Context, runID, stepID string) (*StepExecution, error)
```

Retrieves a step execution by run ID and step ID. Returns `ErrStepExecutionNotFound` if not found.

#### `UpdateStepExecution`

```go
UpdateStepExecution(ctx context.Context, exec *StepExecution) error
```

Updates an existing step execution. Called on status transitions (running, retrying, completed, failed, skipped).

#### `ListStepExecutions`

```go
ListStepExecutions(ctx context.Context, runID string) ([]*StepExecution, error)
```

Returns all step executions for a run, sorted by `ExecutionIndex` ascending.

### Step Outputs

#### `SaveStepOutput`

```go
SaveStepOutput(ctx context.Context, runID, stepID string, output []byte) error
```

Stores the JSON output bytes of a completed step. Used for inter-step data passing.

#### `LoadStepOutput`

```go
LoadStepOutput(ctx context.Context, runID, stepID string) ([]byte, error)
```

Retrieves the output of a step. Returns `ErrStepOutputNotFound` if not found.

### Workflow State

#### `SaveState`

```go
SaveState(ctx context.Context, runID, key string, value []byte) error
```

Stores a key-value pair. Values are JSON-encoded bytes.

#### `LoadState`

```go
LoadState(ctx context.Context, runID, key string) ([]byte, error)
```

Retrieves a state value by key. Returns `ErrStateNotFound` if not found.

#### `DeleteState`

```go
DeleteState(ctx context.Context, runID, key string) error
```

Removes a key from state. Should not return an error if the key doesn't exist.

#### `GetAllState`

```go
GetAllState(ctx context.Context, runID string) (map[string][]byte, error)
```

Returns all state key-value pairs for a run. Returns an empty map (not `nil`) if no state exists.

### Queries

#### `CountRunsByStatus`

```go
CountRunsByStatus(ctx context.Context, resourceID string, status RunStatus) (int, error)
```

Counts workflow runs for a given resource ID and status.

## RunFilter

```go
type RunFilter struct {
    WorkflowID string
    Status     *RunStatus
    ResourceID string
    Limit      int
}
```

| Field | Type | Description |
|-------|------|-------------|
| `WorkflowID` | `string` | Filter by workflow ID (empty = all) |
| `Status` | `*RunStatus` | Filter by status (`nil` = any) |
| `ResourceID` | `string` | Filter by resource ID (empty = all) |
| `Limit` | `int` | Max results (`0` = unlimited) |

## Sentinel Errors

Use these sentinel errors in your store implementations for "not found" cases:

```go
var (
    ErrRunNotFound           = errors.New("workflow run not found")
    ErrStepExecutionNotFound = errors.New("step execution not found")
    ErrStepOutputNotFound    = errors.New("step output not found")
    ErrStateNotFound         = errors.New("state not found")
)
```

The engine and accessors check for these specific errors.

## Available Implementations

| Store | Package | Use Case |
|-------|---------|----------|
| [MemoryStore](../storage/memory-store.md) | `github.com/sicko7947/gorkflow/store` | Testing, development |
| [LibSQLStore](../storage/libsql-store.md) | `github.com/sicko7947/gorkflow/store` | Production (local SQLite or remote Turso) |
| [Custom](../storage/custom-store.md) | Your package | Any backend (PostgreSQL, Redis, DynamoDB, etc.) |

---

**Next**: Learn about [Configuration](configuration.md) →
