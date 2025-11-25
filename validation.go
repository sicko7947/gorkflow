package gorkflow

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// validationConfig holds internal validation settings
type validationConfig struct {
	validateInput  bool
	validateOutput bool
	validator      *validator.Validate
}

// defaultValidator is the shared validator instance used by all steps
var defaultValidator = validator.New()

// defaultValidationConfig is used when validation is enabled
var defaultValidationConfig = &validationConfig{
	validateInput:  true,
	validateOutput: true,
	validator:      defaultValidator,
}

// validateStruct validates a struct using the validator
func (vc *validationConfig) validateStruct(v interface{}) error {
	if vc == nil || vc.validator == nil {
		return nil
	}

	if err := vc.validator.Struct(v); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return newValidationError(validationErrors)
		}
		return err
	}
	return nil
}

// validateInputData unmarshals and validates input data
func validateInputData[T any](data []byte, config *validationConfig) (T, error) {
	var input T

	// Unmarshal
	if err := json.Unmarshal(data, &input); err != nil {
		return input, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	// Validate if enabled
	if config != nil && config.validateInput {
		if err := config.validateStruct(input); err != nil {
			return input, fmt.Errorf("input validation failed: %w", err)
		}
	}

	return input, nil
}

// validateOutputData validates and marshals output data
func validateOutputData[T any](output T, config *validationConfig) ([]byte, error) {
	// Validate if enabled
	if config != nil && config.validateOutput {
		if err := config.validateStruct(output); err != nil {
			return nil, fmt.Errorf("output validation failed: %w", err)
		}
	}

	// Marshal
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return outputBytes, nil
}

// validationError wraps validator.ValidationErrors with formatted output
type validationError struct {
	errors validator.ValidationErrors
}

func newValidationError(errs validator.ValidationErrors) *validationError {
	return &validationError{errors: errs}
}

func (e *validationError) Error() string {
	if len(e.errors) == 0 {
		return "validation failed"
	}

	// Format validation errors in a readable way
	msg := "validation failed:\n"
	for _, err := range e.errors {
		msg += fmt.Sprintf("  - field '%s' failed on '%s' tag", err.Field(), err.Tag())
		if err.Param() != "" {
			msg += fmt.Sprintf(" (param: %s)", err.Param())
		}
		msg += fmt.Sprintf(": got value '%v'\n", err.Value())
	}
	return msg
}

func (e *validationError) Unwrap() error {
	return e.errors
}

// WithCustomValidator sets a custom validator instance for a step
func WithCustomValidator(v *validator.Validate) StepOption {
	return stepOptionFunc(func(s interface{}) {
		if step, ok := s.(interface{ SetCustomValidator(*validator.Validate) }); ok {
			step.SetCustomValidator(v)
		}
	})
}

// WithoutValidation disables validation for a step (validation is enabled by default)
func WithoutValidation() StepOption {
	return stepOptionFunc(func(s interface{}) {
		type validationDisabler interface {
			DisableValidation()
		}
		if step, ok := s.(validationDisabler); ok {
			step.DisableValidation()
		}
	})
}
