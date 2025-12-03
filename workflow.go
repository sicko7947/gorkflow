package gorkflow

import (
	"fmt"
	"time"
)

// Workflow represents the complete workflow blueprint
type Workflow struct {
	id          string
	name        string
	description string
	version     string

	// Steps registered by ID
	steps map[string]StepExecutor

	// Execution graph
	graph *ExecutionGraph

	// Default config
	config ExecutionConfig

	// Metadata
	tags      map[string]string
	createdAt time.Time

	// Custom context
	customContext any
}

// ID returns the workflow ID
func (w *Workflow) ID() string {
	return w.id
}

// Name returns the workflow name
func (w *Workflow) Name() string {
	return w.name
}

// Description returns the workflow description
func (w *Workflow) Description() string {
	return w.description
}

// Version returns the workflow version
func (w *Workflow) Version() string {
	return w.version
}

// Graph returns the execution graph
func (w *Workflow) Graph() *ExecutionGraph {
	return w.graph
}

// GetStep retrieves a step by ID
func (w *Workflow) GetStep(stepID string) (StepExecutor, error) {
	step, exists := w.steps[stepID]
	if !exists {
		return nil, fmt.Errorf("step %s not found in workflow", stepID)
	}
	return step, nil
}

// GetAllSteps returns all steps
func (w *Workflow) GetAllSteps() map[string]StepExecutor {
	return w.steps
}

// GetConfig returns the default execution config
func (w *Workflow) GetConfig() ExecutionConfig {
	return w.config
}

// GetContext returns the custom context
func (w *Workflow) GetContext() any {
	return w.customContext
}

// WorkflowOption configures a workflow
type WorkflowOption func(*Workflow)

// WithContext sets a custom context for the workflow
func WithContext(ctx any) WorkflowOption {
	return func(w *Workflow) {
		w.customContext = ctx
	}
}

// NewWorkflowInstance creates a new workflow instance
func NewWorkflowInstance(id, name string, opts ...WorkflowOption) *Workflow {
	w := &Workflow{
		id:        id,
		name:      name,
		version:   "1.0",
		steps:     make(map[string]StepExecutor),
		graph:     NewExecutionGraph(),
		config:    DefaultExecutionConfig,
		tags:      make(map[string]string),
		createdAt: time.Now(),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// SetDescription sets the workflow description
func (w *Workflow) SetDescription(description string) {
	w.description = description
}

// SetVersion sets the workflow version
func (w *Workflow) SetVersion(version string) {
	w.version = version
}

// SetConfig sets the execution config
func (w *Workflow) SetConfig(config ExecutionConfig) {
	w.config = config
}

// SetTags sets the workflow tags
func (w *Workflow) SetTags(tags map[string]string) {
	w.tags = tags
}

// AddStep registers a step in the workflow and adds it to the graph
func (w *Workflow) AddStep(step StepExecutor) {
	w.steps[step.GetID()] = step
	// Also ensure node exists in graph
	// Default to Sequential, builder can update type if needed
	w.graph.AddNode(step.GetID(), NodeTypeSequential)
}

// SetContext sets the custom context for the workflow
func (w *Workflow) SetContext(ctx any) {
	w.customContext = ctx
}

// Validate performs comprehensive validation on the workflow
func (w *Workflow) Validate() error {
	// Validate graph structure
	if err := w.graph.Validate(); err != nil {
		return fmt.Errorf("invalid workflow graph: %w", err)
	}

	// Validate all steps in graph exist in workflow
	for stepID := range w.graph.Nodes {
		if _, exists := w.steps[stepID]; !exists {
			return fmt.Errorf("step %s referenced in graph but not registered", stepID)
		}
	}

	return nil
}
