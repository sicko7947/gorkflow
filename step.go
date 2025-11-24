package gorkflow

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// StepHandler is the user-defined function signature for step logic
type StepHandler[TIn, TOut any] func(ctx *StepContext, input TIn) (TOut, error)

// Step is a generic, type-safe step definition
type Step[TIn, TOut any] struct {
	// Identity
	ID          string
	Name        string
	Description string

	// The actual step logic (user-defined function)
	Handler StepHandler[TIn, TOut]

	// Execution configuration
	Config ExecutionConfig

	// Type information (for runtime reflection/validation)
	inputType  reflect.Type
	outputType reflect.Type
}

// StepExecutor is the interface the engine works with (polymorphic)
type StepExecutor interface {
	// Metadata
	GetID() string
	GetName() string
	GetDescription() string
	GetConfig() ExecutionConfig

	// Type information
	InputType() reflect.Type
	OutputType() reflect.Type

	// Execution (works with serializable data)
	Execute(ctx *StepContext, input []byte) (output []byte, err error)

	// Validation
	ValidateInput(data []byte) error
	ValidateOutput(data []byte) error
}

// NewStep creates a new type-safe step
func NewStep[TIn, TOut any](
	id, name string,
	handler StepHandler[TIn, TOut],
	opts ...StepOption,
) *Step[TIn, TOut] {
	s := &Step[TIn, TOut]{
		ID:         id,
		Name:       name,
		Handler:    handler,
		Config:     DefaultExecutionConfig,
		inputType:  reflect.TypeOf((*TIn)(nil)).Elem(),
		outputType: reflect.TypeOf((*TOut)(nil)).Elem(),
	}

	// Apply options
	for _, opt := range opts {
		opt.applyStep(s)
	}

	return s
}

// Implement StepExecutor interface

func (s *Step[TIn, TOut]) GetID() string {
	return s.ID
}

func (s *Step[TIn, TOut]) GetName() string {
	return s.Name
}

func (s *Step[TIn, TOut]) GetDescription() string {
	return s.Description
}

func (s *Step[TIn, TOut]) GetConfig() ExecutionConfig {
	return s.Config
}

func (s *Step[TIn, TOut]) InputType() reflect.Type {
	return s.inputType
}

func (s *Step[TIn, TOut]) OutputType() reflect.Type {
	return s.outputType
}

// Execute runs the step handler with type-safe marshaling
func (s *Step[TIn, TOut]) Execute(ctx *StepContext, inputBytes []byte) ([]byte, error) {
	// Unmarshal input
	var input TIn
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	// Execute user's handler
	output, err := s.Handler(ctx, input)
	if err != nil {
		return nil, err
	}

	// Marshal output
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}

	return outputBytes, nil
}

// ValidateInput validates that data can be unmarshaled to TIn
func (s *Step[TIn, TOut]) ValidateInput(data []byte) error {
	var input TIn
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("invalid input for step %s: %w", s.ID, err)
	}
	return nil
}

// ValidateOutput validates that data can be unmarshaled to TOut
func (s *Step[TIn, TOut]) ValidateOutput(data []byte) error {
	var output TOut
	if err := json.Unmarshal(data, &output); err != nil {
		return fmt.Errorf("invalid output for step %s: %w", s.ID, err)
	}
	return nil
}

// Configuration setters (for functional options)

func (s *Step[TIn, TOut]) SetMaxRetries(max int) {
	s.Config.MaxRetries = max
}

func (s *Step[TIn, TOut]) SetTimeout(seconds int) {
	s.Config.TimeoutSeconds = seconds
}

func (s *Step[TIn, TOut]) SetBackoff(strategy BackoffStrategy) {
	s.Config.RetryBackoff = strategy
}

func (s *Step[TIn, TOut]) SetRetryDelay(ms int) {
	s.Config.RetryDelayMs = ms
}

func (s *Step[TIn, TOut]) SetContinueOnError(continueOnError bool) {
	s.Config.ContinueOnError = continueOnError
}

// Condition is a function that determines if a step should execute
type Condition func(ctx *StepContext) (bool, error)

// ConditionalStep wraps a step with a condition
type ConditionalStep[TIn, TOut any] struct {
	Step      *Step[TIn, TOut]
	Condition Condition
	Default   *TOut
}

// Implement StepExecutor interface for ConditionalStep

func (cs *ConditionalStep[TIn, TOut]) GetID() string {
	return cs.Step.GetID()
}

func (cs *ConditionalStep[TIn, TOut]) GetName() string {
	return cs.Step.GetName()
}

func (cs *ConditionalStep[TIn, TOut]) GetDescription() string {
	return cs.Step.GetDescription()
}

func (cs *ConditionalStep[TIn, TOut]) GetConfig() ExecutionConfig {
	return cs.Step.GetConfig()
}

func (cs *ConditionalStep[TIn, TOut]) InputType() reflect.Type {
	return cs.Step.InputType()
}

func (cs *ConditionalStep[TIn, TOut]) OutputType() reflect.Type {
	return cs.Step.OutputType()
}

func (cs *ConditionalStep[TIn, TOut]) Execute(ctx *StepContext, inputBytes []byte) ([]byte, error) {
	// Evaluate condition
	shouldRun, err := cs.Condition(ctx)
	if err != nil {
		return nil, fmt.Errorf("condition evaluation failed: %w", err)
	}

	if !shouldRun {
		LogStepSkipped(ctx.Logger, ctx.RunID, ctx.StepID, "condition_not_met")
		// Step skipped - return default or zero value
		if cs.Default != nil {
			return json.Marshal(cs.Default)
		}
		var zero TOut
		return json.Marshal(zero)
	}

	// Execute the wrapped step
	return cs.Step.Execute(ctx, inputBytes)
}

func (cs *ConditionalStep[TIn, TOut]) ValidateInput(data []byte) error {
	return cs.Step.ValidateInput(data)
}

func (cs *ConditionalStep[TIn, TOut]) ValidateOutput(data []byte) error {
	return cs.Step.ValidateOutput(data)
}

// NewConditionalStep creates a conditional step wrapper
func NewConditionalStep[TIn, TOut any](
	step *Step[TIn, TOut],
	condition Condition,
	defaultValue *TOut,
) *ConditionalStep[TIn, TOut] {
	return &ConditionalStep[TIn, TOut]{
		Step:      step,
		Condition: condition,
		Default:   defaultValue,
	}
}

// conditionalStepWrapper wraps any StepExecutor with conditional execution logic
// This is used by the builder API to provide builder-level conditional support
type conditionalStepWrapper struct {
	step         StepExecutor
	condition    Condition
	defaultValue any
}

func (w *conditionalStepWrapper) GetID() string {
	return w.step.GetID()
}

func (w *conditionalStepWrapper) GetName() string {
	return w.step.GetName()
}

func (w *conditionalStepWrapper) GetDescription() string {
	return w.step.GetDescription()
}

func (w *conditionalStepWrapper) GetConfig() ExecutionConfig {
	return w.step.GetConfig()
}

func (w *conditionalStepWrapper) InputType() reflect.Type {
	return w.step.InputType()
}

func (w *conditionalStepWrapper) OutputType() reflect.Type {
	return w.step.OutputType()
}

func (w *conditionalStepWrapper) Execute(ctx *StepContext, inputBytes []byte) ([]byte, error) {
	// Evaluate condition
	shouldRun, err := w.condition(ctx)
	if err != nil {
		return nil, fmt.Errorf("condition evaluation failed: %w", err)
	}

	if !shouldRun {
		LogStepSkipped(ctx.Logger, ctx.RunID, ctx.StepID, "condition_not_met")
		// Step skipped - return default or zero value
		if w.defaultValue != nil {
			return json.Marshal(w.defaultValue)
		}
		// Return zero value for the output type
		zeroVal := reflect.Zero(w.step.OutputType()).Interface()
		return json.Marshal(zeroVal)
	}

	// Execute the wrapped step
	return w.step.Execute(ctx, inputBytes)
}

func (w *conditionalStepWrapper) ValidateInput(data []byte) error {
	return w.step.ValidateInput(data)
}

func (w *conditionalStepWrapper) ValidateOutput(data []byte) error {
	return w.step.ValidateOutput(data)
}

// WrapStepWithCondition wraps a StepExecutor with conditional execution logic
// This is the type-erased version used by the builder API
// For type-safe conditional steps, use NewConditionalStep directly
func WrapStepWithCondition(step StepExecutor, condition Condition, defaultValue any) StepExecutor {
	return &conditionalStepWrapper{
		step:         step,
		condition:    condition,
		defaultValue: defaultValue,
	}
}
