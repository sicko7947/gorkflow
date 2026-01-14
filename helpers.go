package gorkflow

import (
	"encoding/json"
	"fmt"
)

// ToPtr returns a pointer to the given value.
// This is useful for creating pointers to literals or converting values to pointers.
func ToPtr[T any](v T) *T {
	return &v
}

// GetRunContext retrieves and deserializes the custom context from a WorkflowRun
func GetRunContext[T any](run *WorkflowRun) (T, error) {
	var zero T
	if len(run.Context) == 0 {
		return zero, fmt.Errorf("workflow run has no context")
	}

	var result T
	if err := json.Unmarshal(run.Context, &result); err != nil {
		return zero, fmt.Errorf("failed to unmarshal context: %w", err)
	}
	return result, nil
}
