# Project Structure

Learn how to organize your Gorkflow projects for maintainability and scalability.

## Recommended Structure

Here's a recommended project structure for Gorkflow applications:

```
my-workflow-app/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── workflows/               # Workflow definitions
│   │   ├── registration/
│   │   │   ├── workflow.go      # Workflow builder
│   │   │   ├── steps.go         # Step implementations
│   │   │   └── types.go         # Input/output types
│   │   └── payment/
│   │       ├── workflow.go
│   │       ├── steps.go
│   │       └── types.go
│   ├── steps/                   # Reusable steps
│   │   ├── email/
│   │   │   └── send.go          # Email sending step
│   │   ├── database/
│   │   │   ├── create.go
│   │   │   └── update.go
│   │   └── validation/
│   │       └── validate.go
│   ├── services/                # Business logic services
│   │   ├── user_service.go
│   │   └── email_service.go
│   └── config/
│       └── config.go            # Configuration
├── pkg/                         # Public packages (if any)
├── scripts/                     # Helper scripts
│   └── create-dynamodb-table.sh
├── go.mod
├── go.sum
└── README.md
```

## File Organization

### Workflow Package Structure

Each workflow should be in its own package:

**`internal/workflows/registration/types.go`**

```go
package registration

// Input and output type definitions
type RegistrationInput struct {
    Email    string `json:"email" validate:"required,email"`
    Username string `json:"username" validate:"required,min=3"`
}

type RegistrationOutput struct {
    UserID string `json:"userId"`
}
```

**`internal/workflows/registration/steps.go`**

```go
package registration

import "github.com/sicko7947/gorkflow"

// Step constructors
func NewValidateStep() *gorkflow.Step[RegistrationInput, ValidationOutput] {
    return gorkflow.NewStep("validate", "Validate Input", validateHandler)
}

func validateHandler(ctx *gorkflow.StepContext, input RegistrationInput) (ValidationOutput, error) {
    // Implementation
}
```

**`internal/workflows/registration/workflow.go`**

```go
package registration

import "github.com/sicko7947/gorkflow"

// Workflow constructor
func New() (*gorkflow.Workflow, error) {
    return gorkflow.NewWorkflow("registration", "User Registration").
        Sequence(
            NewValidateStep(),
            NewCreateUserStep(),
            NewSendEmailStep(),
        ).
        Build()
}
```

### Main Application

**`cmd/server/main.go`**

```go
package main

import (
    "context"
    "log"

    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"

    "your-module/internal/workflows/registration"
    "your-module/internal/config"
)

func main() {
    // Load configuration
    cfg := config.Load()

    // Initialize store
    store, err := initStore(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Create engine
    eng := engine.NewEngine(store)

    // Register workflows
    workflows := map[string]*gorkflow.Workflow{
        "registration": registration.New(),
    }

    // Start server or workflow execution
    // ...
}

func initStore(cfg *config.Config) (store.Store, error) {
    switch cfg.StoreType {
    case "memory":
        return store.NewMemoryStore(), nil
    case "dynamodb":
        return store.NewDynamoDBStore(/* ... */)
    case "libsql":
        return store.NewLibSQLStore(cfg.DatabaseURL)
    default:
        return store.NewMemoryStore(), nil
    }
}
```

## Reusable Steps

Create reusable steps that can be shared across workflows:

**`internal/steps/email/send.go`**

```go
package email

import (
    "github.com/sicko7947/gorkflow"
)

type EmailInput struct {
    To      string `json:"to" validate:"required,email"`
    Subject string `json:"subject" validate:"required"`
    Body    string `json:"body" validate:"required"`
}

type EmailOutput struct {
    Sent    bool   `json:"sent"`
    EmailID string `json:"emailId"`
}

func NewSendStep(svc EmailService) *gorkflow.Step[EmailInput, EmailOutput] {
    return gorkflow.NewStep(
        "send_email",
        "Send Email",
        func(ctx *gorkflow.StepContext, input EmailInput) (EmailOutput, error) {
            emailID, err := svc.Send(input.To, input.Subject, input.Body)
            if err != nil {
                return EmailOutput{}, err
            }
            return EmailOutput{Sent: true, EmailID: emailID}, nil
        },
        gorkflow.WithRetries(3),
    )
}
```

Use in multiple workflows:

```go
// In registration workflow
emailStep := email.NewSendStep(emailService)

// In password reset workflow
emailStep := email.NewSendStep(emailService)
```

## Configuration Management

**`internal/config/config.go`**

```go
package config

import (
    "os"
    "github.com/sicko7947/gorkflow"
)

type Config struct {
    StoreType    string
    DatabaseURL  string
    WorkflowDefaults gorkflow.ExecutionConfig
}

func Load() *Config {
    return &Config{
        StoreType:   getEnv("STORE_TYPE", "memory"),
        DatabaseURL: getEnv("DATABASE_URL", ""),
        WorkflowDefaults: gorkflow.ExecutionConfig{
            MaxRetries:     parseInt(getEnv("MAX_RETRIES", "3")),
            RetryDelayMs:   parseInt(getEnv("RETRY_DELAY_MS", "1000")),
            TimeoutSeconds: parseInt(getEnv("TIMEOUT_SECONDS", "300")),
        },
    }
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
```

## Dependency Injection

Use dependency injection for better testability:

```go
// Define interfaces
type UserService interface {
    Create(email, username string) (string, error)
}

type EmailService interface {
    Send(to, subject, body string) (string, error)
}

// Step constructor accepts dependencies
func NewCreateUserStep(svc UserService) *gorkflow.Step[UserInput, UserOutput] {
    return gorkflow.NewStep(
        "create_user",
        "Create User",
        func(ctx *gorkflow.StepContext, input UserInput) (UserOutput, error) {
            userID, err := svc.Create(input.Email, input.Username)
            if err != nil {
                return UserOutput{}, err
            }
            return UserOutput{UserID: userID}, nil
        },
    )
}
```

## Testing Structure

```
my-workflow-app/
├── internal/
│   └── workflows/
│       └── registration/
│           ├── workflow.go
│           ├── workflow_test.go       # Workflow tests
│           ├── steps.go
│           └── steps_test.go          # Step tests
```

**`workflow_test.go`**

```go
package registration_test

import (
    "context"
    "testing"

    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"

    "your-module/internal/workflows/registration"
)

func TestRegistrationWorkflow(t *testing.T) {
    wf, err := registration.New()
    if err != nil {
        t.Fatal(err)
    }

    store := store.NewMemoryStore()
    eng := engine.NewEngine(store)

    input := registration.RegistrationInput{
        Email:    "test@example.com",
        Username: "testuser",
    }

    ctx := context.Background()
    runID, err := eng.StartWorkflow(ctx, wf, input)
    if err != nil {
        t.Fatal(err)
    }

    run, err := eng.GetRun(ctx, runID)
    if err != nil {
        t.Fatal(err)
    }

    if run.Status != "completed" {
        t.Errorf("Expected status completed, got %s", run.Status)
    }
}
```

## Best Practices

### 1. Separate Concerns

✅ **Do**: Keep workflow definitions separate from business logic

```go
// workflow.go - just orchestration
func New(userSvc UserService) (*gorkflow.Workflow, error) {
    return gorkflow.NewWorkflow("registration", "Registration").
        Sequence(NewValidateStep(), NewCreateUserStep(userSvc)).
        Build()
}

// services/user_service.go - business logic
func (s *UserService) CreateUser(...) { /* ... */ }
```

❌ **Don't**: Mix business logic in workflow definitions

### 2. Use Package-Level Constructors

✅ **Do**: Export workflow constructors

```go
package registration

func New() (*gorkflow.Workflow, error) { /* ... */ }
```

### 3. Keep Types Close

✅ **Do**: Define types in the same package as the workflow

```go
package registration

type RegistrationInput struct { /* ... */ }
type RegistrationOutput struct { /* ... */ }

func New() (*gorkflow.Workflow, error) { /* ... */ }
```

### 4. Use Internal Packages

✅ **Do**: Put workflow-specific code in `internal/`

- Prevents external imports
- Keeps implementation details private

### 5. Configuration as Code

✅ **Do**: Make workflows configurable

```go
func New(cfg Config) (*gorkflow.Workflow, error) {
    return gorkflow.NewWorkflow("reg", "Registration").
        WithConfig(cfg.ExecutionConfig).
        Sequence(/* ... */).
        Build()
}
```

## Example: Multi-Workflow Application

```go
// cmd/server/main.go
package main

import (
    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"

    "app/internal/workflows/registration"
    "app/internal/workflows/payment"
    "app/internal/workflows/notification"
)

type WorkflowRegistry struct {
    engine    *engine.Engine
    workflows map[string]*gorkflow.Workflow
}

func NewRegistry(store store.Store) (*WorkflowRegistry, error) {
    eng := engine.NewEngine(store)

    workflows := make(map[string]*gorkflow.Workflow)

    // Register all workflows
    if wf, err := registration.New(); err == nil {
        workflows["registration"] = wf
    }

    if wf, err := payment.New(); err == nil {
        workflows["payment"] = wf
    }

    if wf, err := notification.New(); err == nil {
        workflows["notification"] = wf
    }

    return &WorkflowRegistry{
        engine: eng,
        workflows: workflows,
    }, nil
}

func (r *WorkflowRegistry) Execute(name string, input any) (string, error) {
    wf, ok := r.workflows[name]
    if !ok {
        return "", fmt.Errorf("workflow %s not found", name)
    }

    return r.engine.StartWorkflow(context.Background(), wf, input)
}
```

---

**Next**: Learn about [Workflows](../core-concepts/workflows.md) →
