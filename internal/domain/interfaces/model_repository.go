package interfaces

import (
	"context"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

type ModelRepository interface {
	CreateModel(ctx context.Context, model *entities.Model) error
	UpdateModel(ctx context.Context, model *entities.Model) error
	DeleteModel(ctx context.Context, id string) error
	GetModel(ctx context.Context, id string) (*entities.Model, error)
	ListModels(ctx context.Context) ([]*entities.Model, error)
	GetModelsByProvider(ctx context.Context, providerID string) ([]*entities.Model, error)
}
