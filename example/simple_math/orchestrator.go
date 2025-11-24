package simple_math

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	workflow "github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
)

// Orchestrator handles the execution of simple math workflows
type Orchestrator struct {
	workflow *workflow.Workflow
	engine   *engine.Engine
	logger   zerolog.Logger
}

// NewOrchestrator creates a new simple math workflow orchestrator
func NewOrchestrator(
	store workflow.WorkflowStore,
	logger zerolog.Logger,
	config engine.EngineConfig,
) (*Orchestrator, error) {
	// Create simple math workflow
	wf, err := NewSimpleMathWorkflow()
	if err != nil {
		return nil, fmt.Errorf("failed to create simple math workflow: %w", err)
	}

	// Create engine with logger and config options
	eng := engine.NewEngine(store,
		engine.WithLogger(logger),
		engine.WithConfig(config),
	)

	return &Orchestrator{
		workflow: wf,
		engine:   eng,
		logger:   logger,
	}, nil
}

// StartWorkflow initiates a new simple math workflow execution
func (o *Orchestrator) StartWorkflow(
	ctx context.Context,
	input WorkflowInput,
) (string, error) {
	o.logger.Info().
		Int("val1", input.Val1).
		Int("val2", input.Val2).
		Int("mult", input.Mult).
		Msg("Starting simple math workflow")

	// Start workflow execution
	runID, err := o.engine.StartWorkflow(
		ctx,
		o.workflow,
		input,
		workflow.WithTags(map[string]string{
			"type": "simple_math",
		}),
	)

	if err != nil {
		return "", fmt.Errorf("failed to start workflow: %w", err)
	}

	o.logger.Info().
		Str("run_id", runID).
		Msg("Simple math workflow started successfully")

	return runID, nil
}

// GetWorkflowStatus retrieves the status and results of a workflow run
func (o *Orchestrator) GetWorkflowStatus(
	ctx context.Context,
	runID string,
) (*WorkflowStatus, error) {
	// Get run metadata
	run, err := o.engine.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow run: %w", err)
	}

	// Get step executions for detailed progress
	stepExecs, err := o.engine.GetStepExecutions(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get step executions: %w", err)
	}

	status := &WorkflowStatus{
		WorkflowRun:    run,
		StepExecutions: stepExecs,
	}

	// If completed, parse output
	if run.Status == workflow.RunStatusCompleted && len(run.Output) > 0 {
		var output FormatOutput
		if err := json.Unmarshal(run.Output, &output); err != nil {
			o.logger.Warn().
				Err(err).
				Str("run_id", runID).
				Msg("Failed to parse workflow output")
		} else {
			status.Output = &output
		}
	}

	return status, nil
}

// CancelWorkflow cancels a running workflow
func (o *Orchestrator) CancelWorkflow(ctx context.Context, runID string) error {
	return o.engine.Cancel(ctx, runID)
}

// WorkflowStatus represents the current status of a simple math workflow run
type WorkflowStatus struct {
	*workflow.WorkflowRun
	StepExecutions []*workflow.StepExecution `json:"stepExecutions,omitempty"`
	Output         *FormatOutput             `json:"output,omitempty"`
}
