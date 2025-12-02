package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rs/zerolog"
	workflow "github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/builder"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/store"
)

// Simple input/output types
type MathInput struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddOutput struct {
	Result int `json:"result"`
}

type MultiplyOutput struct {
	Result int `json:"result"`
}

type FinalOutput struct {
	Message string `json:"message"`
	Value   int    `json:"value"`
}

// Step 1: Add two numbers
func NewAddStep() *workflow.Step[MathInput, AddOutput] {
	return workflow.NewStep(
		"add",
		"Add Numbers",
		func(ctx *workflow.StepContext, input MathInput) (AddOutput, error) {
			ctx.Logger.Info().
				Int("a", input.A).
				Int("b", input.B).
				Msg("Adding numbers")

			result := input.A + input.B
			return AddOutput{Result: result}, nil
		},
	)
}

// Step 2: Multiply result by 2
func NewMultiplyStep() *workflow.Step[AddOutput, MultiplyOutput] {
	return workflow.NewStep(
		"multiply",
		"Multiply by 2",
		func(ctx *workflow.StepContext, input AddOutput) (MultiplyOutput, error) {
			ctx.Logger.Info().
				Int("input", input.Result).
				Msg("Multiplying by 2")

			result := input.Result * 2
			return MultiplyOutput{Result: result}, nil
		},
	)
}

// Step 3: Format final output
func NewFormatStep() *workflow.Step[MultiplyOutput, FinalOutput] {
	return workflow.NewStep(
		"format",
		"Format Result",
		func(ctx *workflow.StepContext, input MultiplyOutput) (FinalOutput, error) {
			ctx.Logger.Info().
				Int("value", input.Result).
				Msg("Formatting result")

			return FinalOutput{
				Message: "Calculation complete!",
				Value:   input.Result,
			}, nil
		},
	)
}

func main() {
	// Create logger
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
		With().
		Timestamp().
		Logger()

	// Create LibSQL store with local file
	dbPath := "file:./workflow.db"
	libsqlStore, err := store.NewLibSQLStore(dbPath)
	if err != nil {
		log.Fatal("Failed to create LibSQL store:", err)
	}
	defer libsqlStore.Close()

	logger.Info().Str("db_path", dbPath).Msg("LibSQL store initialized")

	// Create workflow
	wf, err := builder.NewWorkflow("simple_math", "Simple Math Workflow").
		WithDescription("A simple workflow demonstrating LibSQL persistence").
		WithVersion("1.0").
		Sequence(
			NewAddStep(),
			NewMultiplyStep(),
			NewFormatStep(),
		).
		Build()

	if err != nil {
		log.Fatal("Failed to build workflow:", err)
	}

	// Create engine with LibSQL store
	eng := engine.NewEngine(libsqlStore, engine.WithLogger(logger))

	ctx := context.Background()

	// Run workflow with input: (5 + 3) * 2 = 16
	fmt.Println("\n=== Running Workflow: (5 + 3) * 2 ===")
	input := MathInput{A: 5, B: 3}

	runID, err := eng.StartWorkflow(ctx, wf, input, workflow.WithSynchronousExecution())
	if err != nil {
		logger.Error().Err(err).Msg("Workflow failed")
		return
	}

	logger.Info().Str("run_id", runID).Msg("Workflow completed successfully")

	// Retrieve and display the workflow run
	run, err := eng.GetRun(ctx, runID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get run")
		return
	}

	logger.Info().
		Str("status", string(run.Status)).
		Str("output", string(run.Output)).
		Msg("Workflow result")

	// List all step executions (demonstrating persistence)
	fmt.Println("\n=== Step Execution History (from database) ===")
	steps, err := libsqlStore.ListStepExecutions(ctx, runID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list step executions")
		return
	}

	for i, step := range steps {
		logger.Info().
			Int("index", i+1).
			Str("step_id", step.StepID).
			Str("status", string(step.Status)).
			Int("execution_index", step.ExecutionIndex).
			Int64("duration_ms", step.DurationMs).
			Msg("Step execution")
	}

	// Run another workflow to demonstrate persistence across runs
	fmt.Println("\n=== Running Second Workflow: (10 + 20) * 2 ===")
	input2 := MathInput{A: 10, B: 20}

	runID2, err := eng.StartWorkflow(ctx, wf, input2, workflow.WithSynchronousExecution())
	if err != nil {
		logger.Error().Err(err).Msg("Workflow failed")
		return
	}

	logger.Info().Str("run_id", runID2).Msg("Second workflow completed")

	// List all workflow runs
	fmt.Println("\n=== All Workflow Runs (from database) ===")
	runs, err := libsqlStore.ListRuns(ctx, workflow.RunFilter{
		WorkflowID: "simple_math",
	})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to list runs")
		return
	}

	for i, r := range runs {
		logger.Info().
			Int("index", i+1).
			Str("run_id", r.RunID).
			Str("status", string(r.Status)).
			Time("created_at", r.CreatedAt).
			Str("output", string(r.Output)).
			Msg("Workflow run")
	}

	fmt.Println("\n✅ All data has been persisted to workflow.db")
	fmt.Println("💡 You can inspect the database using: sqlite3 workflow.db")
}
