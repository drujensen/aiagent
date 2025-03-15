package services

import (
	"context"
	"fmt"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

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
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return provider, nil
}

func (s *providerService) UpdateProvider(ctx context.Context, id string, name string, providerType entities.ProviderType, baseURL, apiKeyName string, models []entities.ModelPricing) (*entities.Provider, error) {
	existingProvider, err := s.providerRepo.GetProvider(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("provider not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get existing provider: %w", err)
	}

	existingProvider.Name = name
	existingProvider.Type = providerType
	existingProvider.BaseURL = baseURL
	existingProvider.APIKeyName = apiKeyName
	existingProvider.Models = models

	if err := s.providerRepo.UpdateProvider(ctx, existingProvider); err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	return existingProvider, nil
}

func (s *providerService) GetProvider(ctx context.Context, id string) (*entities.Provider, error) {
	provider, err := s.providerRepo.GetProvider(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("provider not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return provider, nil
}

func (s *providerService) GetProviderByType(ctx context.Context, providerType entities.ProviderType) (*entities.Provider, error) {
	provider, err := s.providerRepo.GetProviderByType(ctx, providerType)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("provider not found for type: %s", providerType)
		}
		return nil, fmt.Errorf("failed to get provider by type: %w", err)
	}

	return provider, nil
}

func (s *providerService) ListProviders(ctx context.Context) ([]*entities.Provider, error) {
	providers, err := s.providerRepo.ListProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	return providers, nil
}

func (s *providerService) DeleteProvider(ctx context.Context, id string) error {
	err := s.providerRepo.DeleteProvider(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("provider not found: %s", id)
		}
		return fmt.Errorf("failed to delete provider: %w", err)
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
		return fmt.Errorf("failed to list providers: %w", err)
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
					Name:              "gpt-4o",
					InputPricePerMille: 5.0,
					OutputPricePerMille: 15.0,
					ContextWindow:     128000,
				},
				{
					Name:              "gpt-4o-mini",
					InputPricePerMille: 2.0,
					OutputPricePerMille: 6.0,
					ContextWindow:     128000,
				},
				{
					Name:              "gpt-4-turbo",
					InputPricePerMille: 10.0,
					OutputPricePerMille: 30.0,
					ContextWindow:     128000,
				},
				{
					Name:              "gpt-3.5-turbo",
					InputPricePerMille: 0.5,
					OutputPricePerMille: 1.5,
					ContextWindow:     16000,
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
					Name:              "claude-3-5-sonnet",
					InputPricePerMille: 3.0,
					OutputPricePerMille: 15.0,
					ContextWindow:     200000,
				},
				{
					Name:              "claude-3-opus",
					InputPricePerMille: 15.0,
					OutputPricePerMille: 75.0,
					ContextWindow:     200000,
				},
				{
					Name:              "claude-3-sonnet",
					InputPricePerMille: 3.0,
					OutputPricePerMille: 15.0,
					ContextWindow:     200000,
				},
				{
					Name:              "claude-3-haiku",
					InputPricePerMille: 0.25,
					OutputPricePerMille: 1.25,
					ContextWindow:     200000,
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
					Name:              "grok-1",
					InputPricePerMille: 2.0,
					OutputPricePerMille: 6.0,
					ContextWindow:     128000,
				},
				{
					Name:              "grok-2",
					InputPricePerMille: 2.5,
					OutputPricePerMille: 7.5,
					ContextWindow:     128000,
				},
				{
					Name:              "grok-3",
					InputPricePerMille: 0.0, // Pricing not available yet
					OutputPricePerMille: 0.0, // Pricing not available yet
					ContextWindow:     1000000, // Estimated
				},
			},
		},
		{
			name:       "Google",
			type_:      entities.ProviderGoogle,
			baseURL:    "https://generativelanguage.googleapis.com",
			apiKeyName: "Google API Key",
			models: []entities.ModelPricing{
				{
					Name:              "gemini-1.5-pro",
					InputPricePerMille: 3.5,
					OutputPricePerMille: 10.5,
					ContextWindow:     1000000,
				},
				{
					Name:              "gemini-1.5-flash",
					InputPricePerMille: 0.35,
					OutputPricePerMille: 1.05,
					ContextWindow:     1000000,
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
					Name:              "deepseek-coder",
					InputPricePerMille: 0.2,
					OutputPricePerMille: 0.8,
					ContextWindow:     32000,
				},
				{
					Name:              "deepseek-r1-lite",
					InputPricePerMille: 0.15,
					OutputPricePerMille: 0.45,
					ContextWindow:     128000,
				},
				{
					Name:              "deepseek-r1",
					InputPricePerMille: 0.3,
					OutputPricePerMille: 0.9,
					ContextWindow:     128000,
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
					Name:              "llama3",
					InputPricePerMille: 0.0,
					OutputPricePerMille: 0.0,
					ContextWindow:     8192,
				},
				{
					Name:              "mistral",
					InputPricePerMille: 0.0,
					OutputPricePerMille: 0.0,
					ContextWindow:     8192,
				},
				{
					Name:              "phi3",
					InputPricePerMille: 0.0,
					OutputPricePerMille: 0.0,
					ContextWindow:     4096,
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
			return fmt.Errorf("failed to create provider %s: %w", p.name, err)
		}
		s.logger.Info("Created default provider", zap.String("name", p.name))
	}

	return nil
}