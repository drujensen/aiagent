package repositories_mongo

import (
	"context"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoModelRepository struct {
	collection *mongo.Collection
}

func NewMongoModelRepository(collection *mongo.Collection) *MongoModelRepository {
	return &MongoModelRepository{
		collection: collection,
	}
}

func (r *MongoModelRepository) ListModels(ctx context.Context) ([]*entities.Model, error) {
	var models []*entities.Model
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.InternalErrorf("failed to list models: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var model entities.Model
		if err := cursor.Decode(&model); err != nil {
			return nil, errors.InternalErrorf("failed to decode model: %v", err)
		}
		models = append(models, &model)
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.InternalErrorf("failed to list models: %v", err)
	}

	return models, nil
}

func (r *MongoModelRepository) GetModel(ctx context.Context, id string) (*entities.Model, error) {
	var model entities.Model
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&model)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("model not found")
	}
	if err != nil {
		return nil, errors.InternalErrorf("failed to get model: %v", err)
	}

	return &model, nil
}

func (r *MongoModelRepository) CreateModel(ctx context.Context, model *entities.Model) error {
	_, err := r.collection.InsertOne(ctx, model)
	if err != nil {
		return errors.InternalErrorf("failed to create model: %v", err)
	}

	return nil
}

func (r *MongoModelRepository) UpdateModel(ctx context.Context, model *entities.Model) error {
	model.UpdatedAt = time.Now()

	update, err := bson.Marshal(bson.M{
		"$set": model,
	})
	if err != nil {
		return errors.InternalErrorf("failed to marshal model: %v", err)
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": model.ID}, update)
	if err != nil {
		return errors.InternalErrorf("failed to update model: %v", err)
	}
	if result.MatchedCount == 0 {
		return errors.NotFoundErrorf("model not found: %s", model.ID)
	}

	return nil
}

func (r *MongoModelRepository) DeleteModel(ctx context.Context, id string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return errors.InternalErrorf("failed to delete model: %v", err)
	}
	if result.DeletedCount == 0 {
		return errors.NotFoundErrorf("model not found: %s", id)
	}

	return nil
}

var _ interfaces.ModelRepository = (*MongoModelRepository)(nil)
