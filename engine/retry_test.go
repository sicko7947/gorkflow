package engine

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sicko7947/gorkflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_RetrySuccess(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptCount := int32(0)

	// Step that fails twice, then succeeds
	retryStep := gorkflow.NewStep("retry", "Retry Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 3 {
				return DiscoverOutput{}, errors.New("temporary failure")
			}
			return DiscoverOutput{Companies: []string{"Success"}, Count: 1}, nil
		},
		gorkflow.WithRetries(3),
		gorkflow.WithRetryDelay(100*time.Millisecond),
	)

	wf, err := gorkflow.NewWorkflow("retry_test", "Retry Test").
		ThenStep(retryStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	// Should eventually succeed
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount))

	// Check step execution
	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, gorkflow.StepStatusCompleted, steps[0].Status)
	assert.Equal(t, 2, steps[0].Attempt) // 0-indexed, so attempt 2 = 3rd try
}

func TestEngine_RetryExhaustion(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptCount := int32(0)

	// Step that always fails
	alwaysFailStep := gorkflow.NewStep("fail", "Always Fail",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			atomic.AddInt32(&attemptCount, 1)
			return DiscoverOutput{}, errors.New("persistent failure")
		},
		gorkflow.WithRetries(3),
		gorkflow.WithRetryDelay(50*time.Millisecond),
	)

	wf, err := gorkflow.NewWorkflow("exhaust_test", "Exhaust Test").
		ThenStep(alwaysFailStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	// Should fail after all retries
	assert.Equal(t, gorkflow.RunStatusFailed, run.Status)
	assert.Equal(t, int32(4), atomic.LoadInt32(&attemptCount)) // Initial attempt + 3 retries

	// Check step execution
	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, gorkflow.StepStatusFailed, steps[0].Status)
	assert.NotNil(t, steps[0].Error)
}

func TestEngine_LinearBackoff(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptTimes := make([]time.Time, 0, 4)
	attemptCount := int32(0)

	retryStep := gorkflow.NewStep("backoff", "Backoff Test",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			attemptTimes = append(attemptTimes, time.Now())
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 4 {
				return DiscoverOutput{}, errors.New("retry")
			}
			return DiscoverOutput{Companies: []string{"Done"}, Count: 1}, nil
		},
		gorkflow.WithRetries(3),
		gorkflow.WithRetryDelay(200*time.Millisecond),
		gorkflow.WithBackoff(gorkflow.BackoffLinear),
	)

	wf, err := gorkflow.NewWorkflow("linear_backoff", "Linear Backoff").
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

	retryStep := gorkflow.NewStep("exp_backoff", "Exponential Backoff",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			attemptTimes = append(attemptTimes, time.Now())
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 4 {
				return DiscoverOutput{}, errors.New("retry")
			}
			return DiscoverOutput{Companies: []string{"Done"}, Count: 1}, nil
		},
		gorkflow.WithRetries(3),
		gorkflow.WithRetryDelay(100*time.Millisecond),
		gorkflow.WithBackoff(gorkflow.BackoffExponential),
	)

	wf, err := gorkflow.NewWorkflow("exp_backoff", "Exponential Backoff").
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

	retryStep := gorkflow.NewStep("no_backoff", "No Backoff",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			attemptTimes = append(attemptTimes, time.Now())
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 3 {
				return DiscoverOutput{}, errors.New("retry")
			}
			return DiscoverOutput{Companies: []string{"Done"}, Count: 1}, nil
		},
		gorkflow.WithRetries(2),
		gorkflow.WithRetryDelay(100*time.Millisecond),
		gorkflow.WithBackoff(gorkflow.BackoffNone),
	)

	wf, err := gorkflow.NewWorkflow("no_backoff", "No Backoff").
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
	slowStep := gorkflow.NewStep("slow", "Slow Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			// Try to run for 5 seconds but will be interrupted by timeout
			select {
			case <-time.After(5 * time.Second):
				return DiscoverOutput{Companies: []string{"Done"}, Count: 1}, nil
			case <-ctx.Done():
				return DiscoverOutput{}, ctx.Err()
			}
		},
		gorkflow.WithTimeout(500*time.Millisecond),
		gorkflow.WithRetries(0), // No retries to make test faster
	)

	wf, err := gorkflow.NewWorkflow("timeout_test", "Timeout Test").
		ThenStep(slowStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	// Should fail due to timeout
	assert.Equal(t, gorkflow.RunStatusFailed, run.Status)

	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, gorkflow.StepStatusFailed, steps[0].Status)
}

func TestEngine_TimeoutWithRetry(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptCount := int32(0)

	// Step that times out twice, then succeeds quickly
	timeoutRetryStep := gorkflow.NewStep("timeout_retry", "Timeout Retry",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
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
		gorkflow.WithTimeout(500*time.Millisecond),
		gorkflow.WithRetries(3),
		gorkflow.WithRetryDelay(100*time.Millisecond),
	)

	wf, err := gorkflow.NewWorkflow("timeout_retry_test", "Timeout Retry Test").
		ThenStep(timeoutRetryStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 15*time.Second)

	// Should eventually succeed after retries
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount))
}

func TestEngine_ContinueOnError(t *testing.T) {
	engine, _ := createTestEngine(t)

	failStep := gorkflow.NewStep("fail", "Fail Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			return DiscoverOutput{}, errors.New("step failed")
		},
		gorkflow.WithRetries(0),
		gorkflow.WithContinueOnError(true),
	)

	successStep := gorkflow.NewStep("success", "Success Step",
		func(ctx *gorkflow.StepContext, input EnrichInput) (EnrichOutput, error) {
			return EnrichOutput{Enriched: map[string]interface{}{"result": "success"}}, nil
		},
	)

	wf, err := gorkflow.NewWorkflow("continue_on_error", "Continue On Error").
		ThenStep(failStep).
		ThenStep(successStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	// Workflow should complete despite first step failing
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)

	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, gorkflow.StepStatusFailed, steps[0].Status)
	assert.Equal(t, gorkflow.StepStatusCompleted, steps[1].Status)
}
