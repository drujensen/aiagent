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

type MongoToolRepository struct {
	collection *mongo.Collection
}

func NewMongoToolRepository(collection *mongo.Collection) *MongoToolRepository {
	return &MongoToolRepository{
		collection: collection,
	}
}

func (r *MongoToolRepository) CreateTool(ctx context.Context, tool *entities.Tool) error {
	tool.CreatedAt = time.Now()
	tool.UpdatedAt = time.Now()
	tool.ID = primitive.NewObjectID()

	result, err := r.collection.InsertOne(ctx, tool)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		tool.ID = oid
	} else {
		return mongo.ErrInvalidIndexValue
	}

	return nil
}

func (r *MongoToolRepository) GetTool(ctx context.Context, id string) (*entities.Tool, error) {
	var tool entities.Tool
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, mongo.ErrNoDocuments
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&tool)
	if err == mongo.ErrNoDocuments {
		return nil, mongo.ErrNoDocuments
	}
	if err != nil {
		return nil, err
	}

	return &tool, nil
}

func (r *MongoToolRepository) UpdateTool(ctx context.Context, tool *entities.Tool) error {
	tool.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(tool.ID.Hex())
	if err != nil {
		return mongo.ErrNoDocuments
	}

	update := bson.M{
		"$set": bson.M{
			"name":        tool.Name,
			"description": tool.Description,
			"updated_at":  tool.UpdatedAt,
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

func (r *MongoToolRepository) DeleteTool(ctx context.Context, id string) error {
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

var _ interfaces.ToolRepository = (*MongoToolRepository)(nil)
