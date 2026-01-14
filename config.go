package gorkflow

import "time"

// ExecutionConfig holds step-level execution parameters
type ExecutionConfig struct {
	// Retry policy
	MaxRetries   int             `json:"max_retries,omitempty"`
	RetryDelayMs int             `json:"retry_delay_ms,omitempty"`
	RetryBackoff BackoffStrategy `json:"retry_backoff,omitempty"`

	// Timeout
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`

	// Concurrency (for parallel execution in future)
	MaxConcurrency int `json:"max_concurrency,omitempty"`

	// Failure behavior
	ContinueOnError bool    `json:"continue_on_error,omitempty"`
	FallbackStepID  *string `json:"fallback_step_id,omitempty"`
}

// BackoffStrategy defines retry backoff behavior
type BackoffStrategy string

const (
	BackoffLinear      BackoffStrategy = "LINEAR"
	BackoffExponential BackoffStrategy = "EXPONENTIAL"
	BackoffNone        BackoffStrategy = "NONE"
)

// DefaultExecutionConfig provides sensible defaults
var DefaultExecutionConfig = ExecutionConfig{
	MaxRetries:      3,
	RetryDelayMs:    1000,
	RetryBackoff:    BackoffLinear,
	TimeoutSeconds:  30,
	MaxConcurrency:  1,
	ContinueOnError: false,
}

// EngineConfig holds engine-level configuration
type EngineConfig struct {
	MaxConcurrentWorkflows int           `json:"max_concurrent_workflows"`
	DefaultTimeout         time.Duration `json:"default_timeout"`
}

// DefaultEngineConfig provides engine defaults
var DefaultEngineConfig = EngineConfig{
	MaxConcurrentWorkflows: 10,
	DefaultTimeout:         5 * time.Minute,
}

// StepOption allows functional configuration of steps
type StepOption interface {
	applyStep(step interface{})
}

type stepOptionFunc func(interface{})

func (f stepOptionFunc) applyStep(step interface{}) {
	f(step)
}

// WithRetries sets the maximum retry attempts
func WithRetries(max int) StepOption {
	return stepOptionFunc(func(s interface{}) {
		if step, ok := s.(interface{ SetMaxRetries(int) }); ok {
			step.SetMaxRetries(max)
		}
	})
}

// WithTimeout sets the step timeout
func WithTimeout(d time.Duration) StepOption {
	return stepOptionFunc(func(s interface{}) {
		if step, ok := s.(interface{ SetTimeout(int) }); ok {
			step.SetTimeout(int(d.Seconds()))
		}
	})
}

// WithBackoff sets the retry backoff strategy
func WithBackoff(strategy BackoffStrategy) StepOption {
	return stepOptionFunc(func(s interface{}) {
		if step, ok := s.(interface{ SetBackoff(BackoffStrategy) }); ok {
			step.SetBackoff(strategy)
		}
	})
}

// WithRetryDelay sets the base retry delay
func WithRetryDelay(d time.Duration) StepOption {
	return stepOptionFunc(func(s interface{}) {
		if step, ok := s.(interface{ SetRetryDelay(int) }); ok {
			step.SetRetryDelay(int(d.Milliseconds()))
		}
	})
}

// WithContinueOnError allows workflow to continue even if step fails
func WithContinueOnError(continueOnError bool) StepOption {
	return stepOptionFunc(func(s interface{}) {
		if step, ok := s.(interface{ SetContinueOnError(bool) }); ok {
			step.SetContinueOnError(continueOnError)
		}
	})
}

// CalculateBackoff calculates the backoff delay for a retry attempt.
// It supports three strategies:
//   - EXPONENTIAL: baseDelay * 2^(attempt-1)
//   - LINEAR: baseDelay * attempt
//   - NONE: no backoff delay
//
// Parameters:
//   - baseDelayMs: the base delay in milliseconds
//   - attempt: the current retry attempt (0-based, where 0 = first attempt)
//   - strategy: the backoff strategy ("EXPONENTIAL", "LINEAR", "NONE")
//
// Returns the calculated delay duration. Returns 0 for attempt 0.
func CalculateBackoff(baseDelayMs int, attempt int, strategy string) time.Duration {
	if attempt == 0 {
		return 0
	}

	baseDelay := time.Duration(baseDelayMs) * time.Millisecond

	switch strategy {
	case "EXPONENTIAL":
		// Exponential: baseDelay * 2^(attempt-1)
		multiplier := 1 << (attempt - 1) // 2^(attempt-1)
		return baseDelay * time.Duration(multiplier)
	case "LINEAR":
		// Linear: baseDelay * attempt
		return baseDelay * time.Duration(attempt)
	case "NONE":
		// No backoff
		return 0
	default:
		// Default to linear
		return baseDelay * time.Duration(attempt)
	}
}
