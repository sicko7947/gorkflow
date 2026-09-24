package gorkflow_test

import (
	"fmt"
	"testing"

	"github.com/sicko7947/gorkflow"
)

func BenchmarkGraph_TopologicalSort_10Nodes(b *testing.B) {
	graph := buildLinearGraph(10)

	// Warm up cache
	graph.TopologicalSort()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGraph_TopologicalSort_10Nodes_Cold(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		graph := buildLinearGraph(10)
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGraph_TopologicalSort_100Nodes(b *testing.B) {
	graph := buildLinearGraph(100)

	// Warm up cache (mirrors 10-node variant for a fair hot-path comparison)
	graph.TopologicalSort()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.TopologicalSort()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGraph_ComputeLevels_FanOut_16(b *testing.B) {
	// 1 entry node → 16 parallel leaves
	graph := gorkflow.NewExecutionGraph()
	graph.AddNode("entry", gorkflow.NodeTypeSequential)
	for i := 0; i < 16; i++ {
		id := fmt.Sprintf("leaf-%d", i)
		graph.AddNode(id, gorkflow.NodeTypeParallel)
		graph.AddEdge("entry", id)
	}
	graph.SetEntryPoint("entry")

	// Warm up cache
	graph.ComputeLevels()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := graph.ComputeLevels()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGraph_ComputeLevels_FanOut_16_Cold(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		graph := gorkflow.NewExecutionGraph()
		graph.AddNode("entry", gorkflow.NodeTypeSequential)
		for j := 0; j < 16; j++ {
			id := fmt.Sprintf("leaf-%d", j)
			graph.AddNode(id, gorkflow.NodeTypeParallel)
			graph.AddEdge("entry", id)
		}
		graph.SetEntryPoint("entry")
		if _, err := graph.ComputeLevels(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGraph_Clone_10Nodes(b *testing.B) {
	graph := buildLinearGraph(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = graph.Clone()
	}
}

// Shortcuts reach nodes before their longest dependency path is discovered.
func BenchmarkGraph_ComputeLevels_Shortcuts_100_Cold(b *testing.B) {
	graph := buildLinearGraph(100)
	for i := 2; i < 100; i++ {
		if err := graph.AddEdge("step-0", fmt.Sprintf("step-%d", i)); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Invalidate the caches without including graph construction in the timing.
		if err := graph.SetEntryPoint("step-0"); err != nil {
			b.Fatal(err)
		}
		if _, err := graph.ComputeLevels(); err != nil {
			b.Fatal(err)
		}
	}
}

func buildLinearGraph(n int) *gorkflow.ExecutionGraph {
	graph := gorkflow.NewExecutionGraph()
	for i := 0; i < n; i++ {
		graph.AddNode(fmt.Sprintf("step-%d", i), gorkflow.NodeTypeSequential)
	}
	for i := 0; i < n-1; i++ {
		graph.AddEdge(fmt.Sprintf("step-%d", i), fmt.Sprintf("step-%d", i+1))
	}
	graph.SetEntryPoint("step-0")
	return graph
}
