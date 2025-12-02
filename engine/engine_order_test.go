package engine_test

import (
	"context"
	"testing"

	"github.com/sicko7947/gorkflow"
	"github.com/sicko7947/gorkflow/engine"
	"github.com/sicko7947/gorkflow/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepExecutionOrder(t *testing.T) {
	// Setup
	s := store.NewMemoryStore()
	e := engine.NewEngine(s)
	ctx := context.Background()

	// Create a workflow with multiple steps
	wf := gorkflow.NewWorkflowInstance("order-test", "1.0")

	step1 := gorkflow.NewStep("step-1", "Step 1", func(ctx *gorkflow.StepContext, input []byte) ([]byte, error) {
		return []byte("output-1"), nil
	}, gorkflow.WithoutValidation())

	step2 := gorkflow.NewStep("step-2", "Step 2", func(ctx *gorkflow.StepContext, input []byte) ([]byte, error) {
		return []byte("output-2"), nil
	}, gorkflow.WithoutValidation())

	step3 := gorkflow.NewStep("step-3", "Step 3", func(ctx *gorkflow.StepContext, input []byte) ([]byte, error) {
		return []byte("output-3"), nil
	}, gorkflow.WithoutValidation())

	wf.AddStep(step1)
	wf.AddStep(step2)
	wf.AddStep(step3)

	// Add nodes to graph
	wf.Graph().AddNode("step-1", gorkflow.NodeTypeSequential)
	wf.Graph().AddNode("step-2", gorkflow.NodeTypeSequential)
	wf.Graph().AddNode("step-3", gorkflow.NodeTypeSequential)

	// Define order: 1 -> 2 -> 3
	wf.Graph().SetEntryPoint("step-1")
	wf.Graph().AddEdge("step-1", "step-2")
	wf.Graph().AddEdge("step-2", "step-3")

	// Run workflow
	runID, err := e.StartWorkflow(ctx, wf, nil, gorkflow.WithSynchronousExecution())
	require.NoError(t, err)

	// Get step executions
	execs, err := e.GetStepExecutions(ctx, runID)
	require.NoError(t, err)
	require.Len(t, execs, 3)

	// Verify order
	assert.Equal(t, "step-1", execs[0].StepID)
	assert.Equal(t, "step-2", execs[1].StepID)
	assert.Equal(t, "step-3", execs[2].StepID)

	// Verify ExecutionIndex
	assert.Equal(t, 0, execs[0].ExecutionIndex)
	assert.Equal(t, 1, execs[1].ExecutionIndex)
	assert.Equal(t, 2, execs[2].ExecutionIndex)
}
