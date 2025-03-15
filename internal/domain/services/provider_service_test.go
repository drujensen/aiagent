package services

import (
	"context"
	"testing"

	"aiagent/internal/domain/entities"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// Mock repository for testing
type MockProviderRepository struct {
	mock.Mock
}

func (m *MockProviderRepository) CreateProvider(ctx context.Context, provider *entities.Provider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockProviderRepository) UpdateProvider(ctx context.Context, provider *entities.Provider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockProviderRepository) GetProvider(ctx context.Context, id string) (*entities.Provider, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Provider), args.Error(1)
}

func (m *MockProviderRepository) GetProviderByType(ctx context.Context, providerType entities.ProviderType) (*entities.Provider, error) {
	args := m.Called(ctx, providerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Provider), args.Error(1)
}

func (m *MockProviderRepository) ListProviders(ctx context.Context) ([]*entities.Provider, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Provider), args.Error(1)
}

func (m *MockProviderRepository) DeleteProvider(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestProviderServiceGetProviderNotFound(t *testing.T) {
	// Setup
	mockRepo := new(MockProviderRepository)
	logger, _ := zap.NewDevelopment()
	service := NewProviderService(mockRepo, logger)
	ctx := context.Background()
	
	notFoundID := primitive.NewObjectID().Hex()
	
	// Test case: Provider not found, but found alternative by listing
	mockRepo.On("GetProvider", ctx, notFoundID).Return(nil, mongo.ErrNoDocuments)
	
	providerID := primitive.NewObjectID()
	alternativeProvider := &entities.Provider{
		ID:   providerID,
		Name: "Alternative Provider",
		Type: entities.ProviderOpenAI,
		Models: []entities.ModelPricing{
			{Name: "test-model"},
		},
	}
	
	mockRepo.On("ListProviders", ctx).Return([]*entities.Provider{alternativeProvider}, nil)
	
	// Execute
	result, err := service.GetProvider(ctx, notFoundID)
	
	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, alternativeProvider.ID, result.ID)
	
	mockRepo.AssertExpectations(t)
}