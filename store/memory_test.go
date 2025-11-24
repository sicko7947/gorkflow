package store

import (
	"context"
	"testing"
	"time"

	"github.com/sicko7947/gorkflow"
)

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	if store == nil {
		t.Fatal("NewMemoryStore() returned nil")
	}

	// Verify it implements the interface
	var _ workflow.WorkflowStore = store
}

func TestMemoryStore_CreateRun(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		ResourceID: "resource-1",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := store.CreateRun(ctx, run)
	if err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	// Verify run can be retrieved
	retrieved, err := store.GetRun(ctx, run.RunID)
	if err != nil {
		t.Fatalf("GetRun() failed: %v", err)
	}

	if retrieved.RunID != run.RunID {
		t.Errorf("Retrieved run ID = %s, want %s", retrieved.RunID, run.RunID)
	}
}

func TestMemoryStore_CreateRun_Duplicate(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// First create should succeed
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("First CreateRun() failed: %v", err)
	}

	// Second create with same ID should fail
	err := store.CreateRun(ctx, run)
	if err == nil {
		t.Error("CreateRun() with duplicate ID should have failed")
	}
}

func TestMemoryStore_GetRun_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, err := store.GetRun(ctx, "non-existent")
	if err == nil {
		t.Error("GetRun() with non-existent ID should have failed")
	}
}

func TestMemoryStore_UpdateRun(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	// Update status
	run.Status = workflow.RunStatusRunning
	if err := store.UpdateRun(ctx, run); err != nil {
		t.Fatalf("UpdateRun() failed: %v", err)
	}

	// Verify update
	retrieved, err := store.GetRun(ctx, run.RunID)
	if err != nil {
		t.Fatalf("GetRun() failed: %v", err)
	}

	if retrieved.Status != workflow.RunStatusRunning {
		t.Errorf("Updated status = %s, want %s", retrieved.Status, workflow.RunStatusRunning)
	}
}

func TestMemoryStore_UpdateRun_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:  "non-existent",
		Status: workflow.RunStatusRunning,
	}

	err := store.UpdateRun(ctx, run)
	if err == nil {
		t.Error("UpdateRun() with non-existent ID should have failed")
	}
}

func TestMemoryStore_UpdateRunStatus(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	// Update status with error
	wfErr := &workflow.WorkflowError{
		Code:    "TEST_ERROR",
		Message: "test error",
	}

	if err := store.UpdateRunStatus(ctx, run.RunID, workflow.RunStatusFailed, wfErr); err != nil {
		t.Fatalf("UpdateRunStatus() failed: %v", err)
	}

	// Verify update
	retrieved, err := store.GetRun(ctx, run.RunID)
	if err != nil {
		t.Fatalf("GetRun() failed: %v", err)
	}

	if retrieved.Status != workflow.RunStatusFailed {
		t.Errorf("Updated status = %s, want %s", retrieved.Status, workflow.RunStatusFailed)
	}

	if retrieved.Error == nil {
		t.Error("Error should not be nil")
	} else if retrieved.Error.Code != "TEST_ERROR" {
		t.Errorf("Error code = %s, want TEST_ERROR", retrieved.Error.Code)
	}
}

func TestMemoryStore_ListRuns(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create multiple runs
	runs := []*workflow.WorkflowRun{
		{
			RunID:      "run-1",
			WorkflowID: "workflow-1",
			ResourceID: "resource-1",
			Status:     workflow.RunStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			RunID:      "run-2",
			WorkflowID: "workflow-1",
			ResourceID: "resource-1",
			Status:     workflow.RunStatusRunning,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			RunID:      "run-3",
			WorkflowID: "workflow-2",
			ResourceID: "resource-2",
			Status:     workflow.RunStatusCompleted,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	for _, run := range runs {
		if err := store.CreateRun(ctx, run); err != nil {
			t.Fatalf("CreateRun() failed: %v", err)
		}
	}

	tests := []struct {
		name   string
		filter workflow.RunFilter
		want   int
	}{
		{
			name:   "no filter",
			filter: workflow.RunFilter{},
			want:   3,
		},
		{
			name: "filter by workflow ID",
			filter: workflow.RunFilter{
				WorkflowID: "workflow-1",
			},
			want: 2,
		},
		{
			name: "filter by status",
			filter: workflow.RunFilter{
				Status: &[]workflow.RunStatus{workflow.RunStatusPending}[0],
			},
			want: 1,
		},
		{
			name: "filter by resource ID",
			filter: workflow.RunFilter{
				ResourceID: "resource-1",
			},
			want: 2,
		},
		{
			name: "filter with limit",
			filter: workflow.RunFilter{
				Limit: 2,
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.ListRuns(ctx, tt.filter)
			if err != nil {
				t.Fatalf("ListRuns() failed: %v", err)
			}

			if len(results) != tt.want {
				t.Errorf("ListRuns() returned %d runs, want %d", len(results), tt.want)
			}
		})
	}
}

func TestMemoryStore_CreateStepExecution(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create a run first
	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	exec := &workflow.StepExecution{
		RunID:     "test-run-1",
		StepID:    "step-1",
		Status:    workflow.StepStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.CreateStepExecution(ctx, exec); err != nil {
		t.Fatalf("CreateStepExecution() failed: %v", err)
	}

	// Verify retrieval
	retrieved, err := store.GetStepExecution(ctx, exec.RunID, exec.StepID)
	if err != nil {
		t.Fatalf("GetStepExecution() failed: %v", err)
	}

	if retrieved.StepID != exec.StepID {
		t.Errorf("Retrieved step ID = %s, want %s", retrieved.StepID, exec.StepID)
	}
}

func TestMemoryStore_GetStepExecution_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, err := store.GetStepExecution(ctx, "non-existent-run", "non-existent-step")
	if err == nil {
		t.Error("GetStepExecution() with non-existent ID should have failed")
	}
}

func TestMemoryStore_UpdateStepExecution(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	exec := &workflow.StepExecution{
		RunID:     "test-run-1",
		StepID:    "step-1",
		Status:    workflow.StepStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := store.CreateStepExecution(ctx, exec); err != nil {
		t.Fatalf("CreateStepExecution() failed: %v", err)
	}

	// Update status
	exec.Status = workflow.StepStatusCompleted
	if err := store.UpdateStepExecution(ctx, exec); err != nil {
		t.Fatalf("UpdateStepExecution() failed: %v", err)
	}

	// Verify update
	retrieved, err := store.GetStepExecution(ctx, exec.RunID, exec.StepID)
	if err != nil {
		t.Fatalf("GetStepExecution() failed: %v", err)
	}

	if retrieved.Status != workflow.StepStatusCompleted {
		t.Errorf("Updated status = %s, want %s", retrieved.Status, workflow.StepStatusCompleted)
	}
}

func TestMemoryStore_ListStepExecutions(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	// Create multiple step executions
	steps := []string{"step-1", "step-2", "step-3"}
	for _, stepID := range steps {
		exec := &workflow.StepExecution{
			RunID:     "test-run-1",
			StepID:    stepID,
			Status:    workflow.StepStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := store.CreateStepExecution(ctx, exec); err != nil {
			t.Fatalf("CreateStepExecution() failed: %v", err)
		}
	}

	// List all executions
	executions, err := store.ListStepExecutions(ctx, "test-run-1")
	if err != nil {
		t.Fatalf("ListStepExecutions() failed: %v", err)
	}

	if len(executions) != 3 {
		t.Errorf("ListStepExecutions() returned %d executions, want 3", len(executions))
	}
}

func TestMemoryStore_ListStepExecutions_EmptyRun(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	executions, err := store.ListStepExecutions(ctx, "non-existent-run")
	if err != nil {
		t.Fatalf("ListStepExecutions() failed: %v", err)
	}

	if len(executions) != 0 {
		t.Errorf("ListStepExecutions() for non-existent run should return empty list, got %d", len(executions))
	}
}

func TestMemoryStore_SaveAndLoadStepOutput(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	output := []byte(`{"result": "success"}`)

	// Save output
	if err := store.SaveStepOutput(ctx, "test-run-1", "step-1", output); err != nil {
		t.Fatalf("SaveStepOutput() failed: %v", err)
	}

	// Load output
	loaded, err := store.LoadStepOutput(ctx, "test-run-1", "step-1")
	if err != nil {
		t.Fatalf("LoadStepOutput() failed: %v", err)
	}

	if string(loaded) != string(output) {
		t.Errorf("LoadStepOutput() = %s, want %s", string(loaded), string(output))
	}
}

func TestMemoryStore_LoadStepOutput_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, err := store.LoadStepOutput(ctx, "non-existent-run", "non-existent-step")
	if err == nil {
		t.Error("LoadStepOutput() with non-existent ID should have failed")
	}
}

func TestMemoryStore_SaveAndLoadState(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	value := []byte(`{"key": "value"}`)

	// Save state
	if err := store.SaveState(ctx, "test-run-1", "my-key", value); err != nil {
		t.Fatalf("SaveState() failed: %v", err)
	}

	// Load state
	loaded, err := store.LoadState(ctx, "test-run-1", "my-key")
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	if string(loaded) != string(value) {
		t.Errorf("LoadState() = %s, want %s", string(loaded), string(value))
	}
}

func TestMemoryStore_LoadState_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	_, err := store.LoadState(ctx, "test-run-1", "non-existent-key")
	if err == nil {
		t.Error("LoadState() with non-existent key should have failed")
	}
}

func TestMemoryStore_DeleteState(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	value := []byte(`{"key": "value"}`)

	// Save state
	if err := store.SaveState(ctx, "test-run-1", "my-key", value); err != nil {
		t.Fatalf("SaveState() failed: %v", err)
	}

	// Delete state
	if err := store.DeleteState(ctx, "test-run-1", "my-key"); err != nil {
		t.Fatalf("DeleteState() failed: %v", err)
	}

	// Verify deletion
	_, err := store.LoadState(ctx, "test-run-1", "my-key")
	if err == nil {
		t.Error("LoadState() after DeleteState() should have failed")
	}
}

func TestMemoryStore_GetAllState(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	run := &workflow.WorkflowRun{
		RunID:      "test-run-1",
		WorkflowID: "test-workflow",
		Status:     workflow.RunStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun() failed: %v", err)
	}

	// Save multiple state values
	states := map[string][]byte{
		"key1": []byte(`{"value": 1}`),
		"key2": []byte(`{"value": 2}`),
		"key3": []byte(`{"value": 3}`),
	}

	for key, value := range states {
		if err := store.SaveState(ctx, "test-run-1", key, value); err != nil {
			t.Fatalf("SaveState() failed: %v", err)
		}
	}

	// Get all state
	allState, err := store.GetAllState(ctx, "test-run-1")
	if err != nil {
		t.Fatalf("GetAllState() failed: %v", err)
	}

	if len(allState) != 3 {
		t.Errorf("GetAllState() returned %d items, want 3", len(allState))
	}

	for key, expectedValue := range states {
		value, ok := allState[key]
		if !ok {
			t.Errorf("GetAllState() missing key %s", key)
			continue
		}
		if string(value) != string(expectedValue) {
			t.Errorf("GetAllState()[%s] = %s, want %s", key, string(value), string(expectedValue))
		}
	}
}

func TestMemoryStore_GetAllState_EmptyRun(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	allState, err := store.GetAllState(ctx, "non-existent-run")
	if err != nil {
		t.Fatalf("GetAllState() failed: %v", err)
	}

	if len(allState) != 0 {
		t.Errorf("GetAllState() for non-existent run should return empty map, got %d items", len(allState))
	}
}

func TestMemoryStore_CountRunsByStatus(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create multiple runs
	runs := []*workflow.WorkflowRun{
		{
			RunID:      "run-1",
			WorkflowID: "workflow-1",
			ResourceID: "resource-1",
			Status:     workflow.RunStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			RunID:      "run-2",
			WorkflowID: "workflow-1",
			ResourceID: "resource-1",
			Status:     workflow.RunStatusRunning,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			RunID:      "run-3",
			WorkflowID: "workflow-2",
			ResourceID: "resource-1",
			Status:     workflow.RunStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			RunID:      "run-4",
			WorkflowID: "workflow-3",
			ResourceID: "resource-2",
			Status:     workflow.RunStatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	for _, run := range runs {
		if err := store.CreateRun(ctx, run); err != nil {
			t.Fatalf("CreateRun() failed: %v", err)
		}
	}

	tests := []struct {
		name       string
		resourceID string
		status     workflow.RunStatus
		want       int
	}{
		{
			name:       "count pending for resource-1",
			resourceID: "resource-1",
			status:     workflow.RunStatusPending,
			want:       2,
		},
		{
			name:       "count running for resource-1",
			resourceID: "resource-1",
			status:     workflow.RunStatusRunning,
			want:       1,
		},
		{
			name:       "count pending for resource-2",
			resourceID: "resource-2",
			status:     workflow.RunStatusPending,
			want:       1,
		},
		{
			name:       "count completed for resource-1",
			resourceID: "resource-1",
			status:     workflow.RunStatusCompleted,
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := store.CountRunsByStatus(ctx, tt.resourceID, tt.status)
			if err != nil {
				t.Fatalf("CountRunsByStatus() failed: %v", err)
			}

			if count != tt.want {
				t.Errorf("CountRunsByStatus() = %d, want %d", count, tt.want)
			}
		})
	}
}

func TestMemoryStore_ThreadSafety(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			run := &workflow.WorkflowRun{
				RunID:      string(rune('A' + id)),
				WorkflowID: "test-workflow",
				Status:     workflow.RunStatusPending,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			_ = store.CreateRun(ctx, run)
			_, _ = store.GetRun(ctx, run.RunID)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
