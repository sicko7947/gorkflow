package conditional

import (
	"fmt"

	workflow "github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/builder"
)

// NewConditionalWorkflow demonstrates conditional step execution using ThenStepIf
// This workflow conditionally executes steps based on input flags
func NewConditionalWorkflow() (*workflow.Workflow, error) {
	// Condition: Only double if EnableDoubling flag is true in state
	shouldDouble := func(ctx *workflow.StepContext) (bool, error) {
		var enableDoubling bool
		if err := ctx.State.Get("enable_doubling", &enableDoubling); err != nil {
			ctx.Logger.Warn().Err(err).Msg("Failed to get enable_doubling from state, defaulting to false")
			return false, nil
		}

		ctx.Logger.Info().Bool("enable_doubling", enableDoubling).Msg("Evaluating doubling condition")
		return enableDoubling, nil
	}

	// Condition: Only format if EnableFormatting flag is true in state
	shouldFormat := func(ctx *workflow.StepContext) (bool, error) {
		var enableFormatting bool
		if err := ctx.State.Get("enable_formatting", &enableFormatting); err != nil {
			ctx.Logger.Warn().Err(err).Msg("Failed to get enable_formatting from state, defaulting to false")
			return false, nil
		}

		ctx.Logger.Info().Bool("enable_formatting", enableFormatting).Msg("Evaluating formatting condition")
		return enableFormatting, nil
	}

	// Default output when doubling is skipped
	doubleDefault := &DoubleOutput{
		Value:   0,
		Doubled: false,
		Message: "Doubling was skipped",
	}

	// Build workflow with conditional steps using ThenStepIf
	wf, err := builder.NewWorkflow("conditional_example", "Conditional Execution Example").
		WithDescription("Demonstrates conditional step execution with ThenStepIf").
		WithVersion("1.0").
		WithConfig(workflow.ExecutionConfig{
			MaxRetries:     2,
			RetryDelayMs:   1000,
			TimeoutSeconds: 10,
		}).
		// Step 1: Setup - extract flags from input
		ThenStep(NewSetupStep()).
		// Step 2: Conditionally double the value (using ThenStepIf)
		ThenStepIf(NewDoubleStep(), shouldDouble, doubleDefault).
		// Step 3: Conditionally format the result (using ThenStepIf)
		ThenStepIf(NewConditionalFormatStep(), shouldFormat, nil).
		Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build conditional workflow: %w", err)
	}

	return wf, nil
}
