package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"github.com/drujensen/aiagent/internal/impl/config"
	"github.com/drujensen/aiagent/internal/impl/modelsdev"
	"go.uber.org/zap"
)

// cleanModelDisplayName cleans up model names by removing dates and normalizing format
func cleanModelDisplayName(modelID string) string {
	// Remove date patterns like -20241022, -2024-10-22, etc.
	datePattern1 := regexp.MustCompile(`-\d{8}$`)             // -20241022
	datePattern2 := regexp.MustCompile(`-\d{4}-\d{2}-\d{2}$`) // -2024-10-22
	cleaned := datePattern1.ReplaceAllString(modelID, "")
	cleaned = datePattern2.ReplaceAllString(cleaned, "")

	// Also remove "-latest" suffix
	cleaned = strings.ReplaceAll(cleaned, "-latest", "")

	// Replace hyphens and underscores with spaces for better readability
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	cleaned = strings.ReplaceAll(cleaned, "_", " ")

	// Capitalize words for better display
	words := strings.Fields(cleaned)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// generateModelDisplayName creates the clean display name for models
func generateModelDisplayName(providerName, modelID string) string {
	return cleanModelDisplayName(modelID)
}

// Test examples:
// cleanModelDisplayName("claude-3-5-haiku-20241022") -> "Claude 3 5 Haiku"
// cleanModelDisplayName("claude-3-5-haiku-latest") -> "Claude 3 5 Haiku"
// cleanModelDisplayName("gpt-4o-2024-05-13") -> "Gpt 4o"

type ModelRefreshService interface {
	RefreshAllProviders(ctx context.Context) error
	RefreshProvider(ctx context.Context, providerID string) error
	SyncAllModels(ctx context.Context) error
	GetLastRefreshTime(ctx context.Context) (*time.Time, error)
}

type modelRefreshService struct {
	providerRepo    interfaces.ProviderRepository
	modelRepo       interfaces.ModelRepository
	modelsDevClient *modelsdev.ModelsDevClient
	globalConfig    *config.GlobalConfig
	logger          *zap.Logger
}

func NewModelRefreshService(
	providerRepo interfaces.ProviderRepository,
	modelRepo interfaces.ModelRepository,
	modelsDevClient *modelsdev.ModelsDevClient,
	globalConfig *config.GlobalConfig,
	logger *zap.Logger,
) *modelRefreshService {
	return &modelRefreshService{
		providerRepo:    providerRepo,
		modelRepo:       modelRepo,
		modelsDevClient: modelsDevClient,
		globalConfig:    globalConfig,
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

	// Handle drujensen provider specially - it doesn't exist in models.dev
	if provider.Type == entities.ProviderDrujensen {
		pricing := entities.ModelPricing{
			Name:                "qwen3-coder:latest",
			InputPricePerMille:  0, // 0 pricing as requested
			OutputPricePerMille: 0, // 0 pricing as requested
			ContextWindow:       64000,
		}
		providerToUpdate.Models = append(providerToUpdate.Models, pricing)
	} else {
		for _, modelData := range (*fetched)[string(provider.Type)].Models {
			pricing := entities.ModelPricing{
				Name:                modelData.ID,
				InputPricePerMille:  modelData.Cost.Input,  // Keep as per million tokens
				OutputPricePerMille: modelData.Cost.Output, // Keep as per million tokens
				ContextWindow:       modelData.Limit.Context,
			}

			providerToUpdate.Models = append(providerToUpdate.Models, pricing)
		}
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

func (s *modelRefreshService) SyncAllModels(ctx context.Context) error {
	s.logger.Info("Starting full model sync")

	// Fetch latest data from models.dev
	modelsDevData, err := s.modelsDevClient.Fetch()
	if err != nil {
		return fmt.Errorf("failed to fetch models.dev data: %w", err)
	}

	// First refresh all providers to get latest pricing data
	if err := s.RefreshAllProviders(ctx); err != nil {
		return err
	}

	providers, err := s.providerRepo.ListProviders(ctx)
	if err != nil {
		return err
	}

	totalCreated := 0
	totalUpdated := 0
	totalDeleted := 0

	for _, provider := range providers {
		created, updated, deleted, err := s.syncProviderModels(ctx, provider, modelsDevData)
		if err != nil {
			s.logger.Error("Failed to sync models for provider",
				zap.String("provider_id", provider.ID),
				zap.String("provider_name", provider.Name),
				zap.Error(err))
			continue
		}
		totalCreated += created
		totalUpdated += updated
		totalDeleted += deleted
	}

	s.logger.Info("Completed model sync",
		zap.Int("models_created", totalCreated),
		zap.Int("models_updated", totalUpdated),
		zap.Int("models_deleted", totalDeleted))

	return nil
}

func (s *modelRefreshService) syncProviderModels(ctx context.Context, provider *entities.Provider, modelsDevData *modelsdev.ModelsDevResponse) (int, int, int, error) {
	// Get existing models for this provider
	existingModels, err := s.modelRepo.GetModelsByProvider(ctx, provider.ID)
	if err != nil {
		return 0, 0, 0, err
	}

	// Create map of existing models by model name for quick lookup
	existingByName := make(map[string]*entities.Model)
	for _, model := range existingModels {
		existingByName[model.ModelName] = model
	}

	created := 0
	updated := 0

	// Process each model in provider pricing data
	for _, pricing := range provider.Models {
		// Get the full model metadata from models.dev data
		var modelData modelsdev.ModelData
		if providerData, exists := (*modelsDevData)[string(provider.Type)]; exists {
			if md, exists := providerData.Models[pricing.Name]; exists {
				modelData = md
			}
		}

		if existingModel, exists := existingByName[pricing.Name]; exists {
			// Update existing model with latest pricing data
			needsUpdate := false
			expectedDisplayName := generateModelDisplayName(provider.Name, pricing.Name)
			if existingModel.Name != expectedDisplayName {
				existingModel.Name = expectedDisplayName
				needsUpdate = true
			}
			if existingModel.ContextWindow == nil || *existingModel.ContextWindow != pricing.ContextWindow {
				existingModel.ContextWindow = &pricing.ContextWindow
				needsUpdate = true
			}
			if needsUpdate {
				existingModel.UpdatedAt = time.Now()
				if err := s.modelRepo.UpdateModel(ctx, existingModel); err != nil {
					s.logger.Error("Failed to update model",
						zap.String("model_id", existingModel.ID),
						zap.String("model_name", existingModel.ModelName),
						zap.Error(err))
				} else {
					updated++
				}
			}
			// Remove from map to track which ones we've processed
			delete(existingByName, pricing.Name)
		} else {
			// Create new model
			maxTokens := int(float64(pricing.ContextWindow) * s.globalConfig.DefaultMaxTokensRatio)
			displayName := generateModelDisplayName(provider.Name, pricing.Name)
			model := entities.NewModel(
				displayName,                        // Display name
				provider.ID,                        // Provider ID
				provider.Type,                      // Provider type
				pricing.Name,                       // Model name
				"",                                 // API key (empty - resolved via provider)
				&s.globalConfig.DefaultTemperature, // Temperature
				&maxTokens,                         // Max tokens
				&pricing.ContextWindow,             // Context window
				"",                                 // Reasoning effort
				modelData.Family,                   // Family
				modelData.Reasoning,                // Reasoning capability
				modelData.ToolCall,                 // Tool call capability
				modelData.Temperature,              // Temperature capability
				modelData.Attachment,               // Attachment capability
				modelData.StructuredOutput,         // Structured output capability
			)

			if err := s.modelRepo.CreateModel(ctx, model); err != nil {
				s.logger.Error("Failed to create model",
					zap.String("provider_name", provider.Name),
					zap.String("model_name", pricing.Name),
					zap.Error(err))
			} else {
				created++
			}
		}
	}

	// Delete models that no longer exist in provider pricing
	deleted := 0
	for _, model := range existingByName {
		if err := s.modelRepo.DeleteModel(ctx, model.ID); err != nil {
			s.logger.Error("Failed to delete orphaned model",
				zap.String("model_id", model.ID),
				zap.String("model_name", model.ModelName),
				zap.Error(err))
		} else {
			deleted++
		}
	}

	return created, updated, deleted, nil
}

var _ ModelRefreshService = (*modelRefreshService)(nil)
