# System Overview

High-level architecture of the Gorkflow workflow engine.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Application                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐    ┌──────────────────┐                   │
│  │  WorkflowBuilder │───▶│     Workflow     │                   │
│  │   (Fluent API)   │    │   (Definition)   │                   │
│  └──────────────────┘    └────────┬─────────┘                   │
│                                   │                              │
│                                   ▼                              │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                        Engine                             │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │   │
│  │  │  Executor   │  │   Logger    │  │  Configuration  │   │   │
│  │  │ (Step Run)  │  │ (zerolog)   │  │   (Timeouts,    │   │   │
│  │  │             │  │             │  │    Retries)     │   │   │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘   │   │
│  └──────────────────────────┬───────────────────────────────┘   │
│                             │                                    │
│                             ▼                                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                   WorkflowStore                           │   │
│  │              (Persistence Interface)                      │   │
│  └──────────────────────────┬───────────────────────────────┘   │
│                             │                                    │
└─────────────────────────────┼────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
       ┌───────────┐   ┌───────────┐   ┌───────────┐
       │  Memory   │   │  LibSQL   │   │  Custom   │
       │   Store   │   │   Store   │   │   Store   │
       └───────────┘   └───────────┘   └───────────┘
```

## Core Components

### 1. Workflow Definition Layer

**Files**: `workflow.go`, `workflow_builder.go`, `step.go`, `graph.go`

- **Workflow**: Blueprint containing steps and execution graph
- **WorkflowBuilder**: Fluent API for constructing workflows
- **Step**: Type-safe unit of work with handler function
- **ExecutionGraph**: DAG defining step dependencies and execution order

### 2. Execution Engine

**Files**: `engine/engine.go`, `engine/executor.go`

- **Engine**: Orchestrates workflow execution lifecycle
- **Executor**: Handles individual step execution with retry/timeout logic

### 3. Configuration

**Files**: `config.go`

- **ExecutionConfig**: Step-level settings (retries, timeouts, backoff)
- **EngineConfig**: Engine-level settings (concurrency limits)
- **StartOptions**: Runtime options for workflow execution

### 4. Context & State

**Files**: `context.go`

- **StepContext**: Rich context passed to step handlers
- **StepDataAccessor**: Access to previous step outputs
- **StateAccessor**: Workflow-level state management

### 5. Persistence

**Files**: `store_interface.go`, `store/`

- **WorkflowStore**: Interface for persistence operations
- **MemoryStore**: In-memory implementation for testing
- **LibSQLStore**: SQLite/Turso database persistence

### 6. Supporting Components

- **models.go**: Core data models (WorkflowRun, StepExecution)
- **errors.go**: Error types and codes
- **validation.go**: Input/output validation
- **logging.go**: Structured logging helpers
- **helpers.go**: Utility functions

## Data Flow

```
1. Build Phase
   WorkflowBuilder → Workflow (with ExecutionGraph)

2. Start Phase
   Engine.StartWorkflow() → Creates WorkflowRun → Persists to Store

3. Execution Phase
   For each step in topological order:
   ├── Load previous step output
   ├── Create StepContext
   ├── Execute step handler (with retry/timeout)
   ├── Validate output
   ├── Persist step execution and output
   └── Update workflow progress

4. Completion Phase
   Update WorkflowRun status → Log completion
```

## Key Design Decisions

### Type Safety with Generics

Steps use Go generics for compile-time type safety:

```go
type Step[TIn, TOut any] struct {
    Handler func(ctx *StepContext, input TIn) (TOut, error)
}
```

### DAG-Based Execution

Workflows are directed acyclic graphs enabling:
- Sequential execution
- Parallel execution
- Conditional branching

### Pluggable Storage

The `WorkflowStore` interface allows different backends:
- In-memory for testing
- LibSQL for production
- Custom implementations

### Validation by Default

Input/output validation is enabled by default using struct tags, ensuring data integrity throughout workflow execution.

---

**Next**: Learn about [Execution Flow](execution-flow.md) →
