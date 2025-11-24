package parallel

import (
	"fmt"

	"time"

	"github.com/sicko7947/gorkflow"
)

func NewStartStep() *gorkflow.Step[WorkflowInput, WorkflowInput] {
	return gorkflow.NewStep(
		"start",
		"Start Workflow",
		func(ctx *gorkflow.StepContext, input WorkflowInput) (WorkflowInput, error) {
			ctx.Logger.Info().Msg("Starting parallel workflow")
			return input, nil
		},
	)
}

func NewAddStep() *gorkflow.Step[WorkflowInput, AddOutput] {
	return gorkflow.NewStep(
		"add",
		"Add Numbers",
		func(ctx *gorkflow.StepContext, input WorkflowInput) (AddOutput, error) {
			sum := input.Val1 + input.Val2
			ctx.Logger.Info().Int("val1", input.Val1).Int("val2", input.Val2).Int("sum", sum).Msg("Adding numbers")
			return AddOutput{Value: sum, Mult: input.Mult}, nil
		},
	)
}

func NewMultiplyStep() *gorkflow.Step[WorkflowInput, MultiplyOutput] {
	return gorkflow.NewStep(
		"multiply",
		"Multiply Result",
		func(ctx *gorkflow.StepContext, input WorkflowInput) (MultiplyOutput, error) {
			prod := input.Val1 * input.Mult
			ctx.Logger.Info().Int("val1", input.Val1).Int("mult", input.Mult).Int("product", prod).Msg("Multiplying result")
			return MultiplyOutput{Value: prod}, nil
		},
		// Override default configuration for this step
		gorkflow.WithRetries(5),
		gorkflow.WithBackoff(gorkflow.BackoffExponential),
		gorkflow.WithTimeout(5*time.Second),
	)
}

func NewFormatStep() *gorkflow.Step[MultiplyOutput, FormatOutput] {
	return gorkflow.NewStep(
		"format",
		"Format Output",
		func(ctx *gorkflow.StepContext, input MultiplyOutput) (FormatOutput, error) {
			msg := fmt.Sprintf("The final result is %d", input.Value)
			ctx.Logger.Info().Str("message", msg).Msg("Formatting output")
			return FormatOutput{Message: msg}, nil
		},
	)
}
