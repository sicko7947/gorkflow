# Custom Store

Implement the `WorkflowStore` interface to use any storage backend with Gorkflow.

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

| Method | Description |
|--------|-------------|
| `CreateRun` | Persist a new `WorkflowRun`. Should fail if the run ID already exists. |
| `GetRun` | Retrieve a run by ID. Return `ErrRunNotFound` if not found. |
| `UpdateRun` | Update all fields of an existing run. Return `ErrRunNotFound` if not found. |
| `UpdateRunStatus` | Update only the status and error fields of a run. |
| `ListRuns` | List runs matching the `RunFilter`. Apply `WorkflowID`, `Status`, `ResourceID`, and `Limit` filters. |

### Step Executions

| Method | Description |
|--------|-------------|
| `CreateStepExecution` | Persist a new step execution record. |
| `GetStepExecution` | Retrieve a step execution by run ID and step ID. Return `ErrStepExecutionNotFound` if not found. |
| `UpdateStepExecution` | Update an existing step execution record. |
| `ListStepExecutions` | List all step executions for a run, sorted by `ExecutionIndex`. |

### Step Outputs

| Method | Description |
|--------|-------------|
| `SaveStepOutput` | Store the JSON output bytes of a completed step. |
| `LoadStepOutput` | Retrieve the output of a step. Return `ErrStepOutputNotFound` if not found. |

### Workflow State

| Method | Description |
|--------|-------------|
| `SaveState` | Store a key-value pair (value is JSON bytes). |
| `LoadState` | Retrieve a value by key. Return `ErrStateNotFound` if not found. |
| `DeleteState` | Remove a key from state. Should not error if key doesn't exist. |
| `GetAllState` | Return all state key-value pairs for a run. Return empty map if no state exists. |

### Queries

| Method | Description |
|--------|-------------|
| `CountRunsByStatus` | Count runs for a resource ID with a given status. |

## Sentinel Errors

Use these sentinel errors for "not found" cases:

```go
var (
    ErrRunNotFound           = errors.New("workflow run not found")
    ErrStepExecutionNotFound = errors.New("step execution not found")
    ErrStepOutputNotFound    = errors.New("step output not found")
    ErrStateNotFound         = errors.New("state not found")
)
```

## Implementation Example

Here's a skeleton for a PostgreSQL-backed store:

```go
package pgstore

import (
    "context"
    "database/sql"

    "github.com/sicko7947/gorkflow"
)

type PostgresStore struct {
    db *sql.DB
}

func NewPostgresStore(db *sql.DB) gorkflow.WorkflowStore {
    return &PostgresStore{db: db}
}

func (s *PostgresStore) CreateRun(ctx context.Context, run *gorkflow.WorkflowRun) error {
    _, err := s.db.ExecContext(ctx,
        `INSERT INTO workflow_runs (run_id, workflow_id, workflow_version, status, progress, created_at, updated_at, input, resource_id)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
        run.RunID, run.WorkflowID, run.WorkflowVersion,
        run.Status, run.Progress, run.CreatedAt, run.UpdatedAt,
        run.Input, run.ResourceID,
    )
    return err
}

func (s *PostgresStore) GetRun(ctx context.Context, runID string) (*gorkflow.WorkflowRun, error) {
    run := &gorkflow.WorkflowRun{}
    err := s.db.QueryRowContext(ctx,
        `SELECT run_id, workflow_id, workflow_version, status, progress,
                created_at, started_at, completed_at, updated_at,
                input, output, resource_id
         FROM workflow_runs WHERE run_id = $1`, runID,
    ).Scan(
        &run.RunID, &run.WorkflowID, &run.WorkflowVersion,
        &run.Status, &run.Progress,
        &run.CreatedAt, &run.StartedAt, &run.CompletedAt, &run.UpdatedAt,
        &run.Input, &run.Output, &run.ResourceID,
    )
    if err == sql.ErrNoRows {
        return nil, gorkflow.ErrRunNotFound
    }
    return run, err
}

// ... implement remaining methods
```

## Implementation Guidelines

1. **Use the context** — all methods receive a `context.Context`. Pass it to database calls so timeouts and cancellation propagate.

2. **Return sentinel errors** — use `ErrRunNotFound`, `ErrStepExecutionNotFound`, etc. for missing resources. The engine checks for these.

3. **Handle concurrency** — the engine may call store methods from multiple goroutines. Ensure your implementation is thread-safe.

4. **JSON bytes** — `Input`, `Output`, and state values are `[]byte` containing JSON. Store them as-is (BLOB, JSONB, TEXT) without re-encoding.

5. **Sort `ListStepExecutions`** — return results sorted by `ExecutionIndex` ascending.

6. **Sort `ListRuns`** — return results sorted by `CreatedAt` descending.

## Using Your Custom Store

```go
pgStore := pgstore.NewPostgresStore(db)
eng := engine.NewEngine(pgStore)

// Use the engine normally
runID, err := eng.StartWorkflow(ctx, wf, input)
```

---

**See also**: [Memory Store](memory-store.md) | [LibSQL Store](libsql-store.md)
