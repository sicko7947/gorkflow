# Project Index: gorkflow

**Generated:** 2025-11-24
**Module:** github.com/sicko7947/gorkflow
**Go Version:** 1.25.3

---

## 📁 Project Structure

```
gorkflow/
├── builder/              # Fluent workflow builder API
│   ├── builder.go
│   ├── builder_test.go
│   ├── options.go
│   └── validation.go
├── engine/               # Workflow execution engine
│   ├── engine.go
│   ├── engine_integration_test.go
│   ├── executor.go
│   ├── traverser.go
│   ├── backoff.go
│   ├── retry_test.go
│   └── conditional_test.go
├── store/                # Persistence layer
│   ├── store.go
│   ├── schema.go
│   ├── schema_test.go
│   ├── memory.go
│   └── memory_test.go
├── example/              # Complete working example
│   ├── workflow.go
│   ├── steps.go
│   ├── orchestrator.go
│   ├── types.go
│   └── GUIDE.md
├── workflow.go           # Core workflow definition
├── step.go               # Step definition & execution
├── graph.go              # Execution graph (DAG)
├── models.go             # Data models
├── config.go             # Configuration types
├── context.go            # Execution context
├── errors.go             # Error handling
├── helpers.go            # Utility functions
├── logging.go            # Logging utilities
├── store_interface.go    # Store interface
└── README.md             # Main documentation
```

---

## 🚀 Entry Points

### Main Package

- **Package:** `workflow` (root)
- **Purpose:** Core workflow engine and types
- **Key Functions:**
  - `NewStep[TIn, TOut]()` - Create type-safe workflow steps
  - `NewWorkflowInstance()` - Create workflow instance

### Builder Package

- **Package:** `builder`
- **Entry:** `builder.NewWorkflow()`
- **Purpose:** Fluent API for constructing workflows
- **Pattern:** Builder pattern with method chaining

### Engine Package

- **Package:** `engine`
- **Entry:** `engine.NewEngine()`
- **Purpose:** Orchestrate workflow execution
- **Key Methods:**
  - `StartWorkflow()` - Begin workflow execution
  - `GetRun()` - Retrieve workflow status
  - `Cancel()` - Cancel running workflow

### Store Package

- **Package:** `store`
- **Implementations:**
  - `NewMemoryStore()` - In-memory storage

### Example Package

- **Package:** `simple_math` (example)
- **Entry:** `NewSimpleMathWorkflow()`
- **Purpose:** Complete reference implementation
- **See:** `example/GUIDE.md`

---

## 📦 Core Modules

### Module: Workflow Definition

**Files:** `workflow.go`, `step.go`, `graph.go`

**Exports:**

- `type Workflow` - Workflow blueprint
- `type Step[TIn, TOut]` - Generic step definition
- `type ExecutionGraph` - DAG representation
- `NewWorkflowInstance()` - Create workflow
- `NewStep[TIn, TOut]()` - Create typed step

**Purpose:** Define workflow structure, steps, and execution graph

---

### Module: Workflow Builder

**Files:** `builder/builder.go`, `builder/options.go`, `builder/validation.go`

**Exports:**

- `type WorkflowBuilder` - Fluent builder
- `NewWorkflow()` - Start building
- `WithDescription()`, `WithVersion()`, `WithConfig()` - Configuration
- `Sequence()` - Chain steps sequentially
- `Parallel()` - Execute steps in parallel
- `ThenStep()` - Add single step
- `Build()` - Finalize workflow

**Purpose:** Fluent API for constructing complex workflows

---

### Module: Execution Engine

**Files:** `engine/engine.go`, `engine/executor.go`, `engine/traverser.go`

**Exports:**

- `type Engine` - Execution orchestrator
- `type EngineConfig` - Engine configuration
- `NewEngine()` - Create engine
- `StartWorkflow()` - Begin execution
- `GetRun()` - Query workflow status
- `GetStepExecutions()` - Get step details
- `Cancel()` - Cancel workflow

**Purpose:** Execute workflows, manage retries, handle state transitions

---

### Module: Retry & Backoff

**Files:** `engine/backoff.go`

**Exports:**

- `type BackoffStrategy` - Backoff algorithm
- `BackoffLinear`, `BackoffExponential`, `BackoffNone` - Strategies
- Backoff calculation logic

**Purpose:** Implement retry policies with configurable backoff

---

### Module: Graph Traversal

**Files:** `engine/traverser.go`

**Exports:**

- `type GraphTraverser` - Graph traversal logic
- `GetExecutionOrder()` - Topological sort
- Dependency resolution

**Purpose:** Determine execution order from DAG

---

### Module: Persistence Layer

**Files:** `store/store.go`, `store/schema.go`

**Exports:**

- `type WorkflowStore` interface
- `CreateRun()`, `UpdateRun()`, `GetRun()`
- `CreateStepExecution()`, `UpdateStepExecution()`, `GetStepExecutions()`
- `SetState()`, `GetState()`, `DeleteState()`

**Purpose:** Abstraction for workflow state persistence

---

### Module: Memory Store

**Files:** `store/memory.go`

**Exports:**

- `type MemoryStore` - In-memory implementation
- `NewMemoryStore()` - Create store

**Purpose:** Fast in-memory storage for development and testing

---

### Module: Data Models

**Files:** `models.go`

**Exports:**

- `type WorkflowRun` - Workflow execution record
- `type StepExecution` - Step execution record
- `type RunStatus` - Workflow status enum
- `type StepStatus` - Step status enum
- `type TriggerInfo` - Workflow trigger metadata
- `type WorkflowError` - Error information

**Purpose:** Core data structures for workflow state

---

### Module: Configuration

**Files:** `config.go`

**Exports:**

- `type ExecutionConfig` - Step/workflow configuration
- `type BackoffStrategy` - Retry backoff types
- `type StepOption` - Functional options
- `WithRetries()`, `WithBackoff()`, `WithTimeout()` - Option constructors
- `DefaultExecutionConfig`, `DefaultEngineConfig` - Defaults

**Purpose:** Configuration types and functional options

---

### Module: Execution Context

**Files:** `context.go`

**Exports:**

- `type StepContext` - Step execution context
- `type StepOutputAccessor` - Access previous step outputs
- `type StateAccessor` - Access workflow state
- Context creation and management

**Purpose:** Provide execution context to step handlers

---

### Module: Error Handling

**Files:** `errors.go`

**Exports:**

- `type WorkflowError` - Workflow-specific errors
- Error classification and formatting
- Retry-related error types

**Purpose:** Structured error handling

---

## 🔧 Configuration

### Environment Variables

Not required for library usage. Configuration via code.

### Key Configuration Files

- `go.mod` - Module dependencies
- `go.sum` - Dependency checksums

---

## 📚 Documentation

- `README.md` - Comprehensive guide with examples
- `example/GUIDE.md` - Step-by-step example walkthrough
- `PROJECT_INDEX.md` - This file (repository index)

---

## 🧪 Test Coverage

### Unit Tests

- `*_test.go` files throughout codebase
- Builder tests: `builder/builder_test.go`
- Store tests: `store/memory_test.go`, `store/schema_test.go`
- Retry tests: `engine/retry_test.go`
- Conditional tests: `engine/conditional_test.go`

### Integration Tests

- Engine integration: `engine/engine_integration_test.go`

### Test Execution

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Integration tests only
go test -tags=integration ./store/...
```

---

## 🔗 Key Dependencies

### Core Dependencies

- **google/uuid** (v1.6.0) - UUID generation for run IDs
- **rs/zerolog** (v1.34.0) - Structured logging

### Test Dependencies

- **stretchr/testify** (v1.11.1) - Testing assertions

### Go Version

- Requires Go 1.21+ (uses generics)

---

## 📝 Quick Start

### 1. Install

```bash
go get github.com/sicko7947/gorkflow
```

### 2. Create a Simple Workflow

```go
import (
    workflow "github.com/sicko7947/gorkflow"
    "github.com/sicko7947/gorkflow"
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
)

// Define step
step := workflow.NewStep(
    "hello",
    "Say Hello",
    func(ctx *workflow.StepContext, input string) (string, error) {
        return "Hello, " + input, nil
    },
)

// Build workflow
wf, _ := builder.NewWorkflow("greeting", "Greeting Workflow").
    Sequence(step).
    Build()

// Execute
store := store.NewMemoryStore()
logger := zerolog.New(os.Stdout)
eng := engine.NewEngine(store, logger, engine.DefaultEngineConfig)

runID, _ := eng.StartWorkflow(context.Background(), wf, "World")
```

### 3. Explore Example

```bash
cd example/
cat GUIDE.md
```

---

## 🎯 Use Cases

### Sequential Workflows

Chain steps that depend on previous outputs

### Parallel Workflows

Execute independent steps concurrently

### Data Processing Pipelines

Transform data through multiple stages

### ETL Operations

Extract, transform, and load data with retries

### Orchestration Tasks

Coordinate multiple services or operations

### State Machines

Implement complex state transitions

---

## 🏗️ Architecture Highlights

### Type Safety

- Generic step definitions: `Step[TIn, TOut]`
- Compile-time type checking
- Automatic marshaling/unmarshaling

### DAG-Based Execution

- Directed Acyclic Graph representation
- Topological sorting for execution order
- Cycle detection and validation

### Retry Mechanisms

- Configurable retry policies
- Linear and exponential backoff
- Per-step timeout support

### State Persistence

- Pluggable store interface
- In-memory for development
- LibSQL for persistence

### Observability

- Structured logging (zerolog)
- Progress tracking (0.0 to 1.0)
- Step execution history

---

## 📊 Key Metrics

- **Total Go Files:** 34
- **Core Packages:** 4 (root, builder, engine, store)
- **Store Implementations:** 2 (Memory, LibSQL)
- **Test Files:** 9
- **Example Files:** 5
- **Documentation:** 3 files

---

## 🔄 Workflow Lifecycle

```
1. Define Steps → NewStep[TIn, TOut]()
2. Build Workflow → builder.NewWorkflow().Sequence(...).Build()
3. Create Engine → engine.NewEngine(store, logger, config)
4. Start Execution → engine.StartWorkflow(ctx, wf, input)
5. Monitor Progress → engine.GetRun(ctx, runID)
6. Complete/Fail → Automatic state transitions
```

---

## 🚦 Status Enums

### Workflow Run Status

- `PENDING` - Created, not started
- `RUNNING` - Currently executing
- `COMPLETED` - Successfully finished
- `FAILED` - Execution failed
- `CANCELLED` - Manually cancelled

### Step Execution Status

- `PENDING` - Queued for execution
- `RUNNING` - Currently executing
- `COMPLETED` - Successfully finished
- `FAILED` - Execution failed
- `SKIPPED` - Skipped due to conditions
- `RETRYING` - In retry loop

---

## 📖 Additional Resources

### Official Documentation

- Main README: Comprehensive usage guide
- Example Guide: Step-by-step tutorial

### Code Examples

- `example/` directory: Complete working implementation
- `*_test.go` files: Usage patterns and edge cases

---

**Index Version:** 1.0
**Last Updated:** 2025-11-24
**Maintainer:** sicko7947
