package repositories

import (
	"context"
	"fmt"

	"aiagent/internal/domain/entities"
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

func (r *mongoProviderRepository) CreateProvider(ctx context.Context, provider *entities.Provider) error {
	_, err := r.collection.InsertOne(ctx, provider)
	return err
}

func (r *mongoProviderRepository) UpdateProvider(ctx context.Context, provider *entities.Provider) error {
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": provider.ID}, provider)
	return err
}

func (r *mongoProviderRepository) GetProvider(ctx context.Context, id string) (*entities.Provider, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	var provider entities.Provider
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&provider)
	if err != nil {
		return nil, err
	}

	return &provider, nil
}

func (r *mongoProviderRepository) GetProviderByType(ctx context.Context, providerType entities.ProviderType) (*entities.Provider, error) {
	var provider entities.Provider
	err := r.collection.FindOne(ctx, bson.M{"type": providerType}).Decode(&provider)
	if err != nil {
		return nil, err
	}

	return &provider, nil
}

func (r *mongoProviderRepository) ListProviders(ctx context.Context) ([]*entities.Provider, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var providers []*entities.Provider
	if err = cursor.All(ctx, &providers); err != nil {
		return nil, err
	}

	return providers, nil
}

func (r *mongoProviderRepository) DeleteProvider(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}
