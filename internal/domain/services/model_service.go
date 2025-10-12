package services

import (
	"context"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type ModelService interface {
	ListModels(ctx context.Context) ([]*entities.Model, error)
	GetModel(ctx context.Context, id string) (*entities.Model, error)
}

type modelService struct {
	modelRepo interfaces.ModelRepository
	logger    *zap.Logger
}

func NewModelService(modelRepo interfaces.ModelRepository, logger *zap.Logger) ModelService {
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

// verify interface implementation
var _ ModelService = &modelService{}
