package gorkflow

import (
	"time"

	"github.com/rs/zerolog"
)

// Log event names as per spec §9
const (
	// Workflow-level events
	EventWorkflowCreated   = "workflow_created"
	EventWorkflowStarted   = "workflow_started"
	EventWorkflowProgress  = "workflow_progress"
	EventWorkflowCompleted = "workflow_completed"
	EventWorkflowFailed    = "workflow_failed"
	EventWorkflowCancelled = "workflow_cancelled"

	// Step-level events
	EventStepStarted   = "step_started"
	EventStepRetrying  = "step_retrying"
	EventStepCompleted = "step_completed"
	EventStepFailed    = "step_failed"
	EventStepSkipped   = "step_skipped"

	// Persistence events
	EventPersistenceError = "persistence_error"
)

// LogWorkflowCreated logs when a workflow run is created
func LogWorkflowCreated(logger zerolog.Logger, runID, workflowID, resourceID string) {
	logger.Info().
		Str("event", EventWorkflowCreated).
		Str("run_id", runID).
		Str("workflow_id", workflowID).
		Str("resource_id", resourceID).
		Msg("Workflow run created")
}

// LogWorkflowStarted logs when a workflow starts execution
func LogWorkflowStarted(logger zerolog.Logger, runID, workflowID, resourceID string) {
	logger.Info().
		Str("event", EventWorkflowStarted).
		Str("run_id", runID).
		Str("workflow_id", workflowID).
		Str("resource_id", resourceID).
		Msg("Workflow started")
}

// LogWorkflowProgress logs workflow execution progress
func LogWorkflowProgress(logger zerolog.Logger, runID string, progress float64) {
	logger.Debug().
		Str("event", EventWorkflowProgress).
		Str("run_id", runID).
		Float64("progress", progress).
		Msg("Workflow progress updated")
}

// LogWorkflowCompleted logs successful workflow completion
func LogWorkflowCompleted(logger zerolog.Logger, runID string, duration time.Duration) {
	logger.Info().
		Str("event", EventWorkflowCompleted).
		Str("run_id", runID).
		Dur("duration", duration).
		Msg("Workflow completed")
}

// LogWorkflowFailed logs workflow failure
func LogWorkflowFailed(logger zerolog.Logger, runID string, err error) {
	logger.Error().
		Str("event", EventWorkflowFailed).
		Str("run_id", runID).
		Err(err).
		Msg("Workflow failed")
}

// LogWorkflowCancelled logs workflow cancellation
func LogWorkflowCancelled(logger zerolog.Logger, runID string) {
	logger.Warn().
		Str("event", EventWorkflowCancelled).
		Str("run_id", runID).
		Msg("Workflow cancelled")
}

// LogStepStarted logs when a step starts execution
func LogStepStarted(logger zerolog.Logger, runID, stepID, stepName string, stepNum, totalSteps int) {
	logger.Info().
		Str("event", EventStepStarted).
		Str("run_id", runID).
		Str("step_id", stepID).
		Str("step_name", stepName).
		Int("step_num", stepNum).
		Int("total_steps", totalSteps).
		Msg("Step started")
}

// LogStepRetrying logs when a step is being retried
func LogStepRetrying(logger zerolog.Logger, runID, stepID string, attempt int, delay time.Duration) {
	logger.Warn().
		Str("event", EventStepRetrying).
		Str("run_id", runID).
		Str("step_id", stepID).
		Int("attempt", attempt).
		Dur("delay", delay).
		Msg("Step retrying")
}

// LogStepCompleted logs successful step completion
func LogStepCompleted(logger zerolog.Logger, runID, stepID string, durationMs int64, attempts int) {
	logger.Info().
		Str("event", EventStepCompleted).
		Str("run_id", runID).
		Str("step_id", stepID).
		Int64("duration_ms", durationMs).
		Int("attempts", attempts).
		Msg("Step completed")
}

// LogStepFailed logs step failure
func LogStepFailed(logger zerolog.Logger, runID, stepID string, err error, attempt int, durationMs int64) {
	logger.Error().
		Str("event", EventStepFailed).
		Str("run_id", runID).
		Str("step_id", stepID).
		Err(err).
		Int("attempt", attempt).
		Int64("duration_ms", durationMs).
		Msg("Step failed")
}

// LogStepSkipped logs when a conditional step is skipped
func LogStepSkipped(logger zerolog.Logger, runID, stepID, reason string) {
	logger.Info().
		Str("event", EventStepSkipped).
		Str("run_id", runID).
		Str("step_id", stepID).
		Str("reason", reason).
		Msg("Step skipped")
}

// LogPersistenceError logs errors during persistence operations
func LogPersistenceError(logger zerolog.Logger, runID, operation string, err error) {
	logger.Error().
		Str("event", EventPersistenceError).
		Str("run_id", runID).
		Str("operation", operation).
		Err(err).
		Msg("Persistence error")
}

// WorkflowLogger creates a logger enriched with workflow context
func WorkflowLogger(baseLogger zerolog.Logger, runID, workflowID, resourceID string) zerolog.Logger {
	return baseLogger.With().
		Str("run_id", runID).
		Str("workflow_id", workflowID).
		Str("resource_id", resourceID).
		Logger()
}

// StepLogger creates a logger enriched with step context
func StepLogger(workflowLogger zerolog.Logger, stepID, stepName string, attempt int) zerolog.Logger {
	return workflowLogger.With().
		Str("step_id", stepID).
		Str("step_name", stepName).
		Int("attempt", attempt).
		Logger()
}
