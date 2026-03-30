# Type Safety

Gorkflow uses Go generics to provide compile-time type safety for step inputs and outputs, while using JSON serialization at runtime for flexibility.

## Overview

Every step is parameterized with input and output types:

```go
step := gorkflow.NewStep[InputType, OutputType]("id", "name", handler)
```

Type parameters are typically inferred from the handler signature, so you don't need to specify them explicitly:

```go
step := gorkflow.NewStep("id", "name",
    func(ctx *gorkflow.StepContext, input InputType) (OutputType, error) {
        // ...
    },
)
```

## Compile-Time Safety

### Handler Signature

The handler function enforces type constraints at compile time:

```go
type StepHandler[TIn, TOut any] func(ctx *StepContext, input TIn) (TOut, error)
```

This means:
- The input parameter type is known at compile time
- The return type is known at compile time
- You cannot accidentally return the wrong type

```go
// Compile error: cannot use string as OutputType
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    return "wrong type", nil  // ← won't compile
}
```

### Generic Helper Functions

Type-safe helpers use generics to prevent type mismatches:

```go
// Output retrieval — T must match the actual output type
output, err := gorkflow.GetOutput[UserData](ctx, "fetch-user")

// Input retrieval
input, err := gorkflow.GetInput[OrderRequest](ctx, "validate-order")

// State access
count, err := gorkflow.GetTyped[int](ctx.State, "counter")
err := gorkflow.SetTyped(ctx.State, "counter", 42)

// Custom context
deps, err := gorkflow.GetContext[*AppDeps](ctx)
```

### Conditional Steps

`NewConditionalStep` preserves type safety:

```go
// The default value must be *TOut
step := gorkflow.NewStep[MyInput, MyOutput]("id", "name", handler)
conditionalStep := gorkflow.NewConditionalStep(step, condition, &MyOutput{Default: true})
```

## Runtime Behavior

### JSON Serialization

At the boundary between steps, data is serialized to `[]byte` (JSON). The `StepExecutor` interface works with raw bytes:

```go
type StepExecutor interface {
    Execute(ctx *StepContext, input []byte) (output []byte, err error)
    // ...
}
```

Inside `Step[TIn, TOut].Execute`:
1. Input `[]byte` is unmarshaled to `TIn`
2. The handler runs with typed `TIn` and returns typed `TOut`
3. Output `TOut` is marshaled back to `[]byte`

This design allows the engine to work with a single interface (`StepExecutor`) while each step maintains its own type safety.

### Type Information at Runtime

Each step stores its input and output types via `reflect.Type`:

```go
func (s *Step[TIn, TOut]) InputType() reflect.Type {
    return s.inputType  // reflect.TypeOf((*TIn)(nil)).Elem()
}

func (s *Step[TIn, TOut]) OutputType() reflect.Type {
    return s.outputType
}
```

This is used by the conditional step wrapper to determine if input/output types match (for pass-through behavior when a step is skipped).

### Validation

Struct validation tags are checked at runtime after JSON deserialization:

```go
type UserInput struct {
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"required,gte=18"`
}
```

Validation happens at two points:
1. **Input validation** — after unmarshaling, before the handler runs
2. **Output validation** — after the handler returns, before marshaling

Only struct types are validated. Primitive types and non-struct types pass through without validation.

## Type Compatibility Between Steps

Steps are chained through JSON serialization. The output of one step becomes the input of the next. Types must be **JSON-compatible**, not identical:

```go
// Step 1 output
type UserOutput struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Step 2 input — compatible because it's a subset of the JSON
type NotifyInput struct {
    Email string `json:"email"`
}
```

This works because `{"name":"Alice","email":"alice@example.com"}` can be unmarshaled into `NotifyInput` — the `name` field is silently ignored.

### Mismatched Types

If types are incompatible, the step will fail at runtime during JSON unmarshaling:

```go
// Step 1 outputs a string
type Step1Output struct {
    Count int `json:"count"`
}

// Step 2 expects a different structure
type Step2Input struct {
    Items []string `json:"items"`  // Won't get populated from {"count": 5}
}
```

The step will still run (JSON unmarshaling won't fail for missing fields), but the input will have zero values for unmatched fields. Add `validate:"required"` tags to catch this at runtime.

## Design Tradeoffs

| Aspect | Compile-Time | Runtime |
|--------|-------------|---------|
| Handler signature | Fully type-checked | N/A |
| Step chaining | Not checked (JSON boundary) | Checked via validation tags |
| Output retrieval | Generic return type | JSON deserialization can fail |
| State access | Generic helpers available | JSON deserialization can fail |

The hybrid approach gives you:
- **Safety within a step** — the handler is fully typed
- **Flexibility between steps** — JSON allows structural subtyping
- **Validation at boundaries** — struct tags catch mismatches at runtime

---

**Next**: Learn about [State Management](state-management.md) →
