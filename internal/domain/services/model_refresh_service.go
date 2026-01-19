package services

import (
	"context"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"github.com/drujensen/aiagent/internal/impl/modelsdev"
	"go.uber.org/zap"
)

type ModelRefreshService interface {
	RefreshAllProviders(ctx context.Context) error
	RefreshProvider(ctx context.Context, providerID string) error
	GetLastRefreshTime(ctx context.Context) (*time.Time, error)
}

type modelRefreshService struct {
	providerRepo    interfaces.ProviderRepository
	modelsDevClient *modelsdev.ModelsDevClient
	logger          *zap.Logger
}

func NewModelRefreshService(
	providerRepo interfaces.ProviderRepository,
	modelsDevClient *modelsdev.ModelsDevClient,
	logger *zap.Logger,
) *modelRefreshService {
	return &modelRefreshService{
		providerRepo:    providerRepo,
		modelsDevClient: modelsDevClient,
		logger:          logger,
	}
}

func (s *modelRefreshService) RefreshAllProviders(ctx context.Context) error {
	s.logger.Info("Starting full provider refresh from models.dev")

	providers, err := s.providerRepo.ListProviders(ctx)
	if err != nil {
		return err
	}

	for _, provider := range providers {
		if err := s.refreshProvider(ctx, provider); err != nil {
			s.logger.Error("Failed to refresh provider",
				zap.String("provider_id", provider.ID),
				zap.String("name", provider.Name),
				zap.Error(err))
		}
	}

	s.logger.Info("Completed provider refresh", zap.Int("providers_updated", len(providers)))
	return nil
}

func (s *modelRefreshService) RefreshProvider(ctx context.Context, providerID string) error {
	provider, err := s.providerRepo.GetProvider(ctx, providerID)
	if err != nil {
		return err
	}

	if err := s.refreshProvider(ctx, provider); err != nil {
		return err
	}

	s.logger.Info("Refreshed provider", zap.String("provider_id", provider.ID), zap.String("name", provider.Name))
	return nil
}

func (s *modelRefreshService) refreshProvider(ctx context.Context, provider *entities.Provider) error {
	if provider.Type == entities.ProviderGeneric {
		s.logger.Debug("Skipping generic provider refresh",
			zap.String("provider_id", provider.ID),
			zap.String("name", provider.Name))
		return nil
	}

	fetched, err := s.modelsDevClient.Fetch()
	if err != nil {
		return err
	}

	providerToUpdate := &entities.Provider{
		ID:         provider.ID,
		Name:       provider.Name,
		Type:       provider.Type,
		BaseURL:    provider.BaseURL,
		APIKeyName: provider.APIKeyName,
		Models:     make([]entities.ModelPricing, 0),
	}

	for _, modelData := range (*fetched)[string(provider.Type)].Models {
		pricing := entities.ModelPricing{
			Name:                modelData.ID,
			InputPricePerMille:  modelData.Cost.Input,
			OutputPricePerMille: modelData.Cost.Output,
			ContextWindow:       modelData.Limit.Context,
		}

		providerToUpdate.Models = append(providerToUpdate.Models, pricing)
	}

	providerToUpdate.UpdatedAt = time.Now()
	if err := s.providerRepo.UpdateProvider(ctx, providerToUpdate); err != nil {
		return err
	}

	s.logger.Info("Updated provider with models.dev data",
		zap.String("provider_id", provider.ID),
		zap.String("provider_type", string(provider.Type)),
		zap.Int("models_count", len(providerToUpdate.Models)))
	return nil
}

func (s *modelRefreshService) GetLastRefreshTime(ctx context.Context) (*time.Time, error) {
	return s.modelsDevClient.GetLastRefreshTime()
}

var _ ModelRefreshService = (*modelRefreshService)(nil)
