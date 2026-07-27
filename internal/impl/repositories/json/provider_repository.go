package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
)

type JsonProviderRepository struct {
	mu       sync.RWMutex
	filePath string
	data     []*entities.Provider
}

func NewJSONProviderRepository(storageDir string) (interfaces.ProviderRepository, error) {
	filePath := filepath.Join(storageDir, "providers.json")
	repo := &JsonProviderRepository{
		filePath: filePath,
		data:     []*entities.Provider{},
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func copyProvider(p *entities.Provider) *entities.Provider {
	return &entities.Provider{
		ID:         p.ID,
		Name:       p.Name,
		Type:       p.Type,
		BaseURL:    p.BaseURL,
		APIKeyName: p.APIKeyName,
		Models:     slices.Clone(p.Models),
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

// load and save are only ever called by exported methods that already hold
// r.mu, so neither acquires the lock itself.
func (r *JsonProviderRepository) load() error {
	data, err := os.ReadFile(r.filePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet, start with empty data
	}
	if err != nil {
		return errors.InternalErrorf("failed to read providers.json: %v", err)
	}

	var providers []*entities.Provider
	if err := json.Unmarshal(data, &providers); err != nil {
		return errors.InternalErrorf("failed to unmarshal providers.json: %v", err)
	}

	// Validate UUIDs and ensure uniqueness
	seenIDs := make(map[string]struct{})
	for _, provider := range providers {
		if provider.ID == "" {
			return errors.InternalErrorf("provider %s is missing an ID", provider.Name)
		}
		if _, err := uuid.Parse(provider.ID); err != nil {
			return errors.InternalErrorf("provider %s has an invalid UUID: %v", provider.Name, err)
		}
		if _, exists := seenIDs[provider.ID]; exists {
			return errors.InternalErrorf("duplicate provider ID found: %s", provider.ID)
		}
		seenIDs[provider.ID] = struct{}{}
	}

	r.data = providers
	return nil
}

func (r *JsonProviderRepository) save() error {
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return errors.InternalErrorf("failed to marshal providers: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return errors.InternalErrorf("failed to create directory: %v", err)
	}

	return atomicWriteFile(r.filePath, data)
}

func (r *JsonProviderRepository) ListProviders(ctx context.Context) ([]*entities.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providersCopy := make([]*entities.Provider, len(r.data))
	for i, p := range r.data {
		providersCopy[i] = copyProvider(p)
	}
	return providersCopy, nil
}

func (r *JsonProviderRepository) GetProvider(ctx context.Context, id string) (*entities.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, provider := range r.data {
		if provider.ID == id {
			return copyProvider(provider), nil
		}
	}
	return nil, errors.NotFoundErrorf("provider not found: %s", id)
}

func (r *JsonProviderRepository) CreateProvider(ctx context.Context, provider *entities.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if provider.ID == "" {
		provider.ID = uuid.New().String()
	}

	// Check for duplicate ID
	for _, existing := range r.data {
		if existing.ID == provider.ID {
			return errors.DuplicateErrorf("provider with ID %s already exists", provider.ID)
		}
	}

	r.data = append(r.data, copyProvider(provider))
	return r.save()
}

func (r *JsonProviderRepository) UpdateProvider(ctx context.Context, provider *entities.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, existing := range r.data {
		if existing.ID == provider.ID {
			r.data[i] = copyProvider(provider)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("provider not found: %s", provider.ID)
}

var _ interfaces.ProviderRepository = (*JsonProviderRepository)(nil)
