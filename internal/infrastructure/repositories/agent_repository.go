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
 * This file implements the MongoDB-backed AgentRepository for the AI Workflow Automation Platform.
 * It provides concrete implementations of the AgentRepository interface defined in the domain layer,
 * managing CRUD operations for AIAgent entities stored in MongoDB's 'agents' collection.
 *
 * Key features:
 * - MongoDB Integration: Persists and retrieves hierarchical AIAgent data.
 * - Domain Alignment: Operates on entities.AIAgent and matches the AgentRepository interface.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides the AIAgent entity definition.
 * - aiagent/internal/domain/repositories: Provides the AgentRepository interface and ErrNotFound.
 * - go.mongodb.org/mongo-driver/mongo: MongoDB driver for database operations.
 *
 * @notes
 * - Handles complex fields like Configuration (map) and Tools (string slice) transparently.
 * - Ensures hierarchy fields (ParentID, ChildrenIDs) are stored correctly.
 * - Validation (e.g., hierarchy cycles) is assumed to be handled in the service layer.
 */

// MongoAgentRepository is the MongoDB implementation of the AgentRepository interface.
// It handles CRUD operations for AIAgent entities using a MongoDB collection.
type MongoAgentRepository struct {
	collection *mongo.Collection
}

// NewMongoAgentRepository creates a new MongoAgentRepository instance with the given collection.
//
// Parameters:
// - collection: The MongoDB collection handle for the 'agents' collection.
//
// Returns:
// - *MongoAgentRepository: A new instance ready to manage AIAgent entities.
func NewMongoAgentRepository(collection *mongo.Collection) *MongoAgentRepository {
	return &MongoAgentRepository{
		collection: collection,
	}
}

// CreateAgent inserts a new agent into the MongoDB collection and sets the agent’s ID.
// It initializes CreatedAt and UpdatedAt to the current time.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - agent: Pointer to the AIAgent entity to insert; its ID is updated upon success.
//
// Returns:
// - error: Nil on success, or an error if insertion fails (e.g., database error).
func (r *MongoAgentRepository) CreateAgent(ctx context.Context, agent *entities.AIAgent) error {
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()
	// Ensure ID is empty to let MongoDB generate it
	agent.ID = ""

	result, err := r.collection.InsertOne(ctx, agent)
	if err != nil {
		return err
	}

	// Set the generated ObjectID as a hex string on the agent
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		agent.ID = oid.Hex()
	} else {
		return mongo.ErrInvalidIndexValue
	}

	return nil
}

// GetAgent retrieves an agent by its ID from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the agent to retrieve.
//
// Returns:
// - *entities.AIAgent: The retrieved agent, or nil if not found or an error occurs.
// - error: Nil on success, ErrNotFound if the agent doesn’t exist, or another error otherwise.
func (r *MongoAgentRepository) GetAgent(ctx context.Context, id string) (*entities.AIAgent, error) {
	var agent entities.AIAgent
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, repositories.ErrNotFound
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&agent)
	if err == mongo.ErrNoDocuments {
		return nil, repositories.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &agent, nil
}

// UpdateAgent updates an existing agent in the MongoDB collection.
// It sets UpdatedAt to the current time and updates all fields.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - agent: Pointer to the AIAgent entity with updated fields; must have a valid ID.
//
// Returns:
// - error: Nil on success, ErrNotFound if the agent doesn’t exist, or another error otherwise.
func (r *MongoAgentRepository) UpdateAgent(ctx context.Context, agent *entities.AIAgent) error {
	agent.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(agent.ID)
	if err != nil {
		return repositories.ErrNotFound
	}

	update := bson.M{
		"$set": bson.M{
			"name":                      agent.Name,
			"prompt":                    agent.Prompt,
			"tools":                     agent.Tools,
			"configuration":             agent.Configuration,
			"parent_id":                 agent.ParentID,
			"children_ids":              agent.ChildrenIDs,
			"human_interaction_enabled": agent.HumanInteractionEnabled,
			"updated_at":                agent.UpdatedAt,
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

// DeleteAgent deletes an agent by its ID from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the agent to delete.
//
// Returns:
// - error: Nil on success, ErrNotFound if the agent doesn’t exist, or another error otherwise.
func (r *MongoAgentRepository) DeleteAgent(ctx context.Context, id string) error {
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

// ListAgents retrieves all agents from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
//
// Returns:
// - []*entities.AIAgent: Slice of all agents, empty if none exist.
// - error: Nil on success, or an error if retrieval fails (e.g., database error).
func (r *MongoAgentRepository) ListAgents(ctx context.Context) ([]*entities.AIAgent, error) {
	var agents []*entities.AIAgent
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var agent entities.AIAgent
		if err := cursor.Decode(&agent); err != nil {
			return nil, err
		}
		agents = append(agents, &agent)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return agents, nil
}

// Notes:
// - Edge case: Invalid ParentID or ChildrenIDs references are not validated here; handled in AgentService.
// - Assumption: The 'agents' collection exists; created implicitly by MongoDB on first insert.
// - Limitation: No pagination in ListAgents; extendable if large agent datasets are expected.
