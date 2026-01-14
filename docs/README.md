# Gorkflow Documentation

Welcome to the comprehensive documentation for **Gorkflow** - a powerful, type-safe, and flexible workflow orchestration engine for Go.

## 📚 Documentation Overview

This documentation is organized into the following sections:

### 🚀 Getting Started

- [Installation](getting-started/installation.md) - Get up and running with Gorkflow
- [Quick Start](getting-started/quick-start.md) - Build your first workflow in minutes
- [First Workflow](getting-started/first-workflow.md) - Step-by-step tutorial
- [Project Structure](getting-started/project-structure.md) - Organizing your workflow project

### 💡 Core Concepts

- [Workflows](core-concepts/workflows.md) - Understanding workflow definitions
- [Steps](core-concepts/steps.md) - Building blocks of workflows
- [Execution Graph](core-concepts/execution-graph.md) - DAG-based execution model
- [Type Safety](core-concepts/type-safety.md) - Leveraging Go generics for type-safe workflows
- [State Management](core-concepts/state-management.md) - Managing workflow and step state
- [Validation](core-concepts/validation.md) - Input/output validation with struct tags
- [Context](core-concepts/context.md) - Custom workflow context

### 📖 API Reference

- [Workflow Builder](api-reference/workflow-builder.md) - Fluent API for building workflows
- [Step API](api-reference/step-api.md) - Creating and configuring steps
- [Engine API](api-reference/engine-api.md) - Workflow execution engine
- [Store Interface](api-reference/store-interface.md) - Persistence layer interface
- [Configuration](api-reference/configuration.md) - Configuration options

### 🎯 Advanced Usage

- [Parallel Execution](advanced-usage/parallel-execution.md) - Running steps in parallel
- [Conditional Execution](advanced-usage/conditional-execution.md) - Dynamic workflow paths
- [Retry Strategies](advanced-usage/retry-strategies.md) - Configuring retries and backoff
- [Timeouts](advanced-usage/timeouts.md) - Per-step and workflow-level timeouts
- [Error Handling](advanced-usage/error-handling.md) - Graceful error management
- [Cancellation](advanced-usage/cancellation.md) - Cancelling running workflows
- [Tags and Metadata](advanced-usage/tags-and-metadata.md) - Organizing workflow runs
- [Logging](advanced-usage/logging.md) - Structured logging with zerolog

### 💾 Storage Backends

- [Overview](storage/overview.md) - Available storage options
- [Memory Store](storage/memory-store.md) - In-memory storage for testing
- [LibSQL Store](storage/libsql-store.md) - SQLite/Turso database persistence
- [Custom Store](storage/custom-store.md) - Implementing custom storage backends

### 📝 Examples & Tutorials

- [Sequential Workflows](examples/sequential.md) - Basic sequential execution
- [Parallel Workflows](examples/parallel.md) - Parallel step execution
- [Conditional Workflows](examples/conditional.md) - Runtime conditional logic
- [Validation Example](examples/validation.md) - Input/output validation
- [Persistence Example](examples/persistence.md) - Using LibSQL for persistence
- [Real-World Patterns](examples/real-world-patterns.md) - Common workflow patterns

### 🔧 Troubleshooting

- [Common Issues](troubleshooting/common-issues.md) - Frequently encountered problems
- [Debugging](troubleshooting/debugging.md) - Debugging workflow execution
- [Performance](troubleshooting/performance.md) - Performance optimization tips
- [FAQ](troubleshooting/faq.md) - Frequently asked questions

### 🏗️ Architecture

- [System Overview](architecture/system-overview.md) - High-level architecture
- [Execution Flow](architecture/execution-flow.md) - How workflows execute
- [Graph Traversal](architecture/graph-traversal.md) - DAG traversal algorithm
- [Storage Layer](architecture/storage-layer.md) - Persistence architecture
- [Design Decisions](architecture/design-decisions.md) - Key architectural choices

## 🌟 Key Features

- **🎯 Type-Safe Step Definitions** - Strongly-typed input/output using Go generics
- **✅ Built-in Validation** - Automatic validation using struct tags
- **📊 DAG-Based Execution** - Directed acyclic graphs with sequential and parallel execution
- **🔄 Smart Retry Logic** - Configurable retry policies with backoff strategies
- **💾 Persistent State** - Pluggable storage backends (LibSQL, in-memory)
- **⚡ Parallel Execution** - Execute independent steps concurrently
- **🔍 Progress Tracking** - Real-time workflow and step-level monitoring
- **⏱️ Timeout Support** - Per-step and workflow-level timeouts
- **🏗️ Fluent Builder API** - Easy-to-use builder pattern
- **📝 Structured Logging** - Comprehensive execution logs with zerolog
- **🎨 Conditional Branching** - Runtime conditional step execution
- **🛑 Graceful Cancellation** - Cancel workflows safely

## 🚀 Quick Example

```go
// Define your types
type CalculationInput struct {
    A int `json:"a"`
    B int `json:"b"`
}

type SumOutput struct {
    Sum int `json:"sum"`
}

// Create steps
addStep := gorkflow.NewStep(
    "add", "Add Numbers",
    func(ctx *gorkflow.StepContext, input CalculationInput) (SumOutput, error) {
        return SumOutput{Sum: input.A + input.B}, nil
    },
)

// Build workflow
wf, _ := gorkflow.NewWorkflow("calc", "Calculator").
    Sequence(addStep).
    Build()

// Execute
store := store.NewMemoryStore()
eng := engine.NewEngine(store)
runID, _ := eng.StartWorkflow(context.Background(), wf, CalculationInput{A: 10, B: 5})
```

## 📦 Installation

```bash
go get github.com/sicko7947/gorkflow
```

## 📖 Where to Start

1. **New to Gorkflow?** → Start with [Quick Start](getting-started/quick-start.md)
2. **Building your first workflow?** → Check out [First Workflow Tutorial](getting-started/first-workflow.md)
3. **Need specific features?** → Browse [Advanced Usage](advanced-usage/parallel-execution.md)
4. **Looking for examples?** → Explore [Examples & Tutorials](examples/sequential.md)
5. **Integration questions?** → See [Storage Backends](storage/overview.md)

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](../README.md#contributing) for details.

## 📄 License

Gorkflow is licensed under the MIT License. See [LICENSE](../LICENSE) for details.

## 🔗 Links

- [GitHub Repository](https://github.com/sicko7947/gorkflow)
- [Report Issues](https://github.com/sicko7947/gorkflow/issues)
- [Example Code](../example/)

---

**Ready to build powerful workflows?** Start with our [Quick Start Guide](getting-started/quick-start.md)! 🚀
