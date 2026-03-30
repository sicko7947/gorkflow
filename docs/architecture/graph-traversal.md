# Graph Traversal

How Gorkflow traverses the execution graph to determine step ordering, detect cycles, and compute parallel levels.

## Graph Structure

The `ExecutionGraph` is a directed acyclic graph (DAG) stored as an adjacency list:

```go
type ExecutionGraph struct {
    EntryPoint string
    Nodes      map[string]*GraphNode
    sortCache  []string
    levelCache [][]string
}

type GraphNode struct {
    StepID   string
    Type     NodeType
    Next     []string   // Outgoing edges
    Previous []string   // Incoming edges
}
```

Each node stores both forward (`Next`) and backward (`Previous`) references for efficient traversal in either direction.

## Topological Sort

The engine uses topological sorting to determine the order in which steps execute. This guarantees every step runs after all its dependencies.

### Algorithm

Gorkflow uses **DFS-based topological sort**:

```
function TopologicalSort(graph):
    visited = {}
    result = []

    function visit(node):
        if node in visited: return
        visited[node] = true

        for each successor in node.Next:
            visit(successor)

        result = [node] + result    // prepend

    visit(entryPoint)
    return result
```

### Example

Given this graph:

```
A → B → D
A → C → D
```

The DFS visits: A → B → D (backtrack) → C → D (already visited)

Result: `[A, B, C, D]` or `[A, C, B, D]` (both are valid topological orderings)

### Caching

The sort result is cached in `sortCache`. The cache is invalidated whenever the graph is modified:
- `AddNode`
- `AddEdge`
- `SetEntryPoint`
- `UpdateNodeType`

Subsequent calls to `TopologicalSort()` return the cached result without recomputing.

## Cycle Detection

Cycles are detected during `Validate()` using **DFS with a recursion stack**:

```
function hasCycle(node, visited, recStack):
    visited[node] = true
    recStack[node] = true

    for each successor in node.Next:
        if successor not in visited:
            if hasCycle(successor, visited, recStack):
                return true
        else if successor in recStack:
            return true    // Back edge = cycle

    recStack[node] = false
    return false
```

### How It Works

- **visited** — tracks nodes that have been fully explored
- **recStack** — tracks nodes in the current DFS path

If we encounter a node that is already in `recStack`, we've found a back edge, which means a cycle exists.

### When Validation Runs

`Validate()` is called by:
1. `Build()` — during workflow construction
2. `TopologicalSort()` — before computing the sort order

A graph with cycles will fail validation with: `"execution graph contains cycles"`.

## Reachability Check

After cycle detection, `Validate()` checks that all nodes are reachable from the entry point:

```
function getReachableNodes(start):
    reachable = {}

    function dfs(node):
        reachable[node] = true
        for each successor in node.Next:
            if successor not in reachable:
                dfs(successor)

    dfs(start)
    return reachable
```

If `len(reachable) != len(allNodes)`, some nodes are disconnected and the graph is invalid: `"not all nodes are reachable from entry point"`.

## Graph Query Methods

### `GetNextSteps`

Returns the immediate successors of a node (outgoing edges):

```go
next, err := graph.GetNextSteps("step-a")
// Returns: ["step-b", "step-c"]
```

### `GetPreviousSteps`

Returns the immediate predecessors of a node (incoming edges):

```go
prev, err := graph.GetPreviousSteps("step-d")
// Returns: ["step-b", "step-c"]
```

The engine uses this to resolve input for each step — it loads the output of the first predecessor.

### `IsTerminal`

Returns `true` if a node has no outgoing edges (it's a leaf node):

```go
if graph.IsTerminal("final-step") {
    // This is the last step in the workflow
}
```

## How the Engine Uses the Graph

### Step Execution Flow

```
1. TopologicalSort() → [A, B, C, D]

2. For each step in order:
   a. Check context cancellation
   b. Resolve input:
      - First step → uses workflow input
      - Other steps → GetPreviousSteps(stepID) → LoadStepOutput(prevStep)
   c. Execute step
   d. Save output
   e. Update progress
```

### Input Resolution

The engine resolves step input using `GetPreviousSteps`:

```go
prevSteps, _ := graph.GetPreviousSteps(stepID)
if len(prevSteps) > 0 {
    stepInput = store.LoadStepOutput(runID, prevSteps[0])
}
```

Currently, only the first predecessor's output is used as input. Multi-parent aggregation (fan-in) passes the first parent's output.

## Graph Patterns and Traversal

### Linear Chain

```
A → B → C
```

Sort: `[A, B, C]`
Each step gets the previous step's output.

### Fan-Out

```
    ┌→ B
A ──┤
    └→ C
```

Sort: `[A, B, C]` (or `[A, C, B]`)
Both B and C get A's output.

### Fan-In

```
B ──┐
    ├→ D
C ──┘
```

Sort: `[..., B, C, D]` (B and C before D)
D gets the output of its first predecessor (B or C, depending on sort order).

### Diamond

```
    ┌→ B ──┐
A ──┤      ├→ D
    └→ C ──┘
```

Sort: `[A, B, C, D]`
D gets B's output (first predecessor in the graph).

## Complexity

| Operation | Time Complexity |
|-----------|----------------|
| `TopologicalSort` | O(V + E) |
| `Validate` (cycle detection) | O(V + E) |
| `Validate` (reachability) | O(V + E) |
| `GetNextSteps` | O(1) |
| `GetPreviousSteps` | O(1) |
| `IsTerminal` | O(1) |
| `AddNode` | O(1) |
| `AddEdge` | O(1) |

Where V = number of nodes, E = number of edges.

---

**See also**: [Execution Graph](../core-concepts/execution-graph.md) | [Execution Flow](execution-flow.md)
