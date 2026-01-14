package gorkflow_test

import (
	"context"
	"fmt"
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
