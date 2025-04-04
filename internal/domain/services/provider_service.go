package services

import (
	"context"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type ProviderService interface {
	CreateProvider(ctx context.Context, name string, providerType entities.ProviderType, baseURL, apiKeyName string, models []entities.ModelPricing) (*entities.Provider, error)
	UpdateProvider(ctx context.Context, id string, name string, providerType entities.ProviderType, baseURL, apiKeyName string, models []entities.ModelPricing) (*entities.Provider, error)
	GetProvider(ctx context.Context, id string) (*entities.Provider, error)
	GetProviderByType(ctx context.Context, providerType entities.ProviderType) (*entities.Provider, error)
	ListProviders(ctx context.Context) ([]*entities.Provider, error)
	DeleteProvider(ctx context.Context, id string) error
	InitializeDefaultProviders(ctx context.Context) error
	ResetDefaultProviders(ctx context.Context) error
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

func (s *providerService) CreateProvider(ctx context.Context, name string, providerType entities.ProviderType, baseURL, apiKeyName string, models []entities.ModelPricing) (*entities.Provider, error) {
	provider := entities.NewProvider(name, providerType, baseURL, apiKeyName, models)

	if err := s.providerRepo.CreateProvider(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

func (s *providerService) UpdateProvider(ctx context.Context, id string, name string, providerType entities.ProviderType, baseURL, apiKeyName string, models []entities.ModelPricing) (*entities.Provider, error) {
	existingProvider, err := s.providerRepo.GetProvider(ctx, id)
	if err != nil {
		return nil, err
	}

	existingProvider.Name = name
	existingProvider.Type = providerType
	existingProvider.BaseURL = baseURL
	existingProvider.APIKeyName = apiKeyName
	existingProvider.Models = models

	if err := s.providerRepo.UpdateProvider(ctx, existingProvider); err != nil {
		return nil, err
	}

	return existingProvider, nil
}

func (s *providerService) GetProvider(ctx context.Context, id string) (*entities.Provider, error) {
	provider, err := s.providerRepo.GetProvider(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			s.logger.Warn("Provider not found by ID, checking if we can find one by type", zap.String("provider_id", id))

			// Validate the ID format
			if _, idErr := primitive.ObjectIDFromHex(id); idErr != nil {
				return nil, errors.ValidationErrorf("invalid provider ID: %s", id)
			}

			// Get providers and check if any match the ID format
			providers, listErr := s.providerRepo.ListProviders(ctx)
			if listErr != nil {
				return nil, err
			}

			// Look for matching provider by ID or provider type
			for _, p := range providers {
				if p.ID.Hex() == id || (p.Type != "" && len(p.Models) > 0) {
					s.logger.Info("Found alternative provider",
						zap.String("provider_name", p.Name),
						zap.String("provider_type", string(p.Type)))
					return p, nil
				}
			}

			// Still nothing found
			return nil, errors.NotFoundErrorf("provider not found: %s", id)
		}
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

func (s *providerService) DeleteProvider(ctx context.Context, id string) error {
	err := s.providerRepo.DeleteProvider(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

// InitializeDefaultProviders creates default provider configurations if they don't exist
func (s *providerService) InitializeDefaultProviders(ctx context.Context) error {
	s.logger.Info("Initializing default providers (if needed)")
	// Always try to initialize to make sure we have default providers
	return s.createDefaultProviders(ctx, false)
}

// ResetDefaultProviders deletes all existing providers and creates the default ones
func (s *providerService) ResetDefaultProviders(ctx context.Context) error {
	return s.createDefaultProviders(ctx, true)
}

// createDefaultProviders does the actual work of creating default providers
func (s *providerService) createDefaultProviders(ctx context.Context, forceReset bool) error {
	providers, err := s.providerRepo.ListProviders(ctx)
	if err != nil {
		return err
	}

	// Check if we already have providers with models
	var hasValidProviders bool
	if len(providers) > 0 {
		// Check if at least one provider has models
		for _, p := range providers {
			if len(p.Models) > 0 {
				hasValidProviders = true
				break
			}
		}

		s.logger.Info("Providers status",
			zap.Int("count", len(providers)),
			zap.Bool("hasValidProviders", hasValidProviders),
			zap.Bool("forceReset", forceReset))
	}

	// If we already have providers with models and not forcing, don't initialize
	if hasValidProviders && !forceReset {
		return nil
	}

	// If forcing, clear existing providers
	if forceReset && len(providers) > 0 {
		s.logger.Info("Forcing reset of providers")
		// Delete existing providers
		for _, p := range providers {
			if err := s.providerRepo.DeleteProvider(ctx, p.ID.Hex()); err != nil {
				s.logger.Warn("Failed to delete provider during reset",
					zap.String("id", p.ID.Hex()),
					zap.Error(err))
			}
		}
	}

	// Create default providers
	defaultProviders := []struct {
		name       string
		type_      entities.ProviderType
		baseURL    string
		apiKeyName string
		models     []entities.ModelPricing
	}{
		{
			name:       "OpenAI",
			type_:      entities.ProviderOpenAI,
			baseURL:    "https://api.openai.com",
			apiKeyName: "OpenAI API Key",
			models: []entities.ModelPricing{
				{
					Name:                "gpt-4o",
					InputPricePerMille:  2.50,  // $2/M input tokens (unchanged per OpenAI pricing trends)
					OutputPricePerMille: 10.00, // $10/M output tokens
					ContextWindow:       128000,
				},
				{
					Name:                "gpt-4o-mini",
					InputPricePerMille:  0.15, // Updated to $0.15/M input (per trends and X posts)
					OutputPricePerMille: 0.60, // Updated to $0.60/M output
					ContextWindow:       128000,
				},
			},
		},
		{
			name:       "Anthropic",
			type_:      entities.ProviderAnthropic,
			baseURL:    "https://api.anthropic.com",
			apiKeyName: "Anthropic API Key",
			models: []entities.ModelPricing{
				{
					Name:                "claude-3-7-sonnet-20250219", // Full model name with date suffix
					InputPricePerMille:  3.00,                         // $3/M input (per Anthropic updates)
					OutputPricePerMille: 15.00,                        // $15/M output
					ContextWindow:       200000,
				},
				{
					Name:                "claude-3-5-sonnet-20240620", // Added date suffix
					InputPricePerMille:  3.00,                         // $3/M input
					OutputPricePerMille: 15.00,                        // $15/M output
					ContextWindow:       200000,
				},
				{
					Name:                "claude-3-haiku-20240307", // Added date suffix
					InputPricePerMille:  0.25,                      // $0.25/M input
					OutputPricePerMille: 1.25,                      // $1.25/M output
					ContextWindow:       200000,
				},
				{
					Name:                "claude-3-opus-20240229", // Added date suffix
					InputPricePerMille:  15.00,                    // $15/M input
					OutputPricePerMille: 75.00,                    // $75/M output
					ContextWindow:       200000,
				},
			},
		},
		{
			name:       "X.AI",
			type_:      entities.ProviderXAI,
			baseURL:    "https://api.x.ai",
			apiKeyName: "X.AI API Key",
			models: []entities.ModelPricing{
				{
					Name:                "grok-1",
					InputPricePerMille:  2.00,
					OutputPricePerMille: 6.00,
					ContextWindow:       128000,
				},
				{
					Name:                "grok-2",
					InputPricePerMille:  2.50,
					OutputPricePerMille: 7.50,
					ContextWindow:       128000,
				},
				{
					Name:                "grok-3",
					InputPricePerMille:  0.00,
					OutputPricePerMille: 0.00,
					ContextWindow:       1000000,
				},
			},
		},
		{
			name:       "Google",
			type_:      entities.ProviderGoogle,
			baseURL:    "https://generativelanguage.googleapis.com",
			apiKeyName: "Gemini API Key",
			models: []entities.ModelPricing{
				{
					Name:                "gemini-2.0-pro-preview-03-25",
					InputPricePerMille:  2.50,
					OutputPricePerMille: 10.00,
					ContextWindow:       1000000,
				},
				{
					Name:                "gemini-2.0-flash",
					InputPricePerMille:  0.10,
					OutputPricePerMille: 0.40,
					ContextWindow:       1000000,
				},
				{
					Name:                "gemini-2.0-flash-lite",
					InputPricePerMille:  0.075,
					OutputPricePerMille: 0.30,
					ContextWindow:       1000000,
				},
				{
					Name:                "gemma-3-27b-it",
					InputPricePerMille:  0.00,
					OutputPricePerMille: 0.00,
					ContextWindow:       128000,
				},
			},
		},
		{
			name:       "DeepSeek",
			type_:      entities.ProviderDeepseek,
			baseURL:    "https://api.deepseek.com",
			apiKeyName: "DeepSeek API Key",
			models: []entities.ModelPricing{
				{
					Name:                "deepseek-chat",
					InputPricePerMille:  0.07, // $0.07/M input (unchanged)
					OutputPricePerMille: 1.10, // $1.10/M output
					ContextWindow:       64000,
				},
				{
					Name:                "deepseek-reasoner",
					InputPricePerMille:  0.55, // $0.27/M input (per Web ID 8)
					OutputPricePerMille: 2.19, // $1.10/M output
					ContextWindow:       64000,
				},
			},
		},
		{
			name:       "Ollama",
			type_:      entities.ProviderOllama,
			baseURL:    "http://localhost:11434",
			apiKeyName: "Local API Key (optional)",
			models: []entities.ModelPricing{
				{
					Name:                "llama3.1:8b", // Updated to latest Llama version
					InputPricePerMille:  0.00,          // Free (local hosting)
					OutputPricePerMille: 0.00,
					ContextWindow:       8192,
				},
				{
					Name:                "mistral-nemo:12b",
					InputPricePerMille:  0.00, // Free
					OutputPricePerMille: 0.00,
					ContextWindow:       8192,
				},
				{
					Name:                "qwen2.5-coder:14b",
					InputPricePerMille:  0.00, // Free
					OutputPricePerMille: 0.00,
					ContextWindow:       8192,
				},
				{
					Name:                "qwen2.5:3b", // Added distilled DeepSeek model
					InputPricePerMille:  0.00,         // Free via Ollama
					OutputPricePerMille: 0.00,
					ContextWindow:       8192,
				},
			},
		},
		{
			name:       "Custom Provider",
			type_:      entities.ProviderGeneric,
			baseURL:    "",
			apiKeyName: "API Key",
			models:     []entities.ModelPricing{},
		},
	}

	for _, p := range defaultProviders {
		provider := entities.NewProvider(p.name, p.type_, p.baseURL, p.apiKeyName, p.models)
		if err := s.providerRepo.CreateProvider(ctx, provider); err != nil {
			return errors.InternalErrorf("failed to create provider %s: %v", p.name, err)
		}
		s.logger.Info("Created default provider", zap.String("name", p.name))
	}

	return nil
}
