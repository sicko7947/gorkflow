package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sicko7947/gorkflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_SimpleSequentialWorkflow(t *testing.T) {
	engine, _ := createTestEngine(t)

	discoverStep := gorkflow.NewStep("discover", "Discover Companies", discoverCompanies)
	enrichStep := gorkflow.NewStep("enrich", "Enrich Companies", enrichCompanies)
	filterStep := gorkflow.NewStep("filter", "Filter Companies", filterCompanies)

	wf, err := gorkflow.NewWorkflow("sequential_test", "Sequential Test").
		ThenStep(discoverStep).
		ThenStep(enrichStep).
		ThenStep(filterStep).
		Build()
	require.NoError(t, err)

	input := DiscoverInput{
		Query: "tech companies",
		Limit: 10,
	}

	runID, err := engine.StartWorkflow(context.Background(), wf, input)
	require.NoError(t, err)
	require.NotEmpty(t, runID)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)
	assert.Equal(t, 1.0, run.Progress)
	assert.NotNil(t, run.CompletedAt)

	steps, err := engine.GetStepExecutions(context.Background(), runID)
	require.NoError(t, err)
	assert.Len(t, steps, 3)

	for _, step := range steps {
		assert.Equal(t, gorkflow.StepStatusCompleted, step.Status)
		assert.NotNil(t, step.CompletedAt)
	}
}

func TestEngine_WorkflowWithFailure(t *testing.T) {
	engine, _ := createTestEngine(t)

	failingStep := gorkflow.NewStep("failing", "Failing Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			return DiscoverOutput{}, errors.New("intentional failure")
		},
		gorkflow.WithRetries(2),
	)

	wf, err := gorkflow.NewWorkflow("failing_workflow", "Failing Workflow").
		ThenStep(failingStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	assert.Equal(t, gorkflow.RunStatusFailed, run.Status)
	assert.NotNil(t, run.Error)
	assert.Contains(t, run.Error.Message, "intentional failure")
}

func TestEngine_WorkflowProgress(t *testing.T) {
	engine, _ := createTestEngine(t)

	slowStep1 := gorkflow.NewStep("slow1", "Slow Step 1",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			time.Sleep(500 * time.Millisecond)
			return DiscoverOutput{Companies: []string{"A"}, Count: 1}, nil
		},
	)

	slowStep2 := gorkflow.NewStep("slow2", "Slow Step 2",
		func(ctx *gorkflow.StepContext, input EnrichInput) (EnrichOutput, error) {
			time.Sleep(500 * time.Millisecond)
			return EnrichOutput{Enriched: map[string]any{"A": "data"}}, nil
		},
	)

	wf, err := gorkflow.NewWorkflow("progress_test", "Progress Test").
		ThenStep(slowStep1).
		ThenStep(slowStep2).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	time.Sleep(600 * time.Millisecond)
	run, _ := engine.GetRun(context.Background(), runID)
	assert.Greater(t, run.Progress, 0.0)
	assert.Less(t, run.Progress, 1.0)

	run = waitForCompletion(t, engine, runID, 10*time.Second)
	assert.Equal(t, 1.0, run.Progress)
}

func TestEngine_StepOutputPassing(t *testing.T) {
	engine, wfStore := createTestEngine(t)

	discoverStep := gorkflow.NewStep("discover", "Discover", discoverCompanies)
	enrichStep := gorkflow.NewStep("enrich", "Enrich", enrichCompanies)

	wf, err := gorkflow.NewWorkflow("output_test", "Output Test").
		ThenStep(discoverStep).
		ThenStep(enrichStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	waitForCompletion(t, engine, runID, 10*time.Second)

	discoverOutputBytes, err := wfStore.LoadStepOutput(context.Background(), runID, "discover")
	require.NoError(t, err)

	var discoverOutput DiscoverOutput
	json.Unmarshal(discoverOutputBytes, &discoverOutput)
	assert.Greater(t, discoverOutput.Count, 0)
	assert.NotEmpty(t, discoverOutput.Companies)

	enrichOutputBytes, err := wfStore.LoadStepOutput(context.Background(), runID, "enrich")
	require.NoError(t, err)

	var enrichOutput EnrichOutput
	json.Unmarshal(enrichOutputBytes, &enrichOutput)
	assert.NotEmpty(t, enrichOutput.Enriched)
}

func TestEngine_GetStepExecutions(t *testing.T) {
	engine, _ := createTestEngine(t)

	discoverStep := gorkflow.NewStep("discover", "Discover", discoverCompanies)
	enrichStep := gorkflow.NewStep("enrich", "Enrich", enrichCompanies)

	wf, err := gorkflow.NewWorkflow("test", "Test").
		ThenStep(discoverStep).
		ThenStep(enrichStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	waitForCompletion(t, engine, runID, 10*time.Second)

	steps, err := engine.GetStepExecutions(context.Background(), runID)
	require.NoError(t, err)

	assert.Len(t, steps, 2)

	for _, step := range steps {
		assert.NotEmpty(t, step.StepID)
		assert.NotNil(t, step.StartedAt)
		assert.NotNil(t, step.CompletedAt)
		assert.GreaterOrEqual(t, step.DurationMs, int64(0))
	}
}

func TestEngine_ListRuns(t *testing.T) {
	engine, _ := createTestEngine(t)

	step := gorkflow.NewStep("test", "Test", discoverCompanies)

	wf, err := gorkflow.NewWorkflow("diamond_test", "Diamond Test").
		ThenStep(step).
		Build()
	require.NoError(t, err)

	runID1, _ := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test1", Limit: 10})
	runID2, _ := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test2", Limit: 10})

	waitForCompletion(t, engine, runID1, 10*time.Second)
	waitForCompletion(t, engine, runID2, 10*time.Second)

	filter := gorkflow.RunFilter{
		WorkflowID: "diamond_test",
		Limit:      10,
	}

	runs, err := engine.ListRuns(context.Background(), filter)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(runs), 2)
}

func TestEngine_Cancel(t *testing.T) {
	engine, _ := createTestEngine(t)

	longStep := gorkflow.NewStep("long", "Long Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			select {
			case <-ctx.Done():
				return DiscoverOutput{}, ctx.Err()
			case <-time.After(10 * time.Second):
				return DiscoverOutput{Companies: []string{"A"}, Count: 1}, nil
			}
		},
	)

	wf, err := gorkflow.NewWorkflow("concurrency_test", "Concurrency Test").
		ThenStep(longStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	err = engine.Cancel(context.Background(), runID)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)
	run, _ := engine.GetRun(context.Background(), runID)
	assert.Equal(t, gorkflow.RunStatusCancelled, run.Status)
}

func TestEngine_WorkflowState(t *testing.T) {
	engine, wfStore := createTestEngine(t)

	statefulStep := gorkflow.NewStep("stateful", "Stateful Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			ctx.State.Set("timestamp", time.Now().Unix())
			ctx.State.Set("query", input.Query)

			var query string
			ctx.State.Get("query", &query)

			return DiscoverOutput{Companies: []string{query}, Count: 1}, nil
		},
	)

	wf, err := gorkflow.NewWorkflow("state_test", "State Test").
		ThenStep(statefulStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test-query", Limit: 10})
	require.NoError(t, err)

	waitForCompletion(t, engine, runID, 10*time.Second)

	allState, err := wfStore.GetAllState(context.Background(), runID)
	require.NoError(t, err)

	assert.Contains(t, allState, "timestamp")
	assert.Contains(t, allState, "query")
}

func TestEngine_ConditionalStep_Skipped_PassThrough(t *testing.T) {
	engine, _ := createTestEngine(t)

	step1 := gorkflow.NewStep("step1", "Step 1", func(ctx *gorkflow.StepContext, input any) (string, error) {
		return "step1 output", nil
	})

	step2 := gorkflow.NewStep("step2", "Step 2", func(ctx *gorkflow.StepContext, input string) (string, error) {
		return "step2 output", nil
	})

	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		return false, nil
	}

	wf, err := gorkflow.NewWorkflow("test_wf", "Test Workflow").
		ThenStep(step1).
		ThenStepIf(step2, condition, nil).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, nil)
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)

	steps, err := engine.GetStepExecutions(context.Background(), runID)
	require.NoError(t, err)

	var step2Exec *gorkflow.StepExecution
	for _, s := range steps {
		if s.StepID == "step2" {
			step2Exec = s
			break
		}
	}
	require.NotNil(t, step2Exec)
	assert.Equal(t, gorkflow.StepStatusCompleted, step2Exec.Status)

	var output string
	err = json.Unmarshal(step2Exec.Output, &output)
	require.NoError(t, err)
	assert.Equal(t, "step1 output", output)
}

func TestEngine_ParallelStepExecution(t *testing.T) {
	engine, _ := createTestEngine(t)

	stepA := gorkflow.NewStep("A", "Step A", func(ctx *gorkflow.StepContext, input any) (string, error) {
		return "output-A", nil
	})

	stepB := gorkflow.NewStep("B", "Step B", func(ctx *gorkflow.StepContext, input string) (string, error) {
		if input != "output-A" {
			return "", fmt.Errorf("step B expected 'output-A', got '%s'", input)
		}
		return "output-B", nil
	})

	stepC := gorkflow.NewStep("C", "Step C", func(ctx *gorkflow.StepContext, input string) (string, error) {
		if input != "output-A" {
			return "", fmt.Errorf("step C expected 'output-A', got '%s'", input)
		}
		return "output-C", nil
	})

	wf, err := gorkflow.NewWorkflow("parallel_test", "Parallel Test").
		ThenStep(stepA).
		Parallel(stepB, stepC).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, nil)
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)

	if run.Status == gorkflow.RunStatusFailed {
		t.Logf("Workflow failed: %s", run.Error.Message)
	}
}
