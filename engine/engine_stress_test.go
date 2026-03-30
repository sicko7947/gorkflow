//go:build stress

package engine_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stressEngine(t *testing.T) (*engine.Engine, gorkflow.WorkflowStore) {
	t.Helper()
	wfStore := store.NewMemoryStore()
	eng := engine.NewEngine(wfStore, engine.WithConfig(gorkflow.EngineConfig{
		MaxConcurrentWorkflows: 200,
		DefaultTimeout:         5 * time.Minute,
	}))
	return eng, wfStore
}

func stressWaitForCompletion(t *testing.T, eng *engine.Engine, runID string, timeout time.Duration) *gorkflow.WorkflowRun {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for workflow %s", runID)
			return nil
		case <-ticker.C:
			run, err := eng.GetRun(context.Background(), runID)
			require.NoError(t, err)
			if run.Status.IsTerminal() {
				return run
			}
		}
	}
}

func TestStress_100ConcurrentWorkflows(t *testing.T) {
	t.Parallel()
	eng, _ := stressEngine(t)

	step := gorkflow.NewStep("noop", "Noop", func(ctx *gorkflow.StepContext, input any) (any, error) {
		return "done", nil
	})

	wf, err := gorkflow.NewWorkflow("stress-concurrent", "Stress Concurrent").
		ThenStep(step).
		Build()
	require.NoError(t, err)

	const count = 100
	runIDs := make([]string, count)
	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(idx int) {
			defer wg.Done()
			id, err := eng.StartWorkflow(context.Background(), wf, nil)
			require.NoError(t, err)
			runIDs[idx] = id
		}(i)
	}
	wg.Wait()

	for i, runID := range runIDs {
		run := stressWaitForCompletion(t, eng, runID, 30*time.Second)
		assert.Equal(t, gorkflow.RunStatusCompleted, run.Status, "workflow %d (run %s) not completed", i, runID)
	}
}

func TestStress_WorkflowWith100Steps(t *testing.T) {
	t.Parallel()
	eng, _ := stressEngine(t)

	steps := make([]gorkflow.StepExecutor, 100)
	for i := range steps {
		id := fmt.Sprintf("step-%d", i)
		steps[i] = gorkflow.NewStep(id, id, func(ctx *gorkflow.StepContext, input any) (any, error) {
			return input, nil
		})
	}

	builder := gorkflow.NewWorkflow("stress-100-steps", "Stress 100 Steps").ThenStep(steps[0])
	for i := 1; i < len(steps); i++ {
		builder = builder.ThenStep(steps[i])
	}
	wf, err := builder.Build()
	require.NoError(t, err)

	runID, err := eng.StartWorkflow(context.Background(), wf, "start")
	require.NoError(t, err)

	run := stressWaitForCompletion(t, eng, runID, 60*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)
	assert.Equal(t, 1.0, run.Progress)
}

func TestStress_ConcurrentCancelAndComplete(t *testing.T) {
	t.Parallel()
	eng, _ := stressEngine(t)

	step := gorkflow.NewStep("work", "Work", func(ctx *gorkflow.StepContext, input any) (any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return "done", nil
		}
	})

	wf, err := gorkflow.NewWorkflow("stress-cancel", "Stress Cancel").
		ThenStep(step).
		Build()
	require.NoError(t, err)

	runID, err := eng.StartWorkflow(context.Background(), wf, nil)
	require.NoError(t, err)

	// Immediately try to cancel
	_ = eng.Cancel(context.Background(), runID)

	run := stressWaitForCompletion(t, eng, runID, 10*time.Second)
	// Both are valid outcomes — no panics is what matters
	assert.Contains(t, []gorkflow.RunStatus{gorkflow.RunStatusCompleted, gorkflow.RunStatusCancelled}, run.Status)
}

func TestStress_Store_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	wfStore := store.NewMemoryStore()
	ctx := context.Background()

	run := &gorkflow.WorkflowRun{
		RunID:      "stress-run",
		WorkflowID: "stress-wf",
		Status:     gorkflow.RunStatusRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, wfStore.CreateRun(ctx, run))

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", idx)
			value := []byte(fmt.Sprintf("value-%d", idx))

			err := wfStore.SaveState(ctx, "stress-run", key, value)
			assert.NoError(t, err)

			loaded, err := wfStore.LoadState(ctx, "stress-run", key)
			assert.NoError(t, err)
			assert.Equal(t, value, loaded)
		}(i)
	}
	wg.Wait()
}

func TestStress_Retries_HighVolume(t *testing.T) {
	t.Parallel()
	eng, _ := stressEngine(t)

	const workflowCount = 20

	counters := make([]*int32, workflowCount)
	workflows := make([]*gorkflow.Workflow, workflowCount)

	for i := 0; i < workflowCount; i++ {
		counter := new(int32)
		counters[i] = counter
		idx := i

		step := gorkflow.NewStep(
			fmt.Sprintf("retry-step-%d", idx),
			fmt.Sprintf("Retry Step %d", idx),
			func(ctx *gorkflow.StepContext, input any) (any, error) {
				count := atomic.AddInt32(counters[idx], 1)
				if count < 3 {
					return nil, fmt.Errorf("attempt %d failed", count)
				}
				return "success", nil
			},
			gorkflow.WithRetries(3),
			gorkflow.WithRetryDelay(10*time.Millisecond),
		)

		wf, err := gorkflow.NewWorkflow(
			fmt.Sprintf("stress-retry-%d", idx),
			fmt.Sprintf("Stress Retry %d", idx),
		).ThenStep(step).Build()
		require.NoError(t, err)
		workflows[idx] = wf
	}

	runIDs := make([]string, workflowCount)
	for i := 0; i < workflowCount; i++ {
		id, err := eng.StartWorkflow(context.Background(), workflows[i], nil)
		require.NoError(t, err)
		runIDs[i] = id
	}

	for i, runID := range runIDs {
		run := stressWaitForCompletion(t, eng, runID, 30*time.Second)
		assert.Equal(t, gorkflow.RunStatusCompleted, run.Status, "workflow %d did not complete", i)
	}
}

func TestStress_ParallelLevel_20Steps(t *testing.T) {
	t.Parallel()
	eng, _ := stressEngine(t)

	entry := gorkflow.NewStep("entry", "Entry", func(ctx *gorkflow.StepContext, input any) (any, error) {
		return input, nil
	})

	leaves := make([]gorkflow.StepExecutor, 20)
	for i := range leaves {
		id := fmt.Sprintf("leaf-%d", i)
		leaves[i] = gorkflow.NewStep(id, id, func(ctx *gorkflow.StepContext, input any) (any, error) {
			time.Sleep(10 * time.Millisecond)
			return ctx.StepID, nil
		})
	}

	wf, err := gorkflow.NewWorkflow("stress-parallel-20", "Stress Parallel 20").
		ThenStep(entry).
		Parallel(leaves...).
		Build()
	require.NoError(t, err)

	runID, err := eng.StartWorkflow(context.Background(), wf, "input")
	require.NoError(t, err)

	run := stressWaitForCompletion(t, eng, runID, 30*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)
	assert.Equal(t, 1.0, run.Progress)
}
