package repositories

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoAgentRepository struct {
	collection *mongo.Collection
}

func NewMongoAgentRepository(collection *mongo.Collection) *MongoAgentRepository {
	return &MongoAgentRepository{
		collection: collection,
	}
}

func (r *MongoAgentRepository) ListAgents(ctx context.Context) ([]*entities.Agent, error) {
	var agents []*entities.Agent
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.InternalErrorf("failed to list agents: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var agent entities.Agent
		if err := cursor.Decode(&agent); err != nil {
			return nil, errors.InternalErrorf("failed to decode agent: %v", err)
		}
		agents = append(agents, &agent)
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.InternalErrorf("failed to list agents: %v", err)
	}

	return agents, nil
}

func (r *MongoAgentRepository) GetAgent(ctx context.Context, id string) (*entities.Agent, error) {
	var agent entities.Agent
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.NotFoundErrorf("agent not found")
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&agent)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("agent not found")
	}
	if err != nil {
		return nil, errors.InternalErrorf("failed to get agent: %v", err)
	}

	return &agent, nil
}

func (r *MongoAgentRepository) CreateAgent(ctx context.Context, agent *entities.Agent) error {
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()
	agent.ID = primitive.NewObjectID()

	result, err := r.collection.InsertOne(ctx, agent)
	if err != nil {
		return errors.InternalErrorf("failed to create agent: %v", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		agent.ID = oid
	} else {
		return errors.ValidationErrorf("failed to convert InsertedID to ObjectID")
	}

	return nil
}

func (r *MongoAgentRepository) UpdateAgent(ctx context.Context, agent *entities.Agent) error {
	agent.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(agent.ID.Hex())
	if err != nil {
		return errors.NotFoundErrorf("agent not found: %s", agent.ID.Hex())
	}

	// Convert the agent struct to BSON
	update, err := bson.Marshal(bson.M{
		"$set": agent,
	})
	if err != nil {
		return errors.InternalErrorf("failed to marshal agent: %v", err)
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return errors.InternalErrorf("failed to update agent: %v", err)
	}
	if result.MatchedCount == 0 {
		return errors.NotFoundErrorf("agent not found: %s", agent.ID.Hex())
	}

	return nil
}

func (r *MongoAgentRepository) DeleteAgent(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.NotFoundErrorf("agent not found")
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return errors.InternalErrorf("failed to delete agent: %v", err)
	}
	if result.DeletedCount == 0 {
		return errors.NotFoundErrorf("agent not found: %s", id)
	}

	return nil
}

var _ interfaces.AgentRepository = (*MongoAgentRepository)(nil)
