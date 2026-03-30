package store_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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
	// Create the store first so initSchema runs and tables exist before truncation.
	s, err := store.NewPostgresStore(dsn)
	require.NoError(t, err)
	truncatePostgresTables(t, dsn)
	t.Cleanup(func() { s.Close() })
	return s
}

// truncatePostgresTables clears all tables using a single connection (not a pool).
func truncatePostgresTables(t *testing.T, dsn string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	defer conn.Close(ctx)

	// workflow_state, step_outputs, step_executions all FK-reference workflow_runs,
	// so truncating in dependency order (or using RESTART IDENTITY CASCADE) is safe.
	_, err = conn.Exec(ctx, `
		TRUNCATE TABLE workflow_state, step_outputs, step_executions, workflow_runs
		RESTART IDENTITY
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

func TestPostgres_UpdateRun_NotFound(t *testing.T) {
	s := newTestPostgresStore(t)
	run := &gorkflow.WorkflowRun{
		RunID: "ghost-run", WorkflowID: "wf-1",
		Status: gorkflow.RunStatusRunning, UpdatedAt: time.Now().UTC(),
	}
	err := s.UpdateRun(context.Background(), run)
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

func TestPostgres_ListRuns_WithWorkflowIDFilter(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	for i, wfID := range []string{"wf-a", "wf-a", "wf-b"} {
		run := &gorkflow.WorkflowRun{
			RunID:      fmt.Sprintf("pg-list-wf-%d", i),
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

func TestPostgres_ListRuns_WithStatusFilter(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	statuses := []gorkflow.RunStatus{
		gorkflow.RunStatusCompleted,
		gorkflow.RunStatusFailed,
		gorkflow.RunStatusCompleted,
	}
	for i, st := range statuses {
		st := st
		run := &gorkflow.WorkflowRun{
			RunID: fmt.Sprintf("pg-list-status-%d", i), WorkflowID: "wf-1",
			Status: st, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, s.CreateRun(ctx, run))
	}

	runs, err := s.ListRuns(ctx, gorkflow.RunFilter{Status: statusPtr(gorkflow.RunStatusCompleted)})
	require.NoError(t, err)
	assert.Len(t, runs, 2)
}

func TestPostgres_ListRuns_WithLimit(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		run := &gorkflow.WorkflowRun{
			RunID: fmt.Sprintf("pg-list-limit-%d", i), WorkflowID: "wf-1",
			Status: gorkflow.RunStatusCompleted, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, s.CreateRun(ctx, run))
	}

	runs, err := s.ListRuns(ctx, gorkflow.RunFilter{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, runs, 3)
}

func TestPostgres_ListRuns_Empty(t *testing.T) {
	s := newTestPostgresStore(t)
	runs, err := s.ListRuns(context.Background(), gorkflow.RunFilter{WorkflowID: "no-such-wf"})
	require.NoError(t, err)
	assert.Empty(t, runs)
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

	// PENDING → RUNNING
	now := time.Now().UTC()
	exec.Status = gorkflow.StepStatusRunning
	exec.StartedAt = &now
	require.NoError(t, s.UpdateStepExecution(ctx, exec))

	got, err := s.GetStepExecution(ctx, "pg-step-run", "step-1")
	require.NoError(t, err)
	assert.Equal(t, gorkflow.StepStatusRunning, got.Status)

	// RUNNING → COMPLETED
	completedAt := time.Now().UTC()
	exec.Status = gorkflow.StepStatusCompleted
	exec.CompletedAt = &completedAt
	require.NoError(t, s.UpdateStepExecution(ctx, exec))

	got, err = s.GetStepExecution(ctx, "pg-step-run", "step-1")
	require.NoError(t, err)
	assert.Equal(t, gorkflow.StepStatusCompleted, got.Status)
}

func TestPostgres_GetStepExecution_NotFound(t *testing.T) {
	s := newTestPostgresStore(t)
	_, err := s.GetStepExecution(context.Background(), "no-run", "no-step")
	assert.ErrorIs(t, err, gorkflow.ErrStepExecutionNotFound)
}

func TestPostgres_UpdateStepExecution_NotFound(t *testing.T) {
	s := newTestPostgresStore(t)
	exec := &gorkflow.StepExecution{
		RunID: "ghost-run", StepID: "ghost-step",
		Status: gorkflow.StepStatusRunning,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	err := s.UpdateStepExecution(context.Background(), exec)
	assert.ErrorIs(t, err, gorkflow.ErrStepExecutionNotFound)
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

func TestPostgres_StepOutputs_Binary(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	run := &gorkflow.WorkflowRun{
		RunID: "pg-binary-run", WorkflowID: "wf-1",
		Status: gorkflow.RunStatusRunning, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.CreateRun(ctx, run))

	// Verify BYTEA handles arbitrary binary data including null bytes.
	binary := []byte("binary data \x00\x01\x02\xff\xfe")
	require.NoError(t, s.SaveStepOutput(ctx, "pg-binary-run", "step-bin", binary))

	got, err := s.LoadStepOutput(ctx, "pg-binary-run", "step-bin")
	require.NoError(t, err)
	assert.Equal(t, binary, got)
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

	// Upsert
	require.NoError(t, s.SaveState(ctx, "pg-state-run", "key1", []byte(`"updated"`)))
	got, err = s.LoadState(ctx, "pg-state-run", "key1")
	require.NoError(t, err)
	assert.Equal(t, []byte(`"updated"`), got)

	require.NoError(t, s.DeleteState(ctx, "pg-state-run", "key1"))
	_, err = s.LoadState(ctx, "pg-state-run", "key1")
	assert.ErrorIs(t, err, gorkflow.ErrStateNotFound)
}

func TestPostgres_LoadState_NotFound(t *testing.T) {
	s := newTestPostgresStore(t)
	_, err := s.LoadState(context.Background(), "no-run", "no-key")
	assert.ErrorIs(t, err, gorkflow.ErrStateNotFound)
}

func TestPostgres_State_Binary(t *testing.T) {
	s := newTestPostgresStore(t)
	ctx := context.Background()

	run := &gorkflow.WorkflowRun{
		RunID: "pg-state-bin-run", WorkflowID: "wf-1",
		Status: gorkflow.RunStatusRunning, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, s.CreateRun(ctx, run))

	binary := []byte("state binary \x00\x01\x02\xff")
	require.NoError(t, s.SaveState(ctx, "pg-state-bin-run", "bin-key", binary))

	got, err := s.LoadState(ctx, "pg-state-bin-run", "bin-key")
	require.NoError(t, err)
	assert.Equal(t, binary, got)
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
	// Opening a second store re-runs initSchema — must not error.
	s, err := store.NewPostgresStore(dsn)
	require.NoError(t, err)
	s.Close()
}

// statusPtr is a helper to take the address of a RunStatus value.
func statusPtr(s gorkflow.RunStatus) *gorkflow.RunStatus { return &s }
