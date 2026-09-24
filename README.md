# Gorkflow

A powerful, type-safe workflow orchestration engine for Go with built-in state persistence, retries, and DAG-based execution.

[![Go Reference](https://pkg.go.dev/badge/github.com/sicko7947/gorkflow.svg)](https://pkg.go.dev/github.com/sicko7947/gorkflow)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

📚 **[Full Documentation](https://cai139193541.gitbook.io/gorkflow/)** | [Quick Start](https://cai139193541.gitbook.io/gorkflow/getting-started/quick-start) | [Examples](example/)

## Overview

Gorkflow is a lightweight, type-safe workflow orchestration engine that makes it easy to build complex workflows in Go. Using generics for compile-time type safety and a fluent builder API, you can create robust workflows with minimal boilerplate.

## ✨ Key Features

- **🎯 Type-Safe** - Strongly-typed steps using Go generics
- **✅ Auto-Validation** - Built-in input/output validation with struct tags
- **📊 DAG-Based** - Sequential, parallel, and conditional execution
- **🔄 Smart Retries** - Configurable retry policies with backoff strategies
- **💾 Persistent** - Multiple storage backends (LibSQL/SQLite, in-memory)
- **⚡ Parallel** - Execute independent steps concurrently
- **🎨 Conditional** - Dynamic workflow paths based on runtime data
- **🏗️ Fluent API** - Easy-to-use builder pattern

## 🚀 Quick Start

### Installation

```bash
go get github.com/sicko7947/gorkflow
```

### Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/sicko7947/gorkflow"
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
)

// Define types
type Input struct {
    A int `json:"a"`
    B int `json:"b"`
}

type Output struct {
    Sum int `json:"sum"`
}

func main() {
    // Create a step
    addStep := gorkflow.NewStep(
        "add", "Add Numbers",
        func(ctx *gorkflow.StepContext, input Input) (Output, error) {
            return Output{Sum: input.A + input.B}, nil
        },
    )

    // Build workflow
    wf, err := gorkflow.NewWorkflow("calc", "Calculator").
        Sequence(addStep).
        Build()

    if err != nil {
        panic(err)
    }

    // Execute and wait for completion
    workflowStore := store.NewMemoryStore()
    eng := engine.NewEngine(workflowStore)
    runID, err := eng.StartWorkflow(context.Background(), wf, Input{A: 10, B: 5},
        gorkflow.WithSynchronousExecution())
    if err != nil {
        panic(err)
    }

    // Get result
    run, err := eng.GetRun(context.Background(), runID)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Status: %s, Output: %s\n", run.Status, run.Output)
}
```

## 📖 Documentation

Visit our **[comprehensive documentation](https://cai139193541.gitbook.io/gorkflow/)** for:

- **[Getting Started](https://cai139193541.gitbook.io/gorkflow/getting-started/installation)** - Installation, quick start, and tutorials
- **[Core Concepts](https://cai139193541.gitbook.io/gorkflow/core-concepts/workflows)** - Workflows, steps, validation, state management
- **[Advanced Usage](https://cai139193541.gitbook.io/gorkflow/advanced-usage/parallel-execution)** - Parallel execution, conditionals, error handling
- **[Storage Backends](https://cai139193541.gitbook.io/gorkflow/storage/overview)** - LibSQL, in-memory stores
- **[API Reference](https://cai139193541.gitbook.io/gorkflow/api-reference/workflow-builder)** - Complete API documentation

## 💡 Examples

Check out the [examples/](example/) directory for complete, runnable examples:

- **[Sequential](example/sequential/)** - Basic sequential workflow
- **[Parallel](example/parallel/)** - Parallel step execution
- **[Conditional](example/conditional/)** - Runtime conditional logic
- **[Validation](example/validation/)** - Input/output validation
- **[LibSQL Persistence](example/libsql_persistence/)** - Persistent storage

## 🏗️ Architecture

```
┌─────────────┐
│  Workflow   │  Define your workflow with type-safe steps
└──────┬──────┘
       │
       ↓
┌─────────────┐
│   Engine    │  Orchestrates execution with retries
└──────┬──────┘
       │
       ↓
┌─────────────┐
│    Store    │  Persists state (Memory/LibSQL)
└─────────────┘
```

## 🔧 Core Concepts

### Type-Safe Steps

```go
step := gorkflow.NewStep(
    "process",
    "Process Data",
    func(ctx *gorkflow.StepContext, input UserInput) (UserOutput, error) {
        // Fully type-safe - input and output are validated
        return UserOutput{UserID: "123"}, nil
    },
    gorkflow.WithRetries(3),
    gorkflow.WithTimeout(30 * time.Second),
)
```

### Workflow Patterns

**Sequential:**

```go
wf, _ := gorkflow.NewWorkflow("seq", "Sequential").
    Sequence(step1, step2, step3).
    Build()
```

**Parallel:**

```go
wf, _ := gorkflow.NewWorkflow("parallel", "Parallel").
    Parallel(stepA, stepB, stepC).
    ThenStep(aggregateStep).
    Build()
```

**Conditional:**

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var flag bool
    ctx.State.Get("process_data", &flag)
    return flag, nil
}

wf, _ := gorkflow.NewWorkflow("cond", "Conditional").
    ThenStepIf(processStep, condition, nil).
    Build()
```

### Storage Backends

| Backend      | Use Case                | Setup                                     |
| ------------ | ----------------------- | ----------------------------------------- |
| **Memory**   | Development, Testing    | `store.NewMemoryStore()`                  |
| **LibSQL**   | Small-Medium Apps, Edge | `store.NewLibSQLStore("file:./db")`       |

See [Storage Documentation](https://cai139193541.gitbook.io/gorkflow/storage/overview) for details.

## 📦 Requirements

- **Go 1.26.0+** (see `go.mod`)
- Optional: LibSQL client (for SQLite/Turso)

## 🤝 Contributing

Contributions welcome! Run `go test ./...`, `go test -race ./...`, and `go vet ./...` before submitting changes.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

Extracted from the Tendor Email Agent project and refactored into a standalone workflow engine library.

## 📞 Support

- **Documentation**: https://cai139193541.gitbook.io/gorkflow/
- **Issues**: https://github.com/sicko7947/gorkflow/issues
- **Discussions**: https://github.com/sicko7947/gorkflow/discussions

---

**Ready to build powerful workflows?** Check out the [Quick Start Guide](https://cai139193541.gitbook.io/gorkflow/getting-started/quick-start)! 🚀
