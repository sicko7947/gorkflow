package engine_test

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/store"
)

func BenchmarkWorkflow_Sequential_10Steps(b *testing.B) {
	eng, _ := createBenchEngine(b)

	steps := make([]gorkflow.StepExecutor, 10)
	for i := range steps {
		id := fmt.Sprintf("step-%d", i)
		steps[i] = gorkflow.NewStep(id, id, func(ctx *gorkflow.StepContext, input any) (any, error) {
			return input, nil
		})
	}

	builder := gorkflow.NewWorkflow("bench-seq-10", "Bench Sequential 10").ThenStep(steps[0])
	for i := 1; i < len(steps); i++ {
		builder = builder.ThenStep(steps[i])
	}
	wf, err := builder.Build()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runID, err := eng.StartWorkflow(context.Background(), wf, nil)
		if err != nil {
			b.Fatal(err)
		}
		waitForBenchCompletion(b, eng, runID)
	}
}

func BenchmarkWorkflow_Sequential_100Steps(b *testing.B) {
	eng, _ := createBenchEngine(b)

	steps := make([]gorkflow.StepExecutor, 100)
	for i := range steps {
		id := fmt.Sprintf("step-%d", i)
		steps[i] = gorkflow.NewStep(id, id, func(ctx *gorkflow.StepContext, input any) (any, error) {
			return input, nil
		})
	}

	builder := gorkflow.NewWorkflow("bench-seq-100", "Bench Sequential 100").ThenStep(steps[0])
	for i := 1; i < len(steps); i++ {
		builder = builder.ThenStep(steps[i])
	}
	wf, err := builder.Build()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runID, err := eng.StartWorkflow(context.Background(), wf, nil)
		if err != nil {
			b.Fatal(err)
		}
		waitForBenchCompletion(b, eng, runID)
	}
}

func BenchmarkWorkflow_Parallel_FanOut_4(b *testing.B) {
	benchParallelFanOut(b, 4)
}

func BenchmarkWorkflow_Parallel_FanOut_16(b *testing.B) {
	benchParallelFanOut(b, 16)
}

// benchParallelFanOut runs: entry → N parallel noop steps
func benchParallelFanOut(b *testing.B, fanWidth int) {
	b.Helper()
	eng, _ := createBenchEngine(b)

	entry := gorkflow.NewStep("entry", "entry", func(ctx *gorkflow.StepContext, input any) (any, error) {
		return input, nil
	})
	builder := gorkflow.NewWorkflow(
		fmt.Sprintf("bench-fanout-%d", fanWidth),
		fmt.Sprintf("Bench FanOut %d", fanWidth),
	).ThenStep(entry)
	leaves := make([]gorkflow.StepExecutor, fanWidth)
	for i := range leaves {
		id := fmt.Sprintf("leaf-%d", i)
		leaves[i] = gorkflow.NewStep(id, id, func(ctx *gorkflow.StepContext, input any) (any, error) {
			return input, nil
		})
	}
	builder = builder.Parallel(leaves...)
	wf, err := builder.Build()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runID, err := eng.StartWorkflow(context.Background(), wf, nil, gorkflow.WithSynchronousExecution())
		if err != nil {
			b.Fatal(err)
		}
		_ = runID
	}
}

func BenchmarkEngine_StartWorkflow_Async(b *testing.B) {
	eng, _ := createBenchEngine(b)

	step := gorkflow.NewStep("noop", "Noop", func(ctx *gorkflow.StepContext, input any) (any, error) {
		return input, nil
	})

	wf, err := gorkflow.NewWorkflow("bench-async", "Bench Async").
		ThenStep(step).
		Build()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eng.StartWorkflow(context.Background(), wf, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func createBenchEngine(b *testing.B) (*engine.Engine, gorkflow.WorkflowStore) {
	b.Helper()
	wfStore := store.NewMemoryStore()
	nopLogger := zerolog.New(io.Discard)
	eng := engine.NewEngine(wfStore,
		engine.WithLogger(nopLogger),
		engine.WithConfig(gorkflow.EngineConfig{
			MaxConcurrentWorkflows: 1000,
			DefaultTimeout:         5 * time.Minute,
		}),
	)
	return eng, wfStore
}

func waitForBenchCompletion(b *testing.B, eng *engine.Engine, runID string) {
	b.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.Fatal("Timeout waiting for workflow completion")
		case <-ticker.C:
			run, err := eng.GetRun(context.Background(), runID)
			if err != nil {
				b.Fatal(err)
			}
			if run.Status.IsTerminal() {
				return
			}
		}
	}
}
