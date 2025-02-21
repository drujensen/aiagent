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
 * This file implements the MongoDB-backed ToolRepository for the AI Workflow Automation Platform.
 * It provides concrete implementations of the ToolRepository interface defined in the domain layer,
 * managing CRUD operations for Tool entities stored in MongoDB's 'tools' collection.
 *
 * Key features:
 * - MongoDB Integration: Uses the MongoDB driver to persist and retrieve Tool data.
 * - Domain Alignment: Operates on entities.Tool and adheres to the ToolRepository interface.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides the Tool entity definition.
 * - aiagent/internal/domain/repositories: Provides the ToolRepository interface and ErrNotFound.
 * - go.mongodb.org/mongo-driver/mongo: MongoDB driver for database operations.
 *
 * @notes
 * - All methods use the context passed from the caller, respecting caller-defined timeouts.
 * - ObjectID conversions are handled to map between string IDs and MongoDB's primitive.ObjectID.
 * - Errors are returned as per the interface, with ErrNotFound for missing entities.
 */

// MongoToolRepository is the MongoDB implementation of the ToolRepository interface.
// It handles CRUD operations for Tool entities using a MongoDB collection.
type MongoToolRepository struct {
	collection *mongo.Collection
}

// NewMongoToolRepository creates a new MongoToolRepository instance with the given collection.
//
// Parameters:
// - collection: The MongoDB collection handle for the 'tools' collection.
//
// Returns:
// - *MongoToolRepository: A new instance ready to manage Tool entities.
func NewMongoToolRepository(collection *mongo.Collection) *MongoToolRepository {
	return &MongoToolRepository{
		collection: collection,
	}
}

// CreateTool inserts a new tool into the MongoDB collection and sets the tool's ID.
// It initializes CreatedAt and UpdatedAt to the current time.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - tool: Pointer to the Tool entity to insert; its ID is updated upon success.
//
// Returns:
// - error: Nil on success, or an error if insertion fails (e.g., database error).
func (r *MongoToolRepository) CreateTool(ctx context.Context, tool *entities.Tool) error {
	tool.CreatedAt = time.Now()
	tool.UpdatedAt = time.Now()
	// Ensure ID is empty to let MongoDB generate it
	tool.ID = ""

	result, err := r.collection.InsertOne(ctx, tool)
	if err != nil {
		return err
	}

	// Set the generated ObjectID as a hex string on the tool
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		tool.ID = oid.Hex()
	} else {
		return mongo.ErrInvalidIndexValue
	}

	return nil
}

// GetTool retrieves a tool by its ID from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the tool to retrieve.
//
// Returns:
// - *entities.Tool: The retrieved tool, or nil if not found or an error occurs.
// - error: Nil on success, ErrNotFound if the tool doesn’t exist, or another error otherwise.
func (r *MongoToolRepository) GetTool(ctx context.Context, id string) (*entities.Tool, error) {
	var tool entities.Tool
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, repositories.ErrNotFound // Invalid ID treated as not found
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&tool)
	if err == mongo.ErrNoDocuments {
		return nil, repositories.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &tool, nil
}

// UpdateTool updates an existing tool in the MongoDB collection.
// It sets UpdatedAt to the current time and updates specified fields.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - tool: Pointer to the Tool entity with updated fields; must have a valid ID.
//
// Returns:
// - error: Nil on success, ErrNotFound if the tool doesn’t exist, or another error otherwise.
func (r *MongoToolRepository) UpdateTool(ctx context.Context, tool *entities.Tool) error {
	tool.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(tool.ID)
	if err != nil {
		return repositories.ErrNotFound // Invalid ID treated as not found
	}

	update := bson.M{
		"$set": bson.M{
			"name":       tool.Name,
			"category":   tool.Category,
			"updated_at": tool.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return repositories.ErrNotFound
	}

	return nil
}

// DeleteTool deletes a tool by its ID from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the tool to delete.
//
// Returns:
// - error: Nil on success, ErrNotFound if the tool doesn’t exist, or another error otherwise.
func (r *MongoToolRepository) DeleteTool(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return repositories.ErrNotFound // Invalid ID treated as not found
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return repositories.ErrNotFound
	}

	return nil
}

// ListTools retrieves all tools from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
//
// Returns:
// - []*entities.Tool: Slice of all tools, empty if none exist.
// - error: Nil on success, or an error if retrieval fails (e.g., database error).
func (r *MongoToolRepository) ListTools(ctx context.Context) ([]*entities.Tool, error) {
	var tools []*entities.Tool
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var tool entities.Tool
		if err := cursor.Decode(&tool); err != nil {
			return nil, err
		}
		tools = append(tools, &tool)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return tools, nil
}

// Notes:
// - Edge case: Invalid IDs are treated as not found to simplify error handling per the interface.
// - Assumption: The 'tools' collection exists; created implicitly by MongoDB on first insert.
// - Limitation: No pagination in ListTools; suitable for small toolsets, extendable later if needed.
