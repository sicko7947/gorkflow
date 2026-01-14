package engine

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sicko7947/gorkflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_ConditionalStep_ExecutesWhenTrue(t *testing.T) {
	engine, _ := createTestEngine(t)

	baseStep := gorkflow.NewStep("conditional", "Conditional Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			return DiscoverOutput{Companies: []string{"Executed"}, Count: 1}, nil
		},
	)

	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		return true, nil
	}

	condStep := gorkflow.NewConditionalStep(baseStep, condition, nil)

	wf, err := gorkflow.NewWorkflow("conditional_true", "Conditional True").
		ThenStep(condStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)

	steps, _ := engine.GetStepExecutions(context.Background(), runID)
	assert.Equal(t, gorkflow.StepStatusCompleted, steps[0].Status)
}

func TestEngine_ConditionalStep_SkipsWhenFalse(t *testing.T) {
	engine, wfStore := createTestEngine(t)

	baseStep := gorkflow.NewStep("conditional", "Conditional Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			return DiscoverOutput{Companies: []string{"Should Not Execute"}, Count: 1}, nil
		},
	)

	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		return false, nil
	}

	defaultOutput := &DiscoverOutput{Companies: []string{"Default"}, Count: 0}
	condStep := gorkflow.NewConditionalStep(baseStep, condition, defaultOutput)

	wf, err := gorkflow.NewWorkflow("conditional_false", "Conditional False").
		ThenStep(condStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)

	outputBytes, err := wfStore.LoadStepOutput(context.Background(), runID, "conditional")
	require.NoError(t, err)

	var output DiscoverOutput
	json.Unmarshal(outputBytes, &output)
	assert.Equal(t, "Default", output.Companies[0])
	assert.Equal(t, 0, output.Count)
}

func TestEngine_ConditionalStep_BasedOnState(t *testing.T) {
	engine, _ := createTestEngine(t)

	setupStep := gorkflow.NewStep("setup", "Setup Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			ctx.State.Set("should_process", input.Limit > 5)
			return DiscoverOutput{Companies: []string{"Setup"}, Count: 1}, nil
		},
	)

	baseStep := gorkflow.NewStep("process", "Process Step",
		func(ctx *gorkflow.StepContext, input EnrichInput) (EnrichOutput, error) {
			return EnrichOutput{Enriched: map[string]any{"processed": true}}, nil
		},
	)

	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		var shouldProcess bool
		err := ctx.State.Get("should_process", &shouldProcess)
		if err != nil {
			return false, err
		}
		return shouldProcess, nil
	}

	condStep := gorkflow.NewConditionalStep(baseStep, condition, nil)

	wf, err := gorkflow.NewWorkflow("conditional_state", "Conditional State").
		ThenStep(setupStep).
		ThenStep(condStep).
		Build()
	require.NoError(t, err)

	// Test with Limit > 5 (should execute)
	runID1, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run1 := waitForCompletion(t, engine, runID1, 10*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run1.Status)

	// Test with Limit <= 5 (should skip)
	runID2, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 3})
	require.NoError(t, err)

	run2 := waitForCompletion(t, engine, runID2, 10*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run2.Status)
}

func TestEngine_ConditionalStep_BasedOnPreviousOutput(t *testing.T) {
	engine, _ := createTestEngine(t)

	discoverStep := gorkflow.NewStep("discover", "Discover",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			companies := []string{}
			if input.Limit > 0 {
				companies = []string{"CompanyA", "CompanyB"}
			}
			return DiscoverOutput{Companies: companies, Count: len(companies)}, nil
		},
	)

	enrichStep := gorkflow.NewStep("enrich", "Enrich",
		func(ctx *gorkflow.StepContext, input EnrichInput) (EnrichOutput, error) {
			return EnrichOutput{Enriched: map[string]any{"enriched": true}}, nil
		},
	)

	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		discoverOutput, err := gorkflow.GetOutput[DiscoverOutput](ctx, "discover")
		if err != nil {
			return false, err
		}
		return discoverOutput.Count > 0, nil
	}

	condEnrich := gorkflow.NewConditionalStep(enrichStep, condition, nil)

	wf, err := gorkflow.NewWorkflow("conditional_default", "Conditional Default").
		ThenStep(discoverStep).
		ThenStep(condEnrich).
		Build()
	require.NoError(t, err)

	// Test with companies found (should enrich)
	runID1, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)
	run1 := waitForCompletion(t, engine, runID1, 10*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run1.Status)

	// Test with no companies (should skip enrich)
	runID2, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 0})
	require.NoError(t, err)
	run2 := waitForCompletion(t, engine, runID2, 10*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run2.Status)
}

func TestEngine_ConditionalStep_ConditionError(t *testing.T) {
	engine, _ := createTestEngine(t)

	baseStep := gorkflow.NewStep("conditional", "Conditional Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			return DiscoverOutput{Companies: []string{"Test"}, Count: 1}, nil
		},
	)

	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		return false, errors.New("condition evaluation failed")
	}

	condStep := gorkflow.NewConditionalStep(baseStep, condition, nil)

	wf, err := gorkflow.NewWorkflow("conditional_true", "Conditional True").
		ThenStep(condStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	assert.Equal(t, gorkflow.RunStatusFailed, run.Status)
}

func TestEngine_MultipleConditionalSteps(t *testing.T) {
	engine, _ := createTestEngine(t)

	step1 := gorkflow.NewStep("step1", "Step 1",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			ctx.State.Set("value", input.Limit)
			return DiscoverOutput{Companies: []string{"Step1"}, Count: 1}, nil
		},
	)

	cond2Step := gorkflow.NewStep("step2", "Step 2",
		func(ctx *gorkflow.StepContext, input EnrichInput) (EnrichOutput, error) {
			return EnrichOutput{Enriched: map[string]any{"step2": true}}, nil
		},
	)
	cond2 := func(ctx *gorkflow.StepContext) (bool, error) {
		var value int
		ctx.State.Get("value", &value)
		return value > 5, nil
	}
	condStep2 := gorkflow.NewConditionalStep(cond2Step, cond2, nil)

	cond3Step := gorkflow.NewStep("step3", "Step 3",
		func(ctx *gorkflow.StepContext, input FilterInput) (FilterOutput, error) {
			return FilterOutput{Filtered: []string{"Step3"}}, nil
		},
	)
	cond3 := func(ctx *gorkflow.StepContext) (bool, error) {
		var value int
		ctx.State.Get("value", &value)
		return value <= 5, nil
	}
	condStep3 := gorkflow.NewConditionalStep(cond3Step, cond3, nil)

	wf, err := gorkflow.NewWorkflow("complex_conditional", "Complex Conditional").
		ThenStep(step1).
		ThenStep(condStep2).
		ThenStep(condStep3).
		Build()
	require.NoError(t, err)

	// Test with value > 5 (should execute step2)
	runID1, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)
	run1 := waitForCompletion(t, engine, runID1, 10*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run1.Status)

	// Test with value <= 5 (should execute step3)
	runID2, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 3})
	require.NoError(t, err)
	run2 := waitForCompletion(t, engine, runID2, 10*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run2.Status)
}

func TestEngine_ConditionalStep_WithRetry(t *testing.T) {
	engine, _ := createTestEngine(t)

	attemptCount := int32(0)

	baseStep := gorkflow.NewStep("retry_cond", "Retry Conditional",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			count := atomic.AddInt32(&attemptCount, 1)
			if count < 2 {
				return DiscoverOutput{}, errors.New("retry me")
			}
			return DiscoverOutput{Companies: []string{"Success"}, Count: 1}, nil
		},
		gorkflow.WithRetries(2),
	)

	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		return true, nil
	}

	condStep := gorkflow.NewConditionalStep(baseStep, condition, nil)

	wf, err := gorkflow.NewWorkflow("cond_retry", "Conditional Retry").
		ThenStep(condStep).
		Build()
	require.NoError(t, err)

	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	run := waitForCompletion(t, engine, runID, 10*time.Second)

	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attemptCount))
}
