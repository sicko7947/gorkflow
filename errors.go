package gorkflow

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrStepSkipped indicates that a step was skipped due to a condition
	ErrStepSkipped = errors.New("step skipped")

	// Store sentinel errors
	ErrRunNotFound           = errors.New("workflow run not found")
	ErrStepExecutionNotFound = errors.New("step execution not found")
	ErrStepOutputNotFound    = errors.New("step output not found")
	ErrStateNotFound         = errors.New("state not found")
)

// Error codes
const (
	ErrCodeValidation      = "VALIDATION_ERROR"
	ErrCodeNotFound        = "NOT_FOUND"
	ErrCodeTimeout         = "TIMEOUT"
	ErrCodeConcurrency     = "CONCURRENCY_LIMIT"
	ErrCodeExecutionFailed = "EXECUTION_FAILED"
	ErrCodeCancelled       = "CANCELLED"
	ErrCodePanic           = "PANIC"
	ErrCodeInternalError   = "INTERNAL_ERROR"
)

// WorkflowError represents an error during workflow execution
type WorkflowError struct {
	Message   string                 `json:"message"`
	Code      string                 `json:"code"`
	Step      string                 `json:"step,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *WorkflowError) Error() string {
	if e.Step != "" {
		return fmt.Sprintf("[%s] %s (step: %s)", e.Code, e.Message, e.Step)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewWorkflowError creates a new workflow error
func NewWorkflowError(code, message string) *WorkflowError {
	return &WorkflowError{
		Message:   message,
		Code:      code,
		Timestamp: time.Now(),
	}
}

// NewWorkflowErrorWithStep creates a new workflow error with step context
func NewWorkflowErrorWithStep(code, message, step string) *WorkflowError {
	return &WorkflowError{
		Message:   message,
		Code:      code,
		Step:      step,
		Timestamp: time.Now(),
	}
}

// WithDetails adds details to the error
func (e *WorkflowError) WithDetails(details map[string]interface{}) *WorkflowError {
	e.Details = details
	return e
}

// StepError represents an error during step execution
type StepError struct {
	Message   string                 `json:"message"`
	Code      string                 `json:"code"`
	Timestamp time.Time              `json:"timestamp"`
	Attempt   int                    `json:"attempt"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *StepError) Error() string {
	return fmt.Sprintf("[%s] %s (attempt: %d)", e.Code, e.Message, e.Attempt)
}

// NewStepError creates a new step error
func NewStepError(code, message string, attempt int) *StepError {
	return &StepError{
		Message:   message,
		Code:      code,
		Timestamp: time.Now(),
		Attempt:   attempt,
	}
}

// WithDetails adds details to the error
func (e *StepError) WithDetails(details map[string]interface{}) *StepError {
	e.Details = details
	return e
}

// IsConcurrencyError checks if an error is a concurrency limit error
func IsConcurrencyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), ErrCodeConcurrency)
}

// IsTimeoutError checks if an error is a timeout error
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if se, ok := err.(*StepError); ok {
		return se.Code == ErrCodeTimeout
	}
	if we, ok := err.(*WorkflowError); ok {
		return we.Code == ErrCodeTimeout
	}
	return strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "context deadline exceeded")
}
