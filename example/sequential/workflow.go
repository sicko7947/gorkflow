package sequential

import (
	"fmt"

	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/builder"
)

func NewSimpleMathWorkflow() (*gorkflow.Workflow, error) {
	wf, err := builder.NewWorkflow("sequential", "Simple Math Workflow").
		WithDescription("A simple workflow to test the engine").
		WithVersion("1.0").
		WithConfig(gorkflow.ExecutionConfig{
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
