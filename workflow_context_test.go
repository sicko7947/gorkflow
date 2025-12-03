package gorkflow_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/store"
	"github.com/stretchr/testify/assert"
)

type TestContext struct {
	UserID    string
	RequestID string
}

func TestWorkflowWithContext(t *testing.T) {
	// Define a custom context
	customCtx := TestContext{
		UserID:    "user-123",
		RequestID: "req-abc",
	}

	// Define a step that checks the context
	checkContextStep := gorkflow.NewStep(
		"check-context",
		"Check Context",
		func(ctx *gorkflow.StepContext, input string) (string, error) {
			// Retrieve context
			userCtx, err := gorkflow.GetContext[TestContext](ctx)
			if err != nil {
				return "", err
			}

			// Verify context values
			if userCtx.UserID != "user-123" {
				ctx.Logger.Error().Msgf("expected UserID user-123, got %s", userCtx.UserID)
				return "", fmt.Errorf("context mismatch")
			}
			if userCtx.RequestID != "req-abc" {
				ctx.Logger.Error().Msgf("expected RequestID req-abc, got %s", userCtx.RequestID)
				return "", fmt.Errorf("context mismatch")
			}

			return "context verified", nil
		},
	)

	// Build workflow with context
	wf := gorkflow.NewWorkflow("context-test-wf", "Context Test Workflow").
		WithContext(customCtx).
		ThenStep(checkContextStep).
		MustBuild()

	// Run workflow
	eng := engine.NewEngine(store.NewMemoryStore())
	runID, err := eng.StartWorkflow(context.Background(), wf, "start", gorkflow.WithSynchronousExecution())
	assert.NoError(t, err)

	// Check result
	run, err := eng.GetRun(context.Background(), runID)
	assert.NoError(t, err)
	assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)
}
