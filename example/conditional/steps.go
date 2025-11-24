package conditional

import (
	"fmt"

	workflow "github.com/sicko7947/gorkflow"
)

func NewSetupStep() *workflow.Step[ConditionalInput, DoubleInput] {
	return workflow.NewStep(
		"setup",
		"Setup Conditional Flags",
		func(ctx *workflow.StepContext, input ConditionalInput) (DoubleInput, error) {
			// Store flags in state for condition evaluation
			ctx.State.Set("enable_doubling", input.EnableDoubling)
			ctx.State.Set("enable_formatting", input.EnableFormatting)

			ctx.Logger.Info().
				Bool("enable_doubling", input.EnableDoubling).
				Bool("enable_formatting", input.EnableFormatting).
				Msg("Conditional flags set in state")

			// Pass the value to the next step
			return DoubleInput{Value: input.Value}, nil
		},
	)
}

func NewDoubleStep() *workflow.Step[DoubleInput, DoubleOutput] {
	return workflow.NewStep(
		"double",
		"Double the Value",
		func(ctx *workflow.StepContext, input DoubleInput) (DoubleOutput, error) {
			doubled := input.Value * 2
			ctx.Logger.Info().
				Int("original", input.Value).
				Int("doubled", doubled).
				Msg("Doubling value")
			return DoubleOutput{
				Value:   doubled,
				Doubled: true,
				Message: fmt.Sprintf("Doubled %d to %d", input.Value, doubled),
			}, nil
		},
	)
}

func NewConditionalFormatStep() *workflow.Step[ConditionalFormatInput, ConditionalFormatOutput] {
	return workflow.NewStep(
		"conditional_format",
		"Format Result Conditionally",
		func(ctx *workflow.StepContext, input ConditionalFormatInput) (ConditionalFormatOutput, error) {
			formatted := fmt.Sprintf("Final value: %d (doubled: %v)", input.Value, input.Doubled)
			if input.Message != "" {
				formatted = fmt.Sprintf("%s | %s", formatted, input.Message)
			}
			ctx.Logger.Info().
				Str("formatted", formatted).
				Msg("Formatting conditional result")
			return ConditionalFormatOutput{
				Value:     input.Value,
				Formatted: formatted,
				Doubled:   input.Doubled,
			}, nil
		},
	)
}
