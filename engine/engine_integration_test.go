package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types
type DiscoverInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type DiscoverOutput struct {
	Companies []string `json:"companies"`
	Count     int      `json:"count"`
}

type EnrichInput struct {
	Companies []string `json:"companies"`
}

type EnrichOutput struct {
	Enriched map[string]interface{} `json:"enriched"`
}

type FilterInput struct {
	Data map[string]interface{} `json:"data"`
}

type FilterOutput struct {
	Filtered []string `json:"filtered"`
}

// Test step handlers
func discoverCompanies(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
	ctx.Logger.Info().
		Str("query", input.Query).
		Int("limit", input.Limit).
		Msg("Discovering companies")

	companies := []string{"CompanyA", "CompanyB", "CompanyC"}
	return DiscoverOutput{
		Companies: companies,
		Count:     len(companies),
	}, nil
}

func enrichCompanies(ctx *gorkflow.StepContext, input EnrichInput) (EnrichOutput, error) {
	// Access previous step output
	discoverResult, err := gorkflow.GetOutput[DiscoverOutput](ctx, "discover")
	if err != nil {
		return EnrichOutput{}, err
	}

	ctx.Logger.Info().
		Int("companies_count", len(discoverResult.Companies)).
		Msg("Enriching companies")

	enriched := make(map[string]interface{})
	for _, company := range discoverResult.Companies {
		enriched[company] = map[string]interface{}{
			"name":     company,
			"size":     "medium",
			"industry": "tech",
		}
	}

	return EnrichOutput{Enriched: enriched}, nil
}

func filterCompanies(ctx *gorkflow.StepContext, input FilterInput) (FilterOutput, error) {
	enrichResult, err := gorkflow.GetOutput[EnrichOutput](ctx, "enrich")
	if err != nil {
		return FilterOutput{}, err
	}

	ctx.Logger.Info().
		Int("companies_count", len(enrichResult.Enriched)).
		Msg("Filtering companies")

	filtered := []string{"CompanyA", "CompanyB"}
	return FilterOutput{Filtered: filtered}, nil
}

// Helper to create engine
func createTestEngine(t *testing.T) (*Engine, gorkflow.WorkflowStore) {
	wfStore := store.NewMemoryStore()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	engine := NewEngine(wfStore,
		WithLogger(logger),
		WithConfig(gorkflow.EngineConfig{
			MaxConcurrentWorkflows: 10,
			DefaultTimeout:         5 * time.Minute,
		}),
	)
	return engine, wfStore
}

// Helper to wait for workflow completion
func waitForCompletion(t *testing.T, engine *Engine, runID string, timeout time.Duration) *gorkflow.WorkflowRun {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for workflow completion")
		case <-ticker.C:
			run, err := engine.GetRun(context.Background(), runID)
			require.NoError(t, err)

			if run.Status.IsTerminal() {
				return run
			}
		}
	}
}

func TestEngine_SimpleSequentialWorkflow(t *testing.T) {
	engine, _ := createTestEngine(t)

	// Build workflow
	discoverStep := gorkflow.NewStep("discover", "Discover Companies", discoverCompanies)
	enrichStep := gorkflow.NewStep("enrich", "Enrich Companies", enrichCompanies)
	filterStep := gorkflow.NewStep("filter", "Filter Companies", filterCompanies)

	wf, err := gorkflow.NewWorkflow("sequential_test", "Sequential Test").
		ThenStep(discoverStep).
		ThenStep(enrichStep).
		ThenStep(filterStep).
		Build()
	require.NoError(t, err)

	// Execute workflow
	input := DiscoverInput{
		Query: "tech companies",
		Limit: 10,
	}

	runID, err := engine.StartWorkflow(context.Background(), wf, input)
	require.NoError(t, err)
	require.NotEmpty(t, runID)

	// Wait for completion
	run := waitForCompletion(t, engine, runID, 10*time.Second)

	// Verify final status
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)
	assert.Equal(t, 1.0, run.Progress)
	assert.NotNil(t, run.CompletedAt)

	// Verify step executions
	steps, err := engine.GetStepExecutions(context.Background(), runID)
	require.NoError(t, err)
	assert.Len(t, steps, 3)

	// All steps should be completed
	for _, step := range steps {
		assert.Equal(t, gorkflow.StepStatusCompleted, step.Status)
		assert.NotNil(t, step.CompletedAt)
	}
}

func TestEngine_WorkflowWithFailure(t *testing.T) {
	engine, _ := createTestEngine(t)

	// Step that always fails
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

	// Execute
	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	// Wait for completion
	run := waitForCompletion(t, engine, runID, 10*time.Second)

	// Should fail
	assert.Equal(t, gorkflow.RunStatusFailed, run.Status)
	assert.NotNil(t, run.Error)
	assert.Contains(t, run.Error.Message, "intentional failure")
}

func TestEngine_WorkflowProgress(t *testing.T) {
	engine, _ := createTestEngine(t)

	// Slow steps to observe progress
	slowStep1 := gorkflow.NewStep("slow1", "Slow Step 1",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			time.Sleep(500 * time.Millisecond)
			return DiscoverOutput{Companies: []string{"A"}, Count: 1}, nil
		},
	)

	slowStep2 := gorkflow.NewStep("slow2", "Slow Step 2",
		func(ctx *gorkflow.StepContext, input EnrichInput) (EnrichOutput, error) {
			time.Sleep(500 * time.Millisecond)
			return EnrichOutput{Enriched: map[string]interface{}{"A": "data"}}, nil
		},
	)

	wf, err := gorkflow.NewWorkflow("progress_test", "Progress Test").
		ThenStep(slowStep1).
		ThenStep(slowStep2).
		Build()
	require.NoError(t, err)

	// Execute
	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	// Check progress over time
	time.Sleep(600 * time.Millisecond) // After first step
	run, _ := engine.GetRun(context.Background(), runID)
	assert.Greater(t, run.Progress, 0.0)
	assert.Less(t, run.Progress, 1.0)

	// Wait for completion
	run = waitForCompletion(t, engine, runID, 10*time.Second)
	assert.Equal(t, 1.0, run.Progress)
}

func TestEngine_StepOutputPassing(t *testing.T) {
	engine, wfStore := createTestEngine(t)

	// Build workflow
	discoverStep := gorkflow.NewStep("discover", "Discover", discoverCompanies)
	enrichStep := gorkflow.NewStep("enrich", "Enrich", enrichCompanies)

	wf, err := gorkflow.NewWorkflow("output_test", "Output Test").
		ThenStep(discoverStep).
		ThenStep(enrichStep).
		Build()
	require.NoError(t, err)

	// Execute
	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	// Wait for completion
	waitForCompletion(t, engine, runID, 10*time.Second)

	// Verify step outputs are stored
	discoverOutputBytes, err := wfStore.LoadStepOutput(context.Background(), runID, "discover")
	require.NoError(t, err)

	var discoverOutput DiscoverOutput
	json.Unmarshal(discoverOutputBytes, &discoverOutput)
	assert.Greater(t, discoverOutput.Count, 0)
	assert.NotEmpty(t, discoverOutput.Companies)

	// Verify enrich step used discover output
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

	// Get step executions
	steps, err := engine.GetStepExecutions(context.Background(), runID)
	require.NoError(t, err)

	assert.Len(t, steps, 2)

	// Verify step metadata
	for _, step := range steps {
		assert.NotEmpty(t, step.StepID)
		assert.NotNil(t, step.StartedAt)
		assert.NotNil(t, step.CompletedAt)
		// Duration can be 0 for very fast steps (< 1ms), so we just check it's non-negative
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

	// Create multiple runs
	runID1, _ := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test1", Limit: 10})
	runID2, _ := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test2", Limit: 10})

	waitForCompletion(t, engine, runID1, 10*time.Second)
	waitForCompletion(t, engine, runID2, 10*time.Second)

	// List runs
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

	// Long-running step
	longStep := gorkflow.NewStep("long", "Long Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			// Simulate long operation
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

	// Start workflow
	runID, err := engine.StartWorkflow(context.Background(), wf, DiscoverInput{Query: "test", Limit: 10})
	require.NoError(t, err)

	// Wait a bit to ensure it starts
	time.Sleep(200 * time.Millisecond)

	// Cancel
	err = engine.Cancel(context.Background(), runID)
	require.NoError(t, err)

	// Wait and verify cancellation
	time.Sleep(500 * time.Millisecond)
	run, _ := engine.GetRun(context.Background(), runID)
	assert.Equal(t, gorkflow.RunStatusCancelled, run.Status)
}

func TestEngine_WorkflowState(t *testing.T) {
	engine, wfStore := createTestEngine(t)

	// Step that uses state
	statefulStep := gorkflow.NewStep("stateful", "Stateful Step",
		func(ctx *gorkflow.StepContext, input DiscoverInput) (DiscoverOutput, error) {
			// Write to state
			ctx.State.Set("timestamp", time.Now().Unix())
			ctx.State.Set("query", input.Query)

			// Read from state
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

	// Verify state was persisted
	allState, err := wfStore.GetAllState(context.Background(), runID)
	require.NoError(t, err)

	assert.Contains(t, allState, "timestamp")
	assert.Contains(t, allState, "query")
}

func TestEngine_ConditionalStep_Skipped_PassThrough(t *testing.T) {
	engine, _ := createTestEngine(t)

	// Step 1: Returns "step1 output"
	step1 := gorkflow.NewStep("step1", "Step 1", func(ctx *gorkflow.StepContext, input any) (string, error) {
		return "step1 output", nil
	})

	// Step 2: Takes string, returns "step2 output"
	step2 := gorkflow.NewStep("step2", "Step 2", func(ctx *gorkflow.StepContext, input string) (string, error) {
		return "step2 output", nil
	})

	// Condition: Always false
	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		return false, nil
	}

	// Build workflow
	wf, err := gorkflow.NewWorkflow("test_wf", "Test Workflow").
		ThenStep(step1).
		ThenStepIf(step2, condition, nil).
		Build()
	require.NoError(t, err)

	// Execute
	runID, err := engine.StartWorkflow(context.Background(), wf, nil)
	require.NoError(t, err)

	// Wait for completion
	run := waitForCompletion(t, engine, runID, 10*time.Second)
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)

	// Get step executions
	steps, err := engine.GetStepExecutions(context.Background(), runID)
	require.NoError(t, err)

	// Find step2 execution
	var step2Exec *gorkflow.StepExecution
	for _, s := range steps {
		if s.StepID == "step2" {
			step2Exec = s
			break
		}
	}
	require.NotNil(t, step2Exec)
	// Since we treat pass-through as a successful completion (providing output),
	// the status is now COMPLETED.
	assert.Equal(t, gorkflow.StepStatusCompleted, step2Exec.Status)

	// Verify output of step2 matches step1 output (pass-through)
	var output string
	err = json.Unmarshal(step2Exec.Output, &output)
	require.NoError(t, err)
	assert.Equal(t, "step1 output", output)
}

func TestEngine_ParallelStepExecution(t *testing.T) {
	engine, _ := createTestEngine(t)

	// Step A: Returns "output-A"
	stepA := gorkflow.NewStep("A", "Step A", func(ctx *gorkflow.StepContext, input any) (string, error) {
		return "output-A", nil
	})

	// Step B: Expects "output-A"
	stepB := gorkflow.NewStep("B", "Step B", func(ctx *gorkflow.StepContext, input string) (string, error) {
		if input != "output-A" {
			return "", fmt.Errorf("step B expected 'output-A', got '%s'", input)
		}
		return "output-B", nil
	})

	// Step C: Expects "output-A"
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
