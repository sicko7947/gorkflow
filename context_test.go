package gorkflow_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test context types
type TestContext struct {
	UserID    string
	RequestID string
}

type AppContext struct {
	UserID string
	Config map[string]string
}

type TestRunContext struct {
	UserID    string
	RequestID string
	Metadata  map[string]string
}

func TestWorkflowWithContext(t *testing.T) {
	customCtx := TestContext{
		UserID:    "user-123",
		RequestID: "req-abc",
	}

	checkContextStep := gorkflow.NewStep(
		"check-context",
		"Check Context",
		func(ctx *gorkflow.StepContext, input string) (string, error) {
			userCtx, err := gorkflow.GetContext[TestContext](ctx)
			if err != nil {
				return "", err
			}

			if userCtx.UserID != "user-123" {
				return "", fmt.Errorf("expected UserID user-123, got %s", userCtx.UserID)
			}
			if userCtx.RequestID != "req-abc" {
				return "", fmt.Errorf("expected RequestID req-abc, got %s", userCtx.RequestID)
			}

			return "context verified", nil
		},
	)

	wf := gorkflow.NewWorkflow("context-test-wf", "Context Test Workflow").
		WithContext(customCtx).
		ThenStep(checkContextStep).
		MustBuild()

	eng := engine.NewEngine(store.NewMemoryStore())
	runID, err := eng.StartWorkflow(context.Background(), wf, "start", gorkflow.WithSynchronousExecution())
	assert.NoError(t, err)

	run, err := eng.GetRun(context.Background(), runID)
	assert.NoError(t, err)
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)
}

func TestWorkflowWithCustomContext(t *testing.T) {
	stepHandler := func(ctx *gorkflow.StepContext, input string) (string, error) {
		appCtx, err := gorkflow.GetContext[*AppContext](ctx)
		if err != nil {
			return "", err
		}

		if appCtx.UserID != "user-123" {
			return "", fmt.Errorf("unexpected user ID: %s", appCtx.UserID)
		}

		val, ok := appCtx.Config["env"]
		if !ok || val != "production" {
			return "", fmt.Errorf("unexpected config value: %v", appCtx.Config)
		}

		return fmt.Sprintf("Processed for %s in %s", appCtx.UserID, val), nil
	}

	appCtx := &AppContext{
		UserID: "user-123",
		Config: map[string]string{"env": "production"},
	}

	wf := gorkflow.NewWorkflowInstance("test-wf", "Test Workflow", gorkflow.WithContext(appCtx))
	step := gorkflow.NewStep("step-1", "Step 1", stepHandler)
	wf.AddStep(step)
	wf.Graph().AddNode(step.GetID(), gorkflow.NodeTypeSequential)

	s := store.NewMemoryStore()
	eng := engine.NewEngine(s)

	runID, err := eng.StartWorkflow(context.Background(), wf, "start", gorkflow.WithSynchronousExecution())
	require.NoError(t, err)

	run, err := eng.GetRun(context.Background(), runID)
	require.NoError(t, err)
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)

	executions, err := eng.GetStepExecutions(context.Background(), runID)
	require.NoError(t, err)
	require.Len(t, executions, 1)
	assert.Equal(t, "\"Processed for user-123 in production\"", string(executions[0].Output))
}

func TestGetRunContext(t *testing.T) {
	customCtx := TestRunContext{
		UserID:    "user-456",
		RequestID: "req-xyz",
		Metadata: map[string]string{
			"source":  "api",
			"version": "v1",
		},
	}

	step := gorkflow.NewStep(
		"test-step",
		"Test Step",
		func(ctx *gorkflow.StepContext, input string) (string, error) {
			return "ok", nil
		},
	)

	wf := gorkflow.NewWorkflow("context-retrieval-test", "Context Retrieval Test").
		WithContext(customCtx).
		ThenStep(step).
		MustBuild()

	eng := engine.NewEngine(store.NewMemoryStore())
	runID, err := eng.StartWorkflow(context.Background(), wf, "start", gorkflow.WithSynchronousExecution())
	require.NoError(t, err)

	run, err := eng.GetRun(context.Background(), runID)
	require.NoError(t, err)
	assert.NotNil(t, run.Context)

	retrievedCtx, err := gorkflow.GetRunContext[TestRunContext](run)
	require.NoError(t, err)

	assert.Equal(t, customCtx.UserID, retrievedCtx.UserID)
	assert.Equal(t, customCtx.RequestID, retrievedCtx.RequestID)
	assert.Equal(t, customCtx.Metadata, retrievedCtx.Metadata)
}

func TestGetRunContext_NoContext(t *testing.T) {
	step := gorkflow.NewStep(
		"test-step",
		"Test Step",
		func(ctx *gorkflow.StepContext, input string) (string, error) {
			return "ok", nil
		},
	)

	wf := gorkflow.NewWorkflow("no-context-test", "No Context Test").
		ThenStep(step).
		MustBuild()

	eng := engine.NewEngine(store.NewMemoryStore())
	runID, err := eng.StartWorkflow(context.Background(), wf, "start", gorkflow.WithSynchronousExecution())
	require.NoError(t, err)

	run, err := eng.GetRun(context.Background(), runID)
	require.NoError(t, err)

	_, err = gorkflow.GetRunContext[TestRunContext](run)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no context")
}

// stateFailureStore checks that accessors propagate their own context and only
// publish state after successful persistence.
type stateFailureStore struct {
	gorkflow.WorkflowStore
	failSave   bool
	failDelete bool
}

func (s *stateFailureStore) SaveState(ctx context.Context, runID, key string, value []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.failSave {
		return fmt.Errorf("save failed")
	}
	return s.WorkflowStore.SaveState(ctx, runID, key, value)
}

func (s *stateFailureStore) DeleteState(ctx context.Context, runID, key string) error {
	if s.failDelete {
		return fmt.Errorf("delete failed")
	}
	return s.WorkflowStore.DeleteState(ctx, runID, key)
}

func TestStateAccessorFailedWritesPreserveCache(t *testing.T) {
	s := &stateFailureStore{WorkflowStore: store.NewMemoryStore()}
	a := gorkflow.NewStateAccessor("run", s)
	require.NoError(t, a.Set("key", "original"))
	s.failSave = true
	require.Error(t, a.Set("key", "failed"))
	value, err := gorkflow.GetTyped[string](a, "key")
	require.NoError(t, err)
	require.Equal(t, "original", value)
	require.Error(t, a.Set("missing", "failed"))
	require.False(t, a.Has("missing"))

	s.failDelete = true
	require.Error(t, a.Delete("key"))
	// Remove persisted data directly to distinguish retaining the cache from
	// reloading the original value after incorrectly invalidating the cache.
	require.NoError(t, s.WorkflowStore.DeleteState(context.Background(), "run", "key"))
	value, err = gorkflow.GetTyped[string](a, "key")
	require.NoError(t, err)
	require.Equal(t, "original", value)
}

func TestStateAccessorContextViewsShareCache(t *testing.T) {
	s := &stateFailureStore{WorkflowStore: store.NewMemoryStore()}
	a := gorkflow.NewStateAccessor("run", s)
	ctx, cancel := context.WithCancel(context.Background())
	first := gorkflow.WithStateAccessorContext(a, ctx)
	second := gorkflow.WithStateAccessorContext(a, context.Background())
	require.NoError(t, first.Set("key", "first"))
	value, err := gorkflow.GetTyped[string](second, "key")
	require.NoError(t, err)
	require.Equal(t, "first", value)
	cancel()
	require.ErrorIs(t, first.Set("key", "cancelled"), context.Canceled)
	require.NoError(t, second.Set("key", "second"))
	value, err = gorkflow.GetTyped[string](first, "key")
	require.NoError(t, err)
	require.Equal(t, "second", value)
	require.NoError(t, second.Delete("key"))
	require.False(t, first.Has("key"))
}

func TestStateAccessorGetAllRefreshesCache(t *testing.T) {
	s := store.NewMemoryStore()
	a := gorkflow.NewStateAccessor("run", s)
	require.NoError(t, a.Set("deleted", 1))
	require.NoError(t, a.Set("retained", 2))
	require.NoError(t, s.DeleteState(context.Background(), "run", "deleted"))
	all, err := a.GetAll()
	require.NoError(t, err)
	require.False(t, a.Has("deleted"))
	all["retained"][0] = '9'
	value, err := gorkflow.GetTyped[int](a, "retained")
	require.NoError(t, err)
	require.Equal(t, 2, value)
}

func BenchmarkStateAccessorCachedGet(b *testing.B) {
	a := gorkflow.NewStateAccessor("run", store.NewMemoryStore())
	if err := a.Set("key", 42); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var value int
		for pb.Next() {
			if err := a.Get("key", &value); err != nil {
				b.Error(err)
			}
		}
	})
}

func TestStateAccessorConcurrentViewsRemainCoherent(t *testing.T) {
	s := store.NewMemoryStore()
	a := gorkflow.NewStateAccessor("run", s)
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			view := gorkflow.WithStateAccessorContext(a, context.Background())
			for i := 0; i < 100; i++ {
				if err := view.Set("key", worker*100+i); err != nil {
					t.Error(err)
					return
				}
				var value int
				if err := view.Get("key", &value); err != nil {
					t.Error(err)
					return
				}
			}
		}(worker)
	}
	wg.Wait()
	cached, err := gorkflow.GetTyped[int](a, "key")
	require.NoError(t, err)
	persisted, err := s.LoadState(context.Background(), "run", "key")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprint(cached), string(persisted))
}

func BenchmarkStateAccessorSet(b *testing.B) {
	a := gorkflow.NewStateAccessor("run", store.NewMemoryStore())
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := a.Set("key", 42); err != nil {
				b.Error(err)
			}
		}
	})
}
