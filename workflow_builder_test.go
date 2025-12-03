package gorkflow_test

import (
	"testing"

	"github.com/sicko7947/gorkflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test handler
func testHandler(ctx *gorkflow.StepContext, input interface{}) (interface{}, error) {
	return input, nil
}

func TestNewWorkflow(t *testing.T) {
	builder := gorkflow.NewWorkflow("test-workflow", "Test Workflow")

	assert.NotNil(t, builder)
	// We can't access builder.workflow directly as it is private,
	// but we can verify Build() works
	wf, err := builder.Build()
	require.Error(t, err) // Error because no steps
	assert.Nil(t, wf)
}

func TestWorkflowBuilder_WithDescription(t *testing.T) {
	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		WithDescription("A test workflow").
		ThenStep(gorkflow.NewStep("step1", "Step 1", testHandler)).
		Build()

	require.NoError(t, err)
	assert.Equal(t, "A test workflow", wf.Description())
}

func TestWorkflowBuilder_WithVersion(t *testing.T) {
	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		WithVersion("1.0.0").
		ThenStep(gorkflow.NewStep("step1", "Step 1", testHandler)).
		Build()

	require.NoError(t, err)
	assert.Equal(t, "1.0.0", wf.Version())
}

func TestWorkflowBuilder_WithDefaultConfig(t *testing.T) {
	config := gorkflow.ExecutionConfig{
		MaxRetries:     5,
		TimeoutSeconds: 60,
		RetryBackoff:   gorkflow.BackoffExponential,
	}

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		WithConfig(config).
		ThenStep(gorkflow.NewStep("step1", "Step 1", testHandler)).
		Build()

	require.NoError(t, err)
	assert.Equal(t, config, wf.GetConfig())
}

func TestWorkflowBuilder_Sequence(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)
	step2 := gorkflow.NewStep("step2", "Step 2", testHandler)

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		Sequence(step1, step2).
		Build()

	require.NoError(t, err)

	// Verify steps are registered
	retrievedStep1, err := wf.GetStep("step1")
	require.NoError(t, err)
	retrievedStep2, err := wf.GetStep("step2")
	require.NoError(t, err)

	assert.NotNil(t, retrievedStep1)
	assert.NotNil(t, retrievedStep2)
	assert.Equal(t, "step1", retrievedStep1.GetID())
	assert.Equal(t, "step2", retrievedStep2.GetID())

	// Verify graph connection
	graph := wf.Graph()
	nextSteps, err := graph.GetNextSteps("step1")
	require.NoError(t, err)
	assert.Contains(t, nextSteps, "step2")
}

func TestWorkflowBuilder_ThenStep(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)
	step2 := gorkflow.NewStep("step2", "Step 2", testHandler)

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step1).
		ThenStep(step2).
		Build()

	require.NoError(t, err)

	// Verify steps are connected
	graph := wf.Graph()
	nextSteps, err := graph.GetNextSteps("step1")
	require.NoError(t, err)
	assert.Contains(t, nextSteps, "step2")
}

func TestWorkflowBuilder_SetEntryPoint(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)
	step2 := gorkflow.NewStep("step2", "Step 2", testHandler)

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step2).
		ThenStep(step1). // Connect step2 -> step1
		SetEntryPoint("step2").
		Build()

	require.NoError(t, err)
	assert.Equal(t, "step2", wf.Graph().EntryPoint)
}

func TestWorkflowBuilder_AutoEntryPoint(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)
	step2 := gorkflow.NewStep("step2", "Step 2", testHandler)

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step1).
		ThenStep(step2).
		Build()

	require.NoError(t, err)
	// Entry point should be auto-set to first step (step1)
	assert.Equal(t, "step1", wf.Graph().EntryPoint)
}

func TestWorkflowBuilder_Build_EmptyWorkflow(t *testing.T) {
	_, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no entry point") // Updated assertion
}

func TestWorkflowBuilder_Build_NoEntryPoint(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step1).
		Build()

	require.NoError(t, err)
	// Should auto-set entry point to the only step
	assert.Equal(t, "step1", wf.Graph().EntryPoint)
}

func TestWorkflowBuilder_Build_InvalidGraph(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)
	step2 := gorkflow.NewStep("step2", "Step 2", testHandler)

	// Create a cycle: step1 -> step2 -> step1
	_, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step1).
		ThenStep(step2).
		ThenStep(step1).
		Build()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestWorkflowBuilder_MustBuild_Success(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)

	wf := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step1).
		MustBuild()

	assert.NotNil(t, wf)
	assert.Equal(t, "test-workflow", wf.ID())
}

func TestWorkflowBuilder_MustBuild_Panic(t *testing.T) {
	assert.Panics(t, func() {
		gorkflow.NewWorkflow("test-workflow", "Test Workflow").
			MustBuild()
	})
}

func TestWorkflowBuilder_WithTags(t *testing.T) {
	tags := map[string]string{
		"env":     "test",
		"version": "1.0",
	}

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		WithTags(tags).
		ThenStep(gorkflow.NewStep("step1", "Step 1", testHandler)).
		Build()

	require.NoError(t, err)
	assert.NotNil(t, wf)
}

func TestWorkflowBuilder_Parallel(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)
	step2a := gorkflow.NewStep("step2a", "Step 2a", testHandler)
	step2b := gorkflow.NewStep("step2b", "Step 2b", testHandler)
	step3 := gorkflow.NewStep("step3", "Step 3", testHandler)

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step1).
		Parallel(step2a, step2b).
		ThenStep(step3).
		Build()

	require.NoError(t, err)

	// Verify graph structure
	graph := wf.Graph()

	// step1 -> [step2a, step2b]
	nextSteps1, err := graph.GetNextSteps("step1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"step2a", "step2b"}, nextSteps1)

	// step2a -> step3
	nextSteps2a, err := graph.GetNextSteps("step2a")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"step3"}, nextSteps2a)

	// step2b -> step3
	nextSteps2b, err := graph.GetNextSteps("step2b")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"step3"}, nextSteps2b)

	// Check node types
	node2a := graph.Nodes["step2a"]
	assert.Equal(t, gorkflow.NodeTypeParallel, node2a.Type)

	node2b := graph.Nodes["step2b"]
	assert.Equal(t, gorkflow.NodeTypeParallel, node2b.Type)

	node1 := graph.Nodes["step1"]
	assert.Equal(t, gorkflow.NodeTypeSequential, node1.Type)
}

func TestWorkflowBuilder_ThenStepIf(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)
	step2 := gorkflow.NewStep("step2", "Step 2", testHandler)

	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		return true, nil
	}

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step1).
		ThenStepIf(step2, condition, nil).
		Build()

	require.NoError(t, err)

	// Verify step2 is registered
	retrievedStep2, err := wf.GetStep("step2")
	require.NoError(t, err)
	assert.NotNil(t, retrievedStep2)

	// Verify steps are connected
	graph := wf.Graph()
	nextSteps, err := graph.GetNextSteps("step1")
	require.NoError(t, err)
	assert.Contains(t, nextSteps, "step2")
}

func TestWorkflowBuilder_ThenStepIf_WithDefaultValue(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)
	step2 := gorkflow.NewStep("step2", "Step 2", testHandler)

	defaultValue := "default output"
	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		return false, nil
	}

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step1).
		ThenStepIf(step2, condition, defaultValue).
		Build()

	require.NoError(t, err)
	assert.NotNil(t, wf)

	// Verify step2 is registered (even if conditional)
	retrievedStep2, err := wf.GetStep("step2")
	require.NoError(t, err)
	assert.NotNil(t, retrievedStep2)
}

func TestWorkflowBuilder_ThenStepIf_ConditionFailure(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)
	step2 := gorkflow.NewStep("step2", "Step 2", testHandler)

	// Condition that returns an error
	condition := func(ctx *gorkflow.StepContext) (bool, error) {
		return false, assert.AnError
	}

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step1).
		ThenStepIf(step2, condition, nil).
		Build()

	require.NoError(t, err)
	assert.NotNil(t, wf)
}

func TestWorkflowBuilder_MultipleConditionalSteps(t *testing.T) {
	step1 := gorkflow.NewStep("step1", "Step 1", testHandler)
	step2 := gorkflow.NewStep("step2", "Step 2", testHandler)
	step3 := gorkflow.NewStep("step3", "Step 3", testHandler)
	step4 := gorkflow.NewStep("step4", "Step 4", testHandler)

	condition1 := func(ctx *gorkflow.StepContext) (bool, error) {
		return true, nil
	}

	condition2 := func(ctx *gorkflow.StepContext) (bool, error) {
		return false, nil
	}

	wf, err := gorkflow.NewWorkflow("test-workflow", "Test Workflow").
		ThenStep(step1).
		ThenStepIf(step2, condition1, nil).
		ThenStepIf(step3, condition2, "skipped").
		ThenStep(step4).
		Build()

	require.NoError(t, err)

	// Verify all steps are registered
	graph := wf.Graph()
	assert.NotNil(t, graph.Nodes["step1"])
	assert.NotNil(t, graph.Nodes["step2"])
	assert.NotNil(t, graph.Nodes["step3"])
	assert.NotNil(t, graph.Nodes["step4"])

	// Verify sequential connections
	nextSteps1, err := graph.GetNextSteps("step1")
	require.NoError(t, err)
	assert.Contains(t, nextSteps1, "step2")

	nextSteps2, err := graph.GetNextSteps("step2")
	require.NoError(t, err)
	assert.Contains(t, nextSteps2, "step3")

	nextSteps3, err := graph.GetNextSteps("step3")
	require.NoError(t, err)
	assert.Contains(t, nextSteps3, "step4")
}
