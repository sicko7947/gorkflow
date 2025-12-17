package gorkflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutionGraph(t *testing.T) {
	graph := NewExecutionGraph()

	assert.NotNil(t, graph)
	assert.NotNil(t, graph.Nodes)
	assert.Empty(t, graph.Nodes)
	assert.Empty(t, graph.EntryPoint)
}

func TestExecutionGraph_AddNode(t *testing.T) {
	graph := NewExecutionGraph()

	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeParallel)

	assert.Len(t, graph.Nodes, 2)
	assert.NotNil(t, graph.Nodes["step1"])
	assert.NotNil(t, graph.Nodes["step2"])

	assert.Equal(t, NodeTypeSequential, graph.Nodes["step1"].Type)
	assert.Equal(t, NodeTypeParallel, graph.Nodes["step2"].Type)
}

func TestExecutionGraph_AddEdge(t *testing.T) {
	graph := NewExecutionGraph()

	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)

	err := graph.AddEdge("step1", "step2")
	require.NoError(t, err)

	assert.Contains(t, graph.Nodes["step1"].Next, "step2")
}

func TestExecutionGraph_AddEdge_NonExistentFrom(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step2", NodeTypeSequential)

	err := graph.AddEdge("step1", "step2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "step1 not found")
}

func TestExecutionGraph_AddEdge_NonExistentTo(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)

	err := graph.AddEdge("step1", "step2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "step2 not found")
}

func TestExecutionGraph_SetEntryPoint(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)

	err := graph.SetEntryPoint("step1")
	require.NoError(t, err)
	assert.Equal(t, "step1", graph.EntryPoint)
}

func TestExecutionGraph_SetEntryPoint_NonExistent(t *testing.T) {
	graph := NewExecutionGraph()

	err := graph.SetEntryPoint("step1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestExecutionGraph_Validate_NoSteps(t *testing.T) {
	graph := NewExecutionGraph()

	err := graph.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no entry point")
}

func TestExecutionGraph_Validate_NoEntryPoint(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.EntryPoint = "" // Manually clear entry point to test validation

	err := graph.Validate()
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "no entry point")
	}
}

func TestExecutionGraph_Validate_Cycle(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)
	graph.AddNode("step3", NodeTypeSequential)

	graph.AddEdge("step1", "step2")
	graph.AddEdge("step2", "step3")
	graph.AddEdge("step3", "step1") // Creates cycle
	graph.SetEntryPoint("step1")

	err := graph.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestExecutionGraph_Validate_UnreachableNodes(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)
	graph.AddNode("step3", NodeTypeSequential) // Unreachable

	graph.AddEdge("step1", "step2")
	graph.SetEntryPoint("step1")

	err := graph.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not all nodes are reachable")
}

func TestExecutionGraph_Validate_ValidSequential(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)
	graph.AddNode("step3", NodeTypeSequential)

	graph.AddEdge("step1", "step2")
	graph.AddEdge("step2", "step3")
	graph.SetEntryPoint("step1")

	err := graph.Validate()
	assert.NoError(t, err)
}

func TestExecutionGraph_Validate_ValidDiamond(t *testing.T) {
	// Diamond pattern: step1 -> (step2, step3) -> step4
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)
	graph.AddNode("step3", NodeTypeSequential)
	graph.AddNode("step4", NodeTypeSequential)

	graph.AddEdge("step1", "step2")
	graph.AddEdge("step1", "step3")
	graph.AddEdge("step2", "step4")
	graph.AddEdge("step3", "step4")
	graph.SetEntryPoint("step1")

	err := graph.Validate()
	assert.NoError(t, err)
}

func TestExecutionGraph_TopologicalSort_Linear(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)
	graph.AddNode("step3", NodeTypeSequential)

	graph.AddEdge("step1", "step2")
	graph.AddEdge("step2", "step3")
	graph.SetEntryPoint("step1")

	order, err := graph.TopologicalSort()
	require.NoError(t, err)

	assert.Equal(t, []string{"step1", "step2", "step3"}, order)
}

func TestExecutionGraph_TopologicalSort_Diamond(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)
	graph.AddNode("step3", NodeTypeSequential)
	graph.AddNode("step4", NodeTypeSequential)

	graph.AddEdge("step1", "step2")
	graph.AddEdge("step1", "step3")
	graph.AddEdge("step2", "step4")
	graph.AddEdge("step3", "step4")
	graph.SetEntryPoint("step1")

	order, err := graph.TopologicalSort()
	require.NoError(t, err)

	assert.Len(t, order, 4)
	assert.Equal(t, "step1", order[0]) // step1 must be first
	assert.Equal(t, "step4", order[3]) // step4 must be last
	// step2 and step3 can be in any order in the middle
	assert.Contains(t, order[1:3], "step2")
	assert.Contains(t, order[1:3], "step3")
}

func TestExecutionGraph_TopologicalSort_WithCycle(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)

	graph.AddEdge("step1", "step2")
	graph.AddEdge("step2", "step1")
	graph.SetEntryPoint("step1")

	_, err := graph.TopologicalSort()
	assert.Error(t, err)
}

func TestExecutionGraph_GetTopologicalOrder(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)

	graph.AddEdge("step1", "step2")
	graph.SetEntryPoint("step1")

	order, err := graph.TopologicalSort()
	require.NoError(t, err)
	assert.Equal(t, []string{"step1", "step2"}, order)
}

func TestExecutionGraph_GetNextSteps(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)
	graph.AddNode("step3", NodeTypeSequential)

	graph.AddEdge("step1", "step2")
	graph.AddEdge("step1", "step3")

	nextSteps, err := graph.GetNextSteps("step1")
	require.NoError(t, err)
	assert.Len(t, nextSteps, 2)
	assert.Contains(t, nextSteps, "step2")
	assert.Contains(t, nextSteps, "step3")
}

func TestExecutionGraph_GetNextSteps_Terminal(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)

	nextSteps, err := graph.GetNextSteps("step1")
	require.NoError(t, err)
	assert.Empty(t, nextSteps)
}

func TestExecutionGraph_IsTerminal(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)

	graph.AddEdge("step1", "step2")

	assert.False(t, graph.IsTerminal("step1"))
	assert.True(t, graph.IsTerminal("step2"))
}

func TestExecutionGraph_Clone(t *testing.T) {
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)
	graph.AddEdge("step1", "step2")
	graph.SetEntryPoint("step1")

	cloned := graph.Clone()

	assert.Equal(t, graph.EntryPoint, cloned.EntryPoint)
	assert.Len(t, cloned.Nodes, len(graph.Nodes))

	// Verify deep copy
	cloned.AddNode("step3", NodeTypeSequential)
	assert.Len(t, graph.Nodes, 2)
	assert.Len(t, cloned.Nodes, 3)
}

func TestExecutionGraph_ComplexTopologicalSort(t *testing.T) {
	// More complex graph:
	//       step1
	//      /  |  \
	//   step2 step3 step4
	//      \  |  /
	//       step5
	graph := NewExecutionGraph()
	graph.AddNode("step1", NodeTypeSequential)
	graph.AddNode("step2", NodeTypeSequential)
	graph.AddNode("step3", NodeTypeSequential)
	graph.AddNode("step4", NodeTypeSequential)
	graph.AddNode("step5", NodeTypeSequential)

	graph.AddEdge("step1", "step2")
	graph.AddEdge("step1", "step3")
	graph.AddEdge("step1", "step4")
	graph.AddEdge("step2", "step5")
	graph.AddEdge("step3", "step5")
	graph.AddEdge("step4", "step5")
	graph.SetEntryPoint("step1")

	order, err := graph.TopologicalSort()
	require.NoError(t, err)

	assert.Len(t, order, 5)
	assert.Equal(t, "step1", order[0])
	assert.Equal(t, "step5", order[4])
}

func TestNodeType_String(t *testing.T) {
	assert.Equal(t, "SEQUENTIAL", NodeTypeSequential.String())
	assert.Equal(t, "PARALLEL", NodeTypeParallel.String())
	assert.Equal(t, "CONDITIONAL", NodeTypeConditional.String())
}
