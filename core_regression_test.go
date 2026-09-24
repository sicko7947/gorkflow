package gorkflow

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestCustomValidatorIsScopedToStep(t *testing.T) {
	type input struct {
		Value string `validate:"required"`
	}
	handler := func(_ *StepContext, in input) (input, error) { return in, nil }
	before := NewStep("before", "Before", handler)
	custom := validator.New()
	custom.SetTagName("custom")
	step := NewStep("custom", "Custom", handler, WithCustomValidator(custom))
	after := NewStep("after", "After", handler)
	data := []byte(`{"Value":""}`)
	require.NoError(t, step.ValidateInput(data))
	require.Error(t, before.ValidateInput(data))
	require.Error(t, after.ValidateInput(data))
}

func TestComputeLevelsUsesLongestDependencyPath(t *testing.T) {
	graph := NewExecutionGraph()
	for _, id := range []string{"entry", "a", "b", "join", "leaf"} {
		graph.AddNode(id, NodeTypeSequential)
	}
	for _, edge := range [][2]string{{"entry", "join"}, {"entry", "a"}, {"a", "b"}, {"b", "join"}, {"join", "leaf"}} {
		require.NoError(t, graph.AddEdge(edge[0], edge[1]))
	}
	levels, err := graph.ComputeLevels()
	require.NoError(t, err)
	require.Equal(t, [][]string{{"entry"}, {"a"}, {"b"}, {"join"}, {"leaf"}}, levels)
	// Invalidating both caches must account for the new dependency.
	graph.AddNode("tail", NodeTypeSequential)
	require.NoError(t, graph.AddEdge("leaf", "tail"))
	levels, err = graph.ComputeLevels()
	require.NoError(t, err)
	require.Equal(t, []string{"tail"}, levels[5])
}
