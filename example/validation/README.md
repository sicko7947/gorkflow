# Validation Example

This example demonstrates how to use **`go-playground/validator/v10`** for input and output validation in gorkflow workflows.

## Overview

The example shows a user registration workflow with three steps:

1. **Validate User** - Validates registration input and creates a user
2. **Send Email** - Sends a welcome email
3. **Format Result** - Formats the final workflow output

Each step uses struct validation tags to enforce data constraints.

## Validation Tags Used

### UserRegistrationInput

```go
type UserRegistrationInput struct {
    Email    string `validate:"required,email"`
    Username string `validate:"required,min=3,max=20,alphanum"`
    Age      int    `validate:"required,gte=18,lte=120"`
    Password string `validate:"required,min=8"`
}
```

- **Email**: Required, must be valid email format
- **Username**: Required, 3-20 characters, alphanumeric only
- **Age**: Required, between 18 and 120
- **Password**: Required, minimum 8 characters

### ValidatedUserOutput

```go
type ValidatedUserOutput struct {
    UserID   string `validate:"required,uuid4"`
    Email    string `validate:"required,email"`
    Username string `validate:"required"`
    IsActive bool   `json:"isActive"`
}
```

- **UserID**: Required, must be valid UUID v4
- **Email**: Required, must be valid email
- **Username**: Required

### EmailSentOutput

```go
type EmailSentOutput struct {
    MessageID string `validate:"required"`
    Status    string `validate:"required,oneof=sent failed pending"`
    SentTo    string `validate:"required,email"`
}
```

- **MessageID**: Required
- **Status**: Required, must be one of: `sent`, `failed`, `pending`
- **SentTo**: Required, must be valid email

## How to Enable Validation

### Option 1: Use Default Validation Config (Recommended)

```go
step := workflow.NewStep(
    "my_step",
    "My Step",
    handler,
    workflow.WithValidation(workflow.DefaultValidationConfig),
)
```

This enables both input and output validation with fail-on-error behavior.

### Option 2: Custom Validation Config

```go
step := workflow.NewStep(
    "my_step",
    "My Step",
    handler,
    workflow.WithValidation(workflow.ValidationConfig{
        ValidateInput:         true,
        ValidateOutput:        true,
        FailOnValidationError: true,
        CustomValidator:       nil, // Use default validator
    }),
)
```

### Option 3: Input or Output Only

```go
// Validate input only
step := workflow.NewStep(
    "my_step",
    "My Step",
    handler,
    workflow.WithInputValidation(),
)

// Validate output only
step := workflow.NewStep(
    "my_step",
    "My Step",
    handler,
    workflow.WithOutputValidation(),
)
```

### Option 4: Custom Validator Instance

```go
import "github.com/go-playground/validator/v10"

// Create custom validator with custom rules
customValidator := validator.New()
customValidator.RegisterValidation("custom_rule", myCustomValidationFunc)

step := workflow.NewStep(
    "my_step",
    "My Step",
    handler,
    workflow.WithCustomValidator(customValidator),
    workflow.WithValidation(workflow.DefaultValidationConfig),
)
```

## Running the Example

```bash
cd example/validation
go run main.go
```

## Expected Output

The example runs 4 test cases:

### ✅ Example 1: Valid Input

All validation passes, workflow completes successfully.

### ❌ Example 2: Invalid Email

Input validation fails with error:

```
validation failed:
  - field 'Email' failed on 'email' tag: got value 'not-an-email'
```

### ❌ Example 3: Username Too Short

Input validation fails with error:

```
validation failed:
  - field 'Username' failed on 'min' tag (param: 3): got value 'ab'
```

### ❌ Example 4: Age Below Minimum

Input validation fails with error:

```
validation failed:
  - field 'Age' failed on 'gte' tag (param: 18): got value '16'
```

## Validation Error Handling

When validation fails:

1. **Input Validation**: Fails before the step handler executes
2. **Output Validation**: Fails after the step handler executes
3. **Error Format**: Provides detailed information about which field failed and why
4. **Workflow Behavior**:
   - If `FailOnValidationError: true` → Step fails, workflow stops
   - If `FailOnValidationError: false` → Error logged, execution continues

## Common Validation Tags

Here are some commonly used validation tags from `go-playground/validator`:

### String Validators

- `required` - Field must not be empty
- `email` - Must be valid email format
- `url` - Must be valid URL
- `min=N` - Minimum length
- `max=N` - Maximum length
- `len=N` - Exact length
- `alphanum` - Alphanumeric characters only
- `alpha` - Alphabetic characters only
- `numeric` - Numeric characters only
- `oneof=val1 val2` - Must be one of the specified values

### Numeric Validators

- `eq=N` - Equal to N
- `ne=N` - Not equal to N
- `gt=N` - Greater than N
- `gte=N` - Greater than or equal to N
- `lt=N` - Less than N
- `lte=N` - Less than or equal to N

### Format Validators

- `uuid` - Valid UUID (any version)
- `uuid4` - Valid UUID v4
- `ip` - Valid IP address
- `ipv4` - Valid IPv4 address
- `ipv6` - Valid IPv6 address
- `datetime=layout` - Valid datetime with specified layout

### Cross-Field Validators

- `eqfield=Field` - Equal to another field
- `nefield=Field` - Not equal to another field
- `gtfield=Field` - Greater than another field
- `ltfield=Field` - Less than another field

## Benefits of Using Validation

1. **Early Error Detection**: Catch invalid data before processing
2. **Type Safety**: Enforce data contracts between steps
3. **Self-Documenting**: Validation tags serve as documentation
4. **Consistent Error Messages**: Standardized error format
5. **Reduced Boilerplate**: No need to write manual validation code
6. **Production Ready**: Battle-tested validation library

## See Also

- [go-playground/validator Documentation](https://github.com/go-playground/validator)
- [Validation Tag Reference](https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Baked_In_Validators_and_Tags)
- [Custom Validation Functions](https://github.com/go-playground/validator#custom-validation-functions)
