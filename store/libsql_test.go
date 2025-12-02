package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	workflow "github.com/sicko7947/gorkflow"
)

func TestLibSQLStore(t *testing.T) {
	// Use a temporary file for the database
	dbFile := "test_libsql.db"
	defer os.Remove(dbFile)

	// Initialize store
	store, err := NewLibSQLStore("file:" + dbFile)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	t.Run("WorkflowRun", func(t *testing.T) {
		runID := uuid.New().String()
		run := &workflow.WorkflowRun{
			RunID:      runID,
			WorkflowID: "test-wf",
			Status:     workflow.RunStatusRunning,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			ResourceID: "user-123",
		}

		// Create
		err := store.CreateRun(ctx, run)
		require.NoError(t, err)

		// Get
		fetched, err := store.GetRun(ctx, runID)
		require.NoError(t, err)
		assert.Equal(t, run.RunID, fetched.RunID)
		assert.Equal(t, run.Status, fetched.Status)

		// Update
		run.Status = workflow.RunStatusCompleted
		err = store.UpdateRun(ctx, run)
		require.NoError(t, err)

		fetched, err = store.GetRun(ctx, runID)
		require.NoError(t, err)
		assert.Equal(t, workflow.RunStatusCompleted, fetched.Status)

		// List
		runs, err := store.ListRuns(ctx, workflow.RunFilter{
			WorkflowID: "test-wf",
		})
		require.NoError(t, err)
		assert.Len(t, runs, 1)
		assert.Equal(t, runID, runs[0].RunID)
	})

	t.Run("StepExecution", func(t *testing.T) {
		runID := uuid.New().String()
		stepID := "step-1"
		exec := &workflow.StepExecution{
			RunID:     runID,
			StepID:    stepID,
			Status:    workflow.StepStatusPending,
			CreatedAt: time.Now(),
		}

		// Create
		err := store.CreateStepExecution(ctx, exec)
		require.NoError(t, err)

		// Get
		fetched, err := store.GetStepExecution(ctx, runID, stepID)
		require.NoError(t, err)
		assert.Equal(t, exec.RunID, fetched.RunID)
		assert.Equal(t, exec.StepID, fetched.StepID)

		// Update
		exec.Status = workflow.StepStatusCompleted
		now := time.Now()
		exec.CompletedAt = &now
		err = store.UpdateStepExecution(ctx, exec)
		require.NoError(t, err)

		fetched, err = store.GetStepExecution(ctx, runID, stepID)
		require.NoError(t, err)
		assert.Equal(t, workflow.StepStatusCompleted, fetched.Status)

		// List
		execs, err := store.ListStepExecutions(ctx, runID)
		require.NoError(t, err)
		assert.Len(t, execs, 1)
	})

	t.Run("StepOutput", func(t *testing.T) {
		runID := uuid.New().String()
		stepID := "step-1"
		output := []byte("some output data")

		// Save
		err := store.SaveStepOutput(ctx, runID, stepID, output)
		require.NoError(t, err)

		// Load
		loaded, err := store.LoadStepOutput(ctx, runID, stepID)
		require.NoError(t, err)
		assert.Equal(t, output, loaded)
	})

	t.Run("WorkflowState", func(t *testing.T) {
		runID := uuid.New().String()
		key := "my-key"
		value := []byte("my-value")

		// Save
		err := store.SaveState(ctx, runID, key, value)
		require.NoError(t, err)

		// Load
		loaded, err := store.LoadState(ctx, runID, key)
		require.NoError(t, err)
		assert.Equal(t, value, loaded)

		// Get All
		all, err := store.GetAllState(ctx, runID)
		require.NoError(t, err)
		assert.Len(t, all, 1)
		assert.Equal(t, value, all[key])

		// Delete
		err = store.DeleteState(ctx, runID, key)
		require.NoError(t, err)

		_, err = store.LoadState(ctx, runID, key)
		assert.Error(t, err)
	})
}
