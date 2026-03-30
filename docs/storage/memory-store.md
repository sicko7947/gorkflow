# Memory Store

The `MemoryStore` is an in-memory implementation of the `WorkflowStore` interface, designed for testing and development.

## Overview

`MemoryStore` stores all workflow data in Go maps protected by a `sync.RWMutex`. Data does not persist between process restarts.

## Creating a Memory Store

```go
import "github.com/sicko7947/gorkflow/store"

memStore := store.NewMemoryStore()
```

The returned value implements `gorkflow.WorkflowStore` and can be passed to `engine.NewEngine`.

## Usage with Engine

```go
import (
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
)

memStore := store.NewMemoryStore()
eng := engine.NewEngine(memStore)

runID, err := eng.StartWorkflow(ctx, wf, input)
```

## Characteristics

| Property | Behavior |
|----------|----------|
| **Persistence** | In-memory only; lost on process exit |
| **Thread safety** | `sync.RWMutex` for concurrent access |
| **Deep copying** | All reads/writes deep-copy data to prevent mutation through references |
| **Sorting** | `ListRuns` returns results sorted by `CreatedAt` descending; `ListStepExecutions` sorted by `ExecutionIndex` ascending |

## Deep Copy Behavior

`MemoryStore` deep-copies all data on read and write operations. This prevents callers from accidentally mutating stored data through shared pointers:

```go
// The returned run is a deep copy — modifying it won't affect the store
run, _ := memStore.GetRun(ctx, runID)
run.Status = gorkflow.RunStatusFailed // Does NOT change the stored run
```

Deep-copied fields include:
- `Tags` map
- `Input`, `Output`, `Context` byte slices
- `Error` struct and its `Details` map
- `StartedAt`, `CompletedAt` time pointers
- Step output byte slices
- State value byte slices

## When to Use

- **Unit tests** — fast, no external dependencies
- **Integration tests** — test workflow logic without a database
- **Development** — quick iteration without database setup
- **Prototyping** — get started without configuring persistence

## When NOT to Use

- **Production** — data is lost on restart
- **Multi-process** — no shared state between processes
- **Large datasets** — everything lives in memory

For production use, see [LibSQL Store](libsql-store.md) or implement a [Custom Store](custom-store.md).

## Example: Testing a Workflow

```go
func TestOrderWorkflow(t *testing.T) {
    memStore := store.NewMemoryStore()
    eng := engine.NewEngine(memStore)

    wf, err := gorkflow.NewWorkflow("test-order", "Test Order").
        ThenStep(validateStep).
        ThenStep(processStep).
        Build()
    require.NoError(t, err)

    input := OrderInput{OrderID: "test-123", Amount: 99.99}
    runID, err := eng.StartWorkflow(ctx, wf, input,
        gorkflow.WithSynchronousExecution(),
    )
    require.NoError(t, err)

    // Check result
    run, err := eng.GetRun(ctx, runID)
    require.NoError(t, err)
    assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)

    // Inspect step executions
    executions, err := eng.GetStepExecutions(ctx, runID)
    require.NoError(t, err)
    assert.Len(t, executions, 2)
}
```

---

**Next**: Learn about the [LibSQL Store](libsql-store.md) or how to build a [Custom Store](custom-store.md) →
