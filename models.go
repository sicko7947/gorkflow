package gorkflow

import (
	"encoding/json"
	"time"
)

// RunStatus represents the current state of a workflow execution
type RunStatus string

const (
	RunStatusPending   RunStatus = "PENDING"
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusCompleted RunStatus = "COMPLETED"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusCancelled RunStatus = "CANCELLED"
)

// IsTerminal returns true if the status is a final state
func (s RunStatus) IsTerminal() bool {
	return s == RunStatusCompleted || s == RunStatusFailed || s == RunStatusCancelled
}

// String returns the string representation
func (s RunStatus) String() string {
	return string(s)
}

// StepStatus represents the current state of a step execution
type StepStatus string

const (
	StepStatusPending   StepStatus = "PENDING"
	StepStatusRunning   StepStatus = "RUNNING"
	StepStatusCompleted StepStatus = "COMPLETED"
	StepStatusFailed    StepStatus = "FAILED"
	StepStatusSkipped   StepStatus = "SKIPPED"
	StepStatusRetrying  StepStatus = "RETRYING"
)

// IsTerminal returns true if the status is a final state
func (s StepStatus) IsTerminal() bool {
	return s == StepStatusCompleted || s == StepStatusFailed || s == StepStatusSkipped
}

// String returns the string representation
func (s StepStatus) String() string {
	return string(s)
}

// WorkflowRun represents a single workflow execution instance
type WorkflowRun struct {
	// Identity
	RunID           string `json:"runId"`
	WorkflowID      string `json:"workflowId"`
	WorkflowVersion string `json:"workflowVersion"`

	// Status
	Status   RunStatus `json:"status"`
	Progress float64   `json:"progress"` // 0.0 to 1.0

	// Timing
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`

	// Input/Output (serialized as JSON bytes)
	Input  json.RawMessage `json:"input,omitempty"`
	Output json.RawMessage `json:"output,omitempty"`

	// Error handling
	Error *WorkflowError `json:"error,omitempty"`

	// Metadata
	ResourceID string            `json:"resourceId,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`

	// Custom context (serialized as JSON bytes)
	Context json.RawMessage `json:"context,omitempty"`
}

// StepExecution tracks individual step execution within a workflow run
type StepExecution struct {
	// Identity
	RunID          string `json:"runId"`
	StepID         string `json:"stepId"`
	ExecutionIndex int    `json:"executionIndex"` // For tracking across retries

	// Status
	Status StepStatus `json:"status"`

	// Timing
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	DurationMs  int64      `json:"durationMs"`

	// Input/Output (serialized as JSON bytes)
	Input  json.RawMessage `json:"input,omitempty"`
	Output json.RawMessage `json:"output,omitempty"`

	// Error handling
	Error   *StepError `json:"error,omitempty"`
	Attempt int        `json:"attempt"` // Current retry attempt

	// Metadata
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// WorkflowState holds business data separate from execution metadata
type WorkflowState struct {
	RunID     string            `json:"runId"`
	Data      map[string][]byte `json:"data"` // Key-value store (values are JSON)
	UpdatedAt time.Time         `json:"updatedAt"`
}
