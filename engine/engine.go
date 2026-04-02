package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/sicko7947/gorkflow"
)

// Engine orchestrates workflow execution
type Engine struct {
	store      gorkflow.WorkflowStore
	logger     zerolog.Logger
	config     gorkflow.EngineConfig
	activeRuns map[string]context.CancelFunc
	runsMu     sync.Mutex
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
func WithConfig(config gorkflow.EngineConfig) EngineOption {
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
		store:      store,
		logger:     defaultLogger,
		config:     gorkflow.DefaultEngineConfig,
		activeRuns: make(map[string]context.CancelFunc),
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
	input any,
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
		Tags:            options.Tags,
	}

	// Persist run
	if err := e.store.CreateRun(ctx, run); err != nil {
		return "", fmt.Errorf("failed to create workflow run: %w", err)
	}

	gorkflow.LogWorkflowCreated(e.logger, runID, wf.ID(), options.ResourceID)

	// Launch execution in background
	if !options.Synchronous {
		bgCtx, cancel := context.WithCancel(context.Background())
		e.runsMu.Lock()
		e.activeRuns[runID] = cancel
		e.runsMu.Unlock()
		go func() {
			defer func() {
				e.runsMu.Lock()
				delete(e.activeRuns, runID)
				e.runsMu.Unlock()
				cancel()
			}()
			e.executeWorkflow(bgCtx, wf, run)
		}()
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

	// Build execution context - create shared state accessor
	state := gorkflow.NewStateAccessor(run.RunID, e.store)

	// Get execution levels (groups of steps that can run concurrently)
	graph := wf.Graph()
	levels, err := graph.ComputeLevels()
	if err != nil {
		workflowLogger.Error().Err(err).Msg("Failed to compute execution levels")
		return e.failWorkflow(ctx, run, err)
	}

	totalSteps := 0
	for _, level := range levels {
		totalSteps += len(level)
	}

	var completedSteps int64 // atomic counter

	for _, level := range levels {
		// Check for cancellation before each level
		select {
		case <-ctx.Done():
			gorkflow.LogWorkflowCancelled(e.logger, run.RunID)
			return e.cancelWorkflow(ctx, run)
		default:
		}

		if len(level) == 1 {
			// Single step — sequential path
			stepID := level[0]
			step, err := wf.GetStep(stepID)
			if err != nil {
				return e.failWorkflow(ctx, run, err)
			}

			stepInput, err := e.resolveStepInput(ctx, run, wf, stepID, atomic.LoadInt64(&completedSteps) == 0)
			if err != nil {
				return e.failWorkflow(ctx, run, err)
			}

			gorkflow.LogStepStarted(e.logger, run.RunID, stepID, step.GetName(), int(atomic.LoadInt64(&completedSteps))+1, totalSteps)

			result, err := e.executeStep(ctx, run, step, stepInput, state, wf.GetContext(), int(atomic.LoadInt64(&completedSteps)))
			if err != nil {
				if ctx.Err() != nil {
					gorkflow.LogWorkflowCancelled(e.logger, run.RunID)
					return e.cancelWorkflow(ctx, run)
				}
				if step.GetConfig().ContinueOnError {
					workflowLogger.Warn().Err(err).Str("step_id", stepID).Msg("Step failed but continuing")
				} else {
					return e.failWorkflow(ctx, run, err)
				}
			}

			if result != nil && result.Status == gorkflow.StepStatusCompleted {
				run.Output = result.Output
			}

			atomic.AddInt64(&completedSteps, 1)
			progress := float64(atomic.LoadInt64(&completedSteps)) / float64(totalSteps)
			run.Progress = progress
			run.UpdatedAt = time.Now()
			// Progress update is best-effort; a failure here doesn't stop execution.
			if err := e.store.UpdateRun(ctx, run); err != nil {
				gorkflow.LogPersistenceError(e.logger, run.RunID, "update_run_progress", err)
			}
			gorkflow.LogWorkflowProgress(e.logger, run.RunID, progress)

		} else {
			// Multiple steps in this level — run concurrently
			type stepResult struct {
				stepID string
				result *StepExecutionResult
				err    error
			}
			resultsCh := make(chan stepResult, len(level))

			sem := make(chan struct{}, len(level))

			for _, stepID := range level {
				step, err := wf.GetStep(stepID)
				if err != nil {
					resultsCh <- stepResult{stepID: stepID, err: err}
					continue
				}

				stepInput, err := e.resolveStepInput(ctx, run, wf, stepID, false)
				if err != nil {
					resultsCh <- stepResult{stepID: stepID, err: err}
					continue
				}

				execIndex := int(atomic.LoadInt64(&completedSteps))
				go func(sID string, s gorkflow.StepExecutor, input []byte, idx int) {
					sem <- struct{}{}
					defer func() { <-sem }()

					gorkflow.LogStepStarted(e.logger, run.RunID, sID, s.GetName(), idx+1, totalSteps)
					result, err := e.executeStep(ctx, run, s, input, state, wf.GetContext(), idx)
					resultsCh <- stepResult{stepID: sID, result: result, err: err}
				}(stepID, step, stepInput, execIndex)
			}

			// Collect all results
			var fatalErr error
			for range level {
				r := <-resultsCh
				if r.err != nil {
					step, _ := wf.GetStep(r.stepID)
					if step != nil && step.GetConfig().ContinueOnError {
						workflowLogger.Warn().Err(r.err).Str("step_id", r.stepID).Msg("Parallel step failed but continuing")
					} else if fatalErr == nil {
						fatalErr = r.err
					}
				} else if r.result != nil && r.result.Status == gorkflow.StepStatusCompleted {
					run.Output = r.result.Output
				}
				atomic.AddInt64(&completedSteps, 1)
			}

			// Single progress update after all steps in this level complete.
			progress := float64(atomic.LoadInt64(&completedSteps)) / float64(totalSteps)
			run.Progress = progress
			run.UpdatedAt = time.Now()
			if err := e.store.UpdateRun(ctx, run); err != nil {
				gorkflow.LogPersistenceError(e.logger, run.RunID, "update_run_progress", err)
			}
			gorkflow.LogWorkflowProgress(e.logger, run.RunID, progress)

			if fatalErr != nil {
				if ctx.Err() != nil {
					gorkflow.LogWorkflowCancelled(e.logger, run.RunID)
					return e.cancelWorkflow(ctx, run)
				}
				return e.failWorkflow(ctx, run, fatalErr)
			}
		}
	}

	// All steps completed successfully
	return e.completeWorkflow(ctx, run)
}

// resolveStepInput determines what input a step should receive
func (e *Engine) resolveStepInput(ctx context.Context, run *gorkflow.WorkflowRun, wf *gorkflow.Workflow, stepID string, isFirst bool) ([]byte, error) {
	if isFirst {
		return run.Input, nil
	}
	prevSteps, err := wf.Graph().GetPreviousSteps(stepID)
	if err != nil {
		return nil, err
	}
	if len(prevSteps) == 0 {
		return run.Input, nil
	}
	prevStepID := prevSteps[0]
	input, err := e.store.LoadStepOutput(ctx, run.RunID, prevStepID)
	if err != nil {
		prevStep, stepErr := wf.GetStep(prevStepID)
		if stepErr == nil && prevStep.GetConfig().ContinueOnError {
			return []byte("null"), nil
		}
		return nil, err
	}
	return input, nil
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

// LoadStepOutput retrieves the output of a specific step execution.
// Step outputs are stored separately from step execution metadata to avoid
// loading potentially large payloads when listing all executions.
func (e *Engine) LoadStepOutput(ctx context.Context, runID, stepID string) ([]byte, error) {
	return e.store.LoadStepOutput(ctx, runID, stepID)
}

// Cancel cancels a running workflow.
// If an async goroutine is active, signals it and returns immediately — the goroutine
// handles the DB update via its ctx.Done() path, avoiding a double-update.
// If no goroutine is active (sync execution, already finished), updates DB directly.
func (e *Engine) Cancel(ctx context.Context, runID string) error {
	e.runsMu.Lock()
	cancelFn, hasActive := e.activeRuns[runID]
	if hasActive {
		cancelFn()
	}
	e.runsMu.Unlock()

	if hasActive {
		// The goroutine's ctx.Done() path calls cancelWorkflow — don't double-update.
		return nil
	}

	// No active goroutine: update DB directly.
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
