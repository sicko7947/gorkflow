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

	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount))

	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, gorkflow.StepStatusCompleted, steps[0].Status)
	assert.Equal(t, 2, steps[0].Attempt)
}

func TestEngine_RetryExhaustion(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptCount := int32(0)

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

	assert.Equal(t, gorkflow.RunStatusFailed, run.Status)
	assert.Equal(t, int32(4), atomic.LoadInt32(&attemptCount))

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

	require.Len(t, attemptTimes, 4)

	delay1 := attemptTimes[1].Sub(attemptTimes[0])
	delay2 := attemptTimes[2].Sub(attemptTimes[1])
	delay3 := attemptTimes[3].Sub(attemptTimes[2])

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

	require.Len(t, attemptTimes, 4)

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

	require.Len(t, attemptTimes, 3)

	delay1 := attemptTimes[1].Sub(attemptTimes[0])
	delay2 := attemptTimes[2].Sub(attemptTimes[1])

	assert.Less(t, delay1, 50*time.Millisecond)
	assert.Less(t, delay2, 50*time.Millisecond)
}

func TestEngine_Timeout(t *testing.T) {
	engine, _ := createTestEngine(t)

	slowStep := gorkflow.NewStep("slow", "Slow Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			select {
			case <-time.After(5 * time.Second):
				return DiscoverOutput{Companies: []string{"Done"}, Count: 1}, nil
			case <-ctx.Done():
				return DiscoverOutput{}, ctx.Err()
			}
		},
		gorkflow.WithTimeout(500*time.Millisecond),
		gorkflow.WithRetries(0),
	)

	wf, err := gorkflow.NewWorkflow("timeout_test", "Timeout Test").
		ThenStep(slowStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	assert.Equal(t, gorkflow.RunStatusFailed, run.Status)

	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, gorkflow.StepStatusFailed, steps[0].Status)
}

func TestEngine_TimeoutWithRetry(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptCount := int32(0)

	timeoutRetryStep := gorkflow.NewStep("timeout_retry", "Timeout Retry",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 3 {
				select {
				case <-time.After(2 * time.Second):
					return DiscoverOutput{}, nil
				case <-ctx.Done():
					return DiscoverOutput{}, ctx.Err()
				}
			}
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
			return EnrichOutput{Enriched: map[string]any{"result": "success"}}, nil
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

	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)

	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, gorkflow.StepStatusFailed, steps[0].Status)
	assert.Equal(t, gorkflow.StepStatusCompleted, steps[1].Status)
}
