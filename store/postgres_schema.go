package store

import "strings"

const (
	postgresSchemaWorkflowRuns = `
CREATE TABLE IF NOT EXISTS workflow_runs (
	run_id      TEXT PRIMARY KEY,
	workflow_id TEXT        NOT NULL,
	status      TEXT        NOT NULL,
	created_at  TIMESTAMPTZ NOT NULL,
	updated_at  TIMESTAMPTZ NOT NULL,
	resource_id TEXT,
	data        JSONB
);
CREATE INDEX IF NOT EXISTS idx_runs_workflow_status  ON workflow_runs(workflow_id, status);
CREATE INDEX IF NOT EXISTS idx_runs_resource_status  ON workflow_runs(resource_id, status);
CREATE INDEX IF NOT EXISTS idx_runs_updated_at       ON workflow_runs(updated_at);
CREATE INDEX IF NOT EXISTS idx_runs_workflow_created ON workflow_runs(workflow_id, created_at);
`

	postgresSchemaStepExecutions = `
CREATE TABLE IF NOT EXISTS step_executions (
	run_id          TEXT        NOT NULL,
	step_id         TEXT        NOT NULL,
	execution_index INTEGER     NOT NULL DEFAULT 0,
	status          TEXT        NOT NULL,
	created_at      TIMESTAMPTZ NOT NULL,
	started_at      TIMESTAMPTZ,
	completed_at    TIMESTAMPTZ,
	error           TEXT,
	data            JSONB,
	PRIMARY KEY (run_id, step_id)
);
CREATE INDEX IF NOT EXISTS idx_step_executions_run_index ON step_executions(run_id, execution_index);
CREATE INDEX IF NOT EXISTS idx_step_executions_status    ON step_executions(status);
`

	postgresSchemaStepOutputs = `
CREATE TABLE IF NOT EXISTS step_outputs (
	run_id      TEXT  NOT NULL,
	step_id     TEXT  NOT NULL,
	output_data BYTEA,
	created_at  TIMESTAMPTZ DEFAULT NOW(),
	PRIMARY KEY (run_id, step_id)
);
`

	postgresSchemaWorkflowState = `
CREATE TABLE IF NOT EXISTS workflow_state (
	run_id     TEXT  NOT NULL,
	key        TEXT  NOT NULL,
	value      BYTEA,
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	PRIMARY KEY (run_id, key)
);
`
)

// GetPostgresSchema returns the full PostgreSQL schema creation script.
func GetPostgresSchema() string {
	return strings.Join([]string{
		postgresSchemaWorkflowRuns,
		postgresSchemaStepExecutions,
		postgresSchemaStepOutputs,
		postgresSchemaWorkflowState,
	}, "\n")
}
