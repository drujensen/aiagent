package repositories_mongo

import (
	"context"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoProviderRepository struct {
	collection *mongo.Collection
}

func NewMongoProviderRepository(collection *mongo.Collection) *MongoProviderRepository {
	return &MongoProviderRepository{
		collection: collection,
	}
}

func (r *MongoProviderRepository) ListProviders(ctx context.Context) ([]*entities.Provider, error) {
	var providers []*entities.Provider
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.InternalErrorf("failed to list providers: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var provider entities.Provider
		if err := cursor.Decode(&provider); err != nil {
			return nil, errors.InternalErrorf("failed to decode provider: %v", err)
		}
		providers = append(providers, &provider)
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.InternalErrorf("failed to list providers: %v", err)
	}

	return providers, nil
}

func (r *MongoProviderRepository) GetProvider(ctx context.Context, id string) (*entities.Provider, error) {
	var provider entities.Provider
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&provider)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("provider not found")
	}
	if err != nil {
		return nil, errors.InternalErrorf("failed to get provider: %v", err)
	}

	return &provider, nil
}

func (r *MongoProviderRepository) CreateProvider(ctx context.Context, provider *entities.Provider) error {
	_, err := r.collection.InsertOne(ctx, provider)
	if err != nil {
		return errors.InternalErrorf("failed to create provider: %v", err)
	}
	return nil
}

var _ interfaces.ProviderRepository = (*MongoProviderRepository)(nil)
