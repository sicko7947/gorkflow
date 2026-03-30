# State Management

Workflow state allows steps to share data through a persistent key-value store, separate from step inputs and outputs.

## Overview

Every workflow run has an associated state store. Steps access it through the `StateAccessor` interface on `ctx.State`. State is:

- **Persistent** — values are saved to the configured store backend
- **Cached** — reads are cached in memory for the duration of the run
- **JSON-serialized** — values are marshaled to JSON for storage
- **Shared** — all steps in a run share the same state

## StateAccessor Interface

```go
type StateAccessor interface {
    Set(key string, value interface{}) error
    Get(key string, target interface{}) error
    Delete(key string) error
    Has(key string) bool
    GetAll() (map[string][]byte, error)
}
```

## Methods

### `Set`

Stores a value in the workflow state. The value is JSON-marshaled and persisted to the store.

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    ctx.State.Set("user_email", input.Email)
    ctx.State.Set("item_count", 42)
    ctx.State.Set("config", map[string]string{"mode": "fast"})
    return MyOutput{}, nil
}
```

### `Get`

Retrieves a value from state into the target pointer. Returns an error if the key does not exist or deserialization fails.

```go
func handler(ctx *gorkflow.StepContext, input MyInput) (MyOutput, error) {
    var email string
    if err := ctx.State.Get("user_email", &email); err != nil {
        return MyOutput{}, fmt.Errorf("missing user email: %w", err)
    }

    var count int
    ctx.State.Get("item_count", &count)

    return MyOutput{Email: email, Count: count}, nil
}
```

### `Delete`

Removes a key from state.

```go
ctx.State.Delete("temporary_data")
```

### `Has`

Checks if a key exists in state (either in cache or in the store).

```go
if ctx.State.Has("user_preferences") {
    var prefs UserPreferences
    ctx.State.Get("user_preferences", &prefs)
}
```

### `GetAll`

Retrieves all state data as a map of raw JSON bytes. Updates the internal cache.

```go
allState, err := ctx.State.GetAll()
if err != nil {
    return MyOutput{}, err
}
for key, jsonBytes := range allState {
    fmt.Printf("Key: %s, Value: %s\n", key, string(jsonBytes))
}
```

## Typed Helpers

Generic helper functions provide type-safe state access without requiring a pre-declared variable:

### `SetTyped`

```go
func SetTyped[T any](accessor StateAccessor, key string, value T) error
```

```go
err := gorkflow.SetTyped(ctx.State, "counter", 42)
err = gorkflow.SetTyped(ctx.State, "user", User{Name: "Alice", Age: 30})
```

### `GetTyped`

```go
func GetTyped[T any](accessor StateAccessor, key string) (T, error)
```

```go
counter, err := gorkflow.GetTyped[int](ctx.State, "counter")
user, err := gorkflow.GetTyped[User](ctx.State, "user")
```

## Complete Example

```go
package main

import (
    "fmt"
    "github.com/sicko7947/gorkflow"
)

type OrderInput struct {
    OrderID string  `json:"orderId"`
    Amount  float64 `json:"amount"`
}

type OrderOutput struct {
    OrderID string `json:"orderId"`
    Status  string `json:"status"`
}

type TaxResult struct {
    Tax float64 `json:"tax"`
}

func main() {
    // Step 1: Store order data in state
    receiveOrder := gorkflow.NewStep(
        "receive-order",
        "Receive Order",
        func(ctx *gorkflow.StepContext, input OrderInput) (OrderOutput, error) {
            // Save to state for other steps to access
            ctx.State.Set("order_amount", input.Amount)
            ctx.State.Set("order_id", input.OrderID)

            return OrderOutput{OrderID: input.OrderID, Status: "received"}, nil
        },
    )

    // Step 2: Calculate tax using state
    calculateTax := gorkflow.NewStep(
        "calculate-tax",
        "Calculate Tax",
        func(ctx *gorkflow.StepContext, input OrderOutput) (TaxResult, error) {
            // Read from state
            amount, err := gorkflow.GetTyped[float64](ctx.State, "order_amount")
            if err != nil {
                return TaxResult{}, fmt.Errorf("could not get order amount: %w", err)
            }

            tax := amount * 0.08
            ctx.State.Set("tax_amount", tax)

            return TaxResult{Tax: tax}, nil
        },
    )

    wf, _ := gorkflow.NewWorkflow("order-tax", "Order Tax Calculation").
        ThenStep(receiveOrder).
        ThenStep(calculateTax).
        Build()

    // Use with engine...
    _ = wf
}
```

## State vs Step Data

| Feature | State (`ctx.State`) | Step Data (`ctx.Data`) |
|---------|---------------------|------------------------|
| Purpose | Shared key-value data | Step inputs and outputs |
| Access | Any step can read/write any key | Read-only access to other steps' data |
| Lifetime | Entire workflow run | Set when each step completes |
| Use case | Flags, counters, shared config | Passing structured data between steps |

Use **state** when you need to share arbitrary data across non-adjacent steps or accumulate values. Use **step data** when you need to access a specific step's typed output.

## Best Practices

### Use Descriptive Keys

```go
// Good
ctx.State.Set("user_verification_status", "verified")
ctx.State.Set("retry_count_email_send", 3)

// Bad
ctx.State.Set("status", "verified")
ctx.State.Set("count", 3)
```

### Handle Missing Keys

```go
var prefs UserPreferences
if err := ctx.State.Get("user_prefs", &prefs); err != nil {
    // Use defaults instead of failing
    prefs = DefaultUserPreferences
}
```

### Use Typed Helpers for Simple Values

```go
// Instead of:
var count int
ctx.State.Get("count", &count)

// Use:
count, _ := gorkflow.GetTyped[int](ctx.State, "count")
```

### Clean Up Temporary State

```go
// Remove temporary data when no longer needed
ctx.State.Delete("temp_processing_buffer")
```

---

**Next**: Learn about the [StepContext](context.md) →
