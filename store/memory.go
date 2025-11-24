package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/sicko7947/gorkflow"
)

// MemoryStore implements workflow.WorkflowStore using in-memory storage (for testing)
type MemoryStore struct {
	runs           map[string]*workflow.WorkflowRun
	stepExecutions map[string]map[string]*workflow.StepExecution // runID -> stepID -> execution
	stepOutputs    map[string]map[string][]byte                  // runID -> stepID -> output
	state          map[string]map[string][]byte                  // runID -> key -> value
	mu             sync.RWMutex
}

// NewMemoryStore creates a new in-memory workflow store
func NewMemoryStore() workflow.WorkflowStore {
	return &MemoryStore{
		runs:           make(map[string]*workflow.WorkflowRun),
		stepExecutions: make(map[string]map[string]*workflow.StepExecution),
		stepOutputs:    make(map[string]map[string][]byte),
		state:          make(map[string]map[string][]byte),
	}
}

// Workflow run operations

func (s *MemoryStore) CreateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[run.RunID]; exists {
		return fmt.Errorf("workflow run %s already exists", run.RunID)
	}

	// Deep copy
	runCopy := *run
	s.runs[run.RunID] = &runCopy

	// Initialize maps for this run
	s.stepExecutions[run.RunID] = make(map[string]*workflow.StepExecution)
	s.stepOutputs[run.RunID] = make(map[string][]byte)
	s.state[run.RunID] = make(map[string][]byte)

	return nil
}

func (s *MemoryStore) GetRun(ctx context.Context, runID string) (*workflow.WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	run, exists := s.runs[runID]
	if !exists {
		return nil, fmt.Errorf("workflow run %s not found", runID)
	}

	// Deep copy
	runCopy := *run
	return &runCopy, nil
}

func (s *MemoryStore) UpdateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[run.RunID]; !exists {
		return fmt.Errorf("workflow run %s not found", run.RunID)
	}

	// Deep copy
	runCopy := *run
	s.runs[run.RunID] = &runCopy

	return nil
}

func (s *MemoryStore) UpdateRunStatus(ctx context.Context, runID string, status workflow.RunStatus, err *workflow.WorkflowError) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	run, exists := s.runs[runID]
	if !exists {
		return fmt.Errorf("workflow run %s not found", runID)
	}

	run.Status = status
	run.Error = err

	return nil
}

func (s *MemoryStore) ListRuns(ctx context.Context, filter workflow.RunFilter) ([]*workflow.WorkflowRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var runs []*workflow.WorkflowRun

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

		// Deep copy
		runCopy := *run
		runs = append(runs, &runCopy)

		// Apply limit
		if filter.Limit > 0 && len(runs) >= filter.Limit {
			break
		}
	}

	return runs, nil
}

// Step execution operations

func (s *MemoryStore) CreateStepExecution(ctx context.Context, exec *workflow.StepExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.stepExecutions[exec.RunID]; !exists {
		s.stepExecutions[exec.RunID] = make(map[string]*workflow.StepExecution)
	}

	// Deep copy
	execCopy := *exec
	s.stepExecutions[exec.RunID][exec.StepID] = &execCopy

	return nil
}

func (s *MemoryStore) GetStepExecution(ctx context.Context, runID, stepID string) (*workflow.StepExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runExecs, exists := s.stepExecutions[runID]
	if !exists {
		return nil, fmt.Errorf("no step executions for run %s", runID)
	}

	exec, exists := runExecs[stepID]
	if !exists {
		return nil, fmt.Errorf("step execution %s/%s not found", runID, stepID)
	}

	// Deep copy
	execCopy := *exec
	return &execCopy, nil
}

func (s *MemoryStore) UpdateStepExecution(ctx context.Context, exec *workflow.StepExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.stepExecutions[exec.RunID]; !exists {
		return fmt.Errorf("no step executions for run %s", exec.RunID)
	}

	// Deep copy
	execCopy := *exec
	s.stepExecutions[exec.RunID][exec.StepID] = &execCopy

	return nil
}

func (s *MemoryStore) ListStepExecutions(ctx context.Context, runID string) ([]*workflow.StepExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runExecs, exists := s.stepExecutions[runID]
	if !exists {
		return []*workflow.StepExecution{}, nil
	}

	executions := make([]*workflow.StepExecution, 0, len(runExecs))
	for _, exec := range runExecs {
		// Deep copy
		execCopy := *exec
		executions = append(executions, &execCopy)
	}

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
		return nil, fmt.Errorf("no step outputs for run %s", runID)
	}

	output, exists := runOutputs[stepID]
	if !exists {
		return nil, fmt.Errorf("step output %s/%s not found", runID, stepID)
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
		return nil, fmt.Errorf("no state for run %s", runID)
	}

	value, exists := runState[key]
	if !exists {
		return nil, fmt.Errorf("state key %s not found", key)
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
		return fmt.Errorf("no state for run %s", runID)
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
	stateCopy := make(map[string][]byte)
	for k, v := range runState {
		valueCopy := make([]byte, len(v))
		copy(valueCopy, v)
		stateCopy[k] = valueCopy
	}

	return stateCopy, nil
}

// Query operations

func (s *MemoryStore) CountRunsByStatus(ctx context.Context, resourceID string, status workflow.RunStatus) (int, error) {
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
