# LibSQL Store

SQLite-compatible storage using LibSQL, supporting both local files and remote Turso databases.

## Overview

The LibSQL store provides lightweight, serverless-friendly persistence using SQLite or Turso (remote LibSQL).

## Features

- ✅ **File-based** - Simple `.db` file storage
- ✅ **Remote capable** - Works with Turso cloud database
- ✅ **Zero configuration** - Auto-creates schema
- ✅ **SQL access** - Query directly with SQL
- ✅ **Edge-friendly** - Perfect for serverless/edge
- ✅ **Cost-effective** - Free for small workloads
- ✅ **Embeddable** - No separate database server

## Installation

```bash
go get github.com/tursodatabase/libsql-client-go/libsql
```

## Quick Start

### Local SQLite File

```go
import "github.com/sicko7947/gorkflow/store"

// Create store with local file
store, err := store.NewLibSQLStore("file:./workflows.db")
if err != nil {
    panic(err)
}

// Use with engine
eng := engine.NewEngine(store)
```

### Remote Turso Database

```go
// Connect to Turso
store, err := store.NewLibSQLStore(
    "libsql://my-database-user.turso.io?authToken=your-auth-token",
)
```

## Local Setup

### Create Local Database

```go
package main

import (
    "github.com/sicko7947/gorkflow/store"
    "github.com/sicko7947/gorkflow/engine"
)

func main() {
    // Create/open local database file
    store, err := store.NewLibSQLStore("file:./workflows.db")
    if err != nil {
        panic(err)
    }

    // Tables are created automatically!
    eng := engine.NewEngine(store)
}
```

The database file `workflows.db` will be created automatically if it doesn't exist.

## Turso Setup

[Turso](https://turso.tech) is a cloud-hosted LibSQL database.

### 1. Install Turso CLI

```bash
# macOS/Linux
curl -sSfL https://get.tur.so/install.sh | bash

# Or with Homebrew
brew install tursodatabase/tap/turso
```

### 2. Sign Up

```bash
turso auth signup
```

### 3. Create Database

```bash
# Create database
turso db create gorkflow-workflows

# Get connection URL
turso db show gorkflow-workflows --url

# Create auth token
turso db tokens create gorkflow-workflows
```

### 4. Use in Gorkflow

```go
store, err := store.NewLibSQLStore(
    "libsql://gorkflow-workflows-user.turso.io?authToken=eyJhbGci...",
)
```

## Database Schema

The LibSQL store uses four tables:

### workflow_runs

```sql
CREATE TABLE workflow_runs (
    run_id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    workflow_name TEXT NOT NULL,
    status TEXT NOT NULL,
    input BLOB,
    output BLOB,
    error TEXT,
    progress REAL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    completed_at INTEGER,
    tags TEXT
);

CREATE INDEX idx_workflow_runs_workflow_id ON workflow_runs(workflow_id);
CREATE INDEX idx_workflow_runs_status ON workflow_runs(status);
CREATE INDEX idx_workflow_runs_created_at ON workflow_runs(created_at);
```

### step_executions

```sql
CREATE TABLE step_executions (
    execution_id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    step_name TEXT NOT NULL,
    status TEXT NOT NULL,
    input BLOB,
    output BLOB,
    error TEXT,
    retry_count INTEGER DEFAULT 0,
    execution_index INTEGER NOT NULL,
    started_at INTEGER,
    completed_at INTEGER,
    FOREIGN KEY (run_id) REFERENCES workflow_runs(run_id)
);

CREATE INDEX idx_step_executions_run_id ON step_executions(run_id);
CREATE INDEX idx_step_executions_run_step ON step_executions(run_id, execution_index);
```

### step_outputs

```sql
CREATE TABLE step_outputs (
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    output BLOB NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (run_id, step_id),
    FOREIGN KEY (run_id) REFERENCES workflow_runs(run_id)
);
```

### workflow_state

```sql
CREATE TABLE workflow_state (
    run_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value BLOB NOT NULL,
    PRIMARY KEY (run_id, key),
    FOREIGN KEY (run_id) REFERENCES workflow_runs(run_id)
);
```

## Configuration

### Connection Strings

**Local File:**

```
file:./workflows.db               # Relative path
file:/absolute/path/workflows.db  # Absolute path
file:workflows.db                 # Current directory
```

**In-Memory (Testing):**

```
file::memory:                     # In-memory database
```

**Remote Turso:**

```
libsql://database-name.turso.io?authToken=<token>
```

### Environment Variables

```bash
export LIBSQL_URL="libsql://my-db.turso.io?authToken=..."
```

```go
url := os.Getenv("LIBSQL_URL")
store, _ := store.NewLibSQLStore(url)
```

## Direct SQL Access

Since LibSQL is SQL-based, you can query directly:

```go
import "database/sql"

// Open database
db, err := sql.Open("libsql", "file:./workflows.db")

// Query workflows
rows, err := db.Query(`
    SELECT run_id, workflow_name, status, created_at
    FROM workflow_runs
    WHERE status = 'completed'
    ORDER BY created_at DESC
    LIMIT 10
`)

// Query step execution times
rows, err := db.Query(`
    SELECT
        step_name,
        AVG(completed_at - started_at) as avg_duration
    FROM step_executions
    WHERE status = 'completed'
    GROUP BY step_name
`)
```

## Cleanup and Maintenance

### Delete Old Workflows

```go
import "database/sql"

db, _ := sql.Open("libsql", "file:./workflows.db")

// Delete workflows older than 30 days
cutoff := time.Now().Add(-30 * 24 * time.Hour).Unix()
db.Exec("DELETE FROM workflow_runs WHERE created_at < ?", cutoff)
```

### Vacuum Database

```go
// Reclaim space from deleted records
db.Exec("VACUUM")
```

### Backup

**Local File:**

```bash
# Simple file copy
cp workflows.db workflows.backup.db

# Or use SQLite backup
sqlite3 workflows.db ".backup workflows.backup.db"
```

**Turso:**

```bash
# Turso handles backups automatically
# View backups:
turso db show gorkflow-workflows

# Restore from backup:
turso db restore gorkflow-workflows <timestamp>
```

## Turso Features

### Free Tier

- 9 GB total storage
- 500 databases
- 1 billion row reads/month
- 25 million row writes/month

### Replication

```bash
# Create database with replication
turso db create gorkflow-workflows --location ord --location fra

# Automatic read replication to closest region
```

### Branching

```bash
# Create development branch
turso db create gorkflow-dev --from-db gorkflow-workflows

# Test changes in dev environment
```

## Performance

### Benchmarks

| Operation | Local File | Turso (Same Region) |
| --------- | ---------- | ------------------- |
| Write Run | ~1-2ms     | ~15-30ms            |
| Read Run  | ~0.5-1ms   | ~10-20ms            |
| List Runs | ~2-5ms     | ~15-40ms            |

### Optimization

**Use Indexes:**

```sql
CREATE INDEX idx_custom ON workflow_runs(workflow_id, status);
```

**Batch Inserts:**

```go
// Store handles batching automatically
```

**Connection Pooling:**

```go
// LibSQL handles connection pooling
```

## Best Practices

### 1. Use Local Files for Development

```go
store, _ := store.NewLibSQLStore("file:./dev-workflows.db")
```

### 2. Use Turso for Production

```go
store, _ := store.NewLibSQLStore(os.Getenv("TURSO_URL"))
```

### 3. Regular Backups

```bash
# Cron job for local file backups
0 2 * * * cp /app/workflows.db /backups/workflows-$(date +\%Y\%m\%d).db
```

### 4. Monitor Database Size

```go
import "database/sql"

var size int64
db.QueryRow("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").Scan(&size)
fmt.Printf("Database size: %d bytes\n", size)
```

### 5. Clean Up Old Data

```go
// Delete completed workflows older than 90 days
cutoff := time.Now().Add(-90 * 24 * time.Hour).Unix()
db.Exec("DELETE FROM workflow_runs WHERE status = 'completed' AND created_at < ?", cutoff)
```

## Migration from SQLite

LibSQL is compatible with SQLite:

```bash
# Use existing SQLite database
store, _ := store.NewLibSQLStore("file:./existing-sqlite.db")
```

## Troubleshooting

### File Permission Errors

```bash
# Ensure write permissions
chmod 644 workflows.db
chmod 755 $(dirname workflows.db)
```

### Connection Errors (Turso)

```bash
# Verify connection
turso db shell gorkflow-workflows

# Check auth token
turso db tokens list gorkflow-workflows
```

### Database Locked

```bash
# Check for long-running transactions
# Reduce concurrent writes
# Use WAL mode (enabled by default)
```

---

**Next**: Learn about [Memory Store](memory-store.md) or return to [Storage Overview](overview.md) →
