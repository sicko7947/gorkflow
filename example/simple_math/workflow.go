package simple_math

import (
	"fmt"

	workflow "github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/builder"
)

// NewSimpleMathWorkflow constructs the simple math workflow
func NewSimpleMathWorkflow() (*workflow.Workflow, error) {
	wf, err := builder.NewWorkflow("simple_math", "Simple Math Workflow").
		WithDescription("A simple workflow to test the engine").
		WithVersion("1.0").
		WithConfig(workflow.ExecutionConfig{
			MaxRetries:     3,
			RetryDelayMs:   3000,
			TimeoutSeconds: 3,
		}).
		Sequence(
			NewAddStep(),
			NewMultiplyStep(),
			NewFormatStep(),
		).
		Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build workflow: %w", err)
	}

	return wf, nil
}
