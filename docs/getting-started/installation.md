# Installation

Get started with Gorkflow by adding it to your Go project.

## Requirements

- **Go 1.21 or higher** (Gorkflow uses Go generics extensively)
- A Go module-enabled project

## Install Gorkflow

Add Gorkflow to your project using `go get`:

```bash
go get github.com/sicko7947/gorkflow
```

This will add Gorkflow to your `go.mod` file:

```go
require (
    github.com/sicko7947/gorkflow v0.1.0
)
```

## Dependencies

Gorkflow has minimal required dependencies:

### Core Dependencies

```go
require (
    github.com/google/uuid v1.6.0       // UUID generation for workflow runs
    github.com/rs/zerolog v1.34.0      // Structured logging
    github.com/go-playground/validator/v10 v10.12.0 // Validation
)
```

### Optional Dependencies

Depending on which storage backend you choose, you may need additional dependencies:

#### For LibSQL/SQLite Store

```bash
go get github.com/tursodatabase/libsql-client-go/libsql
```

The LibSQL store is included in the main package and works with:

- Local SQLite files (`file:./workflow.db`)
- Remote Turso databases (`libsql://...`)

## Verify Installation

Create a simple test file to verify your installation:

```go
package main

import (
    "fmt"
    workflow "github.com/sicko7947/gorkflow"
)

func main() {
    wf, err := workflow.NewWorkflow("test", "Test Workflow").Build()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Created workflow: %s\n", wf.Name)
}
```

Run it:

```bash
go run main.go
```

You should see:

```
Created workflow: Test Workflow
```

## Next Steps

Now that you have Gorkflow installed, move on to:

- **[Quick Start](quick-start.md)** - Build your first workflow
- **[First Workflow Tutorial](first-workflow.md)** - Step-by-step guide
- **[Project Structure](project-structure.md)** - Organize your code

## Troubleshooting

### Module Resolution Issues

If you encounter module resolution errors:

```bash
go mod tidy
```

### Version Conflicts

If you have dependency conflicts, check your `go.mod` file and update to compatible versions:

```bash
go get -u github.com/sicko7947/gorkflow
go mod tidy
```

### Import Errors

Make sure you're importing the correct package:

```go
import (
    "github.com/sicko7947/gorkflow"           // Main package
    "github.com/sicko7947/gorkflow/engine"    // Engine
    "github.com/sicko7947/gorkflow/store"     // Storage backends
)
```

## IDE Setup

### VS Code

Install the Go extension for the best development experience:

```bash
code --install-extension golang.go
```

### GoLand / IntelliJ IDEA

GoLand has built-in Go support with excellent generics support.

---

**Ready to build your first workflow?** Continue to the [Quick Start Guide](quick-start.md) →
