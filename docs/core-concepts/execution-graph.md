# Execution Graph

The execution graph is a directed acyclic graph (DAG) that defines the order and dependencies of steps in a workflow.

## Overview

Every workflow has an `ExecutionGraph` that determines:

- Which step runs first (entry point)
- Which steps depend on which (edges)
- Which steps can run in parallel (nodes at the same level)
- The overall execution order (topological sort)

## Concepts

### Nodes

Each step in a workflow is represented as a `GraphNode`:

```go
type GraphNode struct {
    StepID   string
    Type     NodeType
    Next     []string   // Outgoing edges (steps that follow)
    Previous []string   // Incoming edges (steps that precede)
}
```

### Node Types

```go
const (
    NodeTypeSequential  NodeType = "SEQUENTIAL"
    NodeTypeParallel    NodeType = "PARALLEL"
    NodeTypeConditional NodeType = "CONDITIONAL"
)
```

- **Sequential** — runs after all predecessors complete, one at a time
- **Parallel** — grouped with siblings at the same level for concurrent execution
- **Conditional** — executes only if a condition is met at runtime

When you use the builder:
- `ThenStep()` adds a `SEQUENTIAL` node
- `Parallel()` adds nodes as `PARALLEL`
- `ThenStepIf()` wraps the step with conditional logic (the node itself is `SEQUENTIAL`)

### Edges

Edges are directed connections from one step to another:

```
A → B    means "B runs after A"
```

Edges are stored as bidirectional references — each node tracks both its `Next` (outgoing) and `Previous` (incoming) edges.

### Entry Point

The first node added to the graph (or explicitly set via `SetEntryPoint`) is the entry point. Execution starts here.

## How the Builder Creates the Graph

```go
wf, _ := gorkflow.NewWorkflow("example", "Example").
    ThenStep(step1).                          // entry point
    Parallel(step2a, step2b, step2c).         // fan-out
    ThenStep(step3).                          // fan-in
    Build()
```

This creates the following graph:

```
step1
├──→ step2a (PARALLEL)
├──→ step2b (PARALLEL)
└──→ step2c (PARALLEL)
      ├──→ step3
      ├──→ step3  (multiple edges converge)
      └──→ step3
```

## Topological Sort

The engine uses topological sorting to determine execution order. This ensures every step runs after all its predecessors.

```go
order, err := graph.TopologicalSort()
// Returns: ["step1", "step2a", "step2b", "step2c", "step3"]
```

The sort is performed using depth-first search (DFS) starting from the entry point. Results are cached — subsequent calls return the cached order until the graph is modified.

### How It Works

1. Start from the entry point
2. Recursively visit all successors (DFS)
3. After visiting all successors of a node, prepend it to the result
4. This produces a valid topological ordering

## Validation

`Build()` calls `graph.Validate()` which checks:

1. **Entry point exists** — the graph must have a valid entry point
2. **No cycles** — DFS-based cycle detection ensures the graph is a DAG
3. **All nodes reachable** — every node must be reachable from the entry point

```go
wf, err := gorkflow.NewWorkflow("my-wf", "My Workflow").
    ThenStep(step1).
    Build()
if err != nil {
    // "execution graph contains cycles"
    // "not all nodes are reachable from entry point"
    // "execution graph has no entry point"
}
```

### Cycle Detection

Uses DFS with a recursion stack. If a node is encountered that is already in the current recursion stack, a cycle exists.

### Reachability Check

BFS/DFS from the entry point counts reachable nodes. If the count doesn't match the total node count, some nodes are disconnected.

## Graph Methods

### Query Methods

| Method | Description |
|--------|-------------|
| `TopologicalSort()` | Returns execution order as `[]string` |
| `GetNextSteps(stepID)` | Returns immediate successors |
| `GetPreviousSteps(stepID)` | Returns immediate predecessors |
| `IsTerminal(stepID)` | Returns `true` if the step has no outgoing edges |
| `Validate()` | Validates graph structure |

### Mutation Methods

| Method | Description |
|--------|-------------|
| `AddNode(stepID, nodeType)` | Adds a node (sets entry point if first) |
| `AddEdge(from, to)` | Adds a directed edge between two nodes |
| `SetEntryPoint(stepID)` | Sets the graph entry point |
| `UpdateNodeType(stepID, nodeType)` | Changes a node's type |
| `Clone()` | Deep-copies the entire graph |

All mutation methods invalidate the sort and level caches.

## Caching

The graph caches:
- **`sortCache`** — result of `TopologicalSort()`
- **`levelCache`** — result of level computation

Both caches are invalidated whenever the graph is modified (`AddNode`, `AddEdge`, `SetEntryPoint`, `UpdateNodeType`).

## Execution Flow

During workflow execution, the engine:

1. Calls `TopologicalSort()` to get the execution order
2. Iterates through steps in order
3. For each step, resolves input from its predecessors via `GetPreviousSteps()`
4. Executes the step
5. Stores the output for downstream steps

```
TopologicalSort → [step1, step2a, step2b, step2c, step3]
                      ↓       ↓       ↓       ↓       ↓
                   execute  execute  execute  execute  execute
```

## Graph Patterns

### Linear Chain

```go
wf.ThenStep(a).ThenStep(b).ThenStep(c)
```

```
A → B → C
```

### Fan-Out / Fan-In

```go
wf.ThenStep(a).Parallel(b, c, d).ThenStep(e)
```

```
    ┌→ B ─┐
A ──┼→ C ──┼→ E
    └→ D ─┘
```

### Conditional Branch

```go
wf.ThenStep(a).ThenStepIf(b, condition, nil).ThenStep(c)
```

```
A → B(?) → C
```

B executes only if the condition is true. If skipped, C still runs (with B's default output or pass-through).

---

**Next**: Learn about [Type Safety](type-safety.md) →
