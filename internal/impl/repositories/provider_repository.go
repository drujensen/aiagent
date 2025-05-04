package repositories

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
)

type jsonProviderRepository struct {
	providers []*entities.Provider
}

func NewJSONProviderRepository(configPath string) (interfaces.ProviderRepository, error) {
	filePath := filepath.Join(configPath, "providers.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.InternalErrorf("failed to read providers.json: %v", err)
	}

	var providers []*entities.Provider
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, errors.InternalErrorf("failed to unmarshal providers.json: %v", err)
	}

	// Validate UUIDs and ensure uniqueness
	seenIDs := make(map[string]struct{})
	for _, provider := range providers {
		if provider.ID == "" {
			return nil, errors.InternalErrorf("provider %s is missing an ID", provider.Name)
		}
		if _, err := uuid.Parse(provider.ID); err != nil {
			return nil, errors.InternalErrorf("provider %s has an invalid UUID: %v", provider.Name, err)
		}
		if _, exists := seenIDs[provider.ID]; exists {
			return nil, errors.InternalErrorf("duplicate provider ID found: %s", provider.ID)
		}
		seenIDs[provider.ID] = struct{}{}
	}

	return &jsonProviderRepository{
		providers: providers,
	}, nil
}

func (r *jsonProviderRepository) ListProviders(ctx context.Context) ([]*entities.Provider, error) {
	providersCopy := make([]*entities.Provider, len(r.providers))
	for i, p := range r.providers {
		providersCopy[i] = &entities.Provider{
			ID:         p.ID,
			Name:       p.Name,
			Type:       p.Type,
			BaseURL:    p.BaseURL,
			APIKeyName: p.APIKeyName,
			Models:     p.Models,
		}
		// Note: We’re not converting to ObjectID here since Provider.ID is a string UUID
		// and Agent.ProviderID conversions are handled elsewhere (e.g., in controllers/services)
	}
	return providersCopy, nil
}

func (r *jsonProviderRepository) GetProvider(ctx context.Context, id string) (*entities.Provider, error) {
	for _, provider := range r.providers {
		if provider.ID == id {
			return &entities.Provider{
				ID:         provider.ID,
				Name:       provider.Name,
				Type:       provider.Type,
				BaseURL:    provider.BaseURL,
				APIKeyName: provider.APIKeyName,
				Models:     provider.Models,
			}, nil
		}
	}
	return nil, errors.NotFoundErrorf("provider not found: %s", id)
}

func (r *jsonProviderRepository) GetProviderByType(ctx context.Context, providerType entities.ProviderType) (*entities.Provider, error) {
	for _, provider := range r.providers {
		if provider.Type == providerType {
			return &entities.Provider{
				ID:         provider.ID,
				Name:       provider.Name,
				Type:       provider.Type,
				BaseURL:    provider.BaseURL,
				APIKeyName: provider.APIKeyName,
				Models:     provider.Models,
			}, nil
		}
	}
	return nil, errors.NotFoundErrorf("provider type not found: %s", providerType)
}

var _ interfaces.ProviderRepository = (*jsonProviderRepository)(nil)
