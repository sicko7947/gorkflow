package store_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/store"
)

func BenchmarkMemoryStore_CreateAndGetRun(b *testing.B) {
	s := store.NewMemoryStore()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runID := fmt.Sprintf("run-%d", i)
		run := &gorkflow.WorkflowRun{
			RunID:      runID,
			WorkflowID: "bench-wf",
			Status:     gorkflow.RunStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := s.CreateRun(ctx, run); err != nil {
			b.Fatal(err)
		}
		if _, err := s.GetRun(ctx, runID); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryStore_SaveLoadOutput(b *testing.B) {
	s := store.NewMemoryStore()
	ctx := context.Background()
	payload := make([]byte, 100)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	// Pre-create a run
	run := &gorkflow.WorkflowRun{
		RunID:      "bench-run",
		WorkflowID: "bench-wf",
		Status:     gorkflow.RunStatusRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.CreateRun(ctx, run); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stepID := fmt.Sprintf("step-%d", i)
		if err := s.SaveStepOutput(ctx, "bench-run", stepID, payload); err != nil {
			b.Fatal(err)
		}
		if _, err := s.LoadStepOutput(ctx, "bench-run", stepID); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryStore_ListRuns_100Runs(b *testing.B) {
	s := store.NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		run := &gorkflow.WorkflowRun{
			RunID:      fmt.Sprintf("run-%d", i),
			WorkflowID: "bench-wf",
			Status:     gorkflow.RunStatusCompleted,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := s.CreateRun(ctx, run); err != nil {
			b.Fatal(err)
		}
	}

	filter := gorkflow.RunFilter{WorkflowID: "bench-wf"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.ListRuns(ctx, filter); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLibSQL_CreateAndGetRun(b *testing.B) {
	s, cleanup := newBenchLibSQLStore(b)
	defer cleanup()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runID := fmt.Sprintf("run-%d", i)
		run := &gorkflow.WorkflowRun{
			RunID:      runID,
			WorkflowID: "bench-wf",
			Status:     gorkflow.RunStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := s.CreateRun(ctx, run); err != nil {
			b.Fatal(err)
		}
		if _, err := s.GetRun(ctx, runID); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLibSQL_SaveLoadOutput(b *testing.B) {
	s, cleanup := newBenchLibSQLStore(b)
	defer cleanup()
	ctx := context.Background()
	payload := make([]byte, 100)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	run := &gorkflow.WorkflowRun{
		RunID: "bench-run", WorkflowID: "bench-wf",
		Status: gorkflow.RunStatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := s.CreateRun(ctx, run); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stepID := fmt.Sprintf("step-%d", i)
		if err := s.SaveStepOutput(ctx, "bench-run", stepID, payload); err != nil {
			b.Fatal(err)
		}
		if _, err := s.LoadStepOutput(ctx, "bench-run", stepID); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLibSQL_UpdateRun_FullSerialize(b *testing.B) {
	s, cleanup := newBenchLibSQLStore(b)
	defer cleanup()
	ctx := context.Background()

	run := &gorkflow.WorkflowRun{
		RunID: "bench-run", WorkflowID: "bench-wf",
		Status: gorkflow.RunStatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := s.CreateRun(ctx, run); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		run.Progress = float64(i%100) / 100.0
		run.UpdatedAt = time.Now()
		if err := s.UpdateRun(ctx, run); err != nil {
			b.Fatal(err)
		}
	}
}

func newBenchLibSQLStore(b *testing.B) (gorkflow.WorkflowStore, func()) {
	b.Helper()
	dbFile := fmt.Sprintf("./bench_gorkflow_%d.db", time.Now().UnixNano())
	s, err := store.NewLibSQLStore("file:" + dbFile)
	if err != nil {
		b.Fatalf("failed to create LibSQL store: %v", err)
	}
	cleanup := func() {
		s.Close()
		os.Remove(dbFile)
	}
	return s, cleanup
}
