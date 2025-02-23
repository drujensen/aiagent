package repositories

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
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

func (r *MongoAgentRepository) CreateAgent(ctx context.Context, agent *entities.Agent) error {
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()
	agent.ID = primitive.NewObjectID()

	result, err := r.collection.InsertOne(ctx, agent)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		agent.ID = oid
	} else {
		return mongo.ErrInvalidIndexValue
	}

	return nil
}

func (r *MongoAgentRepository) GetAgent(ctx context.Context, id string) (*entities.Agent, error) {
	var agent entities.Agent
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, mongo.ErrNoDocuments
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&agent)
	if err == mongo.ErrNoDocuments {
		return nil, mongo.ErrNoDocuments
	}
	if err != nil {
		return nil, err
	}

	return &agent, nil
}

func (r *MongoAgentRepository) UpdateAgent(ctx context.Context, agent *entities.Agent) error {
	agent.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(agent.ID.Hex())
	if err != nil {
		return mongo.ErrNoDocuments
	}

	update := bson.M{
		"$set": bson.M{
			"name":          agent.Name,
			"endpoint":      agent.Endpoint,
			"model":         agent.Model,
			"api_key":       agent.APIKey,
			"system_prompt": agent.SystemPrompt,
			"temperature":   agent.Temperature,
			"max_tokens":    agent.MaxTokens,
			"updated_at":    agent.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *MongoAgentRepository) DeleteAgent(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return mongo.ErrNoDocuments
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (r *MongoAgentRepository) ListAgents(ctx context.Context) ([]*entities.Agent, error) {
	var agents []*entities.Agent
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var agent entities.Agent
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

var _ interfaces.AgentRepository = (*MongoAgentRepository)(nil)
