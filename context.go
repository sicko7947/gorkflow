package gorkflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
)

// StepContext provides rich context to step handlers
type StepContext struct {
	context.Context

	// Execution metadata
	RunID   string
	StepID  string
	Attempt int

	// Logger (enriched with step context)
	Logger zerolog.Logger

	// Access to other step data (inputs and outputs)
	Data StepDataAccessor

	// Access to workflow-level state
	State StateAccessor

	// Custom context (user-defined)
	CustomContext any
}

// GetContext retrieves the custom context from the step context
func GetContext[T any](ctx *StepContext) (T, error) {
	var zero T
	if ctx.CustomContext == nil {
		return zero, fmt.Errorf("custom context is nil")
	}

	val, ok := ctx.CustomContext.(T)
	if !ok {
		return zero, fmt.Errorf("custom context is not of type %T", zero)
	}
	return val, nil
}

// StepDataAccessor provides type-safe access to other step outputs and inputs
type StepDataAccessor interface {
	// GetOutput retrieves output from a specific step
	GetOutput(stepID string, target interface{}) error

	// GetInput retrieves input from a specific step
	GetInput(stepID string, target interface{}) error

	// HasOutput checks if a step has produced output
	HasOutput(stepID string) bool
}

// GetOutput is a generic function for type-safe output retrieval from StepContext
func GetOutput[T any](ctx *StepContext, stepID string) (T, error) {
	var result T
	err := ctx.Data.GetOutput(stepID, &result)
	return result, err
}

// GetInput is a generic function for type-safe input retrieval from StepContext
func GetInput[T any](ctx *StepContext, stepID string) (T, error) {
	var result T
	err := ctx.Data.GetInput(stepID, &result)
	return result, err
}

// StateAccessor provides type-safe access to workflow state
type StateAccessor interface {
	// Set stores a value in the workflow state
	Set(key string, value interface{}) error

	// Get retrieves a value from the workflow state
	Get(key string, target interface{}) error

	// Delete removes a key from the state
	Delete(key string) error

	// Has checks if a key exists
	Has(key string) bool

	// GetAll retrieves all state data
	GetAll() (map[string][]byte, error)
}

// SetTyped is a generic function for type-safe state setting
func SetTyped[T any](accessor StateAccessor, key string, value T) error {
	return accessor.Set(key, value)
}

// GetTyped is a generic function for type-safe state retrieval
func GetTyped[T any](accessor StateAccessor, key string) (T, error) {
	var result T
	err := accessor.Get(key, &result)
	return result, err
}

// stepAccessor implements StepDataAccessor
type stepAccessor struct {
	runID       string
	store       WorkflowStore
	ctx         context.Context
	outputCache map[string][]byte
	inputCache  map[string][]byte
}

// newStepAccessor creates a new step accessor
func newStepAccessor(runID string, wfStore WorkflowStore) StepDataAccessor {
	return &stepAccessor{
		runID:       runID,
		store:       wfStore,
		ctx:         context.Background(),
		outputCache: make(map[string][]byte),
		inputCache:  make(map[string][]byte),
	}
}

// NewStepAccessor creates a new step accessor (exported)
func NewStepAccessor(runID string, wfStore WorkflowStore) StepDataAccessor {
	return newStepAccessor(runID, wfStore)
}

func (a *stepAccessor) GetOutput(stepID string, target interface{}) error {
	// Check cache first
	if data, ok := a.outputCache[stepID]; ok {
		return json.Unmarshal(data, target)
	}

	// Load from store
	data, err := a.store.LoadStepOutput(a.ctx, a.runID, stepID)
	if err != nil {
		return fmt.Errorf("failed to load output for step %s: %w", stepID, err)
	}

	// Cache it
	a.outputCache[stepID] = data

	// Unmarshal
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal output for step %s: %w", stepID, err)
	}

	return nil
}

func (a *stepAccessor) HasOutput(stepID string) bool {
	// Check cache
	if _, ok := a.outputCache[stepID]; ok {
		return true
	}

	// Check store
	_, err := a.store.LoadStepOutput(a.ctx, a.runID, stepID)
	return err == nil
}

func (a *stepAccessor) GetInput(stepID string, target interface{}) error {
	// Check cache first
	if data, ok := a.inputCache[stepID]; ok {
		return json.Unmarshal(data, target)
	}

	// Load step execution to get the input
	exec, err := a.store.GetStepExecution(a.ctx, a.runID, stepID)
	if err != nil {
		return fmt.Errorf("failed to load step execution for step %s: %w", stepID, err)
	}

	if exec.Input == nil {
		return fmt.Errorf("no input found for step %s", stepID)
	}

	// Cache it
	a.inputCache[stepID] = exec.Input

	// Unmarshal
	if err := json.Unmarshal(exec.Input, target); err != nil {
		return fmt.Errorf("failed to unmarshal input for step %s: %w", stepID, err)
	}

	return nil
}

// stateAccessor implements StateAccessor. Views have independent contexts and
// share the cache and operation lock so persistence and cache updates stay ordered.
type stateAccessor struct {
	runID string
	store WorkflowStore
	ctxMu sync.RWMutex
	ctx   context.Context
	state *stateAccessorCache
}

type stateAccessorCache struct {
	mu    sync.RWMutex
	cache map[string][]byte
}

// NewStateAccessor creates a new state accessor.
func NewStateAccessor(runID string, wfStore WorkflowStore) StateAccessor {
	return &stateAccessor{
		runID: runID,
		store: wfStore,
		ctx:   context.Background(),
		state: &stateAccessorCache{cache: make(map[string][]byte)},
	}
}

// WithStateAccessorContext returns a view with its own store-call context and
// the same workflow state cache. Custom StateAccessor implementations are returned
// unchanged.
func WithStateAccessorContext(accessor StateAccessor, ctx context.Context) StateAccessor {
	if a, ok := accessor.(*stateAccessor); ok {
		return &stateAccessor{runID: a.runID, store: a.store, ctx: ctx, state: a.state}
	}
	return accessor
}

func (a *stateAccessor) context() context.Context {
	a.ctxMu.RLock()
	defer a.ctxMu.RUnlock()
	return a.ctx
}

func (a *stateAccessor) Set(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal state value for key %s: %w", key, err)
	}

	a.state.mu.Lock()
	defer a.state.mu.Unlock()
	if err := a.store.SaveState(a.context(), a.runID, key, data); err != nil {
		return fmt.Errorf("failed to save state for key %s: %w", key, err)
	}
	a.state.cache[key] = data
	return nil
}

func (a *stateAccessor) Get(key string, target interface{}) error {
	a.state.mu.RLock()
	data, ok := a.state.cache[key]
	a.state.mu.RUnlock()
	if ok {
		return json.Unmarshal(data, target)
	}

	// Recheck after obtaining the write lock: another view may have loaded or
	// changed the value while this reader waited.
	a.state.mu.Lock()
	data, ok = a.state.cache[key]
	if !ok {
		var err error
		data, err = a.store.LoadState(a.context(), a.runID, key)
		if err != nil {
			a.state.mu.Unlock()
			return fmt.Errorf("failed to load state for key %s: %w", key, err)
		}
		a.state.cache[key] = data
	}
	a.state.mu.Unlock()

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal state for key %s: %w", key, err)
	}
	return nil
}

func (a *stateAccessor) Delete(key string) error {
	a.state.mu.Lock()
	defer a.state.mu.Unlock()
	if err := a.store.DeleteState(a.context(), a.runID, key); err != nil {
		return fmt.Errorf("failed to delete state for key %s: %w", key, err)
	}
	delete(a.state.cache, key)
	return nil
}

func (a *stateAccessor) Has(key string) bool {
	a.state.mu.RLock()
	defer a.state.mu.RUnlock()
	if _, ok := a.state.cache[key]; ok {
		return true
	}
	_, err := a.store.LoadState(a.context(), a.runID, key)
	return err == nil
}

func (a *stateAccessor) GetAll() (map[string][]byte, error) {
	a.state.mu.Lock()
	defer a.state.mu.Unlock()
	data, err := a.store.GetAllState(a.context(), a.runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all state: %w", err)
	}

	// Replace the snapshot to remove deleted keys, and keep caller-owned bytes
	// separate from the shared cache.
	cache := make(map[string][]byte, len(data))
	for k, v := range data {
		cache[k] = append([]byte(nil), v...)
	}
	a.state.cache = cache
	return data, nil
}

// SetStepAccessorCtx updates the context used by a StepDataAccessor for store calls.
func SetStepAccessorCtx(accessor StepDataAccessor, ctx context.Context) {
	if sa, ok := accessor.(*stepAccessor); ok {
		sa.ctx = ctx
	}
}

// SetStateAccessorCtx updates the context used by a StateAccessor for store calls.
// Safe to call concurrently with other accessor methods.
func SetStateAccessorCtx(accessor StateAccessor, ctx context.Context) {
	if sa, ok := accessor.(*stateAccessor); ok {
		sa.ctxMu.Lock()
		sa.ctx = ctx
		sa.ctxMu.Unlock()
	}
}
