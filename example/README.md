# gorkflow Examples

This directory contains complete, runnable examples demonstrating various features of the gorkflow engine.

## Available Examples

### 1. [Simple Math](./simple_math/)
**Demonstrates**: Basic sequential workflow execution

A straightforward example showing:
- Sequential step execution
- Type-safe data passing between steps
- Step configuration (retries, timeouts, backoff strategies)
- Fluent builder API

**Workflow**: Add two numbers → Multiply by factor → Format result

```bash
cd simple_math
cat README.md  # Full documentation
```

### 2. [Conditional Execution](./conditional/)
**Demonstrates**: Runtime conditional step execution

Shows how to use the `ThenStepIf` builder API for conditional execution:
- Builder-level conditional API
- Condition evaluation from workflow state
- Default values when steps are skipped
- Runtime decision-making

**Workflow**: Setup flags → Conditionally double value → Conditionally format

```bash
cd conditional
cat README.md  # Full documentation
```

## Running Examples

Each example is a standalone Go package that can be imported and used:

```go
import (
    "context"
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
    "github.com/sicko7947/gorkflow/example/simple_math"
    "github.com/sicko7947/gorkflow/example/conditional"
)

func main() {
    // Create engine (uses defaults: stdout logger, DefaultEngineConfig)
    eng := engine.NewEngine(store.NewMemoryStore())

    // Run simple math workflow
    mathWf, _ := simple_math.NewSimpleMathWorkflow()
    mathInput := simple_math.WorkflowInput{Val1: 10, Val2: 5, Mult: 2}
    runID1, _ := eng.StartWorkflow(context.Background(), mathWf, mathInput)

    // Run conditional workflow
    condWf, _ := conditional.NewConditionalWorkflow()
    condInput := conditional.ConditionalInput{
        Value:            10,
        EnableDoubling:   true,
        EnableFormatting: true,
    }
    runID2, _ := eng.StartWorkflow(context.Background(), condWf, condInput)
}
```

## Example Structure

Each example follows this structure:

```
example/
├── <example-name>/
│   ├── README.md       # Detailed documentation
│   ├── types.go        # Input/output type definitions
│   ├── steps.go        # Step implementations
│   ├── workflow.go     # Workflow builder function
│   └── orchestrator.go # Workflow orchestrator (encapsulates engine + workflow)
└── README.md           # This file - examples overview
```

Each example includes its own orchestrator that provides a clean API for starting workflows, getting status, and cancelling executions.

## Learning Path

**Recommended order for learning:**

1. **simple_math/** - Start here to understand the basics
2. **conditional/** - Learn conditional execution patterns

Each example builds on concepts from the previous ones.

## Creating Your Own Example

To add a new example:

1. Create a new directory: `example/my_example/`
2. Add your workflow implementation files
3. Create a `README.md` documenting the example
4. Update this file with a link to your example

## Support

For questions about examples or the gorkflow library:
- Check the main [README](../README.md)
- Review example READMEs for specific features
- Open an issue on GitHub
