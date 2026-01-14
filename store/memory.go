package store

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/sicko7947/gorkflow"
)

// MemoryStore implements gorkflow.WorkflowStore using in-memory storage (for testing)
type MemoryStore struct {
	runs           map[string]*gorkflow.WorkflowRun
	stepExecutions map[string]map[string]*gorkflow.StepExecution // runID -> stepID -> execution
	stepOutputs    map[string]map[string][]byte                  // runID -> stepID -> output
	state          map[string]map[string][]byte                  // runID -> key -> value
	mu             sync.RWMutex
}

// NewMemoryStore creates a new in-memory workflow store
func NewMemoryStore() gorkflow.WorkflowStore {
	return &MemoryStore{
		runs:           make(map[string]*gorkflow.WorkflowRun),
		stepExecutions: make(map[string]map[string]*gorkflow.StepExecution),
		stepOutputs:    make(map[string]map[string][]byte),
		state:          make(map[string]map[string][]byte),
	}
}

// deepCopyRun creates a deep copy of a WorkflowRun
func deepCopyRun(run *gorkflow.WorkflowRun) *gorkflow.WorkflowRun {
	if run == nil {
		return nil
	}
	runCopy := *run

	// Deep copy Tags map
	if run.Tags != nil {
		runCopy.Tags = make(map[string]string, len(run.Tags))
		for k, v := range run.Tags {
			runCopy.Tags[k] = v
		}
	}

	// Deep copy Input/Output/Context (json.RawMessage is []byte)
	if run.Input != nil {
		runCopy.Input = make([]byte, len(run.Input))
		copy(runCopy.Input, run.Input)
	}
	if run.Output != nil {
		runCopy.Output = make([]byte, len(run.Output))
		copy(runCopy.Output, run.Output)
	}
	if run.Context != nil {
		runCopy.Context = make([]byte, len(run.Context))
		copy(runCopy.Context, run.Context)
	}

	// Deep copy Error
	if run.Error != nil {
		errCopy := *run.Error
		if run.Error.Details != nil {
			errCopy.Details = make(map[string]any, len(run.Error.Details))
			for k, v := range run.Error.Details {
				errCopy.Details[k] = v
			}
		}
		runCopy.Error = &errCopy
	}

	// Deep copy time pointers
	if run.StartedAt != nil {
		t := *run.StartedAt
		runCopy.StartedAt = &t
	}
	if run.CompletedAt != nil {
		t := *run.CompletedAt
		runCopy.CompletedAt = &t
	}

	return &runCopy
}

// deepCopyStepExecution creates a deep copy of a StepExecution
func deepCopyStepExecution(exec *gorkflow.StepExecution) *gorkflow.StepExecution {
	if exec == nil {
		return nil
	}
	execCopy := *exec
	// Note: exec.Error is an error interface, shallow copy is acceptable
	return &execCopy
}

// Workflow run operations

func (s *MemoryStore) CreateRun(ctx context.Context, run *gorkflow.WorkflowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[run.RunID]; exists {
		return fmt.Errorf("workflow run %s already exists", run.RunID)
	}

	s.runs[run.RunID] = deepCopyRun(run)

	// Initialize maps for this run
	s.stepExecutions[run.RunID] = make(map[string]*gorkflow.StepExecution)
	s.stepOutputs[run.RunID] = make(map[string][]byte)
	s.state[run.RunID] = make(map[string][]byte)

	return nil
}

func (s *MemoryStore) GetRun(ctx context.Context, runID string) (*gorkflow.WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	run, exists := s.runs[runID]
	if !exists {
		return nil, gorkflow.ErrRunNotFound
	}

	return deepCopyRun(run), nil
}

func (s *MemoryStore) UpdateRun(ctx context.Context, run *gorkflow.WorkflowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[run.RunID]; !exists {
		return gorkflow.ErrRunNotFound
	}

	s.runs[run.RunID] = deepCopyRun(run)
	return nil
}

func (s *MemoryStore) UpdateRunStatus(ctx context.Context, runID string, status gorkflow.RunStatus, err *gorkflow.WorkflowError) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, exists := s.runs[runID]
	if !exists {
		return gorkflow.ErrRunNotFound
	}

	run.Status = status
	if err != nil {
		errCopy := *err
		if err.Details != nil {
			errCopy.Details = make(map[string]any, len(err.Details))
			for k, v := range err.Details {
				errCopy.Details[k] = v
			}
		}
		run.Error = &errCopy
	} else {
		run.Error = nil
	}

	return nil
}

func (s *MemoryStore) ListRuns(ctx context.Context, filter gorkflow.RunFilter) ([]*gorkflow.WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var runs []*gorkflow.WorkflowRun

	for _, run := range s.runs {
		// Apply filters
		if filter.WorkflowID != "" && run.WorkflowID != filter.WorkflowID {
			continue
		}
		if filter.Status != nil && run.Status != *filter.Status {
			continue
		}
		if filter.ResourceID != "" && run.ResourceID != filter.ResourceID {
			continue
		}

		runs = append(runs, deepCopyRun(run))
	}

	// Sort by created_at DESC to match LibSQL behavior
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})

	// Apply limit after sorting
	if filter.Limit > 0 && len(runs) > filter.Limit {
		runs = runs[:filter.Limit]
	}

	return runs, nil
}

// Step execution operations

func (s *MemoryStore) CreateStepExecution(ctx context.Context, exec *gorkflow.StepExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.stepExecutions[exec.RunID]; !exists {
		s.stepExecutions[exec.RunID] = make(map[string]*gorkflow.StepExecution)
	}

	s.stepExecutions[exec.RunID][exec.StepID] = deepCopyStepExecution(exec)
	return nil
}

func (s *MemoryStore) GetStepExecution(ctx context.Context, runID, stepID string) (*gorkflow.StepExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runExecs, exists := s.stepExecutions[runID]
	if !exists {
		return nil, gorkflow.ErrStepExecutionNotFound
	}

	exec, exists := runExecs[stepID]
	if !exists {
		return nil, gorkflow.ErrStepExecutionNotFound
	}

	return deepCopyStepExecution(exec), nil
}

func (s *MemoryStore) UpdateStepExecution(ctx context.Context, exec *gorkflow.StepExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.stepExecutions[exec.RunID]; !exists {
		return gorkflow.ErrStepExecutionNotFound
	}

	s.stepExecutions[exec.RunID][exec.StepID] = deepCopyStepExecution(exec)
	return nil
}

func (s *MemoryStore) ListStepExecutions(ctx context.Context, runID string) ([]*gorkflow.StepExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runExecs, exists := s.stepExecutions[runID]
	if !exists {
		return []*gorkflow.StepExecution{}, nil
	}

	executions := make([]*gorkflow.StepExecution, 0, len(runExecs))
	for _, exec := range runExecs {
		executions = append(executions, deepCopyStepExecution(exec))
	}

	// Sort by execution index
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].ExecutionIndex < executions[j].ExecutionIndex
	})

	return executions, nil
}

// Step output operations

func (s *MemoryStore) SaveStepOutput(ctx context.Context, runID, stepID string, output []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.stepOutputs[runID]; !exists {
		s.stepOutputs[runID] = make(map[string][]byte)
	}

	// Copy bytes
	outputCopy := make([]byte, len(output))
	copy(outputCopy, output)
	s.stepOutputs[runID][stepID] = outputCopy

	return nil
}

func (s *MemoryStore) LoadStepOutput(ctx context.Context, runID, stepID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runOutputs, exists := s.stepOutputs[runID]
	if !exists {
		return nil, gorkflow.ErrStepOutputNotFound
	}

	output, exists := runOutputs[stepID]
	if !exists {
		return nil, gorkflow.ErrStepOutputNotFound
	}

	// Copy bytes
	outputCopy := make([]byte, len(output))
	copy(outputCopy, output)
	return outputCopy, nil
}

// State operations

func (s *MemoryStore) SaveState(ctx context.Context, runID, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.state[runID]; !exists {
		s.state[runID] = make(map[string][]byte)
	}

	// Copy bytes
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	s.state[runID][key] = valueCopy

	return nil
}

func (s *MemoryStore) LoadState(ctx context.Context, runID, key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runState, exists := s.state[runID]
	if !exists {
		return nil, gorkflow.ErrStateNotFound
	}

	value, exists := runState[key]
	if !exists {
		return nil, gorkflow.ErrStateNotFound
	}

	// Copy bytes
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	return valueCopy, nil
}

func (s *MemoryStore) DeleteState(ctx context.Context, runID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	runState, exists := s.state[runID]
	if !exists {
		return nil // No error if run doesn't exist
	}

	delete(runState, key)
	return nil
}

func (s *MemoryStore) GetAllState(ctx context.Context, runID string) (map[string][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runState, exists := s.state[runID]
	if !exists {
		return make(map[string][]byte), nil
	}

	// Deep copy
	stateCopy := make(map[string][]byte, len(runState))
	for k, v := range runState {
		valueCopy := make([]byte, len(v))
		copy(valueCopy, v)
		stateCopy[k] = valueCopy
	}

	return stateCopy, nil
}

// Query operations

func (s *MemoryStore) CountRunsByStatus(ctx context.Context, resourceID string, status gorkflow.RunStatus) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, run := range s.runs {
		if run.ResourceID == resourceID && run.Status == status {
			count++
		}
	}

	return count, nil
}
