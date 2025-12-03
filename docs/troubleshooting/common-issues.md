# Troubleshooting

## Common Issues

### Validation Errors

**Problem**: Getting validation errors even though data looks correct

**Solution**:

- Check that validation tags match the data format
- Use `omitempty` for optional fields
- Verify JSON tags match your input

```go
type Input struct {
    Email string `json:"email" validate:"required,email"`  // Must be valid email
    Age   *int   `json:"age" validate:"omitempty,gte=0"`  // Optional, but if present must be >= 0
}
```

### Type Mismatch Errors

**Problem**: `cannot use handler (type func...) as type StepHandler`

**Solution**: Ensure your handler signature matches exactly:

```go
// Correct signature
func(ctx *gorkflow.StepContext, input TInput) (TOutput, error)

// Common mistake - wrong context type
func(ctx context.Context, input TInput) (TOutput, error)  // ❌ Wrong!
```

### Workflow Build Errors

**Problem**: `invalid workflow graph: cycle detected`

**Solution**: Workflows must be directed acyclic graphs (DAGs). Check for circular dependencies:

```go
// ❌ Creates a cycle
.ThenStep(stepA).
 ThenStep(stepB).
 ThenStep(stepA)  // Can't go back to stepA!

// ✅ Correct
.ThenStep(stepA).
 ThenStep(stepB).
 ThenStep(stepC)
```

### Store Connection Errors

**DynamoDB**:

```bash
# Verify credentials
aws sts get-caller-identity

# Check table exists
aws dynamodb describe-table --table-name workflow_executions
```

**LibSQL**:

```bash
# Verify file permissions
ls -l workflows.db

# Test connection
sqlite3 workflows.db "SELECT 1"
```

### Step Execution Timeout

**Problem**: Steps timing out unexpectedly

**Solution**: Increase timeout or optimize step logic

```go
step := gorkflow.NewStep("slow-step", "Slow Step", handler,
    gorkflow.WithTimeout(5 * time.Minute),  // Increase timeout
)
```

### Memory Issues

**Problem**: High memory usage with MemoryStore

**Solution**: Memory store keeps all data in RAM. For production, use DynamoDB or LibSQL:

```go
// Switch to persistent store
store, _ := store.NewLibSQLStore("file:./workflows.db")
```

## Quick Reference

### Check Workflow Status

```go
run, err := eng.GetRun(ctx, runID)
fmt.Printf("Status: %s, Progress: %.0f%%\n", run.Status, run.Progress*100)
if run.Error != nil {
    fmt.Printf("Error: %s\n", *run.Error)
}
```

### Debug Step Execution

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    ctx.Logger.Debug().
        Interface("input", input).
        Msg("Step started")

    // Your logic

    ctx.Logger.Debug().
        Interface("output", output).
        Msg("Step completed")

    return output, nil
}
```

### Test Workflows

```go
func TestMyWorkflow(t *testing.T) {
    store := store.NewMemoryStore()
    eng := engine.NewEngine(store)

    wf, _ := NewMyWorkflow()
    runID, err := eng.StartWorkflow(context.Background(), wf, input)

    if err != nil {
        t.Fatal(err)
    }

    run, _ := eng.GetRun(context.Background(), runID)
    if run.Status != "completed" {
        t.Errorf("Expected completed, got %s", run.Status)
    }
}
```

## FAQ

**Q: Can I modify a workflow after it's built?**  
A: No, workflows are immutable after `Build()`. Create a new workflow for changes.

**Q: How do I handle large payloads?**  
A: Store large data externally (S3, etc.) and pass references through steps.

**Q: Can steps run in different processes?**  
A: No, all steps run in the same process. For distributed execution, consider using a message queue.

**Q: How do I version workflows?**  
A: Use the `WithVersion()` builder method and include version in workflow ID:

```go
gorkflow.NewWorkflow("user-registration-v2", "User Registration").
    WithVersion("2.0.0")
```

**Q: What happens if the process crashes mid-workflow?**  
A: With persistent storage (DynamoDB/LibSQL), you can implement recovery logic. The workflow state is saved after each step.

**Q: How many parallel steps can run?**  
A: Limited by system resources. The engine manages concurrency automatically.

---

For more help, see [Debugging](debugging.md) or [open an issue](https://github.com/sicko7947/gorkflow/issues) →
