# Steps

Steps are the fundamental building blocks of workflows. Each step performs a specific task with type-safe inputs and outputs.

## What is a Step?

A step is a single unit of work in a workflow that:

- Has a unique ID and name
- Takes typed input and produces typed output
- Can be configured with retries, timeouts, and backoffs
- Validates input and output automatically
- Logs execution progress
- Can access workflow state and previous step outputs

## Creating a Step

Use the `NewStep` function with Go generics:

```go
step := gorkflow.NewStep(
    "step-id",          // Unique identifier
    "Step Name",        // Human-readable name
    handlerFunction,    // Your business logic
    options...,         // Optional: WithRetries, WithTimeout, etc.
)
```

### Complete Example

```go
type UserInput struct {
    Email string `json:"email" validate:"required,email"`
}

type UserOutput struct {
    UserID string `json:"userId" validate:"required,uuid4"`
}

step := gorkflow.NewStep(
    "create-user",
    "Create User",
    func(ctx *gorkflow.StepContext, input UserInput) (UserOutput, error) {
        // Your business logic
        userID := uuid.New().String()

        ctx.Logger.Info().
            Str("email", input.Email).
            Str("user_id", userID).
            Msg("User created")

        return UserOutput{UserID: userID}, nil
    },
    gorkflow.WithRetries(3),
    gorkflow.WithTimeout(30 * time.Second),
)
```

## Step Handlers

The handler function signature:

```go
func(ctx *gorkflow.StepContext, input TInput) (TOutput, error)
```

- `ctx` - Step execution context with logger, state, and outputs
- `input` - Typed input (automatically unmarshaled and validated)
- Returns typed output and error

### Handler Context

The `StepContext` provides:

```go
type StepContext struct {
    Context  context.Context    // Go context for cancellation
    Logger   zerolog.Logger     // Structured logger
    State    StateAccessor      // Workflow state
    Outputs  OutputAccessor     // Previous step outputs
    RunID    string            // Workflow run ID
    StepID   string            // Current step ID
}
```

### Using the Context

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    // 1. Use the logger
    ctx.Logger.Info().Msg("Starting processing")

    // 2. Access workflow state
    var counter int
    ctx.State.Get("counter", &counter)
    counter++
    ctx.State.Set(ctx.Context, "counter", counter)

    // 3. Get previous step output
    var prevOutput PreviousStepOutput
    ctx.Outputs.Get(ctx.Context, "previous-step-id", &prevOutput)

    // 4. Check for cancellation
    select {
    case <-ctx.Context.Done():
        return MyOutput{}, ctx.Context.Err()
    default:
        // Continue processing
    }

    return MyOutput{Data: "processed"}, nil
}
```

## Step Configuration

### Retries

Configure how many times a step should retry on failure:

```go
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithRetries(5),  // Retry up to 5 times
)
```

### Timeout

Set maximum execution time:

```go
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithTimeout(60 * time.Second),  // 60 second timeout
)
```

### Backoff Strategy

Control retry delay behavior:

```go
// Linear backoff: delay stays constant
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithRetries(3),
    gorkflow.WithBackoff(gorkflow.BackoffLinear),
)

// Exponential backoff: delay doubles each retry
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithRetries(3),
    gorkflow.WithBackoff(gorkflow.BackoffExponential),
)
```

### Combined Configuration

```go
step := gorkflow.NewStep("critical-step", "Critical Operation", handler,
    gorkflow.WithRetries(5),
    gorkflow.WithTimeout(120 * time.Second),
    gorkflow.WithBackoff(gorkflow.BackoffExponential),
)
```

## Input/Output Validation

Validation is **enabled by default** using struct tags:

```go
type MyInput struct {
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"required,gte=18,lte=120"`
    Username string `json:"username" validate:"required,min=3,max=20,alphanum"`
}

type MyOutput struct {
    UserID string `json:"userId" validate:"required,uuid4"`
}

step := gorkflow.NewStep("my-step", "My Step", handler)
// Validation happens automatically!
```

### Validation Flow

1. **Input validation** - Before handler executes
2. **Handler execution** - Your business logic
3. **Output validation** - After handler executes

### Disable Validation

If needed, you can disable validation:

```go
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithoutValidation(),  // Disable all validation
)
```

See [Validation](validation.md) for comprehensive validation documentation.

## Type Safety

Steps use Go generics for compile-time type safety:

```go
// Type-safe step definition
step := gorkflow.NewStep[UserInput, UserOutput](
    "create-user",
    "Create User",
    func(ctx *gorkflow.StepContext, input UserInput) (UserOutput, error) {
        // input is UserInput (not interface{})
        // return must be UserOutput (not interface{})
        return UserOutput{UserID: "123"}, nil
    },
)
```

Type parameters can usually be inferred:

```go
// Types inferred from handler signature
step := gorkflow.NewStep(
    "create-user",
    "Create User",
    handlerFunction,  // func(ctx, UserInput) (UserOutput, error)
)
```

## Step Patterns

### Simple Transformation

```go
step := gorkflow.NewStep(
    "transform",
    "Transform Data",
    func(ctx *gorkflow.StepContext, input DataInput) (DataOutput, error) {
        return DataOutput{
            Transformed: strings.ToUpper(input.Value),
        }, nil
    },
)
```

### External API Call

```go
step := gorkflow.NewStep(
    "api-call",
    "Call External API",
    func(ctx *gorkflow.StepContext, input APIInput) (APIOutput, error) {
        resp, err := http.Get(input.URL)
        if err != nil {
            return APIOutput{}, err
        }
        defer resp.Body.Close()

        // Process response...
        return APIOutput{Data: data}, nil
    },
    gorkflow.WithRetries(3),  // Retry on network failures
    gorkflow.WithTimeout(30 * time.Second),
)
```

### Database Operation

```go
step := gorkflow.NewStep(
    "db-write",
    "Write to Database",
    func(ctx *gorkflow.StepContext, input DBInput) (DBOutput, error) {
        result, err := db.Exec(
            ctx.Context,
            "INSERT INTO users (email, username) VALUES ($1, $2)",
            input.Email, input.Username,
        )
        if err != nil {
            return DBOutput{}, err
        }

        return DBOutput{RowsAffected: result.RowsAffected()}, nil
    },
    gorkflow.WithRetries(2),  // Retry transient DB errors
)
```

### Aggregation

```go
step := gorkflow.NewStep(
    "aggregate",
    "Aggregate Results",
    func(ctx *gorkflow.StepContext, input AggInput) (AggOutput, error) {
        // Get outputs from multiple previous steps
        var result1 Result1
        var result2 Result2
        var result3 Result3

        ctx.Outputs.Get(ctx.Context, "step1", &result1)
        ctx.Outputs.Get(ctx.Context, "step2", &result2)
        ctx.Outputs.Get(ctx.Context, "step3", &result3)

        // Aggregate
        total := result1.Value + result2.Value + result3.Value

        return AggOutput{Total: total}, nil
    },
)
```

## Conditional Steps

Steps that execute only if a condition is met:

```go
// Define condition
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var shouldProcess bool
    ctx.State.Get("should_process", &shouldProcess)
    return shouldProcess, nil
}

// Create conditional step
baseStep := gorkflow.NewStep("process", "Process Data", handler)
conditionalStep := gorkflow.NewConditionalStep(baseStep, condition, defaultOutput)

// Or use builder API
wf, _ := gorkflow.NewWorkflow("my-wf", "My Workflow").
    ThenStepIf(processStep, condition, nil).
    Build()
```

See [Conditional Execution](../advanced-usage/conditional-execution.md) for details.

## Reusable Steps

Create step factories for reusability:

```go
// Step factory with dependency injection
func NewEmailStep(emailService EmailService) *gorkflow.Step[EmailInput, EmailOutput] {
    return gorkflow.NewStep(
        "send-email",
        "Send Email",
        func(ctx *gorkflow.StepContext, input EmailInput) (EmailOutput, error) {
            emailID, err := emailService.Send(
                input.To,
                input.Subject,
                input.Body,
            )
            if err != nil {
                return EmailOutput{}, err
            }
            return EmailOutput{EmailID: emailID, Sent: true}, nil
        },
        gorkflow.WithRetries(3),
    )
}

// Use in multiple workflows
emailStep1 := NewEmailStep(emailService)
emailStep2 := NewEmailStep(emailService)
```

## Error Handling

### Return Errors

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    if input.Value == "" {
        return MyOutput{}, errors.New("value cannot be empty")
    }

    result, err := externalService.Call(input.Value)
    if err != nil {
        return MyOutput{}, fmt.Errorf("external service failed: %w", err)
    }

    return MyOutput{Result: result}, nil
}
```

### Retry on Error

Errors trigger automatic retries based on step configuration:

```go
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithRetries(3),  // Retry up to 3 times on error
)
```

### Continue on Error

Allow workflow to continue even if step fails:

```go
wf, _ := gorkflow.NewWorkflow("my-wf", "My Workflow").
    WithConfig(gorkflow.ExecutionConfig{
        ContinueOnError: true,  // Don't stop on step failures
    }).
    Build()
```

## Best Practices

### 1. Use Descriptive IDs and Names

✅ **Good**:

```go
gorkflow.NewStep("validate-user-email", "Validate User Email", handler)
```

❌ **Bad**:

```go
gorkflow.NewStep("step1", "Step", handler)
```

### 2. Keep Steps Focused

Each step should do one thing well:

✅ **Good**:

```go
validateStep := gorkflow.NewStep("validate", "Validate", validateHandler)
createStep := gorkflow.NewStep("create", "Create", createHandler)
emailStep := gorkflow.NewStep("email", "Send Email", emailHandler)
```

❌ **Bad**:

```go
// One step doing everything
allInOneStep := gorkflow.NewStep("process", "Do Everything", megaHandler)
```

### 3. Use Appropriate Timeouts

```go
// Quick operation
quickStep := gorkflow.NewStep("quick", "Quick", handler,
    gorkflow.WithTimeout(5 * time.Second),
)

// Long-running operation
longStep := gorkflow.NewStep("long", "Long", handler,
    gorkflow.WithTimeout(5 * time.Minute),
)
```

### 4. Configure Retries for External Calls

```go
// External API call - higher retries
apiStep := gorkflow.NewStep("api", "API Call", handler,
    gorkflow.WithRetries(5),
    gorkflow.WithBackoff(gorkflow.BackoffExponential),
)

// Local computation - lower retries
computeStep := gorkflow.NewStep("compute", "Compute", handler,
    gorkflow.WithRetries(1),
)
```

### 5. Add Validation Tags

```go
type Input struct {
    Email  string `json:"email" validate:"required,email"`
    Age    int    `json:"age" validate:"required,gte=0,lte=150"`
}
```

### 6. Log Meaningful Information

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    ctx.Logger.Info().
        Str("input_id", input.ID).
        Int("count", input.Count).
        Msg("Processing started")

    // ... processing ...

    ctx.Logger.Info().
        Str("output_id", output.ID).
        Msg("Processing completed")

    return output, nil
}
```

---

**Next**: Learn about the [Execution Graph](execution-graph.md) and how steps are orchestrated →
