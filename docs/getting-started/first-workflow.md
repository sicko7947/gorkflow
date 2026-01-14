# First Workflow Tutorial

Learn how to build a complete workflow step-by-step with detailed explanations.

## What We'll Build

A user registration workflow that:

1. Validates user input
2. Creates a database record
3. Sends a welcome email
4. Returns the user profile

## Prerequisites

- Gorkflow installed (`go get github.com/sicko7947/gorkflow`)
- Basic understanding of Go and structs

## Step 1: Project Setup

Create a new directory for your project:

```bash
mkdir user-registration
cd user-registration
go mod init user-registration
go get github.com/sicko7947/gorkflow
```

## Step 2: Define Your Types

Create `types.go`:

```go
package main

// Workflow input
type RegistrationInput struct {
    Email    string `json:"email" validate:"required,email"`
    Username string `json:"username" validate:"required,min=3,max=20,alphanum"`
    Password string `json:"password" validate:"required,min=8"`
}

// Step 1 output
type ValidationOutput struct {
    Email    string `json:"email"`
    Username string `json:"username"`
    Valid    bool   `json:"valid"`
}

// Step 2 output
type UserRecord struct {
    UserID   string `json:"userId" validate:"required,uuid4"`
    Email    string `json:"email"`
    Username string `json:"username"`
    Created  string `json:"created"`
}

// Step 3 output
type EmailResult struct {
    Sent    bool   `json:"sent"`
    EmailID string `json:"emailId"`
}

// Final workflow output
type RegistrationOutput struct {
    UserID   string `json:"userId"`
    Email    string `json:"email"`
    Username string `json:"username"`
    EmailSent bool  `json:"emailSent"`
}
```

**Key Points:**

- Each step has typed input and output
- Validation tags (`validate:...`) enable automatic validation
- JSON tags enable proper serialization

## Step 3: Create Your Steps

Create `steps.go`:

```go
package main

import (
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/sicko7947/gorkflow"
)

// Step 1: Validate registration data
func NewValidationStep() *gorkflow.Step[RegistrationInput, ValidationOutput] {
    return gorkflow.NewStep(
        "validate",
        "Validate Registration",
        func(ctx *gorkflow.StepContext, input RegistrationInput) (ValidationOutput, error) {
            // Validation happens automatically via struct tags!
            ctx.Logger.Info().
                Str("email", input.Email).
                Str("username", input.Username).
                Msg("Validation passed")

            return ValidationOutput{
                Email:    input.Email,
                Username: input.Username,
                Valid:    true,
            }, nil
        },
    )
}

// Step 2: Create user record
func NewCreateUserStep() *gorkflow.Step[ValidationOutput, UserRecord] {
    return gorkflow.NewStep(
        "create_user",
        "Create User Record",
        func(ctx *gorkflow.StepContext, input ValidationOutput) (UserRecord, error) {
            // Simulate database creation
            userID := uuid.New().String()
            created := time.Now().Format(time.RFC3339)

            ctx.Logger.Info().
                Str("user_id", userID).
                Str("username", input.Username).
                Msg("User created")

            return UserRecord{
                UserID:   userID,
                Email:    input.Email,
                Username: input.Username,
                Created:  created,
            }, nil
        },
        // Configure retries for database operations
        gorkflow.WithRetries(3),
        gorkflow.WithTimeout(10*time.Second),
    )
}

// Step 3: Send welcome email
func NewSendEmailStep() *gorkflow.Step[UserRecord, EmailResult] {
    return gorkflow.NewStep(
        "send_email",
        "Send Welcome Email",
        func(ctx *gorkflow.StepContext, input UserRecord) (EmailResult, error) {
            // Simulate email sending
            emailID := uuid.New().String()

            ctx.Logger.Info().
                Str("email", input.Email).
                Str("email_id", emailID).
                Msg("Welcome email sent")

            return EmailResult{
                Sent:    true,
                EmailID: emailID,
            }, nil
        },
        gorkflow.WithRetries(5), // More retries for email service
        gorkflow.WithBackoff(gorkflow.BackoffExponential),
    )
}

// Step 4: Format final output
func NewFormatStep() *gorkflow.Step[EmailResult, RegistrationOutput] {
    return gorkflow.NewStep(
        "format_output",
        "Format Registration Output",
        func(ctx *gorkflow.StepContext, input EmailResult) (RegistrationOutput, error) {
            // Get user data from previous steps using type-safe helper
            userRecord, err := gorkflow.GetOutput[UserRecord](ctx, "create_user")
            if err != nil {
                return RegistrationOutput{}, fmt.Errorf("failed to get user record: %w", err)
            }

            return RegistrationOutput{
                UserID:    userRecord.UserID,
                Email:     userRecord.Email,
                Username:  userRecord.Username,
                EmailSent: input.Sent,
            }, nil
        },
    )
}
```

**Key Points:**

- Each step is a separate function returning `*gorkflow.Step[TIn, TOut]`
- Use `ctx.Logger` for structured logging
- Configure retries and timeouts per step
- Access previous step outputs via `ctx.Data` or type-safe `gorkflow.GetOutput[T](ctx, stepID)`

## Step 4: Build the Workflow

Create `workflow.go`:

```go
package main

import (
    "github.com/sicko7947/gorkflow"
)

func NewRegistrationWorkflow() (*gorkflow.Workflow, error) {
    return gorkflow.NewWorkflow("user-registration", "User Registration").
        WithDescription("Register a new user with validation and email").
        WithVersion("1.0").
        WithConfig(gorkflow.ExecutionConfig{
            MaxRetries:     2,
            RetryDelayMs:   1000,
            RetryBackoff:   gorkflow.BackoffLinear,
            TimeoutSeconds: 60,
        }).
        Sequence(
            NewValidationStep(),
            NewCreateUserStep(),
            NewSendEmailStep(),
            NewFormatStep(),
        ).
        Build()
}
```

**Key Points:**

- Workflow-level configuration provides defaults for all steps
- Steps can override these defaults
- `Sequence()` chains steps together

## Step 5: Execute the Workflow

Create `main.go`:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/sicko7947/gorkflow/engine"
    "github.com/sicko7947/gorkflow/store"
)

func main() {
    // Create workflow
    wf, err := NewRegistrationWorkflow()
    if err != nil {
        log.Fatal("Failed to create workflow:", err)
    }

    // Create storage and engine
    store := store.NewMemoryStore()
    eng := engine.NewEngine(store)

    // Prepare input
    input := RegistrationInput{
        Email:    "user@example.com",
        Username: "johndoe",
        Password: "securepassword123",
    }

    // Start workflow
    ctx := context.Background()
    runID, err := eng.StartWorkflow(ctx, wf, input)
    if err != nil {
        log.Fatal("Failed to start workflow:", err)
    }

    fmt.Printf("Workflow started: %s\n", runID)

    // Get result
    run, err := eng.GetRun(ctx, runID)
    if err != nil {
        log.Fatal("Failed to get run:", err)
    }

    fmt.Printf("Status: %s\n", run.Status)
    fmt.Printf("Progress: %.0f%%\n", run.Progress*100)

    // Parse output
    if run.Output != nil {
        var output RegistrationOutput
        if err := json.Unmarshal(run.Output, &output); err != nil {
            log.Fatal("Failed to parse output:", err)
        }

        fmt.Printf("\nRegistration Complete!\n")
        fmt.Printf("User ID: %s\n", output.UserID)
        fmt.Printf("Username: %s\n", output.Username)
        fmt.Printf("Email: %s\n", output.Email)
        fmt.Printf("Email Sent: %v\n", output.EmailSent)
    }

    if run.Error != nil {
        fmt.Printf("Error: %s\n", run.Error.Message)
    }
}
```

## Step 6: Run It

```bash
go run .
```

**Expected Output:**

```
Workflow started: 550e8400-e29b-41d4-a716-446655440000
Status: completed
Progress: 100%

Registration Complete!
User ID: 123e4567-e89b-12d3-a456-426614174000
Username: johndoe
Email: user@example.com
Email Sent: true
```

## What You've Learned

✅ **Type Safety** - Using generics for compile-time type checking  
✅ **Validation** - Automatic input/output validation with struct tags  
✅ **Step Configuration** - Retries, timeouts, and backoff strategies  
✅ **Data Flow** - Passing data between steps  
✅ **State Access** - Reading outputs from previous steps  
✅ **Error Handling** - Graceful error management  
✅ **Logging** - Structured logging throughout execution

## Next Steps

Now that you've built your first workflow, explore:

- **[Parallel Execution](../advanced-usage/parallel-execution.md)** - Run independent steps in parallel
- **[Conditional Execution](../advanced-usage/conditional-execution.md)** - Add conditional logic
- **[Storage Backends](../storage/overview.md)** - Use persistent storage
- **[Real-World Patterns](../examples/real-world-patterns.md)** - Common workflow patterns

## Common Enhancements

### Add Parallel Steps

```go
.Parallel(
    NewSendEmailStep(),
    NewCreateProfileStep(),
    NewLogActivityStep(),
)
```

### Add Conditional Steps

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var user UserRecord
    ctx.Outputs.Get(ctx.Context, "create_user", &user)
    return user.Email != "", nil
}

.ThenStepIf(NewSendEmailStep(), condition, nil)
```

### Use Persistent Storage

```go
// Use LibSQL
store, _ := store.NewLibSQLStore("file:./workflows.db")
```

---

**Complete!** You've built your first Gorkflow workflow. Check out more [examples](../examples/sequential.md) →
