# StepContext

The `StepContext` is passed to every step handler, providing access to execution metadata, logging, state, previous step data, and custom context.

## Structure

```go
type StepContext struct {
    context.Context                    // Go context for cancellation/timeout

    RunID         string              // Workflow run ID
    StepID        string              // Current step ID
    Attempt       int                 // Current retry attempt (0-based)

    Logger        zerolog.Logger      // Structured logger enriched with step context
    Data          StepDataAccessor    // Access to other steps' inputs and outputs
    State         StateAccessor       // Workflow-level key-value state

    CustomContext any                 // User-defined context value
}
```

## Fields

### `Context`

The embedded `context.Context` carries cancellation signals and the per-step timeout. Use it for any operation that should respect the step's deadline:

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    req, _ := http.NewRequestWithContext(ctx.Context, "GET", url, nil)
    resp, err := http.DefaultClient.Do(req)
    // ...
}
```

### `RunID`

The unique identifier for the current workflow run.

### `StepID`

The ID of the currently executing step.

### `Attempt`

The current retry attempt number (0-based). `0` is the first attempt, `1` is the first retry, etc.

```go
if ctx.Attempt > 0 {
    ctx.Logger.Info().Int("attempt", ctx.Attempt).Msg("Retrying step")
}
```

### `Logger`

A `zerolog.Logger` pre-configured with step context fields (step ID, step name, run ID). Use it for structured logging within handlers.

```go
ctx.Logger.Info().
    Str("order_id", input.OrderID).
    Msg("Processing order")

ctx.Logger.Debug().
    Int("item_count", len(items)).
    Msg("Items loaded")
```

### `Data`

A `StepDataAccessor` for reading inputs and outputs from other steps in the same run. See the helpers below for type-safe access.

### `State`

A `StateAccessor` for reading and writing workflow-level key-value state. See [State Management](state-management.md).

### `CustomContext`

An `any` value set via `WorkflowBuilder.WithContext`. Use the `GetContext[T]` helper for type-safe retrieval.

## Type-Safe Helpers

### `GetContext[T]`

```go
func GetContext[T any](ctx *StepContext) (T, error)
```

Retrieves the custom context with type assertion. Returns an error if the context is `nil` or not of the expected type.

```go
type AppDeps struct {
    DB     *sql.DB
    Cache  *redis.Client
}

// Set during workflow build:
wf, _ := gorkflow.NewWorkflow("app", "App").
    WithContext(&AppDeps{DB: db, Cache: cache}).
    ThenStep(myStep).
    Build()

// Access in handler:
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    deps, err := gorkflow.GetContext[*AppDeps](ctx)
    if err != nil {
        return MyOutput{}, err
    }
    rows, err := deps.DB.QueryContext(ctx.Context, "SELECT ...")
    // ...
}
```

### `GetOutput[T]`

```go
func GetOutput[T any](ctx *StepContext, stepID string) (T, error)
```

Retrieves the output from a previously completed step with type-safe deserialization. The output is loaded from the store (with caching).

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    userData, err := gorkflow.GetOutput[UserData](ctx, "fetch-user")
    if err != nil {
        return MyOutput{}, fmt.Errorf("could not get user data: %w", err)
    }
    // Use userData.Email, userData.Name, etc.
    return MyOutput{Greeting: "Hello " + userData.Name}, nil
}
```

### `GetInput[T]`

```go
func GetInput[T any](ctx *StepContext, stepID string) (T, error)
```

Retrieves the input that was passed to a previously executed step.

```go
originalInput, err := gorkflow.GetInput[OrderRequest](ctx, "validate-order")
if err != nil {
    return MyOutput{}, err
}
```

## StepDataAccessor Interface

The `Data` field implements:

```go
type StepDataAccessor interface {
    GetOutput(stepID string, target interface{}) error
    GetInput(stepID string, target interface{}) error
    HasOutput(stepID string) bool
}
```

### `HasOutput`

Check if a step has produced output before attempting to read it:

```go
if ctx.Data.HasOutput("optional-step") {
    result, _ := gorkflow.GetOutput[OptionalResult](ctx, "optional-step")
    // Use result...
}
```

## Complete Example

```go
type AppContext struct {
    NotificationService *NotificationService
}

type ValidateInput struct {
    Email string `json:"email" validate:"required,email"`
}

type ValidateOutput struct {
    Valid bool   `json:"valid"`
    Email string `json:"email"`
}

type NotifyOutput struct {
    Sent bool `json:"sent"`
}

func main() {
    validateStep := gorkflow.NewStep(
        "validate",
        "Validate Email",
        func(ctx *gorkflow.StepContext, input ValidateInput) (ValidateOutput, error) {
            ctx.Logger.Info().Str("email", input.Email).Msg("Validating")

            // Store in state for later steps
            ctx.State.Set("validated_email", input.Email)

            return ValidateOutput{Valid: true, Email: input.Email}, nil
        },
    )

    notifyStep := gorkflow.NewStep(
        "notify",
        "Send Notification",
        func(ctx *gorkflow.StepContext, input ValidateOutput) (NotifyOutput, error) {
            // Access custom context
            appCtx, err := gorkflow.GetContext[*AppContext](ctx)
            if err != nil {
                return NotifyOutput{}, err
            }

            // Access previous step output
            validated, _ := gorkflow.GetOutput[ValidateOutput](ctx, "validate")

            // Use the logger
            ctx.Logger.Info().
                Str("email", validated.Email).
                Int("attempt", ctx.Attempt).
                Msg("Sending notification")

            err = appCtx.NotificationService.Send(ctx.Context, validated.Email)
            return NotifyOutput{Sent: err == nil}, err
        },
        gorkflow.WithRetries(3),
    )

    wf, _ := gorkflow.NewWorkflow("email-flow", "Email Flow").
        WithContext(&AppContext{NotificationService: notifService}).
        ThenStep(validateStep).
        ThenStep(notifyStep).
        Build()

    _ = wf
}
```

---

**Next**: Learn about [State Management](state-management.md) →
