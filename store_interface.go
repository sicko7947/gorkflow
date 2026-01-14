package gorkflow

import "context"

// WorkflowStore defines the persistence interface for workflows
type WorkflowStore interface {
	// Workflow runs
	CreateRun(ctx context.Context, run *WorkflowRun) error
	GetRun(ctx context.Context, runID string) (*WorkflowRun, error)
	UpdateRun(ctx context.Context, run *WorkflowRun) error
	UpdateRunStatus(ctx context.Context, runID string, status RunStatus, err *WorkflowError) error
	ListRuns(ctx context.Context, filter RunFilter) ([]*WorkflowRun, error)

	// Step executions
	CreateStepExecution(ctx context.Context, exec *StepExecution) error
	GetStepExecution(ctx context.Context, runID, stepID string) (*StepExecution, error)
	UpdateStepExecution(ctx context.Context, exec *StepExecution) error
	ListStepExecutions(ctx context.Context, runID string) ([]*StepExecution, error)

	// Step outputs (for inter-step communication)
	SaveStepOutput(ctx context.Context, runID, stepID string, output []byte) error
	LoadStepOutput(ctx context.Context, runID, stepID string) ([]byte, error)

	// Workflow state
	SaveState(ctx context.Context, runID, key string, value []byte) error
	LoadState(ctx context.Context, runID, key string) ([]byte, error)
	DeleteState(ctx context.Context, runID, key string) error
	GetAllState(ctx context.Context, runID string) (map[string][]byte, error)

	// Queries
	CountRunsByStatus(ctx context.Context, resourceID string, status RunStatus) (int, error)
}

// RunFilter defines filtering criteria for workflow runs
type RunFilter struct {
	WorkflowID string
	Status     *RunStatus
	ResourceID string
	Limit      int
}
