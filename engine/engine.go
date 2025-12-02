package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/sicko7947/gorkflow"
)

// Engine orchestrates workflow execution
type Engine struct {
	store  gorkflow.WorkflowStore
	logger zerolog.Logger
	config EngineConfig
}

// EngineConfig holds engine configuration
type EngineConfig struct {
	MaxConcurrentWorkflows int
	DefaultTimeout         time.Duration
}

// DefaultEngineConfig provides sensible defaults
var DefaultEngineConfig = EngineConfig{
	MaxConcurrentWorkflows: 10,
	DefaultTimeout:         5 * time.Minute,
}

// NewEngine creates a new workflow engine
// EngineOption configures the workflow engine
type EngineOption func(*Engine)

// WithLogger sets a custom logger for the engine
func WithLogger(logger zerolog.Logger) EngineOption {
	return func(e *Engine) {
		e.logger = logger
	}
}

// WithConfig sets a custom configuration for the engine
func WithConfig(config EngineConfig) EngineOption {
	return func(e *Engine) {
		e.config = config
	}
}

// NewEngine creates a new workflow engine with optional configuration
// If no logger is provided, a default stdout logger with Info level is used
// If no config is provided, DefaultEngineConfig is used
func NewEngine(store gorkflow.WorkflowStore, opts ...EngineOption) *Engine {
	// Default logger: pretty console output, Info level
	defaultLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Logger().
		Level(zerolog.InfoLevel)

	eng := &Engine{
		store:  store,
		logger: defaultLogger,
		config: DefaultEngineConfig,
	}

	// Apply options
	for _, opt := range opts {
		opt(eng)
	}

	return eng
}

// StartWorkflow initiates a workflow execution
func (e *Engine) StartWorkflow(
	ctx context.Context,
	wf *gorkflow.Workflow,
	input interface{},
	opts ...gorkflow.StartOption,
) (string, error) {
	// Apply options
	options := &gorkflow.StartOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Generate run ID
	runID := uuid.New().String()

	// Serialize input
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to serialize workflow input: %w", err)
	}

	// Serialize context if present
	var contextBytes json.RawMessage
	if wf.GetContext() != nil {
		contextBytes, err = json.Marshal(wf.GetContext())
		if err != nil {
			return "", fmt.Errorf("failed to serialize workflow context: %w", err)
		}
	}

	// Create workflow run
	now := time.Now()
	run := &gorkflow.WorkflowRun{
		RunID:           runID,
		WorkflowID:      wf.ID(),
		WorkflowVersion: wf.Version(),
		Status:          gorkflow.RunStatusPending,
		Progress:        0.0,
		CreatedAt:       now,
		UpdatedAt:       now,
		Input:           inputBytes,
		Context:         contextBytes,
		ResourceID:      options.ResourceID,
		Trigger: &gorkflow.TriggerInfo{
			Type:      options.TriggerType,
			Source:    options.TriggerSource,
			Timestamp: now,
		},
		Tags: options.Tags,
	}

	// Set TTL if specified
	if options.TTL > 0 {
		run.TTL = time.Now().Add(options.TTL).Unix()
	}

	// Persist run
	if err := e.store.CreateRun(ctx, run); err != nil {
		return "", fmt.Errorf("failed to create workflow run: %w", err)
	}

	gorkflow.LogWorkflowCreated(e.logger, runID, wf.ID(), options.ResourceID)

	// Launch execution in background
	if !options.Synchronous {
		go e.executeWorkflow(context.Background(), wf, run)
	} else {
		return runID, e.executeWorkflow(ctx, wf, run)
	}

	return runID, nil
}

// executeWorkflow runs the workflow (called asynchronously)
func (e *Engine) executeWorkflow(ctx context.Context, wf *gorkflow.Workflow, run *gorkflow.WorkflowRun) error {
	workflowLogger := gorkflow.WorkflowLogger(e.logger, run.RunID, run.WorkflowID, run.ResourceID)

	gorkflow.LogWorkflowStarted(e.logger, run.RunID, run.WorkflowID, run.ResourceID)

	// Update status to running
	startTime := time.Now()
	run.Status = gorkflow.RunStatusRunning
	run.StartedAt = &startTime
	run.UpdatedAt = startTime

	if err := e.store.UpdateRun(ctx, run); err != nil {
		gorkflow.LogPersistenceError(e.logger, run.RunID, "update_run_status", err)
		return err
	}

	// Build execution context - create accessors for state and outputs
	outputs := gorkflow.NewStepOutputAccessor(run.RunID, e.store)
	state := gorkflow.NewStateAccessor(run.RunID, e.store)

	// Get execution order from graph
	graph := wf.Graph()
	traverser := NewGraphTraverser(graph)
	executionOrder, err := traverser.GetExecutionOrder()
	if err != nil {
		workflowLogger.Error().Err(err).Msg("Failed to get execution order")
		return e.failWorkflow(ctx, run, err)
	}

	workflowLogger.Debug().
		Strs("execution_order", executionOrder).
		Msg("Execution order determined")

	totalSteps := len(executionOrder)
	completedSteps := 0

	// Execute steps in order
	for _, stepID := range executionOrder {
		// Check for cancellation
		select {
		case <-ctx.Done():
			gorkflow.LogWorkflowCancelled(e.logger, run.RunID)
			return e.cancelWorkflow(ctx, run)
		default:
		}

		// Get step
		step, err := wf.GetStep(stepID)
		if err != nil {
			workflowLogger.Error().Err(err).Str("step_id", stepID).Msg("Step not found")
			return e.failWorkflow(ctx, run, err)
		}

		gorkflow.LogStepStarted(e.logger, run.RunID, stepID, step.GetName(), completedSteps+1, totalSteps)

		// Prepare input for this step
		var stepInput []byte
		if completedSteps == 0 {
			// First step gets workflow input
			stepInput = run.Input
		} else {
			// Subsequent steps: get output from previous step
			// This assumes a linear chain for now. For complex graphs, we need to resolve dependencies.
			prevStepID := executionOrder[completedSteps-1]
			var err error
			stepInput, err = e.store.LoadStepOutput(ctx, run.RunID, prevStepID)
			if err != nil {
				// Check if previous step had ContinueOnError set
				prevStep, stepErr := wf.GetStep(prevStepID)
				if stepErr == nil && prevStep.GetConfig().ContinueOnError {
					workflowLogger.Warn().
						Str("prev_step_id", prevStepID).
						Msg("Previous step output not found, but ContinueOnError is true. Passing empty input.")
					// Pass JSON null so unmarshaling works (results in zero value)
					stepInput = []byte("null")
				} else {
					workflowLogger.Error().
						Err(err).
						Str("prev_step_id", prevStepID).
						Msg("Failed to load output from previous step")
					return e.failWorkflow(ctx, run, err)
				}
			}
		}

		// Execute step
		_, err = e.executeStep(ctx, run, step, stepInput, outputs, state, wf.GetContext(), completedSteps)
		if err != nil {
			// Check if we should continue on error
			if step.GetConfig().ContinueOnError {
				workflowLogger.Warn().
					Err(err).
					Str("step_id", stepID).
					Msg("Step failed but continuing due to ContinueOnError")
			} else {
				workflowLogger.Error().
					Err(err).
					Str("step_id", stepID).
					Msg("Step failed, stopping workflow")
				return e.failWorkflow(ctx, run, err)
			}
		}

		completedSteps++

		// Update progress
		progress := float64(completedSteps) / float64(totalSteps)
		run.Progress = progress
		run.UpdatedAt = time.Now()

		if err := e.store.UpdateRun(ctx, run); err != nil {
			gorkflow.LogPersistenceError(e.logger, run.RunID, "update_run_progress", err)
		}

		gorkflow.LogWorkflowProgress(e.logger, run.RunID, progress)
	}

	// All steps completed successfully
	return e.completeWorkflow(ctx, run)
}

// completeWorkflow marks workflow as completed
func (e *Engine) completeWorkflow(ctx context.Context, run *gorkflow.WorkflowRun) error {
	completedAt := time.Now()
	run.Status = gorkflow.RunStatusCompleted
	run.Progress = 1.0
	run.CompletedAt = &completedAt
	run.UpdatedAt = completedAt

	if err := e.store.UpdateRun(ctx, run); err != nil {
		return fmt.Errorf("failed to update run on completion: %w", err)
	}

	duration := completedAt.Sub(*run.StartedAt)
	gorkflow.LogWorkflowCompleted(e.logger, run.RunID, duration)

	return nil
}

// failWorkflow marks workflow as failed
func (e *Engine) failWorkflow(ctx context.Context, run *gorkflow.WorkflowRun, err error) error {
	completedAt := time.Now()
	run.Status = gorkflow.RunStatusFailed
	run.CompletedAt = &completedAt
	run.UpdatedAt = completedAt
	run.Error = &gorkflow.WorkflowError{
		Message:   err.Error(),
		Code:      gorkflow.ErrCodeExecutionFailed,
		Timestamp: completedAt,
	}

	if updateErr := e.store.UpdateRun(ctx, run); updateErr != nil {
		gorkflow.LogPersistenceError(e.logger, run.RunID, "update_run_failure", updateErr)
	}

	gorkflow.LogWorkflowFailed(e.logger, run.RunID, err)

	return err
}

// cancelWorkflow marks workflow as cancelled
func (e *Engine) cancelWorkflow(ctx context.Context, run *gorkflow.WorkflowRun) error {
	completedAt := time.Now()
	run.Status = gorkflow.RunStatusCancelled
	run.CompletedAt = &completedAt
	run.UpdatedAt = completedAt

	if err := e.store.UpdateRun(ctx, run); err != nil {
		return fmt.Errorf("failed to update run on cancellation: %w", err)
	}

	gorkflow.LogWorkflowCancelled(e.logger, run.RunID)

	return nil
}

// GetRun retrieves workflow run status
func (e *Engine) GetRun(ctx context.Context, runID string) (*gorkflow.WorkflowRun, error) {
	return e.store.GetRun(ctx, runID)
}

// GetStepExecutions retrieves all step executions for a run
func (e *Engine) GetStepExecutions(ctx context.Context, runID string) ([]*gorkflow.StepExecution, error) {
	return e.store.ListStepExecutions(ctx, runID)
}

// Cancel cancels a running workflow
func (e *Engine) Cancel(ctx context.Context, runID string) error {
	run, err := e.store.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to get run: %w", err)
	}

	if run.Status.IsTerminal() {
		return fmt.Errorf("cannot cancel workflow in %s state", run.Status)
	}

	return e.cancelWorkflow(ctx, run)
}

// ListRuns lists workflow runs with filtering
func (e *Engine) ListRuns(ctx context.Context, filter gorkflow.RunFilter) ([]*gorkflow.WorkflowRun, error) {
	return e.store.ListRuns(ctx, filter)
}
