# Conditional Execution

Execute workflow steps conditionally based on runtime data, enabling dynamic workflow paths.

## Overview

Conditional execution allows steps to run only when specific conditions are met, enabling:

- Dynamic workflow branching
- Optional processing based on flags
- Data-driven workflow paths
- Skipping unnecessary operations

## Basic Conditional Step

Use `ThenStepIf` in the workflow builder:

```go
// Define a condition
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var shouldProcess bool
    ctx.State.Get("should_process", &shouldProcess)
    return shouldProcess, nil
}

// Build workflow with conditional step
wf, _ := gorkflow.NewWorkflow("conditional", "Conditional Workflow").
    ThenStep(setupStep).
    ThenStepIf(processStep, condition, nil).  // Only runs if condition is true
    ThenStep(finalStep).
    Build()
```

## How It Works

1. **Condition Evaluation** - Before executing the step, the condition function is called
2. **True** - Step executes normally
3. **False** - Step is skipped, default value (or zero value) is used as output
4. **Error** - Workflow fails with the condition error

## Complete Example

```go
package main

import (
    "github.com/sicko7947/gorkflow"
)

type SetupInput struct {
    ProcessData  bool `json:"processData"`
    UseFormatted bool `json:"useFormatted"`
}

type SetupOutput struct {
    ProcessData  bool `json:"processData"`
    UseFormatted bool `json:"useFormatted"`
}

type ProcessedData struct {
    Data string `json:"data"`
}

type FormattedData struct {
    Formatted string `json:"formatted"`
}

func main() {
    // Setup step - sets flags in state
    setupStep := gorkflow.NewStep(
        "setup",
        "Setup Flags",
        func(ctx *gorkflow.StepContext, input SetupInput) (SetupOutput, error) {
            // Store flags in workflow state
            ctx.State.Set("process_data", input.ProcessData)
            ctx.State.Set("use_formatted", input.UseFormatted)

            return SetupOutput{
                ProcessData:  input.ProcessData,
                UseFormatted: input.UseFormatted,
            }, nil
        },
    )

    // Processing step - conditional
    processStep := gorkflow.NewStep(
        "process",
        "Process Data",
        func(ctx *gorkflow.StepContext, input SetupOutput) (ProcessedData, error) {
            return ProcessedData{Data: "Processed!"}, nil
        },
    )

    // Formatting step - conditional
    formatStep := gorkflow.NewStep(
        "format",
        "Format Data",
        func(ctx *gorkflow.StepContext, input ProcessedData) (FormattedData, error) {
            return FormattedData{Formatted: "Formatted: " + input.Data}, nil
        },
    )

    // Condition: Should we process?
    shouldProcess := func(ctx *gorkflow.StepContext) (bool, error) {
        var processData bool
        ctx.State.Get("process_data", &processData)
        return processData, nil
    }

    // Condition: Should we format?
    shouldFormat := func(ctx *gorkflow.StepContext) (bool, error) {
        var useFormatted bool
        ctx.State.Get("use_formatted", &useFormatted)
        return useFormatted, nil
    }

    // Build workflow with conditional steps
    wf, _ := gorkflow.NewWorkflow("conditional-wf", "Conditional Workflow").
        ThenStep(setupStep).
        ThenStepIf(processStep, shouldProcess, nil).
        ThenStepIf(formatStep, shouldFormat, nil).
        Build()
}
```

## Condition Functions

A condition function has this signature:

```go
func(ctx *gorkflow.StepContext) (bool, error)
```

### Accessing State

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var flag bool
    err := ctx.State.Get("my_flag", &flag)
    if err != nil {
        return false, err
    }
    return flag, nil
}
```

### Accessing Previous Step Output

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    prevOutput, err := gorkflow.GetOutput[PreviousStepOutput](ctx, "previous-step-id")
    if err != nil {
        return false, err
    }
    return prevOutput.ShouldContinue, nil
}
```

### Complex Conditions

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    // Get multiple values
    var count int
    var enabled bool

    ctx.State.Get("count", &count)
    ctx.State.Get("enabled", &enabled)

    // Complex logic
    return enabled && count > 10, nil
}
```

## Default Values

When a step is skipped, you can provide a default output:

```go
// Default value when step is skipped
defaultOutput := &ProcessedData{
    Data: "Default - not processed",
}

wf, _ := gorkflow.NewWorkflow("wf", "Workflow").
    ThenStepIf(processStep, condition, defaultOutput).
    Build()
```

If no default is provided (nil), the zero value is used:

```go
// Uses zero value: ProcessedData{Data: ""}
wf, _ := gorkflow.NewWorkflow("wf", "Workflow").
    ThenStepIf(processStep, condition, nil).
    Build()
```

## Step-Level Conditional API

Alternative approach using `NewConditionalStep`:

```go
// Create base step
baseStep := gorkflow.NewStep("process", "Process", processHandler)

// Wrap with condition
defaultOutput := &ProcessOutput{Status: "skipped"}
conditionalStep := gorkflow.NewConditionalStep(baseStep, condition, defaultOutput)

// Use in workflow
wf, _ := gorkflow.NewWorkflow("wf", "Workflow").
    ThenStep(conditionalStep).
    Build()
```

This is equivalent to the builder-level `ThenStepIf`.

## Use Cases

### Feature Flags

```go
featureEnabled := func(ctx *gorkflow.StepContext) (bool, error) {
    var features map[string]bool
    ctx.State.Get("features", &features)
    return features["new_processor"], nil
}

wf, _ := gorkflow.NewWorkflow("wf", "Workflow").
    ThenStepIf(newProcessorStep, featureEnabled, nil).
    Build()
```

### Data-Driven Branching

```go
needsEnrichment := func(ctx *gorkflow.StepContext) (bool, error) {
    data, err := gorkflow.GetOutput[UserData](ctx, "fetch-user")
    if err != nil {
        return false, err
    }
    return data.Email != "" && data.Company == "", nil
}

wf, _ := gorkflow.NewWorkflow("wf", "Workflow").
    ThenStep(fetchUserStep).
    ThenStepIf(enrichCompanyDataStep, needsEnrichment, nil).
    ThenStep(saveUserStep).
    Build()
```

### Optional Notifications

```go
shouldNotify := func(ctx *gorkflow.StepContext) (bool, error) {
    result, err := gorkflow.GetOutput[ProcessResult](ctx, "process")
    if err != nil {
        return false, err
    }
    return result.ChangesDetected, nil
}

wf, _ := gorkflow.NewWorkflow("wf", "Workflow").
    ThenStep(processStep).
    ThenStepIf(sendNotificationStep, shouldNotify, nil).
    Build()
```

### Environment-Based Execution

```go
isProduction := func(ctx *gorkflow.StepContext) (bool, error) {
    var env string
    ctx.State.Get("environment", &env)
    return env == "production", nil
}

wf, _ := gorkflow.NewWorkflow("wf", "Workflow").
    ThenStepIf(productionOnlyStep, isProduction, nil).
    Build()
```

## Multiple Conditional Paths

Create different paths based on conditions:

```go
isTypeA := func(ctx *gorkflow.StepContext) (bool, error) {
    var dataType string
    ctx.State.Get("type", &dataType)
    return dataType == "A", nil
}

isTypeB := func(ctx *gorkflow.StepContext) (bool, error) {
    var dataType string
    ctx.State.Get("type", &dataType)
    return dataType == "B", nil
}

wf, _ := gorkflow.NewWorkflow("multi-path", "Multi-Path").
    ThenStep(determineTypeStep).
    ThenStepIf(processTypeA, isTypeA, nil).  // Path A
    ThenStepIf(processTypeB, isTypeB, nil).  // Path B
    ThenStep(mergeResults).
    Build()
```

**Execution Flow:**

```
determineType
    ├─→ processTypeA (if type=A)
    ├─→ processTypeB (if type=B)
    └─→ mergeResults
```

## Conditional Parallel Execution

Combine conditionals with parallel blocks:

```go
wf, _ := gorkflow.NewWorkflow("conditional-parallel", "Conditional Parallel").
    ThenStep(setupStep).
    Parallel(
        gorkflow.NewConditionalStep(stepA, conditionA, nil),
        gorkflow.NewConditionalStep(stepB, conditionB, nil),
        stepC,  // Always runs
    ).
    ThenStep(aggregateStep).
    Build()
```

## Error Handling

### Condition Errors

If a condition returns an error, the workflow fails:

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var flag bool
    err := ctx.State.Get("flag", &flag)
    if err != nil {
        return false, fmt.Errorf("failed to get flag: %w", err)
    }
    return flag, nil
}
```

### Handling Missing Data

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    data, err := gorkflow.GetOutput[SomeData](ctx, "step-id")
    if err != nil {
        // Decide: fail workflow or default to false
        return false, nil  // Default to skipping step
        // OR
        // return false, err  // Fail workflow
    }
    return data.ShouldProcess, nil
}
```

## Best Practices

### 1. Keep Conditions Simple

✅ **Good** - Simple, readable:

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var enabled bool
    ctx.State.Get("feature_enabled", &enabled)
    return enabled, nil
}
```

❌ **Bad** - Complex logic:

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    // 50 lines of complex logic...
}
```

### 2. Use Descriptive Condition Names

```go
shouldSendWelcomeEmail := func(ctx *gorkflow.StepContext) (bool, error) {
    // ...
}

hasValidEmail := func(ctx *gorkflow.StepContext) (bool, error) {
    // ...
}
```

### 3. Provide Meaningful Defaults

```go
defaultOutput := &ProcessOutput{
    Status:  "skipped",
    Reason:" "Condition not met",
    Timestamp: time.Now(),
}

wf, _ := gorkflow.NewWorkflow("wf", "Workflow").
    ThenStepIf(processStep, condition, defaultOutput).
    Build()
```

### 4. Log Condition Evaluation

```go
condition := func(ctx *gorkflow.StepContext) (bool, error) {
    var flag bool
    ctx.State.Get("feature_flag", &flag)

    ctx.Logger.Debug().
        Bool("flag_value", flag).
        Msg("Evaluated condition")

    return flag, nil
}
```

### 5. Test Both Paths

Always test workflows with conditions evaluating to both true and false.

---

**Next**: Explore more patterns in [Examples](../examples/conditional.md) →
