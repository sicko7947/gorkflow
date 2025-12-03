package parallel

import (
	"fmt"

	"github.com/sicko7947/gorkflow"
)

func NewSimpleMathWorkflow() (*gorkflow.Workflow, error) {
	wf, err := gorkflow.NewWorkflow("simple_math", "Simple Math Workflow").
		WithDescription("A simple workflow to test the engine").
		WithVersion("1.0").
		WithConfig(gorkflow.ExecutionConfig{
			MaxRetries:     3,
			RetryDelayMs:   3000,
			TimeoutSeconds: 3,
		}).
		Sequence(
			NewStartStep(),
		).
		Parallel(
			NewAddStep(),
			NewMultiplyStep(),
		).
		Sequence(
			NewFormatStep(),
		).
		Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build workflow: %w", err)
	}

	return wf, nil
}
