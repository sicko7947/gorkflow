# LibSQL Persistence Example

This example demonstrates how to use the LibSQL store for workflow persistence with a local SQLite database file.

## Overview

This example shows:

- Creating a LibSQL store with a local file (`workflow.db`)
- Running workflows with persistent storage
- Retrieving workflow runs and step executions from the database
- Demonstrating that data persists across multiple workflow executions

## Workflow

The example implements a simple math workflow:

1. **Add**: Takes two numbers and adds them
2. **Multiply**: Multiplies the result by 2
3. **Format**: Formats the final output

Example: `(5 + 3) * 2 = 16`

## Running the Example

```bash
# From the example/libsql_persistence directory
go run main.go
```

## What Happens

1. Creates a local SQLite database file: `workflow.db`
2. Runs the first workflow: `(5 + 3) * 2 = 16`
3. Displays step execution history from the database
4. Runs a second workflow: `(10 + 20) * 2 = 60`
5. Lists all workflow runs stored in the database

## Key Features Demonstrated

### LibSQL Store Initialization

```go
dbPath := "file:./workflow.db"
libsqlStore, err := store.NewLibSQLStore(dbPath)
if err != nil {
    log.Fatal("Failed to create LibSQL store:", err)
}
defer libsqlStore.Close()
```

### Persistent Storage

All workflow runs and step executions are automatically saved to the database and can be retrieved later:

```go
// Retrieve workflow run
run, err := eng.GetRun(ctx, runID)

// List all step executions (sorted by execution_index)
steps, err := libsqlStore.ListStepExecutions(ctx, runID)

// List all workflow runs
runs, err := libsqlStore.ListRuns(ctx, workflow.RunFilter{
    WorkflowID: "simple_math",
})
```

## Database Inspection

After running the example, you can inspect the database directly:

```bash
# Open the database with sqlite3
sqlite3 workflow.db

# View tables
.tables

# View workflow runs
SELECT run_id, workflow_id, status, created_at FROM workflow_runs;

# View step executions (sorted by execution_index)
SELECT run_id, step_id, execution_index, status FROM step_executions ORDER BY execution_index;

# Exit
.quit
```

## Using Remote LibSQL (Turso)

To use a remote Turso database instead of a local file:

```go
// Replace the dbPath with your Turso database URL
dbPath := "libsql://your-database.turso.io?authToken=your-token"
libsqlStore, err := store.NewLibSQLStore(dbPath)
```

## Clean Up

To remove the database file:

```bash
rm workflow.db
```

## Benefits of LibSQL Store

- ✅ **Persistent Storage**: Data survives application restarts
- ✅ **Local Development**: Use local SQLite files for development
- ✅ **Production Ready**: Switch to Turso for production
- ✅ **Efficient Sorting**: Step executions are sorted by `execution_index` using database indexes
- ✅ **Query Capabilities**: Full SQL querying support for workflow data
