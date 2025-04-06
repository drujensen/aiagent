package repositories

import (
	"context"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoProviderRepository struct {
	collection *mongo.Collection
}

func NewMongoProviderRepository(collection *mongo.Collection) interfaces.ProviderRepository {
	return &mongoProviderRepository{
		collection: collection,
	}
}

func (r *mongoProviderRepository) ListProviders(ctx context.Context) ([]*entities.Provider, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.InternalErrorf("failed to list providers: %v", err)
	}
	defer cursor.Close(ctx)

	var providers []*entities.Provider
	if err = cursor.All(ctx, &providers); err != nil {
		return nil, errors.InternalErrorf("failed to list providers: %v", err)
	}

	return providers, nil
}

func (r *mongoProviderRepository) GetProvider(ctx context.Context, id string) (*entities.Provider, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.ValidationErrorf("invalid id: %v", err)
	}

	var provider entities.Provider
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&provider)
	if err != nil {
		return nil, errors.NotFoundErrorf("provider not found: %v", err)
	}

	return &provider, nil
}

func (r *mongoProviderRepository) GetProviderByType(ctx context.Context, providerType entities.ProviderType) (*entities.Provider, error) {
	var provider entities.Provider
	err := r.collection.FindOne(ctx, bson.M{"type": providerType}).Decode(&provider)
	if err != nil {
		return nil, errors.NotFoundErrorf("provider not found: %v", err)
	}

	return &provider, nil
}

func (r *mongoProviderRepository) CreateProvider(ctx context.Context, provider *entities.Provider) error {
	_, err := r.collection.InsertOne(ctx, provider)
	if err != nil {
		return errors.InternalErrorf("failed to create provider: %v", err)
	}

	return nil
}

func (r *mongoProviderRepository) UpdateProvider(ctx context.Context, provider *entities.Provider) error {
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": provider.ID}, provider)
	if err != nil {
		return errors.InternalErrorf("failed to update provider: %v", err)
	}

	return nil
}

func (r *mongoProviderRepository) DeleteProvider(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.ValidationErrorf("invalid id: %v", err)
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return errors.InternalErrorf("failed to delete provider: %v", err)
	}

	if result.DeletedCount == 0 {
		return errors.NotFoundErrorf("provider not found: %v", err)
	}

	return nil
}

var _ interfaces.ProviderRepository = (*mongoProviderRepository)(nil)
