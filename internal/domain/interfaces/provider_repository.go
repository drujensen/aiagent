package interfaces

import (
	"context"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

type ProviderRepository interface {
	ListProviders(ctx context.Context) ([]*entities.Provider, error)
	GetProvider(ctx context.Context, id string) (*entities.Provider, error)
	CreateProvider(ctx context.Context, provider *entities.Provider) error
}
