# Gorkflow

A powerful, type-safe, and flexible workflow orchestration engine for Go with built-in state persistence, retries, and DAG-based execution.

## Features

- **🎯 Type-Safe Step Definitions**: Strongly-typed input/output for workflow steps using Go generics
- **📊 DAG-Based Execution**: Define workflows as directed acyclic graphs with sequential and parallel execution
- **🔄 Built-in Retry Logic**: Configurable retry policies with linear and exponential backoff strategies
- **💾 Persistent State Management**: Pluggable storage backends (DynamoDB and in-memory implementations included)
- **⚡ Parallel Execution**: Execute multiple steps concurrently when they don't depend on each other
- **🔍 Progress Tracking**: Monitor workflow execution status and step-level progress in real-time
- **⏱️ Timeout Support**: Per-step timeout configuration to prevent hanging operations
- **🏗️ Fluent Builder API**: Easy-to-use builder pattern for constructing complex workflows
- **📝 Structured Logging**: Built-in integration with zerolog for comprehensive execution logs
- **🎨 Conditional Branching**: Support for conditional step execution based on runtime data
- **🛑 Cancellation Support**: Cancel running workflows gracefully
- **🏷️ Tagging and Metadata**: Add custom tags and metadata to workflow runs for categorization

## Installation

```bash
go get github.com/sicko7947/gorkflow
```

## Quick Start

### 1. Define Your Step Types

```go
package main

import (
    "fmt"
    workflow "github.com/sicko7947/gorkflow"
)

// Input for the workflow
type CalculationInput struct {
    A int `json:"a"`
    B int `json:"b"`
}

// Output from step 1
type SumOutput struct {
    Sum int `json:"sum"`
}

// Output from step 2
type ResultOutput struct {
    Result  int    `json:"result"`
    Message string `json:"message"`
}
```

### 2. Create Steps with Handlers

```go
func NewAddStep() *workflow.Step[CalculationInput, SumOutput] {
    return workflow.NewStep(
        "add",
        "Add Two Numbers",
        func(ctx *workflow.StepContext, input CalculationInput) (SumOutput, error) {
            sum := input.A + input.B
            ctx.Logger.Info().Int("sum", sum).Msg("Addition completed")
            return SumOutput{Sum: sum}, nil
        },
    )
}

func NewFormatStep() *workflow.Step[SumOutput, ResultOutput] {
    return workflow.NewStep(
        "format",
        "Format Result",
        func(ctx *workflow.StepContext, input SumOutput) (ResultOutput, error) {
            message := fmt.Sprintf("The sum is %d", input.Sum)
            return ResultOutput{
                Result:  input.Sum,
                Message: message,
            }, nil
        },
        workflow.WithRetries(5),
        workflow.WithTimeout(5*time.Second),
    )
}
```

### 3. Build Your Workflow

```go
import (
    "github.com/sicko7947/gorkflow/builder"
)

func NewCalculationWorkflow() (*workflow.Workflow, error) {
    wf, err := builder.NewWorkflow("calculation", "Calculation Workflow").
        WithDescription("A simple calculation workflow").
        WithVersion("1.0").
        WithConfig(workflow.ExecutionConfig{
            MaxRetries:     3,
            RetryDelayMs:   1000,
            TimeoutSeconds: 30,
        }).
        Sequence(
            NewAddStep(),
            NewFormatStep(),
        ).
        Build()

    if err != nil {
        return nil, err
    }

    return wf, nil
}
```

### 4. Execute the Workflow

```go
import (
    "context"
    "github.com/rs/zerolog"
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
)

func main() {
    // Create store
    store := store.NewMemoryStore()

    // Create engine with default logger and config
    eng := engine.NewEngine(store)

    // Or with custom logger and config
    // logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
    // eng := engine.NewEngine(store,
    //     engine.WithLogger(logger),
    //     engine.WithConfig(engine.DefaultEngineConfig),
    // )

    // Create workflow
    wf, err := NewCalculationWorkflow()
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to create workflow")
    }

    // Start workflow
    ctx := context.Background()
    runID, err := eng.StartWorkflow(
        ctx,
        wf,
        CalculationInput{A: 10, B: 5},
        workflow.WithTags(map[string]string{
            "type": "calculation",
        }),
    )

    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to start workflow")
    }

    logger.Info().Str("run_id", runID).Msg("Workflow started")

    // Get workflow status
    run, err := eng.GetRun(ctx, runID)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to get workflow status")
    }

    logger.Info().
        Str("status", string(run.Status)).
        Float64("progress", run.Progress).
        Msg("Workflow status")
}
```

## Advanced Features

### Parallel Execution

Execute multiple independent steps in parallel:

```go
wf, err := builder.NewWorkflow("parallel-example", "Parallel Example").
    Parallel(
        NewStep1(),
        NewStep2(),
        NewStep3(),
    ).
    ThenStep(NewMergeStep()).
    Build()
```

### Retry Configuration

Configure step-specific retry behavior:

```go
step := workflow.NewStep(
    "retry-example",
    "Step with Custom Retry",
    handler,
    workflow.WithRetries(5),
    workflow.WithBackoff(workflow.BackoffExponential),
    workflow.WithTimeout(10*time.Second),
)
```

### Conditional Execution

Execute steps conditionally based on runtime evaluation:

```go
// Define a condition function
condition := func(ctx *workflow.StepContext) (bool, error) {
    // Access state or previous step outputs
    var shouldProcess bool
    ctx.State.Get("should_process", &shouldProcess)
    return shouldProcess, nil
}

// Builder-level API (recommended)
processStep := workflow.NewStep("process", "Process Data", processHandler)

wf, err := builder.NewWorkflow("conditional-workflow", "Conditional Workflow").
    ThenStep(checkStep).
    ThenStepIf(processStep, condition, nil).  // ← Builder-level conditional
    ThenStep(finalStep).
    Build()

// Alternative: Step-level API (for type-safe default values)
baseStep := workflow.NewStep("process", "Process Data", processHandler)
defaultOutput := &ProcessOutput{Status: "skipped"}
conditionalStep := workflow.NewConditionalStep(baseStep, condition, defaultOutput)

wf, err := builder.NewWorkflow("conditional-workflow", "Conditional Workflow").
    ThenStep(checkStep).
    ThenStep(conditionalStep).
    Build()
```

**How it works:**
- Condition is evaluated before step execution
- Step executes only if condition returns `true`
- If `false`, uses default value (or zero value if nil)
- Condition errors propagate and fail the workflow

### State Management

Access and modify workflow state during execution:

```go
func handler(ctx *workflow.StepContext, input MyInput) (MyOutput, error) {
    // Get state accessor
    state := ctx.State

    // Set state
    state.Set(ctx.Context, "counter", 42)

    // Get state
    counter, err := state.Get(ctx.Context, "counter")

    // Access previous step output
    outputs := ctx.Outputs
    prevOutput, err := outputs.Get(ctx.Context, "previous-step-id")

    return MyOutput{}, nil
}
```

### Cancellation

Cancel a running workflow:

```go
err := eng.Cancel(ctx, runID)
if err != nil {
    logger.Error().Err(err).Msg("Failed to cancel workflow")
}
```

## Architecture

### Core Components

- **`workflow.Workflow`**: The workflow definition containing steps, execution graph, and configuration
- **`workflow.Step[TIn, TOut]`**: Type-safe step definition with input/output types
- **`engine.Engine`**: Orchestrates workflow execution, handles retries, and manages state transitions
- **`ExecutionGraph`**: DAG representation of workflow steps and their dependencies
- **`WorkflowStore`**: Persistence layer interface for storing workflow runs and step executions

### Execution Flow

```
1. StartWorkflow() → Create WorkflowRun
2. Engine.executeWorkflow() → Traverse execution graph
3. For each step:
   - Create StepExecution
   - Execute handler with retries
   - Store output
   - Update progress
4. Complete workflow → Update final status
```

### Storage Backends

#### DynamoDB Store

Persistent storage using AWS DynamoDB:

```go
import (
    "github.com/sicko7947/gorkflow/store"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

cfg, _ := config.LoadDefaultConfig(context.Background())
client := dynamodb.NewFromConfig(cfg)

store, err := store.NewDynamoDBStore(client, "workflow-table")
```

**Setting up DynamoDB Table**

Use the included helper scripts to manage your DynamoDB table. The scripts accept configuration via environment variables:

**Environment Variables:**
- `AWS_REGION` - AWS region (default: `ap-southeast-2`)
- `AWS_DYNAMODB_TABLE_NAME` - Table name (default: `workflow_executions`)

**Usage:**

```bash
# Using defaults (ap-southeast-2 region, workflow_executions table)
./scripts/create-dynamodb-table.sh

# Custom region and table name
export AWS_REGION=us-east-1
export AWS_DYNAMODB_TABLE_NAME=my_workflow_table
./scripts/create-dynamodb-table.sh

# Or inline
AWS_REGION=eu-west-1 AWS_DYNAMODB_TABLE_NAME=workflows ./scripts/create-dynamodb-table.sh

# Delete table (prompts for confirmation)
./scripts/delete-dynamodb-table.sh
```

**What the create script does:**
- Creates a table with Single Table Design (PK/SK pattern)
- Adds 2 Global Secondary Indexes (GSI1, GSI2) for flexible querying
- Enables TTL on the `ttl` attribute for automatic cleanup
- Uses PAY_PER_REQUEST billing mode
- Tags the table with project metadata

#### Memory Store

In-memory storage for testing and development:

```go
store := store.NewMemoryStore()
```

## Package Structure

```
gorkflow/
├── builder/          # Fluent API for building workflows
│   ├── builder.go    # WorkflowBuilder implementation
│   ├── options.go    # Builder options
│   └── validation.go # Workflow validation
├── engine/           # Workflow execution engine
│   ├── engine.go     # Main engine implementation
│   ├── executor.go   # Step execution logic
│   ├── traverser.go  # Graph traversal
│   ├── backoff.go    # Retry backoff strategies
│   └── *_test.go     # Engine tests
├── store/            # Persistence layer
│   ├── store.go      # Store interface
│   ├── memory.go     # In-memory implementation
│   ├── dynamodb.go   # DynamoDB implementation
│   └── schema.go     # Data models
├── example/          # Example implementations
│   ├── README.md     # Example overview and usage guide
│   ├── simple_math/  # Simple sequential workflow example
│   │   ├── workflow.go      # Workflow definition
│   │   ├── steps.go         # Step implementations
│   │   ├── orchestrator.go  # Workflow orchestrator
│   │   ├── types.go         # Type definitions
│   │   └── README.md        # Example-specific guide
│   └── conditional/  # Conditional execution example
│       ├── workflow.go      # Conditional workflow definition
│       ├── steps.go         # Conditional step implementations
│       ├── orchestrator.go  # Workflow orchestrator
│       ├── types.go         # Type definitions
│       └── README.md        # Example-specific guide
├── workflow.go       # Workflow definition
├── step.go           # Step definition and execution
├── graph.go          # Execution graph
├── models.go         # Core data models
├── config.go         # Configuration types
├── context.go        # Execution context
├── errors.go         # Error types
└── store_interface.go # Store interface definition
```

## Examples

The [example/](example/) directory contains complete working examples demonstrating different workflow patterns:

### Simple Math Workflow

Located in [example/simple_math/](example/simple_math/), demonstrates:
- Sequential step execution
- Data passing between steps
- Type-safe step definitions
- Step configuration (retries, timeouts)
- Workflow orchestration

### Conditional Workflow

Located in [example/conditional/](example/conditional/), demonstrates:
- Conditional step execution with `ThenStepIf`
- Runtime condition evaluation from workflow state
- Default values for skipped steps
- Dynamic workflow paths based on input flags

To explore the examples:

```bash
cd example/
cat README.md  # Read the example overview

cd simple_math/
cat README.md  # Simple math workflow guide

cd ../conditional/
cat README.md  # Conditional workflow guide
```

## Configuration

### Workflow-Level Configuration

Set default execution parameters for all steps:

```go
builder.NewWorkflow("my-workflow", "My Workflow").
    WithConfig(workflow.ExecutionConfig{
        MaxRetries:      3,
        RetryDelayMs:    1000,
        RetryBackoff:    workflow.BackoffLinear,
        TimeoutSeconds:  30,
        ContinueOnError: false,
    })
```

### Step-Level Configuration

Override workflow defaults for specific steps:

```go
workflow.NewStep(
    "my-step",
    "My Step",
    handler,
    workflow.WithRetries(5),
    workflow.WithBackoff(workflow.BackoffExponential),
    workflow.WithTimeout(60*time.Second),
)
```

### Engine Configuration

Configure the execution engine with optional parameters:

```go
// Simple: Use defaults (stdout logger at Info level, DefaultEngineConfig)
eng := engine.NewEngine(store)

// Custom logger only
logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
eng := engine.NewEngine(store, engine.WithLogger(logger))

// Custom config only
eng := engine.NewEngine(store, engine.WithConfig(engine.EngineConfig{
    MaxConcurrentWorkflows: 10,
    DefaultTimeout:         5 * time.Minute,
}))

// Both custom logger and config
eng := engine.NewEngine(store,
    engine.WithLogger(logger),
    engine.WithConfig(customConfig),
)
```

## Testing

Run tests:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests (requires DynamoDB)
go test -tags=integration ./store/...
```

## Requirements

- **Go**: 1.21 or higher (uses generics)
- **AWS SDK v2**: For DynamoDB store integration
- **zerolog**: For structured logging

## Dependencies

```go
require (
    github.com/aws/aws-sdk-go-v2 v1.40.0
    github.com/aws/aws-sdk-go-v2/service/dynamodb v1.53.1
    github.com/google/uuid v1.6.0
    github.com/rs/zerolog v1.34.0
)
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

Extracted from the Tendor Email Agent project and refactored into a standalone, reusable workflow engine library.

## Support

For issues, questions, or contributions, please open an issue on GitHub.
