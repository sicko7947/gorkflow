package builder

import (
	"fmt"

	workflow "github.com/sicko7947/gorkflow"
)

// ValidateWorkflow performs comprehensive validation on a workflow
func ValidateWorkflow(w *workflow.Workflow) error {
	// Validate graph
	if err := w.Graph().Validate(); err != nil {
		return fmt.Errorf("invalid workflow graph: %w", err)
	}

	// Validate all steps exist
	for stepID := range w.Graph().Nodes {
		if _, err := w.GetStep(stepID); err != nil {
			return fmt.Errorf("step %s referenced in graph but not registered", stepID)
		}
	}

	return nil
}

// ValidateGraph validates the execution graph structure
func ValidateGraph(graph *workflow.ExecutionGraph) error {
	return graph.Validate()
}

// ValidateNoCycles checks for cycles in the graph
func ValidateNoCycles(graph *workflow.ExecutionGraph) error {
	if len(graph.Nodes) == 0 {
		return nil
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		node := graph.Nodes[nodeID]
		for _, nextID := range node.Next {
			if !visited[nextID] {
				if hasCycle(nextID) {
					return true
				}
			} else if recStack[nextID] {
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	for nodeID := range graph.Nodes {
		if !visited[nodeID] {
			if hasCycle(nodeID) {
				return fmt.Errorf("cycle detected in execution graph")
			}
		}
	}

	return nil
}

// ValidateReachability ensures all nodes are reachable from the entry point
func ValidateReachability(graph *workflow.ExecutionGraph) error {
	if graph.EntryPoint == "" {
		return fmt.Errorf("no entry point set")
	}

	reachable := make(map[string]bool)
	var visit func(string)
	visit = func(nodeID string) {
		if reachable[nodeID] {
			return
		}
		reachable[nodeID] = true

		node := graph.Nodes[nodeID]
		for _, nextID := range node.Next {
			visit(nextID)
		}
	}

	visit(graph.EntryPoint)

	// Check if all nodes are reachable
	for nodeID := range graph.Nodes {
		if !reachable[nodeID] {
			return fmt.Errorf("node %s is not reachable from entry point", nodeID)
		}
	}

	return nil
}

// ValidateStepReferences ensures all referenced steps exist in the workflow
func ValidateStepReferences(w *workflow.Workflow) error {
	for stepID := range w.Graph().Nodes {
		if _, err := w.GetStep(stepID); err != nil {
			return fmt.Errorf("step %s referenced in graph but not registered", stepID)
		}
	}
	return nil
}
