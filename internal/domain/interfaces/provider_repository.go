package interfaces

import (
	"context"

	"aiagent/internal/domain/entities"
)

type ProviderRepository interface {
	GetProvider(ctx context.Context, id string) (*entities.Provider, error)
	GetProviderByType(ctx context.Context, providerType entities.ProviderType) (*entities.Provider, error)
	ListProviders(ctx context.Context) ([]*entities.Provider, error)
}
