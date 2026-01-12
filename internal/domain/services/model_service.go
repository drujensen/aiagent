package services

import (
	"context"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
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
	return models, nil
}

func (s *modelService) GetModel(ctx context.Context, id string) (*entities.Model, error) {
	if id == "" {
		return nil, errors.ValidationErrorf("model ID is required")
	}

	model, err := s.modelRepo.GetModel(ctx, id)
	if err != nil {
		return nil, err
	}

	return model, nil
}

func (s *modelService) CreateModel(ctx context.Context, model *entities.Model) error {
	if model.ID == "" {
		return errors.ValidationErrorf("model id is required")
	}
	if model.Name == "" {
		return errors.ValidationErrorf("model name is required")
	}
	if model.ModelName == "" {
		return errors.ValidationErrorf("model name is required")
	}

	model.CreatedAt = time.Now()
	model.UpdatedAt = model.CreatedAt

	if err := s.modelRepo.CreateModel(ctx, model); err != nil {
		return err
	}

	return nil
}

func (s *modelService) UpdateModel(ctx context.Context, model *entities.Model) error {
	if model.ID == "" {
		return errors.ValidationErrorf("model ID is required")
	}

	existing, err := s.modelRepo.GetModel(ctx, model.ID)
	if err != nil {
		return err
	}

	if model.Name == "" {
		return errors.ValidationErrorf("model name is required")
	}

	model.CreatedAt = existing.CreatedAt
	model.UpdatedAt = time.Now()

	if err := s.modelRepo.UpdateModel(ctx, model); err != nil {
		return err
	}

	return nil
}

func (s *modelService) DeleteModel(ctx context.Context, id string) error {
	if id == "" {
		return errors.ValidationErrorf("model ID is required")
	}

	err := s.modelRepo.DeleteModel(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *modelService) GetModelsByProvider(ctx context.Context, providerID string) ([]*entities.Model, error) {
	models, err := s.modelRepo.GetModelsByProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	return models, nil
}

var _ ModelService = (*modelService)(nil)
