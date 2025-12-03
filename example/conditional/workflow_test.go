package conditional

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionalWorkflow(t *testing.T) {
	// Initialize engine with memory store
	st := store.NewMemoryStore()
	eng := engine.NewEngine(st)

	// Create workflow
	wf, err := NewConditionalWorkflow()
	require.NoError(t, err)

	tests := []struct {
		name            string
		input           ConditionalInput
		expectedValue   int
		expectFormatted bool
	}{
		{
			name: "Doubling enabled, Result > 10 -> Format runs",
			input: ConditionalInput{
				Value:          6,
				EnableDoubling: true,
			},
			// 6 * 2 = 12. 12 > 10, so format runs.
			expectedValue:   12,
			expectFormatted: true,
		},
		{
			name: "Doubling enabled, Result <= 10 -> Format skipped",
			input: ConditionalInput{
				Value:          5,
				EnableDoubling: true,
			},
			// 5 * 2 = 10. 10 is not > 10, so format skipped.
			expectedValue:   10,
			expectFormatted: false,
		},
		{
			name: "Doubling disabled -> Double skipped (0) -> Format skipped",
			input: ConditionalInput{
				Value:          20,
				EnableDoubling: false,
			},
			// Doubling skipped, returns default value 0.
			// 0 <= 10, so format skipped.
			expectedValue:   0,
			expectFormatted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Run workflow synchronously
			runID, err := eng.StartWorkflow(ctx, wf, tt.input, gorkflow.WithSynchronousExecution())
			require.NoError(t, err)

			// Verify run status
			run, err := eng.GetRun(ctx, runID)
			require.NoError(t, err)
			assert.Equal(t, gorkflow.RunStatusCompleted, run.Status)

			// Get step executions to verify outputs
			executions, err := eng.GetStepExecutions(ctx, runID)
			require.NoError(t, err)

			// Find "double" step execution
			var doubleExec *gorkflow.StepExecution
			for _, exec := range executions {
				if exec.StepID == "double" {
					doubleExec = exec
					break
				}
			}
			require.NotNil(t, doubleExec, "Double step should have executed")

			var doubleOut DoubleOutput
			err = json.Unmarshal(doubleExec.Output, &doubleOut)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedValue, doubleOut.Value)

			// Find "conditional_format" step execution
			var formatExec *gorkflow.StepExecution
			for _, exec := range executions {
				if exec.StepID == "conditional_format" {
					formatExec = exec
					break
				}
			}
			require.NotNil(t, formatExec, "Format step should have executed (even if skipped logic)")

			var formatOut ConditionalFormatOutput
			err = json.Unmarshal(formatExec.Output, &formatOut)
			require.NoError(t, err)

			if tt.expectFormatted {
				assert.NotEmpty(t, formatOut.Formatted, "Should have formatted output")
				assert.Contains(t, formatOut.Formatted, "Final value")
			} else {
				// If skipped, it returns zero value, so Formatted should be empty
				assert.Empty(t, formatOut.Formatted, "Should not have formatted output (skipped)")
			}
		})
	}
}
