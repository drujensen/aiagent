package services

import (
	"context"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type ProviderService interface {
	GetProvider(ctx context.Context, id string) (*entities.Provider, error)
	GetProviderByType(ctx context.Context, providerType entities.ProviderType) (*entities.Provider, error)
	ListProviders(ctx context.Context) ([]*entities.Provider, error)
}

type providerService struct {
	providerRepo interfaces.ProviderRepository
	logger       *zap.Logger
}

func NewProviderService(providerRepo interfaces.ProviderRepository, logger *zap.Logger) ProviderService {
	return &providerService{
		providerRepo: providerRepo,
		logger:       logger,
	}
}

func (s *providerService) GetProvider(ctx context.Context, id string) (*entities.Provider, error) {
	provider, err := s.providerRepo.GetProvider(ctx, id)
	if err != nil {
		return nil, errors.InternalErrorf("failed to get provider: %v", err)
	}
	return provider, nil
}

func (s *providerService) GetProviderByType(ctx context.Context, providerType entities.ProviderType) (*entities.Provider, error) {
	provider, err := s.providerRepo.GetProviderByType(ctx, providerType)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *providerService) ListProviders(ctx context.Context) ([]*entities.Provider, error) {
	providers, err := s.providerRepo.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	return providers, nil
}
