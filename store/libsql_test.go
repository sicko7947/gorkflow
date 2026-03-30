package store

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	workflow "github.com/sicko7947/gorkflow"
)

func newTestLibSQLStore(t *testing.T) *LibSQLStore {
	t.Helper()
	dbFile := "./test_gorkflow.db"
	t.Cleanup(func() {
		os.Remove(dbFile)
	})
	s, err := NewLibSQLStore("file:" + dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})
	return s
}

func TestLibSQL_CreateAndGetRun(t *testing.T) {
	s := newTestLibSQLStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	run := &workflow.WorkflowRun{
		RunID:      uuid.New().String(),
		WorkflowID: "test-wf",
		ResourceID: "resource-1",
		Status:     workflow.RunStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err := s.CreateRun(ctx, run)
	require.NoError(t, err)

	fetched, err := s.GetRun(ctx, run.RunID)
	require.NoError(t, err)
	assert.Equal(t, run.RunID, fetched.RunID)
	assert.Equal(t, run.WorkflowID, fetched.WorkflowID)
	assert.Equal(t, run.ResourceID, fetched.ResourceID)
	assert.Equal(t, run.Status, fetched.Status)
}

func TestLibSQL_UpdateRun(t *testing.T) {
	s := newTestLibSQLStore(t)
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      uuid.New().String(),
		WorkflowID: "test-wf",
		Status:     workflow.RunStatusRunning,
		Progress:   0.0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, s.CreateRun(ctx, run))

	run.Progress = 0.75
	run.Output = []byte(`{"result":"ok"}`)
	run.Status = workflow.RunStatusCompleted
	require.NoError(t, s.UpdateRun(ctx, run))

	fetched, err := s.GetRun(ctx, run.RunID)
	require.NoError(t, err)
	assert.Equal(t, workflow.RunStatusCompleted, fetched.Status)
	assert.Equal(t, 0.75, fetched.Progress)
	assert.JSONEq(t, `{"result":"ok"}`, string(fetched.Output))
}

func TestLibSQL_ListRuns_WithFilter(t *testing.T) {
	s := newTestLibSQLStore(t)
	ctx := context.Background()

	statuses := []workflow.RunStatus{
		workflow.RunStatusPending,
		workflow.RunStatusRunning,
		workflow.RunStatusCompleted,
	}

	for i, status := range statuses {
		run := &workflow.WorkflowRun{
			RunID:      uuid.New().String(),
			WorkflowID: fmt.Sprintf("wf-%d", i),
			Status:     status,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		require.NoError(t, s.CreateRun(ctx, run))
	}

	// Filter by status
	pendingStatus := workflow.RunStatusPending
	runs, err := s.ListRuns(ctx, workflow.RunFilter{Status: &pendingStatus})
	require.NoError(t, err)
	assert.Len(t, runs, 1)
	assert.Equal(t, workflow.RunStatusPending, runs[0].Status)

	// No filter — all 3
	all, err := s.ListRuns(ctx, workflow.RunFilter{})
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestLibSQL_StepExecution_FullLifecycle(t *testing.T) {
	s := newTestLibSQLStore(t)
	ctx := context.Background()

	runID := uuid.New().String()
	stepID := "lifecycle-step"

	exec := &workflow.StepExecution{
		RunID:     runID,
		StepID:    stepID,
		Status:    workflow.StepStatusPending,
		CreatedAt: time.Now(),
	}
	require.NoError(t, s.CreateStepExecution(ctx, exec))

	// Update to RUNNING
	startedAt := time.Now()
	exec.Status = workflow.StepStatusRunning
	exec.StartedAt = &startedAt
	require.NoError(t, s.UpdateStepExecution(ctx, exec))

	fetched, err := s.GetStepExecution(ctx, runID, stepID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StepStatusRunning, fetched.Status)
	assert.NotNil(t, fetched.StartedAt)

	// Update to COMPLETED
	completedAt := time.Now()
	exec.Status = workflow.StepStatusCompleted
	exec.CompletedAt = &completedAt
	require.NoError(t, s.UpdateStepExecution(ctx, exec))

	fetched, err = s.GetStepExecution(ctx, runID, stepID)
	require.NoError(t, err)
	assert.Equal(t, workflow.StepStatusCompleted, fetched.Status)
	assert.NotNil(t, fetched.CompletedAt)
}

func TestLibSQL_StepOutputs_SaveAndLoad(t *testing.T) {
	s := newTestLibSQLStore(t)
	ctx := context.Background()

	runID := uuid.New().String()
	stepID := "output-step"
	output := []byte("hello world binary data \x00\x01\x02")

	require.NoError(t, s.SaveStepOutput(ctx, runID, stepID, output))

	loaded, err := s.LoadStepOutput(ctx, runID, stepID)
	require.NoError(t, err)
	assert.Equal(t, output, loaded)
}

func TestLibSQL_State_SetGetDeleteHas(t *testing.T) {
	s := newTestLibSQLStore(t)
	ctx := context.Background()

	runID := uuid.New().String()
	key := "test-key"
	value := []byte(`{"foo":"bar"}`)

	// Set
	require.NoError(t, s.SaveState(ctx, runID, key, value))

	// Get
	loaded, err := s.LoadState(ctx, runID, key)
	require.NoError(t, err)
	assert.Equal(t, value, loaded)

	// Verify key exists via GetAllState
	all, err := s.GetAllState(ctx, runID)
	require.NoError(t, err)
	assert.Contains(t, all, key)

	// Delete
	require.NoError(t, s.DeleteState(ctx, runID, key))

	// Verify gone
	_, err = s.LoadState(ctx, runID, key)
	assert.Error(t, err)
}

func TestLibSQL_Schema_Idempotent(t *testing.T) {
	dbFile := "./test_gorkflow_idempotent.db"
	t.Cleanup(func() {
		os.Remove(dbFile)
	})

	// First initialization
	s1, err := NewLibSQLStore("file:" + dbFile)
	require.NoError(t, err)
	s1.Close()

	// Second initialization on same DB — should not error
	s2, err := NewLibSQLStore("file:" + dbFile)
	require.NoError(t, err)
	s2.Close()
}

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
