package engine_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/store"
	"github.com/stretchr/testify/require"
)

// Enforce slow-start -> fast-start -> fast-complete -> slow-state-write so
// the regression does not depend on scheduler timing.
type parallelContextStore struct {
	gorkflow.WorkflowStore
	slowStarted   chan struct{}
	fastCompleted chan struct{}
}

func (s *parallelContextStore) CreateStepExecution(ctx context.Context, execution *gorkflow.StepExecution) error {
	if execution.StepID == "fast" {
		select {
		case <-s.slowStarted:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.WorkflowStore.CreateStepExecution(ctx, execution)
}

func (s *parallelContextStore) UpdateStepExecution(ctx context.Context, execution *gorkflow.StepExecution) error {
	err := s.WorkflowStore.UpdateStepExecution(ctx, execution)
	if execution.StepID == "fast" && execution.Status == gorkflow.StepStatusCompleted {
		close(s.fastCompleted)
	}
	return err
}

func (s *parallelContextStore) SaveState(ctx context.Context, runID, key string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.WorkflowStore.SaveState(ctx, runID, key, data)
}

func TestParallelStepsKeepIndependentStateContexts(t *testing.T) {
	s := &parallelContextStore{
		WorkflowStore: store.NewMemoryStore(),
		slowStarted:   make(chan struct{}),
		fastCompleted: make(chan struct{}),
	}
	e := engine.NewEngine(s, engine.WithLogger(zerolog.Nop()))
	entry := gorkflow.NewStep("entry", "entry", func(ctx *gorkflow.StepContext, input any) (any, error) { return input, nil })
	slow := gorkflow.NewStep("slow", "slow", func(ctx *gorkflow.StepContext, input any) (any, error) {
		close(s.slowStarted)
		select {
		case <-s.fastCompleted:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return input, ctx.State.Set("result", "saved")
	}, gorkflow.WithRetries(0))
	fast := gorkflow.NewStep("fast", "fast", func(ctx *gorkflow.StepContext, input any) (any, error) { return input, nil })
	wf, err := gorkflow.NewWorkflow("state-context", "State context").ThenStep(entry).Parallel(slow, fast).Build()
	require.NoError(t, err)
	runID, err := e.StartWorkflow(context.Background(), wf, nil, gorkflow.WithSynchronousExecution())
	require.NoError(t, err)
	data, err := s.LoadState(context.Background(), runID, "result")
	require.NoError(t, err)
	require.JSONEq(t, `"saved"`, string(data))
}
