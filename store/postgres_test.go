package store_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPostgresStore creates a PostgresStore for testing.
// Tests are skipped when GORKFLOW_TEST_POSTGRES_DSN is not set.
//
// Example:
//
//	GORKFLOW_TEST_POSTGRES_DSN="postgres://user:pass@localhost:5432/gorkflow_test?sslmode=disable" go test ./store/...
func newTestPostgresStore(t *testing.T) *store.PostgresStore {
	t.Helper()
	dsn := os.Getenv("GORKFLOW_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set GORKFLOW_TEST_POSTGRES_DSN to run PostgreSQL store tests")
	}

	// Truncate all tables before each test for isolation.
	truncatePostgresTables(t, dsn)

	s, err := store.NewPostgresStore(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func truncatePostgresTables(t *testing.T, dsn string) {
	t.Helper()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	_, err = pool.Exec(ctx, `
		TRUNCATE TABLE workflow_state, step_outputs, step_executions, workflow_runs CASCADE
	`)
	require.NoError(t, err)
}

func TestPostgres_CreateAndGetRun(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	run := &gorkflow.WorkflowRun{
		RunID:      "pg-run-1",
		WorkflowID: "wf-1",
		Status:     gorkflow.RunStatusPending,
		CreatedAt:  time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt:  time.Now().UTC().Truncate(time.Millisecond),
	}
	require.NoError(t, s.CreateRun(ctx, run))

	got, err := s.GetRun(ctx, "pg-run-1")
	require.NoError(t, err)
	assert.Equal(t, run.RunID, got.RunID)
	assert.Equal(t, run.WorkflowID, got.WorkflowID)
	assert.Equal(t, gorkflow.RunStatusPending, got.Status)
}

func TestPostgres_GetRun_NotFound(t *testing.T) {
	s := newTestPostgresStore(t)
	_, err := s.GetRun(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, gorkflow.ErrRunNotFound)
}

func TestPostgres_UpdateRun_StatusTransitions(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	run := &gorkflow.WorkflowRun{
		RunID:      "pg-run-transitions",
		WorkflowID: "wf-1",
		Status:     gorkflow.RunStatusPending,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	require.NoError(t, s.CreateRun(ctx, run))

	for _, status := range []gorkflow.RunStatus{
		gorkflow.RunStatusRunning,
		gorkflow.RunStatusCompleted,
	} {
		run.Status = status
		run.UpdatedAt = time.Now().UTC()
		require.NoError(t, s.UpdateRun(ctx, run))

		got, err := s.GetRun(ctx, run.RunID)
		require.NoError(t, err)
		assert.Equal(t, status, got.Status)
	}
}

func TestPostgres_ListRuns_WithFilter(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	for i, wfID := range []string{"wf-a", "wf-a", "wf-b"} {
		run := &gorkflow.WorkflowRun{
			RunID:      fmt.Sprintf("pg-list-run-%d", i),
			WorkflowID: wfID,
			Status:     gorkflow.RunStatusCompleted,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		require.NoError(t, s.CreateRun(ctx, run))
	}

	runs, err := s.ListRuns(ctx, gorkflow.RunFilter{WorkflowID: "wf-a"})
	require.NoError(t, err)
	assert.Len(t, runs, 2)
}

func TestPostgres_StepExecution_FullLifecycle(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	run := &gorkflow.WorkflowRun{
		RunID: "pg-step-run", WorkflowID: "wf-1",
		Status: gorkflow.RunStatusRunning, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.CreateRun(ctx, run))

	exec := &gorkflow.StepExecution{
		RunID: "pg-step-run", StepID: "step-1",
		ExecutionIndex: 0, Status: gorkflow.StepStatusPending,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.CreateStepExecution(ctx, exec))

	now := time.Now().UTC()
	exec.Status = gorkflow.StepStatusRunning
	exec.StartedAt = &now
	require.NoError(t, s.UpdateStepExecution(ctx, exec))

	got, err := s.GetStepExecution(ctx, "pg-step-run", "step-1")
	require.NoError(t, err)
	assert.Equal(t, gorkflow.StepStatusRunning, got.Status)
}

func TestPostgres_StepOutputs_SaveAndLoad(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	run := &gorkflow.WorkflowRun{
		RunID: "pg-output-run", WorkflowID: "wf-1",
		Status: gorkflow.RunStatusRunning, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.CreateRun(ctx, run))

	payload := []byte(`{"result":"ok"}`)
	require.NoError(t, s.SaveStepOutput(ctx, "pg-output-run", "step-1", payload))

	got, err := s.LoadStepOutput(ctx, "pg-output-run", "step-1")
	require.NoError(t, err)
	assert.Equal(t, payload, got)

	// Upsert overwrites cleanly.
	updated := []byte(`{"result":"updated"}`)
	require.NoError(t, s.SaveStepOutput(ctx, "pg-output-run", "step-1", updated))
	got, err = s.LoadStepOutput(ctx, "pg-output-run", "step-1")
	require.NoError(t, err)
	assert.Equal(t, updated, got)
}

func TestPostgres_LoadStepOutput_NotFound(t *testing.T) {
	s := newTestPostgresStore(t)
	_, err := s.LoadStepOutput(context.Background(), "no-run", "no-step")
	assert.ErrorIs(t, err, gorkflow.ErrStepOutputNotFound)
}

func TestPostgres_State_SetGetDelete(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	run := &gorkflow.WorkflowRun{
		RunID: "pg-state-run", WorkflowID: "wf-1",
		Status: gorkflow.RunStatusRunning, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.CreateRun(ctx, run))

	require.NoError(t, s.SaveState(ctx, "pg-state-run", "key1", []byte(`"value1"`)))

	got, err := s.LoadState(ctx, "pg-state-run", "key1")
	require.NoError(t, err)
	assert.Equal(t, []byte(`"value1"`), got)

	require.NoError(t, s.DeleteState(ctx, "pg-state-run", "key1"))
	_, err = s.LoadState(ctx, "pg-state-run", "key1")
	assert.ErrorIs(t, err, gorkflow.ErrStateNotFound)
}

func TestPostgres_State_GetAll(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	run := &gorkflow.WorkflowRun{
		RunID: "pg-getall-run", WorkflowID: "wf-1",
		Status: gorkflow.RunStatusRunning, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.CreateRun(ctx, run))

	require.NoError(t, s.SaveState(ctx, "pg-getall-run", "a", []byte(`1`)))
	require.NoError(t, s.SaveState(ctx, "pg-getall-run", "b", []byte(`2`)))

	all, err := s.GetAllState(ctx, "pg-getall-run")
	require.NoError(t, err)
	assert.Len(t, all, 2)
	assert.Equal(t, []byte(`1`), all["a"])
	assert.Equal(t, []byte(`2`), all["b"])
}

func TestPostgres_Schema_Idempotent(t *testing.T) {
	dsn := os.Getenv("GORKFLOW_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set GORKFLOW_TEST_POSTGRES_DSN to run PostgreSQL store tests")
	}
	// Opening a second store against the same DSN re-runs initSchema — must not error.
	s, err := store.NewPostgresStore(dsn)
	require.NoError(t, err)
	s.Close()
}
