# Design Decisions

Key architectural choices in Gorkflow and the reasoning behind them.

## Why Go Generics for Steps?

**Decision:** Steps are parameterized with `Step[TIn, TOut]` using Go generics.

**Alternatives considered:**
- `interface{}` for inputs/outputs (pre-generics Go pattern)
- Code generation for type-safe steps

**Why generics:**
- Compile-time type safety within step handlers — the handler function signature enforces correct types
- No code generation step required
- Type inference means most users don't need to specify type parameters explicitly
- The `StepExecutor` interface provides a non-generic boundary for the engine, keeping the orchestration layer simple

**Tradeoff:** Steps are connected through JSON serialization at runtime (not compile-time type-checked between steps). This allows structural subtyping (a step can accept a superset or subset of the previous step's output) but means type mismatches between steps are caught at runtime, not compile time.

## Why a DAG-Based Execution Graph?

**Decision:** Workflows are modeled as directed acyclic graphs (DAGs) with topological sort for execution ordering.

**Alternatives considered:**
- Linear step chains only
- State machine with explicit transitions
- Event-driven execution

**Why DAGs:**
- Natural representation of step dependencies
- Supports both sequential and parallel execution patterns
- Topological sort gives a deterministic execution order
- Cycle detection prevents infinite loops at build time
- Extensible — can add features like fan-in aggregation without changing the core model

**Tradeoff:** DAGs don't support looping or recursive workflows. If you need iteration, implement it within a step handler.

## Why LibSQL/SQLite for Persistence?

**Decision:** The primary production store uses LibSQL (SQLite-compatible), with support for both local files and remote Turso databases.

**Alternatives considered:**
- PostgreSQL as the primary store
- Redis for state management
- Embedded key-value stores (BoltDB, BadgerDB)

**Why LibSQL:**
- Zero-dependency for local development — a SQLite file requires no running database server
- Same store works for development (local file) and production (remote Turso)
- SQL gives rich querying without custom code (filtering runs, counting by status)
- Turso provides remote access with edge replication when needed
- The `WorkflowStore` interface allows custom backends for teams that need PostgreSQL, DynamoDB, etc.

**Tradeoff:** SQLite has write concurrency limitations. For high-throughput multi-process deployments, implement a custom store with PostgreSQL or similar.

## Why JSON Blobs for Step Data?

**Decision:** Step inputs, outputs, and state values are stored as JSON byte slices (`[]byte`), not decomposed into columns.

**Alternatives considered:**
- Structured columns per field
- Protocol Buffers for serialization
- MessagePack for compact binary encoding

**Why JSON blobs:**
- Schema-free — steps can have any input/output shape without store migration
- Human-readable — easy to inspect stored data for debugging
- Go's `encoding/json` is well-supported and handles struct tags
- Compatible with both SQLite JSON functions and application-level access
- Validation is handled at the application layer (struct tags), not the storage layer

**Tradeoff:** No database-level field querying on step data. If you need to query by specific output fields, consider adding custom indexes or a separate analytics store.

## Why a Store Interface?

**Decision:** Persistence is abstracted behind the `WorkflowStore` interface, with `MemoryStore` and `LibSQLStore` as built-in implementations.

**Why:**
- Test isolation — `MemoryStore` makes tests fast and deterministic with no database setup
- Backend flexibility — teams can implement PostgreSQL, DynamoDB, or any other store
- Separation of concerns — the engine handles orchestration, the store handles persistence
- The interface is intentionally simple (CRUD + queries) to make custom implementations straightforward

## Why JSON Serialization Between Steps?

**Decision:** Data flows between steps through JSON marshaling/unmarshaling, not direct Go value passing.

**Alternatives considered:**
- Pass Go values directly through channels
- Use `interface{}` with type assertions

**Why JSON:**
- Enables persistence — step inputs and outputs are stored for debugging and resumability
- Structural subtyping — a step can consume a subset of the previous step's output fields
- Consistent with the store layer — no separate serialization path
- Validation can be applied uniformly (struct tags checked after deserialization)

**Tradeoff:** Serialization overhead for each step boundary. For CPU-intensive pipelines with minimal I/O, this overhead may be noticeable. In practice, step handlers typically do more work than the serialization cost.

## Why Functional Options?

**Decision:** Steps, engine, and workflow starts are configured using the functional options pattern (`WithRetries`, `WithTimeout`, `WithLogger`, etc.).

**Why:**
- Clean API with sensible defaults — `NewStep("id", "name", handler)` works without any options
- Extensible — new options can be added without breaking existing code
- Self-documenting — option names describe what they do
- Composable — options can be combined freely

## Why zerolog?

**Decision:** Gorkflow uses `zerolog` for structured logging.

**Why:**
- Zero-allocation JSON logger — minimal performance impact
- Structured fields make log analysis easy (filter by `run_id`, `step_id`, `event`)
- The step context logger is pre-configured with execution metadata
- Pretty console output for development, JSON output for production

---

**See also**: [System Overview](system-overview.md) | [Storage Layer](storage-layer.md)
