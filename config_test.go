package gorkflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultExecutionConfig(t *testing.T) {
	config := DefaultExecutionConfig

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1000, config.RetryDelayMs)
	assert.Equal(t, BackoffLinear, config.RetryBackoff)
	assert.Equal(t, 30, config.TimeoutSeconds)
	assert.Equal(t, 1, config.MaxConcurrency)
	assert.False(t, config.ContinueOnError)
	assert.Nil(t, config.FallbackStepID)
}

func TestWithRetries(t *testing.T) {
	step := NewStep("test", "Test", testHandler)

	opt := WithRetries(5)
	opt.applyStep(step)

	assert.Equal(t, 5, step.Config.MaxRetries)
}

func TestWithTimeout(t *testing.T) {
	step := NewStep("test", "Test", testHandler)

	opt := WithTimeout(60 * time.Second)
	opt.applyStep(step)

	assert.Equal(t, 60, step.Config.TimeoutSeconds)
}

func TestWithBackoff(t *testing.T) {
	step := NewStep("test", "Test", testHandler)

	opt := WithBackoff(BackoffExponential)
	opt.applyStep(step)

	assert.Equal(t, BackoffExponential, step.Config.RetryBackoff)
}

func TestWithRetryDelay(t *testing.T) {
	step := NewStep("test", "Test", testHandler)

	opt := WithRetryDelay(2 * time.Second)
	opt.applyStep(step)

	assert.Equal(t, 2000, step.Config.RetryDelayMs)
}

func TestWithContinueOnError(t *testing.T) {
	step := NewStep("test", "Test", testHandler)

	opt := WithContinueOnError(true)
	opt.applyStep(step)

	assert.True(t, step.Config.ContinueOnError)
}

func TestStepOptions_Multiple(t *testing.T) {
	step := NewStep("test", "Test", testHandler,
		WithRetries(5),
		WithTimeout(60*time.Second),
		WithBackoff(BackoffExponential),
		WithRetryDelay(2*time.Second),
		WithContinueOnError(true),
	)

	config := step.GetConfig()
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 60, config.TimeoutSeconds)
	assert.Equal(t, BackoffExponential, config.RetryBackoff)
	assert.Equal(t, 2000, config.RetryDelayMs)
	assert.True(t, config.ContinueOnError)
}

func TestBackoffStrategy_String(t *testing.T) {
	assert.Equal(t, "LINEAR", string(BackoffLinear))
	assert.Equal(t, "EXPONENTIAL", string(BackoffExponential))
	assert.Equal(t, "NONE", string(BackoffNone))
}

func TestWithResourceID(t *testing.T) {
	opts := &StartOptions{}
	opt := WithResourceID("resource-123")
	opt(opts)

	assert.Equal(t, "resource-123", opts.ResourceID)
}

func TestWithConcurrencyCheck(t *testing.T) {
	opts := &StartOptions{}
	opt := WithConcurrencyCheck(true)
	opt(opts)

	assert.True(t, opts.CheckConcurrency)
}

func TestWithTTL(t *testing.T) {
	opts := &StartOptions{}
	ttl := 7 * 24 * time.Hour // 7 days
	opt := WithTTL(ttl)
	opt(opts)

	assert.Equal(t, ttl, opts.TTL)
}

func TestWithTags(t *testing.T) {
	opts := &StartOptions{}
	tags := map[string]string{
		"env":     "production",
		"version": "1.0.0",
	}
	opt := WithTags(tags)
	opt(opts)

	assert.Equal(t, tags, opts.Tags)
}

func TestStartOptions_Multiple(t *testing.T) {
	opts := &StartOptions{}

	WithResourceID("resource-123")(opts)
	WithConcurrencyCheck(true)(opts)
	WithTTL(24 * time.Hour)(opts)
	WithTags(map[string]string{"env": "test"})(opts)

	assert.Equal(t, "resource-123", opts.ResourceID)
	assert.True(t, opts.CheckConcurrency)
	assert.Equal(t, 24*time.Hour, opts.TTL)
	assert.Equal(t, "test", opts.Tags["env"])
}

func TestExecutionConfig_CustomValues(t *testing.T) {
	config := ExecutionConfig{
		MaxRetries:      10,
		RetryDelayMs:    5000,
		RetryBackoff:    BackoffExponential,
		TimeoutSeconds:  120,
		MaxConcurrency:  5,
		ContinueOnError: true,
	}

	assert.Equal(t, 10, config.MaxRetries)
	assert.Equal(t, 5000, config.RetryDelayMs)
	assert.Equal(t, BackoffExponential, config.RetryBackoff)
	assert.Equal(t, 120, config.TimeoutSeconds)
	assert.Equal(t, 5, config.MaxConcurrency)
	assert.True(t, config.ContinueOnError)
}

func TestExecutionConfig_WithFallback(t *testing.T) {
	fallbackStep := "fallback-step"
	config := ExecutionConfig{
		MaxRetries:      3,
		TimeoutSeconds:  30,
		FallbackStepID:  &fallbackStep,
		ContinueOnError: true,
	}

	assert.NotNil(t, config.FallbackStepID)
	assert.Equal(t, "fallback-step", *config.FallbackStepID)
}

func TestDefaultEngineConfig(t *testing.T) {
	config := DefaultEngineConfig

	assert.Equal(t, 10, config.MaxConcurrentWorkflows)
	assert.Equal(t, 5*time.Minute, config.DefaultTimeout)
}
