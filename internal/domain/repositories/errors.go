package repositories

import "errors"

// ErrNotFound is returned by repository methods when an entity is not found.
// It is a generic error used across all repository interfaces in this package,
// allowing callers to distinguish "not found" cases from other errors.
//
// Key features:
// - Generic: Applicable to agents, tools, tasks, conversations, and audit logs.
// - Reusable: Defined once at the package level to avoid redeclaration.
//
// Notes:
// - Callers can wrap this error with context (e.g., "agent not found: id=xyz") in services or implementations.
// - Assumption: A single error type suffices for "not found" across all entities; specific types could be added if needed.
var ErrNotFound = errors.New("entity not found")
