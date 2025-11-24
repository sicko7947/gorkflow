package conditional

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	workflow "github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
)

// Orchestrator handles the execution of conditional workflows
type Orchestrator struct {
	workflow *workflow.Workflow
	engine   *engine.Engine
	logger   zerolog.Logger
}

// NewOrchestrator creates a new conditional workflow orchestrator
func NewOrchestrator(
	store workflow.WorkflowStore,
	logger zerolog.Logger,
	config engine.EngineConfig,
) (*Orchestrator, error) {
	// Create conditional workflow
	wf, err := NewConditionalWorkflow()
	if err != nil {
		return nil, fmt.Errorf("failed to create conditional workflow: %w", err)
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

// StartWorkflow initiates a new conditional workflow execution
func (o *Orchestrator) StartWorkflow(
	ctx context.Context,
	input ConditionalInput,
) (string, error) {
	o.logger.Info().
		Int("value", input.Value).
		Bool("enable_doubling", input.EnableDoubling).
		Bool("enable_formatting", input.EnableFormatting).
		Msg("Starting conditional workflow")

	// Start workflow execution
	runID, err := o.engine.StartWorkflow(
		ctx,
		o.workflow,
		input,
		workflow.WithTags(map[string]string{
			"type": "conditional",
		}),
	)

	if err != nil {
		return "", fmt.Errorf("failed to start workflow: %w", err)
	}

	o.logger.Info().
		Str("run_id", runID).
		Msg("Conditional workflow started successfully")

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
		var output ConditionalFormatOutput
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

// WorkflowStatus represents the current status of a conditional workflow run
type WorkflowStatus struct {
	*workflow.WorkflowRun
	StepExecutions []*workflow.StepExecution `json:"stepExecutions,omitempty"`
	Output         *ConditionalFormatOutput  `json:"output,omitempty"`
}
