package store

import (
	"fmt"
	"strings"
)

const (
	// Table names
	TableWorkflowRuns   = "workflow_runs"
	TableStepExecutions = "step_executions"
	TableStepOutputs    = "step_outputs"
	TableWorkflowState  = "workflow_state"
)

// Schema definitions
const (
	schemaWorkflowRuns = `
CREATE TABLE IF NOT EXISTS workflow_runs (
	run_id TEXT PRIMARY KEY,
	workflow_id TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	resource_id TEXT,
	data TEXT
);
CREATE INDEX IF NOT EXISTS idx_runs_workflow_status ON workflow_runs(workflow_id, status);
CREATE INDEX IF NOT EXISTS idx_runs_resource_status ON workflow_runs(resource_id, status);
`

	schemaStepExecutions = `
CREATE TABLE IF NOT EXISTS step_executions (
	run_id TEXT NOT NULL,
	step_id TEXT NOT NULL,
	execution_index INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	started_at DATETIME,
	completed_at DATETIME,
	error TEXT,
	data TEXT,
	PRIMARY KEY (run_id, step_id)
);
CREATE INDEX IF NOT EXISTS idx_step_executions_run_index ON step_executions(run_id, execution_index);
`

	schemaStepOutputs = `
CREATE TABLE IF NOT EXISTS step_outputs (
	run_id TEXT NOT NULL,
	step_id TEXT NOT NULL,
	output_data BLOB,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (run_id, step_id)
);
`

	schemaWorkflowState = `
CREATE TABLE IF NOT EXISTS workflow_state (
	run_id TEXT NOT NULL,
	key TEXT NOT NULL,
	value BLOB,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (run_id, key)
);
`
)

// GetSchema returns the full schema creation script
func GetLibSQLSchema() string {
	return strings.Join([]string{
		schemaWorkflowRuns,
		schemaStepExecutions,
		schemaStepOutputs,
		schemaWorkflowState,
	}, "\n")
}

// Helper to format placeholders for SQL queries
// LibSQL/SQLite uses ? for placeholders
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// Helper to build UPDATE set clause
func buildUpdateSet(cols []string) string {
	var sets []string
	for _, col := range cols {
		sets = append(sets, fmt.Sprintf("%s = ?", col))
	}
	return strings.Join(sets, ", ")
}
