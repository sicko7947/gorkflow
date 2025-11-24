package engine

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	workflow "github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_RetrySuccess(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptCount := int32(0)

	// Step that fails twice, then succeeds
	retryStep := workflow.NewStep("retry", "Retry Step",
		func(ctx *workflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 3 {
				return DiscoverOutput{}, errors.New("temporary failure")
			}
			return DiscoverOutput{Companies: []string{"Success"}, Count: 1}, nil
		},
		workflow.WithRetries(3),
		workflow.WithRetryDelay(100*time.Millisecond),
	)

	wf, err := builder.NewWorkflow("retry_test", "Retry Test").
		ThenStep(retryStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	// Should eventually succeed
	assert.Equal(t, workflow.RunStatusCompleted, run.Status)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount))

	// Check step execution
	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, workflow.StepStatusCompleted, steps[0].Status)
	assert.Equal(t, 2, steps[0].Attempt) // 0-indexed, so attempt 2 = 3rd try
}

func TestEngine_RetryExhaustion(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptCount := int32(0)

	// Step that always fails
	alwaysFailStep := workflow.NewStep("fail", "Always Fail",
		func(ctx *workflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			atomic.AddInt32(&attemptCount, 1)
			return DiscoverOutput{}, errors.New("persistent failure")
		},
		workflow.WithRetries(3),
		workflow.WithRetryDelay(50*time.Millisecond),
	)

	wf, err := builder.NewWorkflow("exhaust_test", "Exhaust Test").
		ThenStep(alwaysFailStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	// Should fail after all retries
	assert.Equal(t, workflow.RunStatusFailed, run.Status)
	assert.Equal(t, int32(4), atomic.LoadInt32(&attemptCount)) // Initial attempt + 3 retries

	// Check step execution
	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, workflow.StepStatusFailed, steps[0].Status)
	assert.NotNil(t, steps[0].Error)
}

func TestEngine_LinearBackoff(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptTimes := make([]time.Time, 0, 4)
	attemptCount := int32(0)

	retryStep := workflow.NewStep("backoff", "Backoff Test",
		func(ctx *workflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			attemptTimes = append(attemptTimes, time.Now())
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 4 {
				return DiscoverOutput{}, errors.New("retry")
			}
			return DiscoverOutput{Companies: []string{"Done"}, Count: 1}, nil
		},
		workflow.WithRetries(3),
		workflow.WithRetryDelay(200*time.Millisecond),
		workflow.WithBackoff(workflow.BackoffLinear),
	)

	wf, err := builder.NewWorkflow("linear_backoff", "Linear Backoff").
		ThenStep(retryStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	waitForCompletion(t, engine, runID, 15*time.Second)

	// Verify linear backoff delays
	require.Len(t, attemptTimes, 4)

	// Delays should increase linearly: 200ms, 400ms, 600ms
	delay1 := attemptTimes[1].Sub(attemptTimes[0])
	delay2 := attemptTimes[2].Sub(attemptTimes[1])
	delay3 := attemptTimes[3].Sub(attemptTimes[2])

	// Allow 50ms tolerance
	tolerance := 50 * time.Millisecond
	assert.InDelta(t, 200*time.Millisecond, delay1, float64(tolerance))
	assert.InDelta(t, 400*time.Millisecond, delay2, float64(tolerance))
	assert.InDelta(t, 600*time.Millisecond, delay3, float64(tolerance))
}

func TestEngine_ExponentialBackoff(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptTimes := make([]time.Time, 0, 4)
	attemptCount := int32(0)

	retryStep := workflow.NewStep("exp_backoff", "Exponential Backoff",
		func(ctx *workflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			attemptTimes = append(attemptTimes, time.Now())
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 4 {
				return DiscoverOutput{}, errors.New("retry")
			}
			return DiscoverOutput{Companies: []string{"Done"}, Count: 1}, nil
		},
		workflow.WithRetries(3),
		workflow.WithRetryDelay(100*time.Millisecond),
		workflow.WithBackoff(workflow.BackoffExponential),
	)

	wf, err := builder.NewWorkflow("exp_backoff", "Exponential Backoff").
		ThenStep(retryStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	waitForCompletion(t, engine, runID, 15*time.Second)

	// Verify exponential backoff
	require.Len(t, attemptTimes, 4)

	// Delays should increase exponentially: 100ms, 200ms, 400ms
	delay1 := attemptTimes[1].Sub(attemptTimes[0])
	delay2 := attemptTimes[2].Sub(attemptTimes[1])
	delay3 := attemptTimes[3].Sub(attemptTimes[2])

	tolerance := 50 * time.Millisecond
	assert.InDelta(t, 100*time.Millisecond, delay1, float64(tolerance))
	assert.InDelta(t, 200*time.Millisecond, delay2, float64(tolerance))
	assert.InDelta(t, 400*time.Millisecond, delay3, float64(tolerance))
}

func TestEngine_NoBackoff(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptTimes := make([]time.Time, 0, 3)
	attemptCount := int32(0)

	retryStep := workflow.NewStep("no_backoff", "No Backoff",
		func(ctx *workflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			attemptTimes = append(attemptTimes, time.Now())
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 3 {
				return DiscoverOutput{}, errors.New("retry")
			}
			return DiscoverOutput{Companies: []string{"Done"}, Count: 1}, nil
		},
		workflow.WithRetries(2),
		workflow.WithRetryDelay(100*time.Millisecond),
		workflow.WithBackoff(workflow.BackoffNone),
	)

	wf, err := builder.NewWorkflow("no_backoff", "No Backoff").
		ThenStep(retryStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	waitForCompletion(t, engine, runID, 10*time.Second)

	// With no backoff, all retries should happen immediately
	require.Len(t, attemptTimes, 3)

	delay1 := attemptTimes[1].Sub(attemptTimes[0])
	delay2 := attemptTimes[2].Sub(attemptTimes[1])

	// Should be minimal delay (just processing time)
	assert.Less(t, delay1, 50*time.Millisecond)
	assert.Less(t, delay2, 50*time.Millisecond)
}

func TestEngine_Timeout(t *testing.T) {
	engine, _ := createTestEngine(t)

	// Step that takes too long
	slowStep := workflow.NewStep("slow", "Slow Step",
		func(ctx *workflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			// Try to run for 5 seconds but will be interrupted by timeout
			select {
			case <-time.After(5 * time.Second):
				return DiscoverOutput{Companies: []string{"Done"}, Count: 1}, nil
			case <-ctx.Done():
				return DiscoverOutput{}, ctx.Err()
			}
		},
		workflow.WithTimeout(500*time.Millisecond),
		workflow.WithRetries(0), // No retries to make test faster
	)

	wf, err := builder.NewWorkflow("timeout_test", "Timeout Test").
		ThenStep(slowStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	// Should fail due to timeout
	assert.Equal(t, workflow.RunStatusFailed, run.Status)

	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, workflow.StepStatusFailed, steps[0].Status)
}

func TestEngine_TimeoutWithRetry(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptCount := int32(0)

	// Step that times out twice, then succeeds quickly
	timeoutRetryStep := workflow.NewStep("timeout_retry", "Timeout Retry",
		func(ctx *workflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 3 {
				// First two attempts timeout
				select {
				case <-time.After(2 * time.Second):
					return DiscoverOutput{}, nil
				case <-ctx.Done():
					return DiscoverOutput{}, ctx.Err()
				}
			}
			// Third attempt succeeds quickly
			return DiscoverOutput{Companies: []string{"Success"}, Count: 1}, nil
		},
		workflow.WithTimeout(500*time.Millisecond),
		workflow.WithRetries(3),
		workflow.WithRetryDelay(100*time.Millisecond),
	)

	wf, err := builder.NewWorkflow("timeout_retry_test", "Timeout Retry Test").
		ThenStep(timeoutRetryStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 15*time.Second)

	// Should eventually succeed after retries
	assert.Equal(t, workflow.RunStatusCompleted, run.Status)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount))
}

func TestEngine_ContinueOnError(t *testing.T) {
	engine, _ := createTestEngine(t)

	failStep := workflow.NewStep("fail", "Fail Step",
		func(ctx *workflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			return DiscoverOutput{}, errors.New("step failed")
		},
		workflow.WithRetries(0),
		workflow.WithContinueOnError(true),
	)

	successStep := workflow.NewStep("success", "Success Step",
		func(ctx *workflow.StepContext, input EnrichInput) (EnrichOutput, error) {
			return EnrichOutput{Enriched: map[string]interface{}{"result": "success"}}, nil
		},
	)

	wf, err := builder.NewWorkflow("continue_on_error", "Continue On Error").
		ThenStep(failStep).
		ThenStep(successStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	// Workflow should complete despite first step failing
	assert.Equal(t, workflow.RunStatusCompleted, run.Status)

	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, workflow.StepStatusFailed, steps[0].Status)
	assert.Equal(t, workflow.StepStatusCompleted, steps[1].Status)
}
