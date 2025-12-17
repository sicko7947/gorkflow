package gorkflow

import (
	"fmt"
)

// ExecutionGraph defines the workflow execution flow
type ExecutionGraph struct {
	EntryPoint string
	Nodes      map[string]*GraphNode
}

// GraphNode represents a node in the execution graph
type GraphNode struct {
	StepID     string
	Type       NodeType
	Next       []string
	Previous   []string
	Conditions []Condition
}

// NewExecutionGraph creates a new execution graph
func NewExecutionGraph() *ExecutionGraph {
	return &ExecutionGraph{
		Nodes: make(map[string]*GraphNode),
	}
}

// AddNode adds a node to the graph
func (g *ExecutionGraph) AddNode(stepID string, nodeType NodeType) {
	if _, exists := g.Nodes[stepID]; !exists {
		g.Nodes[stepID] = &GraphNode{
			StepID:     stepID,
			Type:       nodeType,
			Next:       []string{},
			Previous:   []string{},
			Conditions: []Condition{},
		}
	}

	// Set entry point if this is the first node
	if g.EntryPoint == "" {
		g.EntryPoint = stepID
	}
}

// UpdateNodeType updates the type of an existing node
func (g *ExecutionGraph) UpdateNodeType(stepID string, nodeType NodeType) error {
	node, exists := g.Nodes[stepID]
	if !exists {
		return fmt.Errorf("node %s not found", stepID)
	}
	node.Type = nodeType
	return nil
}

// AddEdge adds a directed edge from one step to another
func (g *ExecutionGraph) AddEdge(fromStepID, toStepID string) error {
	fromNode, exists := g.Nodes[fromStepID]
	if !exists {
		return fmt.Errorf("source node %s not found", fromStepID)
	}

	if _, exists := g.Nodes[toStepID]; !exists {
		return fmt.Errorf("target node %s not found", toStepID)
	}

	// Add edge
	fromNode.Next = append(fromNode.Next, toStepID)

	// Add reverse edge
	toNode := g.Nodes[toStepID]
	toNode.Previous = append(toNode.Previous, fromStepID)

	return nil
}

// SetEntryPoint sets the entry point of the graph
func (g *ExecutionGraph) SetEntryPoint(stepID string) error {
	if _, exists := g.Nodes[stepID]; !exists {
		return fmt.Errorf("step %s not found in graph", stepID)
	}
	g.EntryPoint = stepID
	return nil
}

// Validate validates the graph structure
func (g *ExecutionGraph) Validate() error {
	if g.EntryPoint == "" {
		return fmt.Errorf("execution graph has no entry point")
	}

	if _, exists := g.Nodes[g.EntryPoint]; !exists {
		return fmt.Errorf("entry point %s not found in graph", g.EntryPoint)
	}

	// Check for cycles (simple DFS-based cycle detection)
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for nodeID := range g.Nodes {
		if !visited[nodeID] {
			if g.hasCycle(nodeID, visited, recStack) {
				return fmt.Errorf("execution graph contains cycles")
			}
		}
	}

	// Check that all nodes are reachable from entry point
	reachable := g.getReachableNodes(g.EntryPoint)
	if len(reachable) != len(g.Nodes) {
		return fmt.Errorf("not all nodes are reachable from entry point")
	}

	return nil
}

// hasCycle performs DFS to detect cycles
func (g *ExecutionGraph) hasCycle(nodeID string, visited, recStack map[string]bool) bool {
	visited[nodeID] = true
	recStack[nodeID] = true

	node := g.Nodes[nodeID]
	for _, nextID := range node.Next {
		if !visited[nextID] {
			if g.hasCycle(nextID, visited, recStack) {
				return true
			}
		} else if recStack[nextID] {
			return true
		}
	}

	recStack[nodeID] = false
	return false
}

// getReachableNodes returns all nodes reachable from the given start node
func (g *ExecutionGraph) getReachableNodes(startID string) map[string]bool {
	reachable := make(map[string]bool)
	g.dfsReachable(startID, reachable)
	return reachable
}

// dfsReachable performs DFS to find all reachable nodes
func (g *ExecutionGraph) dfsReachable(nodeID string, reachable map[string]bool) {
	reachable[nodeID] = true

	node := g.Nodes[nodeID]
	for _, nextID := range node.Next {
		if !reachable[nextID] {
			g.dfsReachable(nextID, reachable)
		}
	}
}

// TopologicalSort returns nodes in topological order
func (g *ExecutionGraph) TopologicalSort() ([]string, error) {
	// Check if graph is valid
	if err := g.Validate(); err != nil {
		return nil, err
	}

	visited := make(map[string]bool)
	stack := []string{}

	// Perform topological sort using DFS
	var visit func(string) error
	visit = func(nodeID string) error {
		if visited[nodeID] {
			return nil
		}

		visited[nodeID] = true

		node := g.Nodes[nodeID]
		for _, nextID := range node.Next {
			if err := visit(nextID); err != nil {
				return err
			}
		}

		stack = append([]string{nodeID}, stack...)
		return nil
	}

	// Start from entry point
	if err := visit(g.EntryPoint); err != nil {
		return nil, err
	}

	return stack, nil
}

// GetNextSteps returns the next steps to execute after the given step
func (g *ExecutionGraph) GetNextSteps(stepID string) ([]string, error) {
	node, exists := g.Nodes[stepID]
	if !exists {
		return nil, fmt.Errorf("step %s not found in graph", stepID)
	}

	return node.Next, nil
}

// GetPreviousSteps returns the steps that lead to the given step
func (g *ExecutionGraph) GetPreviousSteps(stepID string) ([]string, error) {
	node, exists := g.Nodes[stepID]
	if !exists {
		return nil, fmt.Errorf("step %s not found in graph", stepID)
	}
	return node.Previous, nil
}

// IsTerminal returns true if the step has no outgoing edges
func (g *ExecutionGraph) IsTerminal(stepID string) bool {
	node, exists := g.Nodes[stepID]
	if !exists {
		return false
	}
	return len(node.Next) == 0
}

// Clone creates a deep copy of the graph
func (g *ExecutionGraph) Clone() *ExecutionGraph {
	clone := &ExecutionGraph{
		EntryPoint: g.EntryPoint,
		Nodes:      make(map[string]*GraphNode),
	}

	for stepID, node := range g.Nodes {
		clone.Nodes[stepID] = &GraphNode{
			StepID:   node.StepID,
			Type:     node.Type,
			Next:     append([]string{}, node.Next...),
			Previous: append([]string{}, node.Previous...),
			// Note: Conditions are not cloned as they're functions
		}
	}

	return clone
}
