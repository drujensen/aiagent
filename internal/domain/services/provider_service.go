package services

import (
	"context"
	"fmt"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"github.com/drujensen/aiagent/internal/impl/config"

	"go.uber.org/zap"
)

type ProviderService interface {
	ListProviders(ctx context.Context) ([]*entities.Provider, error)
	GetProvider(ctx context.Context, id string) (*entities.Provider, error)
	EnsureCustomProviders(ctx context.Context, globalConfig *config.GlobalConfig) error
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

func (s *providerService) ListProviders(ctx context.Context) ([]*entities.Provider, error) {
	providers, err := s.providerRepo.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	return providers, nil
}

func (s *providerService) GetProvider(ctx context.Context, id string) (*entities.Provider, error) {
	provider, err := s.providerRepo.GetProvider(ctx, id)
	if err != nil {
		return nil, errors.InternalErrorf("failed to get provider: %v", err)
	}
	return provider, nil
}

func (s *providerService) EnsureCustomProviders(ctx context.Context, globalConfig *config.GlobalConfig) error {
	// Get existing providers to check for duplicates
	existingProviders, err := s.providerRepo.ListProviders(ctx)
	if err != nil {
		return fmt.Errorf("failed to list existing providers: %w", err)
	}

	// Create a map of existing provider names for quick lookup
	existingNames := make(map[string]bool)
	for _, provider := range existingProviders {
		existingNames[provider.Name] = true
	}

	for providerKey, customConfig := range globalConfig.Providers {
		// Check if provider name already exists
		if existingNames[customConfig.Name] {
			s.logger.Debug("Custom provider name already exists, skipping",
				zap.String("provider_key", providerKey),
				zap.String("name", customConfig.Name))
			continue
		}

		// Provider doesn't exist, create it
		s.logger.Info("Creating custom provider from config",
			zap.String("provider_key", providerKey),
			zap.String("name", customConfig.Name))

		provider := &entities.Provider{
			ID:         "", // Let repository generate UUID
			Name:       customConfig.Name,
			Type:       entities.ProviderType(customConfig.Type),
			BaseURL:    customConfig.BaseURL,
			APIKeyName: customConfig.APIKeyName,
			Models:     []entities.ModelPricing{}, // Will be populated during refresh
		}

		if err := s.providerRepo.CreateProvider(ctx, provider); err != nil {
			return fmt.Errorf("failed to create custom provider %s: %w", providerKey, err)
		}

		s.logger.Info("Created custom provider",
			zap.String("provider_key", providerKey),
			zap.String("provider_id", provider.ID),
			zap.String("name", provider.Name))
	}

	return nil
}
