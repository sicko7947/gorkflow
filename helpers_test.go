package gorkflow_test

import (
	"context"
	"testing"

	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestRunContext struct {
	UserID    string
	RequestID string
	Metadata  map[string]string
}

func TestGetRunContext(t *testing.T) {
	// Define a custom context
	customCtx := TestRunContext{
		UserID:    "user-456",
		RequestID: "req-xyz",
		Metadata: map[string]string{
			"source":  "api",
			"version": "v1",
		},
	}

	// Create a simple step
	step := gorkflow.NewStep(
		"test-step",
		"Test Step",
		func(ctx *gorkflow.StepContext, input string) (string, error) {
			return "ok", nil
		},
	)

	// Build workflow with context
	wf := gorkflow.NewWorkflow("context-retrieval-test", "Context Retrieval Test").
		WithContext(customCtx).
		ThenStep(step).
		MustBuild()

	// Run workflow
	eng := engine.NewEngine(store.NewMemoryStore())
	runID, err := eng.StartWorkflow(context.Background(), wf, "start", gorkflow.WithSynchronousExecution())
	require.NoError(t, err)

	// Get run and verify context is present
	run, err := eng.GetRun(context.Background(), runID)
	require.NoError(t, err)
	assert.NotNil(t, run.Context)

	// Deserialize context
	retrievedCtx, err := gorkflow.GetRunContext[TestRunContext](run)
	require.NoError(t, err)

	// Verify context values
	assert.Equal(t, customCtx.UserID, retrievedCtx.UserID)
	assert.Equal(t, customCtx.RequestID, retrievedCtx.RequestID)
	assert.Equal(t, customCtx.Metadata, retrievedCtx.Metadata)
}

func TestGetRunContext_NoContext(t *testing.T) {
	// Create a workflow without context
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

	// Run workflow
	eng := engine.NewEngine(store.NewMemoryStore())
	runID, err := eng.StartWorkflow(context.Background(), wf, "start", gorkflow.WithSynchronousExecution())
	require.NoError(t, err)

	// Get run
	run, err := eng.GetRun(context.Background(), runID)
	require.NoError(t, err)

	// Try to get context - should fail
	_, err = gorkflow.GetRunContext[TestRunContext](run)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no context")
}
