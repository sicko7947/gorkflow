# gorkflow Examples

This directory contains complete, runnable examples demonstrating various features of the gorkflow engine.

## Available Examples

### 1. [Sequential](./sequential/)

**Demonstrates**: Basic sequential workflow execution

A straightforward example showing:

- Sequential step execution
- Type-safe data passing between steps
- Step configuration (retries, timeouts, backoff strategies)
- Fluent builder API

**Workflow**: Add two numbers → Multiply by factor → Format result

```bash
cd sequential
go run main/main.go
```

### 2. [Parallel](./parallel/)

**Demonstrates**: Parallel workflow execution

Shows how to use the `Parallel` builder API:

- Parallel step execution
- Aggregating results

**Workflow**: Add two numbers AND Multiply result (in parallel) -> Format result

```bash
cd parallel
go run main/main.go
```

### 3. [Conditional Execution](./conditional/)

**Demonstrates**: Runtime conditional step execution

Shows how to use the `ThenStepIf` builder API for conditional execution:

- Builder-level conditional API
- Condition evaluation from workflow state
- Default values when steps are skipped
- Runtime decision-making

**Workflow**: Setup flags → Conditionally double value → Conditionally format

```bash
cd conditional
go run main/main.go
```

### 4. [Validation](./validation/)

**Demonstrates**: Automatic input/output validation

Shows how validation works with struct tags:

- Automatic validation using struct tags
- Email, UUID, and custom validators
- Validation error handling
- Multiple validation scenarios

**Workflow**: Validate user → Send email → Format result

```bash
cd validation
go run main.go
```

### 5. [LibSQL Persistence](./libsql_persistence/)

**Demonstrates**: Persistent workflow storage with LibSQL/SQLite

Shows how to use the LibSQL store for persistence:

- Local SQLite database file storage
- Workflow run persistence
- Step execution history
- Database querying and inspection
- Efficient sorting by execution_index

**Workflow**: Add numbers → Multiply → Format (with persistence)

```bash
cd libsql_persistence
go run main.go
```

## Running Examples

Each example is a standalone Go package that can be run directly.
They start a Fiber HTTP server.

## Example Structure

Each example follows this structure:

```
example/
├── <example-name>/
│   ├── main/
│   │   └── main.go     # Entry point, HTTP server
│   ├── types.go        # Input/output type definitions
│   ├── steps.go        # Step implementations
│   └── workflow.go     # Workflow builder function
└── README.md           # This file - examples overview
```

## Learning Path

**Recommended order for learning:**

1. **sequential/** - Start here to understand the basics
2. **parallel/** - Learn parallel execution
3. **conditional/** - Learn conditional execution patterns
4. **validation/** - Learn input/output validation
5. **libsql_persistence/** - Learn persistent storage with LibSQL

Each example builds on concepts from the previous ones.
