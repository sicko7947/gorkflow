# Validation

Automatic input/output validation using struct tags with `go-playground/validator`.

## Overview

Gorkflow provides **automatic validation by default** for all step inputs and outputs. Simply add validation tags to your structs, and Gorkflow handles the rest.

## How It Works

Validation happens at two points:

1. **Before Step Execution** - Input is validated
2. **After Step Execution** - Output is validated

If validation fails, the step fails with a detailed error message.

## Basic Usage

Add `validate` tags to your structs:

```go
type UserInput struct {
    Email    string `json:"email" validate:"required,email"`
    Username string `json:"username" validate:"required,min=3,max=20,alphanum"`
    Age      int    `json:"age" validate:"required,gte=18,lte=120"`
}

type UserOutput struct {
    UserID string `json:"userId" validate:"required,uuid4"`
    Email  string `json:"email" validate:"required,email"`
}

// Create step - validation is automatic!
step := gorkflow.NewStep(
    "create-user",
    "Create User",
    func(ctx *gorkflow.StepContext, input UserInput) (UserOutput, error) {
        // Input is already validated here
        return UserOutput{
            UserID: uuid.New().String(),
            Email:  input.Email,
        }, nil
    },
)
```

## Common Validation Tags

### String Validation

```go
type StringValidation struct {
    // Required field
    Name string `validate:"required"`

    // Email address
    Email string `validate:"required,email"`

    // URL
    Website string `validate:"url"`

    // Minimum and maximum length
    Username string `validate:"min=3,max=20"`

    // Alphanumeric only
    Code string `validate:"alphanum"`

    // Numeric only
    ID string `validate:"numeric"`

    // UUID
    UserID string `validate:"uuid4"`

    // One of specific values
    Status string `validate:"oneof=active inactive pending"`

    // Regular expression
    PhoneNumber string `validate:"regexp=^[0-9]{10}$"`
}
```

### Number Validation

```go
type NumberValidation struct {
    // Greater than or equal
    Age int `validate:"gte=18"`

    // Less than or equal
    Discount int `validate:"lte=100"`

    // Range
    Score int `validate:"gte=0,lte=100"`

    // Greater than
    Price float64 `validate:"gt=0"`

    // Less than
    Tax float64 `validate:"lt=1"`
}
```

### Slice and Array Validation

```go
type SliceValidation struct {
    // Required, not empty
    Tags []string `validate:"required,min=1"`

    // Maximum items
    Items []string `validate:"max=10"`

    // Validation for each element
    Emails []string `validate:"dive,email"`

    // Unique elements
    IDs []string `validate:"unique"`
}
```

### Struct Validation

```go
type Address struct {
    Street  string `validate:"required"`
    City    string `validate:"required"`
    ZipCode string `validate:"required,len=5"`
}

type UserWithAddress struct {
    Name    string  `validate:"required"`
    Address Address `validate:"required"`  // Nested validation
}
```

### Pointer Validation

```go
type OptionalField struct {
    // Optional field (can be nil)
    MiddleName *string

    // If present, must be valid email
    Email *string `validate:"omitempty,email"`
}
```

## Full Validation Reference

### Basic

| Tag         | Description                | Example                      |
| ----------- | -------------------------- | ---------------------------- |
| `required`  | Field cannot be zero value | `validate:"required"`        |
| `omitempty` | Skip validation if empty   | `validate:"omitempty,email"` |

### String

| Tag        | Description                | Example               |
| ---------- | -------------------------- | --------------------- |
| `email`    | Valid email address        | `validate:"email"`    |
| `url`      | Valid URL                  | `validate:"url"`      |
| `alpha`    | Alphabetic characters only | `validate:"alpha"`    |
| `alphanum` | Alphanumeric only          | `validate:"alphanum"` |
| `numeric`  | Numeric only               | `validate:"numeric"`  |
| `len`      | Exact length               | `validate:"len=10"`   |
| `min`      | Minimum length             | `validate:"min=3"`    |
| `max`      | Maximum length             | `validate:"max=20"`   |

### Numbers

| Tag   | Description           | Example              |
| ----- | --------------------- | -------------------- |
| `eq`  | Equal to              | `validate:"eq=0"`    |
| `ne`  | Not equal to          | `validate:"ne=0"`    |
| `gt`  | Greater than          | `validate:"gt=0"`    |
| `gte` | Greater than or equal | `validate:"gte=18"`  |
| `lt`  | Less than             | `validate:"lt=100"`  |
| `lte` | Less than or equal    | `validate:"lte=100"` |

### Collections

| Tag      | Description           | Example                 |
| -------- | --------------------- | ----------------------- |
| `dive`   | Validate each element | `validate:"dive,email"` |
| `unique` | All elements unique   | `validate:"unique"`     |
| `min`    | Minimum items         | `validate:"min=1"`      |
| `max`    | Maximum items         | `validate:"max=10"`     |

### Special

| Tag      | Description              | Example                           |
| -------- | ------------------------ | --------------------------------- |
| `uuid4`  | Valid UUID v4            | `validate:"uuid4"`                |
| `oneof`  | One of specified values  | `validate:"oneof=red blue green"` |
| `regexp` | Match regular expression | `validate:"regexp=^[0-9]+$"`      |

## Validation Errors

When validation fails, you get detailed error messages:

```go
// Input with invalid email
input := UserInput{
    Email:    "invalid-email",
    Username: "ab",  // Too short (min=3)
    Age:      16,    // Too young (gte=18)
}

// Attempting to run step will fail with:
// validation failed:
//   - field 'Email' failed on 'email' tag: got value 'invalid-email'
//   - field 'Username' failed on 'min' tag (param: 3): got value 'ab'
//   - field 'Age' failed on'gte' tag (param: 18): got value '16'
```

## Custom Validation

### Custom Validator Functions

Create a custom validator:

```go
import "github.com/go-playground/validator/v10"

// Custom validation function
func validatePassword(fl validator.FieldLevel) bool {
    password := fl.Field().String()

    // Custom logic
    hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
    hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
    hasNumber := strings.ContainsAny(password, "0123456789")

    return hasUpper && hasLower && hasNumber
}

// Register custom validator
v := validator.New()
v.RegisterValidation("secure_password", validatePassword)

// Use in step
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithCustomValidator(v),
)
```

Use the custom tag:

```go
type PasswordInput struct {
    Password string `validate:"required,min=8,secure_password"`
}
```

### Struct-Level Validation

Validate relationships between fields:

```go
type DateRange struct {
    StartDate time.Time `validate:"required"`
    EndDate   time.Time `validate:"required"`
}

func validateDateRange(fl validator.StructLevel) {
    dateRange := fl.Current().Interface().(DateRange)

    if dateRange.EndDate.Before(dateRange.StartDate) {
        fl.ReportError(dateRange.EndDate, "EndDate", "EndDate", "gtefield", "StartDate")
    }
}

// Register struct-level validation
v := validator.New()
v.RegisterStructValidation(validateDateRange, DateRange{})
```

## Disabling Validation

If you need to disable validation for a specific step:

```go
step := gorkflow.NewStep("my-step", "My Step", handler,
    gorkflow.WithoutValidation(),  // Disable validation
)
```

> ⚠️ **Warning**: Disabling validation removes type safety guarantees and can lead to runtime errors.

## Best Practices

### 1. Always Validate User Input

```go
type UserInput struct {
    Email    string `validate:"required,email"`
    Password string `validate:"required,min=8"`
}
```

### 2. Validate External Data

```go
type APIResponse struct {
    ID        string `validate:"required,uuid4"`
    Status    string `validate:"required,oneof=success error"`
    Timestamp int64  `validate:"required,gt=0"`
}
```

### 3. Use Appropriate Constraints

```go
type OrderInput struct {
    Quantity int     `validate:"required,gte=1,lte=100"`
    Price    float64 `validate:"required,gt=0"`
}
```

### 4. Validate Nested Structs

```go
type UserWithAddress struct {
    Name    string  `validate:"required"`
    Address Address `validate:"required"`  // Validates nested struct
}
```

### 5. Document Validation Rules

```go
// UserInput represents user registration data.
//
// Validation Rules:
// - Email: Required, must be valid email format
// - Username: Required, 3-20 alphanumeric characters
// - Age: Required, must be 18-120
type UserInput struct {
    Email    string `json:"email" validate:"required,email"`
    Username string `json:"username" validate:"required,min=3,max=20,alphanum"`
    Age      int    `json:"age" validate:"required,gte=18,lte=120"`
}
```

## Common Patterns

### Optional Fields

```go
type OptionalInput struct {
    RequiredField  string  `validate:"required"`
    OptionalField  *string `validate:"omitempty,email"`  // Only validates if present
}
```

### Conditional Validation

```go
type ConditionalInput struct {
    Type  string `validate:"required,oneof=email phone"`
    Email string `validate:"required_if=Type email,omitempty,email"`
    Phone string `validate:"required_if=Type phone,omitempty,e164"`
}
```

### Cross-Field Validation

```go
type PasswordChange struct {
    Password        string `validate:"required,min=8"`
    ConfirmPassword string `validate:"required,eqfield=Password"`
}
```

## Validation Examples

See the [Validation Example](../examples/validation.md) for a complete working example.

---

**Next**: Learn about [State Management](state-management.md) →
