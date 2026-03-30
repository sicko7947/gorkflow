# Storage Layer

The storage layer provides persistence for workflow runs, step executions, step outputs, and workflow state.

## Architecture

```
┌──────────────────────────┐
│        Engine            │
│  (orchestration logic)   │
└─────────┬────────────────┘
          │ WorkflowStore interface
          ▼
┌──────────────────────────┐
│     Store Backend        │
│  ┌─────────┐ ┌────────┐ │
│  │ Memory  │ │ LibSQL │ │
│  │ Store   │ │ Store  │ │
│  └─────────┘ └────────┘ │
└──────────────────────────┘
```

The engine interacts with storage exclusively through the `WorkflowStore` interface. This decouples orchestration logic from persistence details.

## Interface Abstraction

The `WorkflowStore` interface groups operations into five categories:

| Category | Methods | Purpose |
|----------|---------|---------|
| Workflow Runs | `CreateRun`, `GetRun`, `UpdateRun`, `UpdateRunStatus`, `ListRuns` | Track workflow execution lifecycle |
| Step Executions | `CreateStepExecution`, `GetStepExecution`, `UpdateStepExecution`, `ListStepExecutions` | Track individual step status, timing, errors |
| Step Outputs | `SaveStepOutput`, `LoadStepOutput` | Inter-step data passing |
| Workflow State | `SaveState`, `LoadState`, `DeleteState`, `GetAllState` | Shared key-value state |
| Queries | `CountRunsByStatus` | Operational metrics |

See [Store Interface](../api-reference/store-interface.md) for the full interface definition.

## Data Model

### What Gets Stored

```
WorkflowRun (1 per execution)
├── RunID, WorkflowID, Version, Status, Progress
├── Input/Output (JSON blobs)
├── Error (structured WorkflowError)
├── Tags, ResourceID
└── Timing (CreatedAt, StartedAt, CompletedAt)

StepExecution (1 per step per run)
├── RunID, StepID, ExecutionIndex
├── Status, Attempt
├── Input/Output (JSON blobs)
├── Error (structured StepError)
└── Timing (StartedAt, CompletedAt, DurationMs)

StepOutput (1 per completed step)
└── Output bytes (JSON blob)

WorkflowState (N per run)
└── Key → Value (JSON blob)
```

### JSON Blob Approach

Step inputs, outputs, and state values are stored as opaque JSON byte slices. This means:

- **No schema migrations** when step types change
- **Any Go type** can be stored (as long as it's JSON-serializable)
- **Human-readable** for debugging (inspect with any JSON tool)
- **No column-level queries** on step data (filter at the application layer)

## Built-in Implementations

### MemoryStore

**Package:** `github.com/sicko7947/gorkflow/store`

| Property | Value |
|----------|-------|
| Persistence | In-memory only |
| Concurrency | `sync.RWMutex` |
| Deep copying | All reads/writes copy data |
| Use case | Testing, development |

Key behaviors:
- `ListRuns` sorts by `CreatedAt` descending
- `ListStepExecutions` sorts by `ExecutionIndex` ascending
- All data is deep-copied on read and write to prevent mutation through shared references
- Returns empty collections (not `nil`) when no data exists

See [Memory Store](../storage/memory-store.md).

### LibSQLStore

**Package:** `github.com/sicko7947/gorkflow/store`

| Property | Value |
|----------|-------|
| Persistence | SQLite file or remote Turso |
| Connection pool | Configurable (`MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`) |
| Schema management | Auto-creates tables on initialization |
| Use case | Production |

Key behaviors:
- Supports both local files (`file:./local.db`) and remote Turso (`libsql://...`)
- Performance PRAGMAs applied conditionally (some don't work on remote Turso)
- Uses SQL `json_set` for atomic step execution updates
- Cache size configurable (default ~8MB)

Configuration:

```go
opts := store.LibSQLStoreOptions{
    MaxOpenConns:    10,
    MaxIdleConns:    5,
    ConnMaxLifetime: time.Hour,
    CacheSize:       -2000, // 2000 pages
}
store, err := store.NewLibSQLStoreWithOptions("file:./workflow.db", opts)
```

See [LibSQL Store](../storage/libsql-store.md).

## MemoryStore vs LibSQLStore

| Aspect | MemoryStore | LibSQLStore |
|--------|------------|-------------|
| Speed | Fastest (no I/O) | Fast (local SQLite) to moderate (remote Turso) |
| Durability | None (lost on exit) | Full (persisted to file/remote) |
| Multi-process | No | Yes (remote Turso) |
| Setup | Zero | Minimal (file path or Turso URL) |
| Debugging | In-memory only | Query with SQL tools |
| Testing | Ideal | Use for integration tests |

## Caching

The `stepAccessor` and `stateAccessor` (in `context.go`) maintain in-memory caches:

- **Step outputs** — cached after first load from store
- **Step inputs** — cached after first load from store
- **State values** — cached after first load; updated on `Set`

This reduces store calls during a single run, since a step's output may be read multiple times (by conditions, by downstream steps, etc.).

## Custom Implementations

Implement `WorkflowStore` to use any backend. Key considerations:

1. Return sentinel errors (`ErrRunNotFound`, `ErrStepOutputNotFound`, etc.) for missing resources
2. Pass `context.Context` to database calls for timeout/cancellation propagation
3. Ensure thread safety (the engine may call methods concurrently)
4. Sort `ListStepExecutions` by `ExecutionIndex` ascending
5. Sort `ListRuns` by `CreatedAt` descending

See [Custom Store](../storage/custom-store.md) for a full implementation guide.

---

**See also**: [Design Decisions](design-decisions.md) | [System Overview](system-overview.md)
