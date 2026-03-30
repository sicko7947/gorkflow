# Benchmark & Stress Test Results

Baseline results captured on 2026-03-30. Re-run after significant changes to track regressions.

**Environment:** linux/amd64, 32-core x86_64, Go 1.26.1

---

## How to Run

```bash
# Benchmarks
go test -bench=. -benchmem ./...

# Stress tests (build tag required)
go test -tags=stress -run TestStress ./engine/...

# Both with race detector
go test -race -bench=. -benchmem ./...
go test -race -tags=stress -run TestStress ./engine/...
```

---

## Stress Tests

Run with: `go test -race -tags=stress -run TestStress ./engine/...`

| Test | Duration | Result |
|------|----------|--------|
| `TestStress_Store_ConcurrentAccess` | 0.00s | PASS |
| `TestStress_ConcurrentCancelAndComplete` | 0.05s | PASS |
| `TestStress_WorkflowWith100Steps` | 0.11s | PASS |
| `TestStress_ParallelLevel_20Steps` | 0.25s | PASS |
| `TestStress_Retries_HighVolume` | 1.05s | PASS |
| `TestStress_100ConcurrentWorkflows` | 5.05s | PASS |

All 6 stress tests pass with `-race` enabled (zero data races detected).

---

## Graph Benchmarks

Package: `github.com/sicko7947/gorkflow`

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `TopologicalSort_10Nodes` (cache hit) | 9.0 | 0 | 0 |
| `TopologicalSort_10Nodes_Cold` | 9,608 | 4,977 | 93 |
| `TopologicalSort_100Nodes` (cache hit) | 9.0 | 0 | 0 |
| `ComputeLevels_FanOut_16` (cache hit) | 9.0 | 0 | 0 |
| `ComputeLevels_FanOut_16_Cold` | 14,518 | 10,540 | 92 |
| `Clone_10Nodes` | 1,713 | 1,544 | 31 |

Cache hits for `TopologicalSort` and `ComputeLevels` are effectively free (~9 ns) since the graph is immutable after build time.

---

## Engine Benchmarks

Package: `github.com/sicko7947/gorkflow/engine`

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `Sequential_10Steps` | 1,101,452 | 34,873 | 300 |
| `Sequential_100Steps` | 1,113,261 | 311,741 | 2,563 |
| `Parallel_FanOut_4` | 39,323 | 18,583 | 161 |
| `Parallel_FanOut_16` | 122,631 | 59,669 | 495 |
| `StartWorkflow_Async` | 11,748 | 6,298 | 56 |

`Parallel_FanOut_4` at ~39 µs vs `Sequential_10Steps` at ~1.1 ms: parallel fan-out is ~28× faster than an equivalent sequential chain for independent work.

Async workflow startup overhead is ~12 µs (UUID generation + JSON marshal + store write).

---

## Store Benchmarks

Package: `github.com/sicko7947/gorkflow/store`

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `MemoryStore_CreateAndGetRun` | 2,206 | 1,093 | 7 |
| `MemoryStore_SaveLoadOutput` | 664 | 415 | 4 |
| `MemoryStore_ListRuns_100Runs` | 17,530 | 24,952 | 103 |
| `LibSQL_CreateAndGetRun` | 182,490 | 2,910 | 47 |
| `LibSQL_SaveLoadOutput` | 86,804 | 1,114 | 31 |
| `LibSQL_UpdateRun_FullSerialize` | 90,007 | 913 | 17 |

MemoryStore is ~83× faster than LibSQL for create+get (2.2 µs vs 182 µs). LibSQL numbers reflect local file I/O; a remote Turso instance will be higher. Use MemoryStore for development/testing, LibSQL for production persistence.
