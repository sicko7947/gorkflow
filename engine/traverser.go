package engine

import (
	"fmt"

	workflow "github.com/sicko7947/gorkflow"
)

// GraphTraverser handles traversal of the execution graph
type GraphTraverser struct {
	graph *workflow.ExecutionGraph
}

// NewGraphTraverser creates a new graph traverser
func NewGraphTraverser(graph *workflow.ExecutionGraph) *GraphTraverser {
	return &GraphTraverser{
		graph: graph,
	}
}

// GetExecutionOrder returns the step IDs in topological order
func (t *GraphTraverser) GetExecutionOrder() ([]string, error) {
	// Perform topological sort
	return t.graph.TopologicalSort()
}

// GetNextSteps returns the immediate next steps for a given step
func (t *GraphTraverser) GetNextSteps(stepID string) ([]string, error) {
	node, exists := t.graph.Nodes[stepID]
	if !exists {
		return nil, fmt.Errorf("step %s not found in graph", stepID)
	}

	return node.Next, nil
}

// IsParallel checks if a step should be executed in parallel with others
func (t *GraphTraverser) IsParallel(stepID string) bool {
	node, exists := t.graph.Nodes[stepID]
	if !exists {
		return false
	}

	return node.Type == workflow.NodeTypeParallel
}

// HasConditions checks if a step has conditional execution
func (t *GraphTraverser) HasConditions(stepID string) bool {
	node, exists := t.graph.Nodes[stepID]
	if !exists {
		return false
	}

	return node.Type == workflow.NodeTypeConditional && len(node.Conditions) > 0
}
