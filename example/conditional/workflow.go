package conditional

import (
	"fmt"

	"github.com/sicko7947/gorkflow"
)

// NewConditionalWorkflow demonstrates conditional step execution using ThenStepIf
// This workflow conditionally executes steps based on input flags
func NewConditionalWorkflow() (*gorkflow.Workflow, error) {
	// Condition: Only double if EnableDoubling flag is true in state
	shouldDouble := func(ctx *gorkflow.StepContext) (bool, error) {
		var enableDoubling bool
		if err := ctx.State.Get("enable_doubling", &enableDoubling); err != nil {
			ctx.Logger.Warn().Err(err).Msg("Failed to get enable_doubling from state, defaulting to false")
			return false, nil
		}

		ctx.Logger.Info().Bool("enable_doubling", enableDoubling).Msg("Evaluating doubling condition")
		return enableDoubling, nil
	}

	// Condition: Only format if the value from the previous step is > 10
	// This demonstrates checking the output of a previous step in the condition
	shouldFormat := func(ctx *gorkflow.StepContext) (bool, error) {
		// "double" is the ID of the NewDoubleStep()
		doubleOut, err := gorkflow.GetOutput[DoubleOutput](ctx, "double")
		if err != nil {
			ctx.Logger.Warn().Err(err).Msg("Failed to get output from 'double' step")
			return false, nil
		}

		// Only run if value is greater than 10
		shouldRun := doubleOut.Value > 10
		ctx.Logger.Info().
			Int("value", doubleOut.Value).
			Bool("should_run", shouldRun).
			Msg("Evaluating formatting condition based on previous step output")
		return shouldRun, nil
	}

	// Default output when doubling is skipped
	doubleDefault := &DoubleOutput{
		Value:   0,
		Doubled: false,
		Message: "Doubling was skipped",
	}

	// Build workflow with conditional steps using ThenStepIf
	wf, err := gorkflow.NewWorkflow("conditional_example", "Conditional Execution Example").
		WithDescription("Demonstrates conditional step execution with ThenStepIf").
		WithVersion("1.0").
		WithConfig(gorkflow.ExecutionConfig{
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
