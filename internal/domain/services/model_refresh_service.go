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

// modelWithTime holds model data with parsed release time
type modelWithTime struct {
	model       modelsdev.ModelData
	releaseTime time.Time
}

// filterLatestModelsPerFamily filters models to only include the latest release per family and version
func filterLatestModelsPerFamily(models map[string]modelsdev.ModelData) map[string]modelsdev.ModelData {
	familyLatest := make(map[string]modelWithTime)
	datePattern := regexp.MustCompile(`-(\d{4}-\d{2}-\d{2}|\d{8})$`)

	for _, model := range models {
		if model.Family == "" {
			continue // Skip models without family
		}

		// Extract version from model name for grouping
		version := ""
		if matches := regexp.MustCompile(`Claude \w+ ([\d.]+)`).FindStringSubmatch(model.Name); len(matches) > 1 {
			version = matches[1]
		}

		// Create group key: family + version
		groupKey := model.Family
		if version != "" {
			groupKey += "-" + version
		} else {
			groupKey += "-unknown"
		}

		// Try to extract date from model ID first
		var modelTime time.Time
		if matches := datePattern.FindStringSubmatch(model.ID); len(matches) > 1 {
			dateStr := matches[1]
			if len(dateStr) == 8 { // YYYYMMDD format
				if t, err := time.Parse("20060102", dateStr); err == nil {
					modelTime = t
				}
			} else { // YYYY-MM-DD format
				if t, err := time.Parse("2006-01-02", dateStr); err == nil {
					modelTime = t
				}
			}
		}

		// Fall back to release_date if no ID date found
		if modelTime.IsZero() {
			if t, err := time.Parse("2006-01-02", model.ReleaseDate); err != nil {
				continue // Skip models with invalid dates
			} else {
				modelTime = t
			}
		}

		if existing, exists := familyLatest[groupKey]; !exists || modelTime.After(existing.releaseTime) {
			familyLatest[groupKey] = modelWithTime{model: model, releaseTime: modelTime}
		}
	}

	// Reconstruct map with only latest per group
	filtered := make(map[string]modelsdev.ModelData)
	for _, mwt := range familyLatest {
		filtered[mwt.model.ID] = mwt.model
	}

	return filtered
}

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
	// Check if this provider is defined in global config (custom provider)
	// For custom providers, we need to find the config entry by matching names
	var customConfig *config.CustomProviderConfig
	var configKey string
	for key, cfg := range s.globalConfig.Providers {
		if cfg.Name == provider.Name && cfg.Type == "generic" {
			customConfig = &cfg
			configKey = key
			break
		}
	}

	if customConfig != nil {
		s.logger.Debug("Using custom provider config",
			zap.String("provider_id", provider.ID),
			zap.String("provider_name", provider.Name),
			zap.String("config_key", configKey))

		// Update provider with config data
		providerToUpdate := &entities.Provider{
			ID:         provider.ID,
			Name:       customConfig.Name,
			Type:       entities.ProviderType(customConfig.Type),
			BaseURL:    customConfig.BaseURL,
			APIKeyName: customConfig.APIKeyName,
			Models:     make([]entities.ModelPricing, 0),
		}

		// Add models from config
		for modelName, modelConfig := range customConfig.Models {
			pricing := entities.ModelPricing{
				Name:                modelName,
				InputPricePerMille:  modelConfig.InputPricePerMille,
				OutputPricePerMille: modelConfig.OutputPricePerMille,
				ContextWindow:       modelConfig.ContextWindow,
				MaxOutputTokens:     modelConfig.MaxOutputTokens,
			}
			providerToUpdate.Models = append(providerToUpdate.Models, pricing)
		}

		providerToUpdate.UpdatedAt = time.Now()
		if err := s.providerRepo.UpdateProvider(ctx, providerToUpdate); err != nil {
			return err
		}

		s.logger.Info("Updated custom provider from config",
			zap.String("provider_id", provider.ID),
			zap.String("provider_name", provider.Name),
			zap.String("config_key", configKey),
			zap.Int("models_count", len(providerToUpdate.Models)))
		return nil
	}

	// Original logic for built-in providers
	if provider.Type == entities.ProviderGeneric {
		s.logger.Debug("Skipping generic provider refresh (no config found)",
			zap.String("provider_id", provider.ID),
			zap.String("provider_name", provider.Name))
		return nil
	}

	fetched, err := s.modelsDevClient.Fetch()
	if err != nil {
		return err
	}

	// Filter Anthropic models to only include the latest per family
	if anthropicData, exists := (*fetched)["anthropic"]; exists {
		originalCount := len(anthropicData.Models)
		anthropicData.Models = filterLatestModelsPerFamily(anthropicData.Models)
		filteredCount := len(anthropicData.Models)
		s.logger.Info("Filtered Anthropic models in provider refresh",
			zap.Int("original_count", originalCount),
			zap.Int("filtered_count", filteredCount))
		(*fetched)["anthropic"] = anthropicData
	}

	providerToUpdate := &entities.Provider{
		ID:         provider.ID,
		Name:       provider.Name,
		Type:       provider.Type,
		BaseURL:    provider.BaseURL,
		APIKeyName: provider.APIKeyName,
		Models:     make([]entities.ModelPricing, 0),
	}

	// Handle provider key mapping for models.dev
	providerKey := string(provider.Type)
	if provider.Type == entities.ProviderTogether {
		providerKey = "togetherai" // models.dev uses "togetherai" not "together"
	}

	for _, modelData := range (*fetched)[providerKey].Models {
		pricing := entities.ModelPricing{
			Name:                modelData.ID,
			InputPricePerMille:  modelData.Cost.Input,  // Keep as per million tokens
			OutputPricePerMille: modelData.Cost.Output, // Keep as per million tokens
			ContextWindow:       modelData.Limit.Context,
			MaxOutputTokens:     modelData.Limit.Output,
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

func (s *modelRefreshService) SyncAllModels(ctx context.Context) error {
	s.logger.Info("Starting full model sync")

	// Fetch latest data from models.dev
	modelsDevData, err := s.modelsDevClient.Fetch()
	if err != nil {
		return fmt.Errorf("failed to fetch models.dev data: %w", err)
	}

	// Filter Anthropic models to only include the latest per family
	if anthropicData, exists := (*modelsDevData)["anthropic"]; exists {
		originalCount := len(anthropicData.Models)
		anthropicData.Models = filterLatestModelsPerFamily(anthropicData.Models)
		filteredCount := len(anthropicData.Models)
		s.logger.Info("Filtered Anthropic models",
			zap.Int("original_count", originalCount),
			zap.Int("filtered_count", filteredCount))
		(*modelsDevData)["anthropic"] = anthropicData
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
			// Update MaxTokens if pricing has MaxOutputTokens and it's different
			expectedMaxTokens := pricing.MaxOutputTokens
			if expectedMaxTokens <= 0 {
				expectedMaxTokens = int(float64(pricing.ContextWindow) * s.globalConfig.DefaultMaxTokensRatio)
			}
			if existingModel.MaxTokens == nil || *existingModel.MaxTokens != expectedMaxTokens {
				existingModel.MaxTokens = &expectedMaxTokens
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
			maxTokens := pricing.MaxOutputTokens
			if maxTokens <= 0 {
				maxTokens = int(float64(pricing.ContextWindow) * s.globalConfig.DefaultMaxTokensRatio)
			}
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
