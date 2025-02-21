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
 * This file implements the MongoDB-backed ConversationRepository for the AI Workflow Automation Platform.
 * It provides concrete implementations of the ConversationRepository interface defined in the domain layer,
 * managing CRUD operations for Conversation entities stored in MongoDB's 'conversations' collection.
 *
 * Key features:
 * - MongoDB Integration: Persists and retrieves Conversation data with embedded Messages.
 * - Domain Alignment: Operates on entities.Conversation and matches the ConversationRepository interface.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides the Conversation entity definition.
 * - aiagent/internal/domain/repositories: Provides the ConversationRepository interface and ErrNotFound.
 * - go.mongodb.org/mongo-driver/mongo: MongoDB driver for database operations.
 *
 * @notes
 * - Messages are embedded as a slice, updated via UpdateConversation.
 * - TaskID links to a Task, assumed valid by the service layer.
 * - No size limit on Messages; may need consideration for large conversations.
 */

// MongoConversationRepository is the MongoDB implementation of the ConversationRepository interface.
// It handles CRUD operations for Conversation entities using a MongoDB collection.
type MongoConversationRepository struct {
	collection *mongo.Collection
}

// NewMongoConversationRepository creates a new MongoConversationRepository instance with the given collection.
//
// Parameters:
// - collection: The MongoDB collection handle for the 'conversations' collection.
//
// Returns:
// - *MongoConversationRepository: A new instance ready to manage Conversation entities.
func NewMongoConversationRepository(collection *mongo.Collection) *MongoConversationRepository {
	return &MongoConversationRepository{
		collection: collection,
	}
}

// CreateConversation inserts a new conversation into the MongoDB collection and sets the conversation’s ID.
// It initializes CreatedAt and UpdatedAt to the current time.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - conversation: Pointer to the Conversation entity to insert; its ID is updated upon success.
//
// Returns:
// - error: Nil on success, or an error if insertion fails (e.g., database error).
func (r *MongoConversationRepository) CreateConversation(ctx context.Context, conversation *entities.Conversation) error {
	conversation.CreatedAt = time.Now()
	conversation.UpdatedAt = time.Now()
	// Ensure ID is empty to let MongoDB generate it
	conversation.ID = ""

	result, err := r.collection.InsertOne(ctx, conversation)
	if err != nil {
		return err
	}

	// Set the generated ObjectID as a hex string on the conversation
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		conversation.ID = oid.Hex()
	} else {
		return mongo.ErrInvalidIndexValue
	}

	return nil
}

// GetConversation retrieves a conversation by its ID from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the conversation to retrieve.
//
// Returns:
// - *entities.Conversation: The retrieved conversation, or nil if not found or an error occurs.
// - error: Nil on success, ErrNotFound if the conversation doesn’t exist, or another error otherwise.
func (r *MongoConversationRepository) GetConversation(ctx context.Context, id string) (*entities.Conversation, error) {
	var conversation entities.Conversation
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, repositories.ErrNotFound
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&conversation)
	if err == mongo.ErrNoDocuments {
		return nil, repositories.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

// UpdateConversation updates an existing conversation in the MongoDB collection.
// It sets UpdatedAt to the current time and updates all fields, typically appending messages.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - conversation: Pointer to the Conversation entity with updated fields; must have a valid ID.
//
// Returns:
// - error: Nil on success, ErrNotFound if the conversation doesn’t exist, or another error otherwise.
func (r *MongoConversationRepository) UpdateConversation(ctx context.Context, conversation *entities.Conversation) error {
	conversation.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(conversation.ID)
	if err != nil {
		return repositories.ErrNotFound
	}

	update := bson.M{
		"$set": bson.M{
			"task_id":    conversation.TaskID,
			"messages":   conversation.Messages,
			"updated_at": conversation.UpdatedAt,
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

// DeleteConversation deletes a conversation by its ID from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the conversation to delete.
//
// Returns:
// - error: Nil on success, ErrNotFound if the conversation doesn’t exist, or another error otherwise.
func (r *MongoConversationRepository) DeleteConversation(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return repositories.ErrNotFound
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

// ListConversations retrieves all conversations from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
//
// Returns:
// - []*entities.Conversation: Slice of all conversations, empty if none exist.
// - error: Nil on success, or an error if retrieval fails (e.g., database error).
func (r *MongoConversationRepository) ListConversations(ctx context.Context) ([]*entities.Conversation, error) {
	var conversations []*entities.Conversation
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var conversation entities.Conversation
		if err := cursor.Decode(&conversation); err != nil {
			return nil, err
		}
		conversations = append(conversations, &conversation)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return conversations, nil
}

// Notes:
// - Edge case: Large Messages arrays may impact performance; no current limit enforced.
// - Assumption: The 'conversations' collection exists; created implicitly by MongoDB on first insert.
// - Limitation: No filtering for active conversations; extendable for chat UI optimization.
