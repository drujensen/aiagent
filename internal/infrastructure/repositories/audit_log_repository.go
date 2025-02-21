package repositories

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/repositories"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/**
 * @description
 * This file implements the MongoDB-backed AuditLogRepository for the AI Workflow Automation Platform.
 * It provides concrete implementations of the AuditLogRepository interface defined in the domain layer,
 * managing operations for AuditLog entities stored in MongoDB's 'audit_logs' collection.
 *
 * Key features:
 * - MongoDB Integration: Persists and retrieves immutable AuditLog data.
 * - Domain Alignment: Operates on entities.AuditLog and matches the AuditLogRepository interface.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides the AuditLog entity definition.
 * - aiagent/internal/domain/repositories: Provides the AuditLogRepository interface and ErrNotFound.
 * - go.mongodb.org/mongo-driver/mongo: MongoDB driver for database operations.
 *
 * @notes
 * - No Update or Delete methods, as audit logs are immutable per the interface.
 * - Timestamp is set to current time if not provided, ensuring consistency.
 * - AgentID links to an AIAgent, assumed valid by the service layer.
 */

// MongoAuditLogRepository is the MongoDB implementation of the AuditLogRepository interface.
// It handles operations for AuditLog entities using a MongoDB collection.
type MongoAuditLogRepository struct {
	collection *mongo.Collection
}

// NewMongoAuditLogRepository creates a new MongoAuditLogRepository instance with the given collection.
//
// Parameters:
// - collection: The MongoDB collection handle for the 'audit_logs' collection.
//
// Returns:
// - *MongoAuditLogRepository: A new instance ready to manage AuditLog entities.
func NewMongoAuditLogRepository(collection *mongo.Collection) *MongoAuditLogRepository {
	return &MongoAuditLogRepository{
		collection: collection,
	}
}

// CreateAuditLog inserts a new audit log into the MongoDB collection and sets the audit log’s ID.
// It sets Timestamp to the current time if not already set.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - auditLog: Pointer to the AuditLog entity to insert; its ID is updated upon success.
//
// Returns:
// - error: Nil on success, or an error if insertion fails (e.g., database error).
func (r *MongoAuditLogRepository) CreateAuditLog(ctx context.Context, auditLog *entities.AuditLog) error {
	if auditLog.Timestamp.IsZero() {
		auditLog.Timestamp = time.Now()
	}
	// Ensure ID is empty to let MongoDB generate it
	auditLog.ID = ""

	result, err := r.collection.InsertOne(ctx, auditLog)
	if err != nil {
		return err
	}

	// Set the generated ObjectID as a hex string on the audit log
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		auditLog.ID = oid.Hex()
	} else {
		return mongo.ErrInvalidIndexValue
	}

	return nil
}

// GetAuditLog retrieves an audit log by its ID from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the audit log to retrieve.
//
// Returns:
// - *entities.AuditLog: The retrieved audit log, or nil if not found or an error occurs.
// - error: Nil on success, ErrNotFound if the audit log doesn’t exist, or another error otherwise.
func (r *MongoAuditLogRepository) GetAuditLog(ctx context.Context, id string) (*entities.AuditLog, error) {
	var auditLog entities.AuditLog
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, repositories.ErrNotFound
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&auditLog)
	if err == mongo.ErrNoDocuments {
		return nil, repositories.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &auditLog, nil
}

// ListAuditLogs retrieves all audit logs from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
//
// Returns:
// - []*entities.AuditLog: Slice of all audit logs, empty if none exist.
// - error: Nil on success, or an error if retrieval fails (e.g., database error).
func (r *MongoAuditLogRepository) ListAuditLogs(ctx context.Context) ([]*entities.AuditLog, error) {
	var auditLogs []*entities.AuditLog
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var auditLog entities.AuditLog
		if err := cursor.Decode(&auditLog); err != nil {
			return nil, err
		}
		auditLogs = append(auditLogs, &auditLog)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return auditLogs, nil
}

// Notes:
// - Edge case: Large log volumes may require pagination; not currently implemented.
// - Assumption: The 'audit_logs' collection exists; created implicitly by MongoDB on first insert.
// - Limitation: No filtering by AgentID or Timestamp; extendable for monitoring enhancements.
