package simple_math

import (
	"fmt"
	"time"

	workflow "github.com/sicko7947/gorkflow"
)

func NewAddStep() *workflow.Step[WorkflowInput, AddOutput] {
	return workflow.NewStep(
		"add",
		"Add Numbers",
		func(ctx *workflow.StepContext, input WorkflowInput) (AddOutput, error) {
			sum := input.Val1 + input.Val2
			ctx.Logger.Info().Int("val1", input.Val1).Int("val2", input.Val2).Int("sum", sum).Msg("Adding numbers")
			return AddOutput{Value: sum, Mult: input.Mult}, nil
		},
	)
}

func NewMultiplyStep() *workflow.Step[AddOutput, MultiplyOutput] {
	return workflow.NewStep(
		"multiply",
		"Multiply Result",
		func(ctx *workflow.StepContext, input AddOutput) (MultiplyOutput, error) {
			prod := input.Value * input.Mult
			ctx.Logger.Info().Int("value", input.Value).Int("mult", input.Mult).Int("product", prod).Msg("Multiplying result")
			return MultiplyOutput{Value: prod}, nil
		},
		// Override default configuration for this step
		workflow.WithRetries(5),
		workflow.WithBackoff(workflow.BackoffExponential),
		workflow.WithTimeout(5*time.Second),
	)
}

func NewFormatStep() *workflow.Step[MultiplyOutput, FormatOutput] {
	return workflow.NewStep(
		"format",
		"Format Output",
		func(ctx *workflow.StepContext, input MultiplyOutput) (FormatOutput, error) {
			msg := fmt.Sprintf("The final result is %d", input.Value)
			ctx.Logger.Info().Str("message", msg).Msg("Formatting output")
			return FormatOutput{Message: msg}, nil
		},
	)
}
