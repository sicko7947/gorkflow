package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"

	workflow "github.com/sicko7947/gorkflow"
)

// LibSQLStore implements WorkflowStore for LibSQL/SQLite
type LibSQLStore struct {
	db *sql.DB
}

// NewLibSQLStore creates a new LibSQL store
// url can be a local file path (file:./local.db) or a remote Turso URL (libsql://...)
func NewLibSQLStore(url string) (*LibSQLStore, error) {
	db, err := sql.Open("libsql", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &LibSQLStore{db: db}
	if err := store.Init(context.Background()); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// Init creates the necessary tables
func (s *LibSQLStore) Init(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, GetLibSQLSchema())
	if err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}
	return nil
}

// Close closes the database connection
func (s *LibSQLStore) Close() error {
	return s.db.Close()
}

// --- Workflow Runs ---

func (s *LibSQLStore) CreateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal run: %w", err)
	}

	query := `
		INSERT INTO workflow_runs (run_id, workflow_id, status, created_at, updated_at, resource_id, data)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.ExecContext(ctx, query,
		run.RunID,
		run.WorkflowID,
		string(run.Status),
		run.CreatedAt,
		run.UpdatedAt,
		run.ResourceID,
		string(data),
	)
	if err != nil {
		return fmt.Errorf("failed to create run: %w", err)
	}
	return nil
}

func (s *LibSQLStore) GetRun(ctx context.Context, runID string) (*workflow.WorkflowRun, error) {
	query := `SELECT data FROM workflow_runs WHERE run_id = ?`
	var data []byte
	err := s.db.QueryRowContext(ctx, query, runID).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found: %s", runID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	var run workflow.WorkflowRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal run: %w", err)
	}
	return &run, nil
}

func (s *LibSQLStore) UpdateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal run: %w", err)
	}

	query := `
		UPDATE workflow_runs 
		SET status = ?, updated_at = ?, data = ?
		WHERE run_id = ?
	`
	_, err = s.db.ExecContext(ctx, query,
		string(run.Status),
		run.UpdatedAt,
		string(data),
		run.RunID,
	)
	if err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}
	return nil
}

func (s *LibSQLStore) UpdateRunStatus(ctx context.Context, runID string, status workflow.RunStatus, werr *workflow.WorkflowError) error {
	// First get the current run to update its data
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return err
	}

	run.Status = status
	run.UpdatedAt = time.Now()
	if werr != nil {
		run.Error = werr
	}

	return s.UpdateRun(ctx, run)
}

func (s *LibSQLStore) ListRuns(ctx context.Context, filter workflow.RunFilter) ([]*workflow.WorkflowRun, error) {
	var queryBuilder strings.Builder
	var args []interface{}

	queryBuilder.WriteString("SELECT data FROM workflow_runs WHERE 1=1")

	if filter.WorkflowID != "" {
		queryBuilder.WriteString(" AND workflow_id = ?")
		args = append(args, filter.WorkflowID)
	}
	if filter.Status != nil {
		queryBuilder.WriteString(" AND status = ?")
		args = append(args, string(*filter.Status))
	}
	if filter.ResourceID != "" {
		queryBuilder.WriteString(" AND resource_id = ?")
		args = append(args, filter.ResourceID)
	}

	queryBuilder.WriteString(" ORDER BY created_at DESC")

	if filter.Limit > 0 {
		queryBuilder.WriteString(" LIMIT ?")
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}
	defer rows.Close()

	var runs []*workflow.WorkflowRun
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var run workflow.WorkflowRun
		if err := json.Unmarshal(data, &run); err != nil {
			return nil, err
		}
		runs = append(runs, &run)
	}
	return runs, nil
}

// --- Step Executions ---

func (s *LibSQLStore) CreateStepExecution(ctx context.Context, exec *workflow.StepExecution) error {
	data, err := json.Marshal(exec)
	if err != nil {
		return fmt.Errorf("failed to marshal step execution: %w", err)
	}

	query := `
		INSERT INTO step_executions (run_id, step_id, execution_index, status, created_at, started_at, completed_at, error, data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var errStr sql.NullString
	if exec.Error != nil {
		errStr.String = exec.Error.Error()
		errStr.Valid = true
	}

	_, err = s.db.ExecContext(ctx, query,
		exec.RunID,
		exec.StepID,
		exec.ExecutionIndex,
		string(exec.Status),
		exec.CreatedAt,
		exec.StartedAt,
		exec.CompletedAt,
		errStr,
		string(data),
	)
	if err != nil {
		return fmt.Errorf("failed to create step execution: %w", err)
	}
	return nil
}

func (s *LibSQLStore) GetStepExecution(ctx context.Context, runID, stepID string) (*workflow.StepExecution, error) {
	query := `SELECT data FROM step_executions WHERE run_id = ? AND step_id = ?`
	var data []byte
	err := s.db.QueryRowContext(ctx, query, runID, stepID).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("step execution not found: %s/%s", runID, stepID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get step execution: %w", err)
	}

	var exec workflow.StepExecution
	if err := json.Unmarshal(data, &exec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal step execution: %w", err)
	}
	return &exec, nil
}

func (s *LibSQLStore) UpdateStepExecution(ctx context.Context, exec *workflow.StepExecution) error {
	data, err := json.Marshal(exec)
	if err != nil {
		return fmt.Errorf("failed to marshal step execution: %w", err)
	}

	query := `
		UPDATE step_executions 
		SET status = ?, started_at = ?, completed_at = ?, error = ?, data = ?
		WHERE run_id = ? AND step_id = ?
	`
	var errStr sql.NullString
	if exec.Error != nil {
		errStr.String = exec.Error.Error()
		errStr.Valid = true
	}

	_, err = s.db.ExecContext(ctx, query,
		string(exec.Status),
		exec.StartedAt,
		exec.CompletedAt,
		errStr,
		string(data),
		exec.RunID,
		exec.StepID,
	)
	if err != nil {
		return fmt.Errorf("failed to update step execution: %w", err)
	}
	return nil
}

func (s *LibSQLStore) ListStepExecutions(ctx context.Context, runID string) ([]*workflow.StepExecution, error) {
	query := `SELECT data FROM step_executions WHERE run_id = ? ORDER BY execution_index ASC`
	rows, err := s.db.QueryContext(ctx, query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to list step executions: %w", err)
	}
	defer rows.Close()

	var execs []*workflow.StepExecution
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var exec workflow.StepExecution
		if err := json.Unmarshal(data, &exec); err != nil {
			return nil, err
		}
		execs = append(execs, &exec)
	}

	return execs, nil
}

// --- Step Outputs ---

func (s *LibSQLStore) SaveStepOutput(ctx context.Context, runID, stepID string, output []byte) error {
	query := `
		INSERT INTO step_outputs (run_id, step_id, output_data)
		VALUES (?, ?, ?)
		ON CONFLICT(run_id, step_id) DO UPDATE SET output_data = excluded.output_data
	`
	_, err := s.db.ExecContext(ctx, query, runID, stepID, output)
	if err != nil {
		return fmt.Errorf("failed to save step output: %w", err)
	}
	return nil
}

func (s *LibSQLStore) LoadStepOutput(ctx context.Context, runID, stepID string) ([]byte, error) {
	query := `SELECT output_data FROM step_outputs WHERE run_id = ? AND step_id = ?`
	var data []byte
	err := s.db.QueryRowContext(ctx, query, runID, stepID).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("step output not found: %s/%s", runID, stepID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load step output: %w", err)
	}
	return data, nil
}

// --- Workflow State ---

func (s *LibSQLStore) SaveState(ctx context.Context, runID, key string, value []byte) error {
	query := `
		INSERT INTO workflow_state (run_id, key, value, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(run_id, key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`
	_, err := s.db.ExecContext(ctx, query, runID, key, value)
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}
	return nil
}

func (s *LibSQLStore) LoadState(ctx context.Context, runID, key string) ([]byte, error) {
	query := `SELECT value FROM workflow_state WHERE run_id = ? AND key = ?`
	var value []byte
	err := s.db.QueryRowContext(ctx, query, runID, key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("state not found: %s/%s", runID, key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return value, nil
}

func (s *LibSQLStore) DeleteState(ctx context.Context, runID, key string) error {
	query := `DELETE FROM workflow_state WHERE run_id = ? AND key = ?`
	_, err := s.db.ExecContext(ctx, query, runID, key)
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}
	return nil
}

func (s *LibSQLStore) GetAllState(ctx context.Context, runID string) (map[string][]byte, error) {
	query := `SELECT key, value FROM workflow_state WHERE run_id = ?`
	rows, err := s.db.QueryContext(ctx, query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all state: %w", err)
	}
	defer rows.Close()

	state := make(map[string][]byte)
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		state[key] = value
	}
	return state, nil
}

func (s *LibSQLStore) CountRunsByStatus(ctx context.Context, resourceID string, status workflow.RunStatus) (int, error) {
	query := `SELECT COUNT(*) FROM workflow_runs WHERE resource_id = ? AND status = ?`
	var count int
	err := s.db.QueryRowContext(ctx, query, resourceID, string(status)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count runs: %w", err)
	}
	return count, nil
}
