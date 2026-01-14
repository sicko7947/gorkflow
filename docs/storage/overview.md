# Storage Backends Overview

Gorkflow supports multiple storage backends for persisting workflow state. Choose the right backend for your use case.

## Available Backends

| Backend           | Use Case                | Persistence | Scalability     | Setup Complexity |
| ----------------- | ----------------------- | ----------- | --------------- | ---------------- |
| **Memory**        | Development, Testing    | None        | Single Instance | None             |
| **LibSQL/SQLite** | Small-Medium Apps, Edge | File/Remote | Medium          | Low              |

## Memory Store

In-memory storage for development and testing.

### Features

✅ **Zero setup** - No configuration required  
✅ **Fast** - All data in memory  
✅ **Simple** - Perfect for development  
❌ **No persistence** - Data lost on restart  
❌ **Single instance** - Cannot scale horizontally

### Usage

```go
import "github.com/sicko7947/gorkflow/store"

// Create store
store := store.NewMemoryStore()

// Use with engine
eng := engine.NewEngine(store)
```

### When to Use

- Development and testing
- Proof of concepts
- Ephemeral workflows
- Unit/integration tests

See [Memory Store](memory-store.md) for details.

## LibSQL/SQLite Store

File-based or remote SQLite database using LibSQL.

### Features

✅ **File-based persistence** - Data survives restarts  
✅ **Remote capable** - Works with Turso  
✅ **Easy setup** - Single file database  
✅ **SQL queries** - Direct database access  
✅ **Serverless friendly** - Edge computing support  
↔️ **Medium scalability** - Good for small-medium workloads

### Usage

```go
import "github.com/sicko7947/gorkflow/store"

// Local SQLite file
store, err := store.NewLibSQLStore("file:./workflows.db")

// Remote Turso database
store, err := store.NewLibSQLStore("libsql://my-db.turso.io?authToken=...")
```

### When to Use

- Small to medium applications
- Edge computing / serverless
- Single-region deployments
- Direct SQL access needed
- Cost-sensitive projects

See [LibSQL Store](libsql-store.md) for details.

## Comparison

### Performance

| Backend         | Write Latency | Read Latency | Throughput |
| --------------- | ------------- | ------------ | ---------- |
| Memory          | ~0.01ms       | ~0.01ms      | Very High  |
| LibSQL (Local)  | ~1-5ms        | ~0.1-1ms     | Medium     |
| LibSQL (Remote) | ~10-50ms      | ~10-50ms     | Medium     |

### Cost

| Backend        | Development | Production (1M workflows/month) |
| -------------- | ----------- | ------------------------------- |
| Memory         | Free        | Free                            |
| LibSQL (Local) | Free        | Free                            |
| LibSQL (Turso) | Free (5GB)  | ~$15-50                         |

### Scaling

```
Memory Store
  └─ Single Instance
     ├─ Can't scale horizontally
     └─ Limited by RAM

LibSQL Store
  └─ File-based or Remote
     ├─ Single writer (local file)
     ├─ Multiple readers (Turso)
     └─ Regional scaling
```

## Choosing a Backend

### Development

```go
// Use memory store for local development
store := store.NewMemoryStore()
```

### Small Application (< 10k workflows/day)

```go
// Use LibSQL with local file
store, _ := store.NewLibSQLStore("file:./workflows.db")
```

### Medium Application (10k-100k workflows/day)

```go
// Use LibSQL with Turso for reliability
store, _ := store.NewLibSQLStore("libsql://my-db.turso.io?authToken=...")
```

## Migration Between Backends

You can migrate between backends by:

1. Running workflows in both stores temporarily
2. Exporting/importing data
3. Using a migration script

Example migration from Memory to LibSQL:

```go
// Old development setup
memStore := store.NewMemoryStore()

// New production setup
libsqlStore, _ := store.NewLibSQLStore("file:./workflows.db")

// Use libsqlStore instead of memStore
eng := engine.NewEngine(libsqlStore)
```

## Custom Store Implementation

Implement your own storage backend:

```go
type CustomStore struct {
    // Your implementation
}

func (s *CustomStore) SaveRun(ctx context.Context, run *schema.WorkflowRun) error {
    // Implement
}

func (s *CustomStore) GetRun(ctx context.Context, runID string) (*schema.WorkflowRun, error) {
    // Implement
}

// ... implement all Store interface methods
```

See [Custom Store](custom-store.md) for details.

## Best Practices

### 1. Match Backend to Use Case

Development → Memory  
Small/Medium App → LibSQL

### 2. Plan for Growth

Start with LibSQL for persistence needs.

### 3. Test with Production Backend

Test with the same backend you'll use in production.

### 4. Monitor Storage Costs

Track storage usage and costs, especially with cloud backends.

### 5. Implement Cleanup

Remove old workflow runs to manage storage:

```go
// LibSQL: Direct SQL cleanup
db.Exec("DELETE FROM workflow_runs WHERE created_at < ?", cutoffDate)
```

## Environment-Based Configuration

```go
func initStore(env string) (store.Store, error) {
    switch env {
    case "development":
        return store.NewMemoryStore(), nil
    case "staging", "production":
        return store.NewLibSQLStore("file:./workflows.db")
    default:
        return store.NewMemoryStore(), nil
    }
}
```

---

**Next**: Explore specific backends:

- [Memory Store](memory-store.md)
- [LibSQL Store](libsql-store.md)
