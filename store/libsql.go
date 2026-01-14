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

// LibSQLStoreOptions configures the LibSQL store
type LibSQLStoreOptions struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	CacheSize       int // in KB, negative values mean number of pages
}

// DefaultLibSQLStoreOptions returns sensible defaults
func DefaultLibSQLStoreOptions() LibSQLStoreOptions {
	return LibSQLStoreOptions{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		CacheSize:       -2000, // 2000 pages (~8MB with 4KB pages)
	}
}

// LibSQLStore implements WorkflowStore for LibSQL/SQLite
type LibSQLStore struct {
	db *sql.DB
}

// NewLibSQLStore creates a new LibSQL store with default options
// url can be a local file path (file:./local.db) or a remote Turso URL (libsql://...)
func NewLibSQLStore(url string) (*LibSQLStore, error) {
	return NewLibSQLStoreWithOptions(url, DefaultLibSQLStoreOptions())
}

// NewLibSQLStoreWithOptions creates a new LibSQL store with custom options
func NewLibSQLStoreWithOptions(url string, opts LibSQLStoreOptions) (*LibSQLStore, error) {
	db, err := sql.Open("libsql", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(opts.ConnMaxLifetime)

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Detect if this is a remote Turso database (libsql:// or https://)
	isRemote := strings.HasPrefix(url, "libsql://") || strings.HasPrefix(url, "https://")

	store := &LibSQLStore{db: db}
	if err := store.init(ctx, opts, isRemote); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// init creates the necessary tables and sets performance PRAGMAs
func (s *LibSQLStore) init(ctx context.Context, opts LibSQLStoreOptions, isRemote bool) error {
	// PRAGMAs that work on both local and remote
	universalPragmas := []string{
		"PRAGMA foreign_keys = ON",
	}

	// PRAGMAs only for local SQLite (not supported on remote Turso)
	localOnlyPragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
		fmt.Sprintf("PRAGMA cache_size = %d", opts.CacheSize),
	}

	// Apply universal PRAGMAs (fail if these don't work)
	for _, pragma := range universalPragmas {
		if _, err := s.db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("failed to set pragma %q: %w", pragma, err)
		}
	}

	// Apply local-only PRAGMAs (skip for remote, these are managed by Turso)
	if !isRemote {
		for _, pragma := range localOnlyPragmas {
			if _, err := s.db.ExecContext(ctx, pragma); err != nil {
				return fmt.Errorf("failed to set pragma %q: %w", pragma, err)
			}
		}
	}

	_, err := s.db.ExecContext(ctx, GetLibSQLSchema())
	if err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}
	return nil
}

// Init is kept for backward compatibility but delegates to init
// Assumes local database (sets PRAGMAs)
func (s *LibSQLStore) Init(ctx context.Context) error {
	return s.init(ctx, DefaultLibSQLStoreOptions(), false)
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
		return nil, workflow.ErrRunNotFound
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
	now := time.Now()

	if werr != nil {
		werrJSON, err := json.Marshal(werr)
		if err != nil {
			return fmt.Errorf("failed to marshal error: %w", err)
		}

		query := `
			UPDATE workflow_runs 
			SET status = ?, 
			    updated_at = ?, 
			    data = json_set(data, '$.status', ?, '$.updatedAt', ?, '$.error', json(?))
			WHERE run_id = ?
		`
		_, err = s.db.ExecContext(ctx, query,
			string(status),
			now,
			string(status),
			now.Format(time.RFC3339Nano),
			string(werrJSON),
			runID,
		)
		if err != nil {
			return fmt.Errorf("failed to update run status with error: %w", err)
		}
	} else {
		query := `
			UPDATE workflow_runs 
			SET status = ?, 
			    updated_at = ?, 
			    data = json_set(data, '$.status', ?, '$.updatedAt', ?)
			WHERE run_id = ?
		`
		_, err := s.db.ExecContext(ctx, query,
			string(status),
			now,
			string(status),
			now.Format(time.RFC3339Nano),
			runID,
		)
		if err != nil {
			return fmt.Errorf("failed to update run status: %w", err)
		}
	}

	return nil
}

func (s *LibSQLStore) ListRuns(ctx context.Context, filter workflow.RunFilter) ([]*workflow.WorkflowRun, error) {
	var queryBuilder strings.Builder
	var args []any

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
			return nil, fmt.Errorf("failed to scan run: %w", err)
		}
		var run workflow.WorkflowRun
		if err := json.Unmarshal(data, &run); err != nil {
			return nil, fmt.Errorf("failed to unmarshal run: %w", err)
		}
		runs = append(runs, &run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate runs: %w", err)
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
		return nil, workflow.ErrStepExecutionNotFound
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
	var errStr sql.NullString
	if exec.Error != nil {
		errStr.String = exec.Error.Error()
		errStr.Valid = true
	}

	// Use json_set for atomic partial update of status, timing, and error fields
	query := `
		UPDATE step_executions 
		SET status = ?, 
		    started_at = ?, 
		    completed_at = ?, 
		    error = ?,
		    data = json_set(data, 
		        '$.status', ?, 
		        '$.startedAt', ?, 
		        '$.completedAt', ?, 
		        '$.durationMs', ?,
		        '$.attempt', ?,
		        '$.error', json_set('{}', '$.message', ?))
		WHERE run_id = ? AND step_id = ?
	`

	// Prepare timestamp formats for JSON
	var startedAtJSON, completedAtJSON any
	if exec.StartedAt != nil {
		startedAtJSON = exec.StartedAt.Format(time.RFC3339Nano)
	}
	if exec.CompletedAt != nil {
		completedAtJSON = exec.CompletedAt.Format(time.RFC3339Nano)
	}

	var errMsg any
	if exec.Error != nil {
		errMsg = exec.Error.Error()
	}

	_, err := s.db.ExecContext(ctx, query,
		string(exec.Status),
		exec.StartedAt,
		exec.CompletedAt,
		errStr,
		string(exec.Status),
		startedAtJSON,
		completedAtJSON,
		exec.DurationMs,
		exec.Attempt,
		errMsg,
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
			return nil, fmt.Errorf("failed to scan step execution: %w", err)
		}
		var exec workflow.StepExecution
		if err := json.Unmarshal(data, &exec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal step execution: %w", err)
		}
		execs = append(execs, &exec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate step executions: %w", err)
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
		return nil, workflow.ErrStepOutputNotFound
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
		return nil, workflow.ErrStateNotFound
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
			return nil, fmt.Errorf("failed to scan state: %w", err)
		}
		state[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate state: %w", err)
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
