package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	workflow "github.com/sicko7947/gorkflow"
)

// Compile-time interface compliance check.
var _ workflow.WorkflowStore = (*PostgresStore)(nil)

// PostgresStoreOptions configures the PostgreSQL store.
type PostgresStoreOptions struct {
	MaxConns        int32
	MinConns        int32
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultPostgresStoreOptions returns sensible defaults.
func DefaultPostgresStoreOptions() PostgresStoreOptions {
	return PostgresStoreOptions{
		MaxConns:        10,
		MinConns:        2,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
	}
}

// PostgresStore implements WorkflowStore for PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgreSQL store with default options.
// dsn follows the standard PostgreSQL connection string format:
//
//	postgres://user:password@host:5432/dbname?sslmode=disable
func NewPostgresStore(dsn string) (*PostgresStore, error) {
	return NewPostgresStoreWithOptions(dsn, DefaultPostgresStoreOptions())
}

// NewPostgresStoreWithOptions creates a new PostgreSQL store with custom options.
func NewPostgresStoreWithOptions(dsn string, opts PostgresStoreOptions) (*PostgresStore, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	cfg.MaxConns = opts.MaxConns
	cfg.MinConns = opts.MinConns
	cfg.MaxConnLifetime = opts.ConnMaxLifetime
	cfg.MaxConnIdleTime = opts.ConnMaxIdleTime

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &PostgresStore{pool: pool}
	if err := s.initSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// initSchema creates tables and indexes inside a single transaction.
// The entire schema string is passed as one statement so it is never fragmented
// by semicolon-splitting (which would break on comments, string literals, or DO blocks).
func (s *PostgresStore) initSchema(ctx context.Context) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for schema init: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin schema transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	schema := GetPostgresSchema()
	if _, err := tx.Exec(ctx, schema); err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}
	return tx.Commit(ctx)
}

// Close closes the underlying connection pool. Always returns nil;
// present to match the error-returning convention of other stores.
func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

// --- Workflow Runs ---

func (s *PostgresStore) CreateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal run: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO workflow_runs (run_id, workflow_id, status, created_at, updated_at, resource_id, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		run.RunID,
		run.WorkflowID,
		string(run.Status),
		run.CreatedAt,
		run.UpdatedAt,
		run.ResourceID,
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to create run: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetRun(ctx context.Context, runID string) (*workflow.WorkflowRun, error) {
	var data []byte
	err := s.pool.QueryRow(ctx,
		`SELECT data FROM workflow_runs WHERE run_id = $1`, runID,
	).Scan(&data)
	if err == pgx.ErrNoRows {
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

func (s *PostgresStore) UpdateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("failed to marshal run: %w", err)
	}

	tag, err := s.pool.Exec(ctx, `
		UPDATE workflow_runs
		SET status = $1, updated_at = $2, data = $3
		WHERE run_id = $4`,
		string(run.Status),
		run.UpdatedAt,
		data,
		run.RunID,
	)
	if err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return workflow.ErrRunNotFound
	}
	return nil
}

func (s *PostgresStore) ListRuns(ctx context.Context, filter workflow.RunFilter) ([]*workflow.WorkflowRun, error) {
	var sb strings.Builder
	args := make([]any, 0, 4)

	sb.WriteString("SELECT data FROM workflow_runs WHERE 1=1")

	if filter.WorkflowID != "" {
		args = append(args, filter.WorkflowID)
		fmt.Fprintf(&sb, " AND workflow_id = $%d", len(args))
	}
	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		fmt.Fprintf(&sb, " AND status = $%d", len(args))
	}
	if filter.ResourceID != "" {
		args = append(args, filter.ResourceID)
		fmt.Fprintf(&sb, " AND resource_id = $%d", len(args))
	}

	sb.WriteString(" ORDER BY created_at DESC")

	if filter.Limit > 0 {
		args = append(args, filter.Limit)
		fmt.Fprintf(&sb, " LIMIT $%d", len(args))
	}

	rows, err := s.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}
	defer rows.Close()

	runs := make([]*workflow.WorkflowRun, 0, 16)
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

func (s *PostgresStore) CreateStepExecution(ctx context.Context, exec *workflow.StepExecution) error {
	data, err := json.Marshal(exec)
	if err != nil {
		return fmt.Errorf("failed to marshal step execution: %w", err)
	}

	var errMsg *string
	if exec.Error != nil {
		msg := exec.Error.Error()
		errMsg = &msg
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO step_executions
			(run_id, step_id, execution_index, status, created_at, started_at, completed_at, error, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		exec.RunID,
		exec.StepID,
		exec.ExecutionIndex,
		string(exec.Status),
		exec.CreatedAt,
		exec.StartedAt,
		exec.CompletedAt,
		errMsg,
		data,
	)
	if err != nil {
		return fmt.Errorf("failed to create step execution: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetStepExecution(ctx context.Context, runID, stepID string) (*workflow.StepExecution, error) {
	var data []byte
	err := s.pool.QueryRow(ctx,
		`SELECT data FROM step_executions WHERE run_id = $1 AND step_id = $2`, runID, stepID,
	).Scan(&data)
	if err == pgx.ErrNoRows {
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

func (s *PostgresStore) UpdateStepExecution(ctx context.Context, exec *workflow.StepExecution) error {
	data, err := json.Marshal(exec)
	if err != nil {
		return fmt.Errorf("failed to marshal step execution: %w", err)
	}

	var errMsg *string
	if exec.Error != nil {
		msg := exec.Error.Error()
		errMsg = &msg
	}

	tag, err := s.pool.Exec(ctx, `
		UPDATE step_executions
		SET status = $1, started_at = $2, completed_at = $3, error = $4, data = $5
		WHERE run_id = $6 AND step_id = $7`,
		string(exec.Status),
		exec.StartedAt,
		exec.CompletedAt,
		errMsg,
		data,
		exec.RunID,
		exec.StepID,
	)
	if err != nil {
		return fmt.Errorf("failed to update step execution: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return workflow.ErrStepExecutionNotFound
	}
	return nil
}

func (s *PostgresStore) ListStepExecutions(ctx context.Context, runID string) ([]*workflow.StepExecution, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT data FROM step_executions WHERE run_id = $1 ORDER BY execution_index ASC`, runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list step executions: %w", err)
	}
	defer rows.Close()

	execs := make([]*workflow.StepExecution, 0, 16)
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

func (s *PostgresStore) SaveStepOutput(ctx context.Context, runID, stepID string, output []byte) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO step_outputs (run_id, step_id, output_data)
		VALUES ($1, $2, $3)
		ON CONFLICT (run_id, step_id) DO UPDATE SET output_data = EXCLUDED.output_data`,
		runID, stepID, output,
	)
	if err != nil {
		return fmt.Errorf("failed to save step output: %w", err)
	}
	return nil
}

func (s *PostgresStore) LoadStepOutput(ctx context.Context, runID, stepID string) ([]byte, error) {
	var data []byte
	err := s.pool.QueryRow(ctx,
		`SELECT output_data FROM step_outputs WHERE run_id = $1 AND step_id = $2`, runID, stepID,
	).Scan(&data)
	if err == pgx.ErrNoRows {
		return nil, workflow.ErrStepOutputNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load step output: %w", err)
	}
	return data, nil
}

// --- Workflow State ---

func (s *PostgresStore) SaveState(ctx context.Context, runID, key string, value []byte) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO workflow_state (run_id, key, value, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (run_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`,
		runID, key, value, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}
	return nil
}

func (s *PostgresStore) LoadState(ctx context.Context, runID, key string) ([]byte, error) {
	var value []byte
	err := s.pool.QueryRow(ctx,
		`SELECT value FROM workflow_state WHERE run_id = $1 AND key = $2`, runID, key,
	).Scan(&value)
	if err == pgx.ErrNoRows {
		return nil, workflow.ErrStateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return value, nil
}

func (s *PostgresStore) DeleteState(ctx context.Context, runID, key string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM workflow_state WHERE run_id = $1 AND key = $2`, runID, key,
	)
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetAllState(ctx context.Context, runID string) (map[string][]byte, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT key, value FROM workflow_state WHERE run_id = $1`, runID,
	)
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
