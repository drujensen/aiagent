package services

import (
	"context"
	"sort"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errs "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"go.uber.org/zap"
)

type ModelService interface {
	ListModels(ctx context.Context) ([]*entities.Model, error)
	GetModel(ctx context.Context, id string) (*entities.Model, error)
	CreateModel(ctx context.Context, model *entities.Model) error
	UpdateModel(ctx context.Context, model *entities.Model) error
	DeleteModel(ctx context.Context, id string) error
	GetModelsByProvider(ctx context.Context, providerID string) ([]*entities.Model, error)
}

type modelService struct {
	modelRepo interfaces.ModelRepository
	logger    *zap.Logger
}

func NewModelService(modelRepo interfaces.ModelRepository, logger *zap.Logger) *modelService {
	return &modelService{
		modelRepo: modelRepo,
		logger:    logger,
	}
}

func (s *modelService) ListModels(ctx context.Context) ([]*entities.Model, error) {
	models, err := s.modelRepo.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	// Sort models by provider type, then by name
	sort.Slice(models, func(i, j int) bool {
		if models[i].ProviderType != models[j].ProviderType {
			return models[i].ProviderType < models[j].ProviderType
		}
		return models[i].Name < models[j].Name
	})

	return models, nil
}

func (s *modelService) GetModel(ctx context.Context, id string) (*entities.Model, error) {
	if id == "" {
		return nil, errs.ValidationErrorf("model ID is required")
	}

	model, err := s.modelRepo.GetModel(ctx, id)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func (s *modelService) CreateModel(ctx context.Context, model *entities.Model) error {
	if model.ID == "" {
		return errs.ValidationErrorf("model id is required")
	}
	if model.Name == "" {
		return errs.ValidationErrorf("model name is required")
	}
	if model.ModelName == "" {
		return errs.ValidationErrorf("model name is required")
	}

	model.CreatedAt = time.Now()
	model.UpdatedAt = time.Now()

	return s.modelRepo.CreateModel(ctx, model)
}

func (s *modelService) UpdateModel(ctx context.Context, model *entities.Model) error {
	if model.ID == "" {
		return errs.ValidationErrorf("model ID is required")
	}
	if model.Name == "" {
		return errs.ValidationErrorf("model name is required")
	}

	model.UpdatedAt = time.Now()

	return s.modelRepo.UpdateModel(ctx, model)
}

func (s *modelService) DeleteModel(ctx context.Context, id string) error {
	if id == "" {
		return errs.ValidationErrorf("model ID is required")
	}

	return s.modelRepo.DeleteModel(ctx, id)
}

func (s *modelService) GetModelsByProvider(ctx context.Context, providerID string) ([]*entities.Model, error) {
	models, err := s.modelRepo.GetModelsByProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	return models, nil
}

var _ ModelService = (*modelService)(nil)
