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
	RunID           string `json:"runId" dynamodbav:"run_id"`
	WorkflowID      string `json:"workflowId" dynamodbav:"workflow_id"`
	WorkflowVersion string `json:"workflowVersion" dynamodbav:"workflow_version"`

	// Status
	Status   RunStatus `json:"status" dynamodbav:"status"`
	Progress float64   `json:"progress" dynamodbav:"progress"` // 0.0 to 1.0

	// Timing
	CreatedAt   time.Time  `json:"createdAt" dynamodbav:"created_at"`
	StartedAt   *time.Time `json:"startedAt,omitempty" dynamodbav:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty" dynamodbav:"completed_at,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt" dynamodbav:"updated_at"`

	// Input/Output (serialized as JSON bytes)
	Input  json.RawMessage `json:"input,omitempty" dynamodbav:"input,omitempty"`
	Output json.RawMessage `json:"output,omitempty" dynamodbav:"output,omitempty"`

	// Error handling
	Error *WorkflowError `json:"error,omitempty" dynamodbav:"error,omitempty"`

	// Metadata
	ResourceID string            `json:"resourceId,omitempty" dynamodbav:"resource_id,omitempty"`
	Tags       map[string]string `json:"tags,omitempty" dynamodbav:"tags,omitempty"`

	// Custom context (serialized as JSON bytes)
	Context json.RawMessage `json:"context,omitempty" dynamodbav:"context,omitempty"`

	// DynamoDB TTL
	TTL int64 `json:"-" dynamodbav:"ttl,omitempty"`
}

// StepExecution tracks individual step execution within a workflow run
type StepExecution struct {
	// Identity
	RunID          string `json:"runId" dynamodbav:"run_id"`
	StepID         string `json:"stepId" dynamodbav:"step_id"`
	ExecutionIndex int    `json:"executionIndex" dynamodbav:"execution_index"` // For tracking across retries

	// Status
	Status StepStatus `json:"status" dynamodbav:"status"`

	// Timing
	StartedAt   *time.Time `json:"startedAt,omitempty" dynamodbav:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty" dynamodbav:"completed_at,omitempty"`
	DurationMs  int64      `json:"durationMs" dynamodbav:"duration_ms"`

	// Input/Output (serialized as JSON bytes)
	Input  json.RawMessage `json:"input,omitempty" dynamodbav:"input,omitempty"`
	Output json.RawMessage `json:"output,omitempty" dynamodbav:"output,omitempty"`

	// Error handling
	Error   *StepError `json:"error,omitempty" dynamodbav:"error,omitempty"`
	Attempt int        `json:"attempt" dynamodbav:"attempt"` // Current retry attempt

	// Metadata
	CreatedAt time.Time `json:"createdAt" dynamodbav:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" dynamodbav:"updated_at"`
}

// WorkflowState holds business data separate from execution metadata
type WorkflowState struct {
	RunID     string            `json:"runId" dynamodbav:"run_id"`
	Data      map[string][]byte `json:"data" dynamodbav:"data"` // Key-value store (values are JSON)
	UpdatedAt time.Time         `json:"updatedAt" dynamodbav:"updated_at"`
}

// NodeType defines the type of graph node
type NodeType string

const (
	NodeTypeSequential  NodeType = "SEQUENTIAL"
	NodeTypeParallel    NodeType = "PARALLEL"
	NodeTypeConditional NodeType = "CONDITIONAL"
)

// String returns the string representation
func (n NodeType) String() string {
	return string(n)
}
