# Execution Flow

How workflows execute from start to completion.

## Workflow Lifecycle

```
┌─────────────┐
│   PENDING   │ ← StartWorkflow() called, run created
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   RUNNING   │ ← Steps executing
└──────┬──────┘
       │
       ├────────→ CANCELLED (Cancel() called)
       │
       ├────────→ FAILED (Step failed, no ContinueOnError)
       │
       ▼
┌─────────────┐
│  COMPLETED  │ ← All steps finished successfully
└─────────────┘
```

## Execution Sequence

### 1. Workflow Start

```go
runID, err := engine.StartWorkflow(ctx, workflow, input, options...)
```

**What happens:**
1. Generate unique run ID
2. Serialize input to JSON
3. Serialize custom context (if any)
4. Create `WorkflowRun` record with PENDING status
5. Persist to store
6. Launch execution (async or sync based on options)

### 2. Execution Initialization

```go
// Internal: executeWorkflow()
```

**What happens:**
1. Update status to RUNNING
2. Create step data accessor (for inter-step communication)
3. Create state accessor (for workflow state)
4. Get execution order via topological sort
5. Begin step iteration

### 3. Step Execution Loop

For each step in topological order:

```
┌─────────────────────────────────────────────────────────┐
│                    Step Execution                        │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  1. Check for cancellation                               │
│     └─ If cancelled → cancelWorkflow()                   │
│                                                          │
│  2. Resolve step input                                   │
│     ├─ First step → workflow input                       │
│     └─ Other steps → previous step output                │
│                                                          │
│  3. Create StepExecution record (PENDING)                │
│                                                          │
│  4. Build StepContext                                    │
│     ├─ Logger (enriched with step info)                  │
│     ├─ Data accessor                                     │
│     ├─ State accessor                                    │
│     └─ Custom context                                    │
│                                                          │
│  5. Execute with retry loop                              │
│     ├─ Apply backoff delay (if retry)                    │
│     ├─ Update status to RUNNING                          │
│     ├─ Create timeout context                            │
│     ├─ Execute handler (with panic recovery)             │
│     ├─ Check result                                      │
│     │   ├─ Success → break loop                          │
│     │   ├─ Skipped → break loop                          │
│     │   └─ Error → retry or fail                         │
│     └─ Log attempt result                                │
│                                                          │
│  6. Persist step output                                  │
│                                                          │
│  7. Update workflow progress                             │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 4. Step Retry Logic

```
Attempt 0 (initial):
├─ Execute handler
├─ Success? → Done
└─ Error? → Continue to retry

Attempt 1..N (retries):
├─ Calculate backoff delay
│   ├─ LINEAR: baseDelay * attempt
│   ├─ EXPONENTIAL: baseDelay * 2^(attempt-1)
│   └─ NONE: 0
├─ Sleep for delay
├─ Execute handler
├─ Success? → Done
└─ Error? → Continue or fail if max retries reached
```

### 5. Workflow Completion

**Success path:**
```go
// All steps completed
completeWorkflow(ctx, run)
```
- Set status to COMPLETED
- Set progress to 1.0
- Record completion time
- Log completion with duration

**Failure path:**
```go
// Step failed (and ContinueOnError = false)
failWorkflow(ctx, run, err)
```
- Set status to FAILED
- Record error details
- Log failure

**Cancellation path:**
```go
// Cancel() called or context cancelled
cancelWorkflow(ctx, run)
```
- Set status to CANCELLED
- Log cancellation

## Input/Output Flow

```
Workflow Input
     │
     ▼
┌─────────┐     ┌─────────┐     ┌─────────┐
│  Step 1 │────▶│  Step 2 │────▶│  Step 3 │
│         │     │         │     │         │
│ In: WF  │     │ In: S1  │     │ In: S2  │
│ Out: S1 │     │ Out: S2 │     │ Out: S3 │
└─────────┘     └─────────┘     └─────────┘
                                     │
                                     ▼
                              Workflow Output
                              (Last step output)
```

**Key points:**
- First step receives workflow input
- Each subsequent step receives previous step's output
- Workflow output is the last step's output
- All outputs are persisted for debugging/recovery

## Parallel Execution Flow

```
         Step A
           │
     ┌─────┴─────┐
     ▼           ▼
  Step B      Step C
     │           │
     └─────┬─────┘
           ▼
         Step D
```

**Current behavior:**
- Steps B and C both receive Step A's output
- Steps execute in topological order (not truly concurrent yet)
- Step D executes after both B and C complete

## Conditional Execution Flow

```go
condition := func(ctx *StepContext) (bool, error) {
    return shouldExecute, nil
}
```

**When condition returns true:**
- Step executes normally
- Output is step's actual output

**When condition returns false:**
- Step is skipped (status: SKIPPED or COMPLETED with pass-through)
- If input type == output type: input passes through as output
- Otherwise: default value or zero value used

## State Management During Execution

```go
// In step handler
func handler(ctx *StepContext, input MyInput) (MyOutput, error) {
    // Read state
    var counter int
    ctx.State.Get("counter", &counter)
    
    // Write state
    ctx.State.Set("counter", counter + 1)
    
    // Access previous step output
    var prevOutput PrevOutput
    ctx.Data.GetOutput("prev-step", &prevOutput)
    
    return MyOutput{}, nil
}
```

**State is:**
- Persisted after each Set() call
- Available to all subsequent steps
- Scoped to the workflow run

## Error Handling Flow

```
Step Error
    │
    ├─ Retries remaining?
    │   ├─ Yes → Apply backoff, retry
    │   └─ No → Check ContinueOnError
    │
    └─ ContinueOnError?
        ├─ Yes → Log warning, continue to next step
        └─ No → Fail workflow
```

---

**Next**: Learn about [Storage Layer](storage-layer.md) →
