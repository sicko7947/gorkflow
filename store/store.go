package store

// Package store provides persistence implementations for the workflow framework.
// The WorkflowStore interface is defined in the parent workflow package
// (../store_interface.go) to avoid import cycles between the workflow
// and store packages.
//
// This package contains concrete implementations:
//   - MemoryStore: In-memory backend for testing
//   - LibSQLStore: Persistent backend using LibSQL/SQLite
//
// Schema design follows single-table patterns defined in schema.go.
