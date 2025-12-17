package gorkflow

import "time"

// ExecutionConfig holds step-level execution parameters
type ExecutionConfig struct {
	// Retry policy
	MaxRetries   int             `json:"maxRetries"`
	RetryDelayMs int             `json:"retryDelayMs"`
	RetryBackoff BackoffStrategy `json:"retryBackoff"`

	// Timeout
	TimeoutSeconds int `json:"timeoutSeconds"`

	// Concurrency (for parallel execution in future)
	MaxConcurrency int `json:"maxConcurrency"`

	// Failure behavior
	ContinueOnError bool    `json:"continueOnError"`
	FallbackStepID  *string `json:"fallbackStepId,omitempty"`
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
	MaxConcurrentWorkflows int           `json:"maxConcurrentWorkflows"`
	DefaultTimeout         time.Duration `json:"defaultTimeout"`
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

// StartOption allows functional configuration of workflow execution
type StartOption func(*StartOptions)

// StartOptions holds options for starting a workflow
type StartOptions struct {
	ResourceID       string
	CheckConcurrency bool
	TTL              time.Duration
	Tags             map[string]string

	Synchronous bool
}

// WithResourceID sets the resource ID for concurrency control
func WithResourceID(id string) StartOption {
	return func(opts *StartOptions) {
		opts.ResourceID = id
	}
}

// WithConcurrencyCheck enables concurrency checking
func WithConcurrencyCheck(check bool) StartOption {
	return func(opts *StartOptions) {
		opts.CheckConcurrency = check
	}
}

// WithTTL sets the TTL duration for DynamoDB
func WithTTL(ttl time.Duration) StartOption {
	return func(opts *StartOptions) {
		opts.TTL = ttl
	}
}

// WithTags sets custom tags for the workflow run
func WithTags(tags map[string]string) StartOption {
	return func(opts *StartOptions) {
		opts.Tags = tags
	}
}

// WithSynchronousExecution enables synchronous execution
func WithSynchronousExecution() StartOption {
	return func(opts *StartOptions) {
		opts.Synchronous = true
	}
}
