package repositories_mongo

import (
	"context"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/bson"
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
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&agent)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("agent not found")
	}
	if err != nil {
		return nil, errors.InternalErrorf("failed to get agent: %v", err)
	}

	return &agent, nil
}

func (r *MongoAgentRepository) CreateAgent(ctx context.Context, agent *entities.Agent) error {
	_, err := r.collection.InsertOne(ctx, agent)
	if err != nil {
		return errors.InternalErrorf("failed to create agent: %v", err)
	}

	return nil
}

func (r *MongoAgentRepository) UpdateAgent(ctx context.Context, agent *entities.Agent) error {
	agent.UpdatedAt = time.Now()

	update, err := bson.Marshal(bson.M{
		"$set": agent,
	})
	if err != nil {
		return errors.InternalErrorf("failed to marshal agent: %v", err)
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": agent.ID}, update)
	if err != nil {
		return errors.InternalErrorf("failed to update agent: %v", err)
	}
	if result.MatchedCount == 0 {
		return errors.NotFoundErrorf("agent not found: %s", agent.ID)
	}

	return nil
}

func (r *MongoAgentRepository) DeleteAgent(ctx context.Context, id string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return errors.InternalErrorf("failed to delete agent: %v", err)
	}
	if result.DeletedCount == 0 {
		return errors.NotFoundErrorf("agent not found: %s", id)
	}

	return nil
}

var _ interfaces.AgentRepository = (*MongoAgentRepository)(nil)
