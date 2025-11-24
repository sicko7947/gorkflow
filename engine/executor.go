package engine

import (
	"context"
	"fmt"
	"time"

	workflow "github.com/sicko7947/gorkflow"
)

// StepExecutionResult holds the result of a step execution
type StepExecutionResult struct {
	StepID       string
	Output       []byte
	Error        error
	DurationMs   int64
	AttemptsMade int
}

// executeStep runs a single step with retry/timeout logic
func (e *Engine) executeStep(
	ctx context.Context,
	run *workflow.WorkflowRun,
	step workflow.StepExecutor,
	inputBytes []byte,
	outputs workflow.StepOutputAccessor,
	state workflow.StateAccessor,
) (*StepExecutionResult, error) {
	config := step.GetConfig()

	// Create step execution record
	stepExec := &workflow.StepExecution{
		RunID:          run.RunID,
		StepID:         step.GetID(),
		ExecutionIndex: 0,
		Status:         workflow.StepStatusPending,
		Input:          inputBytes,
		StartedAt:      nil,
		CompletedAt:    nil,
		UpdatedAt:      time.Now(),
	}

	if err := e.store.CreateStepExecution(ctx, stepExec); err != nil {
		return nil, fmt.Errorf("failed to create step execution: %w", err)
	}

	// Build step context
	stepLogger := e.logger.With().
		Str("run_id", run.RunID).
		Str("step_id", step.GetID()).
		Str("step_name", step.GetName()).
		Logger()

	stepCtx := &workflow.StepContext{
		Context: ctx,
		RunID:   run.RunID,
		StepID:  step.GetID(),
		Logger:  stepLogger,
		Outputs: outputs,
		State:   state,
	}

	var outputBytes []byte
	var lastErr error
	var attemptsMade int

	// Retry loop
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		attemptsMade = attempt + 1
		stepCtx.Attempt = attempt

		if attempt > 0 {
			stepLogger.Warn().Int("attempt", attempt).Msg("Retrying step")
			stepExec.Status = workflow.StepStatusRetrying
			stepExec.Attempt = attempt
			stepExec.UpdatedAt = time.Now()

			if err := e.store.UpdateStepExecution(ctx, stepExec); err != nil {
				e.logger.Error().Err(err).Msg("Failed to update step execution during retry")
			}

			// Apply backoff
			delay := calculateBackoff(config.RetryDelayMs, attempt, string(config.RetryBackoff))
			if delay > 0 {
				stepLogger.Debug().Dur("delay", delay).Msg("Applying backoff delay")
				time.Sleep(delay)
			}
		}

		// Update to running
		stepExec.Status = workflow.StepStatusRunning
		now := time.Now()
		stepExec.StartedAt = &now
		stepExec.Attempt = attempt
		stepExec.UpdatedAt = now

		if err := e.store.UpdateStepExecution(ctx, stepExec); err != nil {
			e.logger.Error().Err(err).Msg("Failed to update step execution to running")
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

		if lastErr == nil {
			// Success
			stepExec.Status = workflow.StepStatusCompleted
			stepExec.Output = outputBytes
			completedAt := time.Now()
			stepExec.CompletedAt = &completedAt
			stepExec.UpdatedAt = completedAt

			if err := e.store.UpdateStepExecution(ctx, stepExec); err != nil {
				e.logger.Error().Err(err).Msg("Failed to update step execution on success")
			}

			stepLogger.Info().
				Int64("duration_ms", duration.Milliseconds()).
				Int("attempts", attemptsMade).
				Msg("Step completed successfully")

			// Save output for downstream steps
			if err := e.store.SaveStepOutput(ctx, run.RunID, step.GetID(), outputBytes); err != nil {
				e.logger.Error().Err(err).Msg("Failed to save step output")
			}

			return &StepExecutionResult{
				StepID:       step.GetID(),
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

		stepLogger.Error().
			Err(lastErr).
			Int("attempt", attempt).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Step execution failed")
	}

	// All retries exhausted
	stepExec.Status = workflow.StepStatusFailed
	completedAt := time.Now()
	stepExec.CompletedAt = &completedAt
	stepExec.UpdatedAt = completedAt
	stepExec.Error = &workflow.StepError{
		Message: lastErr.Error(),
		Code:    workflow.ErrCodeExecutionFailed,
		Attempt: config.MaxRetries,
	}

	if err := e.store.UpdateStepExecution(ctx, stepExec); err != nil {
		e.logger.Error().Err(err).Msg("Failed to update step execution on failure")
	}

	stepLogger.Error().
		Int("max_retries", config.MaxRetries).
		Int("attempts_made", attemptsMade).
		Msg("Step failed after all retries exhausted")

	return &StepExecutionResult{
		StepID:       step.GetID(),
		Output:       nil,
		Error:        lastErr,
		DurationMs:   stepExec.DurationMs,
		AttemptsMade: attemptsMade,
	}, fmt.Errorf("step %s failed after %d attempts: %w", step.GetID(), attemptsMade, lastErr)
}
