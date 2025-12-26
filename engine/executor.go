package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sicko7947/gorkflow"
)

// StepExecutionResult holds the result of a step execution
type StepExecutionResult struct {
	StepID       string
	Status       gorkflow.StepStatus
	Output       []byte
	Error        error
	DurationMs   int64
	AttemptsMade int
}

// executeStep runs a single step with retry/timeout logic
func (e *Engine) executeStep(
	ctx context.Context,
	run *gorkflow.WorkflowRun,
	step gorkflow.StepExecutor,
	inputBytes []byte,
	outputs gorkflow.StepDataAccessor,
	state gorkflow.StateAccessor,
	customContext any,
	executionIndex int,
) (*StepExecutionResult, error) {
	config := step.GetConfig()

	// Create step execution record
	stepExec := &gorkflow.StepExecution{
		RunID:          run.RunID,
		StepID:         step.GetID(),
		ExecutionIndex: executionIndex,
		Status:         gorkflow.StepStatusPending,
		Input:          inputBytes,
		StartedAt:      nil,
		CompletedAt:    nil,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := e.store.CreateStepExecution(ctx, stepExec); err != nil {
		return nil, fmt.Errorf("failed to create step execution: %w", err)
	}

	// Build step context
	stepLogger := gorkflow.StepLogger(e.logger, step.GetID(), step.GetName(), 0).With().Str("run_id", run.RunID).Logger()

	stepCtx := &gorkflow.StepContext{
		Context:       ctx,
		RunID:         run.RunID,
		StepID:        step.GetID(),
		Logger:        stepLogger,
		Data:          outputs,
		State:         state,
		CustomContext: customContext,
	}

	var outputBytes []byte
	var lastErr error
	var attemptsMade int

	// Retry loop
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		attemptsMade = attempt + 1
		stepCtx.Attempt = attempt

		if attempt > 0 {
			// Apply backoff
			delay := gorkflow.CalculateBackoff(config.RetryDelayMs, attempt, string(config.RetryBackoff))

			gorkflow.LogStepRetrying(e.logger, run.RunID, step.GetID(), attempt, delay)

			stepExec.Status = gorkflow.StepStatusRetrying
			stepExec.Attempt = attempt
			stepExec.UpdatedAt = time.Now()

			if err := e.store.UpdateStepExecution(ctx, stepExec); err != nil {
				gorkflow.LogPersistenceError(e.logger, run.RunID, "update_step_execution_retry", err)
			}

			if delay > 0 {
				time.Sleep(delay)
			}
		}

		// Update to running
		stepExec.Status = gorkflow.StepStatusRunning
		now := time.Now()
		stepExec.StartedAt = &now
		stepExec.Attempt = attempt
		stepExec.UpdatedAt = now

		if err := e.store.UpdateStepExecution(ctx, stepExec); err != nil {
			gorkflow.LogPersistenceError(e.logger, run.RunID, "update_step_execution_running", err)
		}

		// Execute with timeout
		execCtx, cancel := context.WithTimeout(
			ctx,
			time.Duration(config.TimeoutSeconds)*time.Second,
		)

		stepCtx.Context = execCtx
		startTime := time.Now()

		// Execute step (with panic recovery)
		func() {
			defer func() {
				if r := recover(); r != nil {
					lastErr = fmt.Errorf("step panicked: %v", r)
					stepLogger.Error().Interface("panic", r).Msg("Step panicked")
				}
			}()

			outputBytes, lastErr = step.Execute(stepCtx, inputBytes)
		}()

		cancel() // Clean up timeout context
		duration := time.Since(startTime)
		stepExec.DurationMs = duration.Milliseconds()

		// Check if step was skipped
		if errors.Is(lastErr, gorkflow.ErrStepSkipped) {
			stepExec.Status = gorkflow.StepStatusSkipped
			stepExec.Output = outputBytes
			completedAt := time.Now()
			stepExec.CompletedAt = &completedAt
			stepExec.UpdatedAt = completedAt

			if err := e.store.UpdateStepExecution(ctx, stepExec); err != nil {
				gorkflow.LogPersistenceError(e.logger, run.RunID, "update_step_execution_skipped", err)
			}

			// Save output for downstream steps (even if skipped, we might have pass-through output)
			if err := e.store.SaveStepOutput(ctx, run.RunID, step.GetID(), outputBytes); err != nil {
				gorkflow.LogPersistenceError(e.logger, run.RunID, "save_step_output_skipped", err)
			}

			return &StepExecutionResult{
				StepID:       step.GetID(),
				Status:       gorkflow.StepStatusSkipped,
				Output:       outputBytes,
				Error:        nil,
				DurationMs:   duration.Milliseconds(),
				AttemptsMade: attemptsMade,
			}, nil
		}

		if lastErr == nil {
			// Success
			stepExec.Status = gorkflow.StepStatusCompleted
			stepExec.Output = outputBytes
			completedAt := time.Now()
			stepExec.CompletedAt = &completedAt
			stepExec.UpdatedAt = completedAt

			if err := e.store.UpdateStepExecution(ctx, stepExec); err != nil {
				gorkflow.LogPersistenceError(e.logger, run.RunID, "update_step_execution_success", err)
			}

			gorkflow.LogStepCompleted(e.logger, run.RunID, step.GetID(), duration.Milliseconds(), attemptsMade)

			// Save output for downstream steps
			if err := e.store.SaveStepOutput(ctx, run.RunID, step.GetID(), outputBytes); err != nil {
				gorkflow.LogPersistenceError(e.logger, run.RunID, "save_step_output", err)
			}

			return &StepExecutionResult{
				StepID:       step.GetID(),
				Status:       gorkflow.StepStatusCompleted,
				Output:       outputBytes,
				Error:        nil,
				DurationMs:   duration.Milliseconds(),
				AttemptsMade: attemptsMade,
			}, nil
		}

		// Check if error is timeout
		if execCtx.Err() == context.DeadlineExceeded {
			lastErr = fmt.Errorf("step timed out after %d seconds: %w", config.TimeoutSeconds, lastErr)
			stepLogger.Error().
				Int("timeout_seconds", config.TimeoutSeconds).
				Msg("Step execution timed out")
		}

		gorkflow.LogStepFailed(e.logger, run.RunID, step.GetID(), lastErr, attempt, duration.Milliseconds())
	}

	// All retries exhausted
	stepExec.Status = gorkflow.StepStatusFailed
	completedAt := time.Now()
	stepExec.CompletedAt = &completedAt
	stepExec.UpdatedAt = completedAt
	stepExec.Error = &gorkflow.StepError{
		Message: lastErr.Error(),
		Code:    gorkflow.ErrCodeExecutionFailed,
		Attempt: config.MaxRetries,
	}

	if err := e.store.UpdateStepExecution(ctx, stepExec); err != nil {
		gorkflow.LogPersistenceError(e.logger, run.RunID, "update_step_execution_failure", err)
	}

	stepLogger.Error().
		Int("max_retries", config.MaxRetries).
		Int("attempts_made", attemptsMade).
		Msg("Step failed after all retries exhausted")

	return &StepExecutionResult{
		StepID:       step.GetID(),
		Status:       gorkflow.StepStatusFailed,
		Output:       nil,
		Error:        lastErr,
		DurationMs:   stepExec.DurationMs,
		AttemptsMade: attemptsMade,
	}, fmt.Errorf("step %s failed after %d attempts: %w", step.GetID(), attemptsMade, lastErr)
}
