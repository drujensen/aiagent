package repositories

import (
	"context"

	"aiagent/internal/domain/entities"
)

// AuditLogRepository defines the interface for managing AuditLog entities in the domain layer.
// It provides methods to create and retrieve audit logs, abstracting the data storage mechanism
// (e.g., MongoDB). This interface supports monitoring and debugging by tracking agent actions,
// omitting update and delete operations due to the immutable nature of audit logs.
//
// Key features:
// - Append-Only: Includes only Create, Get, and List methods, reflecting audit log immutability.
// - Context Usage: Uses context.Context for timeout and cancellation support.
// - Error Handling: Returns ErrNotFound (from package scope) for non-existent logs in GetAuditLog.
//
// Dependencies:
// - context: For managing request timeouts and cancellations.
// - aiagent/internal/domain/entities: Provides the AuditLog entity definition.
//
// Notes:
// - CreateAuditLog sets the audit log's ID in place, assuming storage generates it.
// - ListAuditLogs returns pointers to optimize memory usage for potentially large log sets.
// - Update and Delete methods are intentionally omitted as audit logs should not be modified.
type AuditLogRepository interface {
	// CreateAuditLog inserts a new audit log into the repository and sets the audit log's ID.
	// The audit log's ID field is updated with the generated ID upon successful insertion.
	// Returns an error if the insertion fails (e.g., invalid AgentID, database error).
	CreateAuditLog(ctx context.Context, auditLog *entities.AuditLog) error

	// GetAuditLog retrieves an audit log by its ID.
	// Returns a pointer to the AuditLog and nil error on success, or nil and ErrNotFound
	// if the audit log does not exist, or nil and another error for other failures (e.g., database error).
	GetAuditLog(ctx context.Context, id string) (*entities.AuditLog, error)

	// ListAuditLogs retrieves all audit logs in the repository.
	// Returns a slice of pointers to AuditLog entities; returns an empty slice if no logs exist,
	// or an error if the retrieval fails (e.g., database connection lost).
	ListAuditLogs(ctx context.Context) ([]*entities.AuditLog, error)
}

// Notes:
// - Error handling assumes implementations provide context in errors (e.g., "audit log not found: id=abc").
// - Edge case: Large log volumes may require pagination, not currently supported.
// - Assumption: AuditLog IDs are strings, consistent with MongoDB ObjectID usage.
// - Limitation: No filtering by AgentID or Timestamp; could be added for enhanced monitoring.
