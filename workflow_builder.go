package gorkflow

import (
	"fmt"
)

// WorkflowBuilder provides a fluent API for building workflows
type WorkflowBuilder struct {
	workflow     *Workflow
	lastStepIDs  []string
	currentChain []string
}

// NewWorkflow creates a new workflow builder
func NewWorkflow(id, name string) *WorkflowBuilder {
	return &WorkflowBuilder{
		workflow:     NewWorkflowInstance(id, name),
		lastStepIDs:  []string{},
		currentChain: []string{},
	}
}

// WithDescription sets the workflow description
func (b *WorkflowBuilder) WithDescription(description string) *WorkflowBuilder {
	b.workflow.SetDescription(description)
	return b
}

// WithVersion sets the workflow version
func (b *WorkflowBuilder) WithVersion(version string) *WorkflowBuilder {
	b.workflow.SetVersion(version)
	return b
}

// WithConfig sets the default execution config
func (b *WorkflowBuilder) WithConfig(config ExecutionConfig) *WorkflowBuilder {
	b.workflow.SetConfig(config)
	return b
}

// WithTags sets workflow tags
func (b *WorkflowBuilder) WithTags(tags map[string]string) *WorkflowBuilder {
	b.workflow.SetTags(tags)
	return b
}

// WithContext sets a custom context for the workflow
func (b *WorkflowBuilder) WithContext(ctx any) *WorkflowBuilder {
	b.workflow.SetContext(ctx)
	return b
}

// ThenStep chains the given step after the last added step
func (b *WorkflowBuilder) ThenStep(step StepExecutor) *WorkflowBuilder {
	stepID := step.GetID()

	// Register step if not already registered
	if _, err := b.workflow.GetStep(stepID); err != nil {
		b.workflow.AddStep(step)
		// Note: AddStep now adds node to graph as Sequential by default
	}

	// Chain from last steps
	for _, lastID := range b.lastStepIDs {
		if err := b.workflow.graph.AddEdge(lastID, stepID); err != nil {
			panic(fmt.Sprintf("failed to add edge: %v", err))
		}
	}

	b.lastStepIDs = []string{stepID}
	b.currentChain = append(b.currentChain, stepID)

	return b
}

// Parallel adds multiple steps that execute in parallel after the last step(s)
func (b *WorkflowBuilder) Parallel(steps ...StepExecutor) *WorkflowBuilder {
	var newLastIDs []string
	for _, step := range steps {
		stepID := step.GetID()

		// Register step if not already registered
		if _, err := b.workflow.GetStep(stepID); err != nil {
			b.workflow.AddStep(step)
			// Update node type to Parallel
			if err := b.workflow.graph.UpdateNodeType(stepID, NodeTypeParallel); err != nil {
				panic(fmt.Sprintf("failed to update node type: %v", err))
			}
		}

		// Chain from last steps
		for _, lastID := range b.lastStepIDs {
			if err := b.workflow.graph.AddEdge(lastID, stepID); err != nil {
				panic(fmt.Sprintf("failed to add edge: %v", err))
			}
		}

		newLastIDs = append(newLastIDs, stepID)
		b.currentChain = append(b.currentChain, stepID)
	}

	b.lastStepIDs = newLastIDs
	return b
}

// Sequence adds multiple steps and chains them together in order
func (b *WorkflowBuilder) Sequence(steps ...StepExecutor) *WorkflowBuilder {
	for _, step := range steps {
		b.ThenStep(step)
	}
	return b
}

// ThenStepIf chains a step with a condition after the last added step
// The step executes only if condition evaluates to true at runtime
// If false, defaultValue is used as output (pass nil for zero value)
//
// Example:
//
//	condition := func(ctx *gorkflow.StepContext) (bool, error) {
//	    var shouldProcess bool
//	    ctx.State.Get("should_process", &shouldProcess)
//	    return shouldProcess, nil
//	}
//	builder.ThenStepIf(processStep, condition, nil)
func (b *WorkflowBuilder) ThenStepIf(step StepExecutor, condition Condition, defaultValue any) *WorkflowBuilder {
	// Wrap the step in a conditional wrapper
	wrappedStep := WrapStepWithCondition(step, condition, defaultValue)
	return b.ThenStep(wrappedStep)
}

// SetEntryPoint sets the workflow entry point explicitly
func (b *WorkflowBuilder) SetEntryPoint(stepID string) *WorkflowBuilder {
	if err := b.workflow.graph.SetEntryPoint(stepID); err != nil {
		panic(fmt.Sprintf("failed to set entry point: %v", err))
	}
	return b
}

// Build finalizes and validates the workflow
func (b *WorkflowBuilder) Build() (*Workflow, error) {
	// Validate graph
	if err := b.workflow.graph.Validate(); err != nil {
		return nil, fmt.Errorf("invalid workflow graph: %w", err)
	}

	// Validate all steps exist
	for stepID := range b.workflow.graph.Nodes {
		step, err := b.workflow.GetStep(stepID)
		if err != nil {
			return nil, fmt.Errorf("step %s referenced in graph but not registered", stepID)
		}

		// Apply workflow config to step if step is using default config
		// This allows workflow-level config (e.g. MaxRetries) to propagate to steps
		// unless the step has been explicitly configured with non-default values.
		if step.GetConfig() == DefaultExecutionConfig {
			step.SetConfig(b.workflow.GetConfig())
		}
	}

	return b.workflow, nil
}

// MustBuild finalizes and validates the workflow, panics on error
func (b *WorkflowBuilder) MustBuild() *Workflow {
	wf, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build workflow: %v", err))
	}
	return wf
}
