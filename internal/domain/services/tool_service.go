package services

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errs"
	"aiagent/internal/domain/interfaces"
	"context"
	"time"

	"go.uber.org/zap"
)

type ToolService interface {
	ListTools() ([]*entities.Tool, error)

	ListToolData(ctx context.Context) ([]*entities.ToolData, error)
	GetToolData(ctx context.Context, id string) (*entities.ToolData, error)
	CreateToolData(ctx context.Context, tool *entities.ToolData) error
	UpdateToolData(ctx context.Context, tool *entities.ToolData) error
	DeleteToolData(ctx context.Context, id string) error
}

type toolService struct {
	toolRepo interfaces.ToolRepository
	logger   *zap.Logger
}

func NewToolService(toolRepo interfaces.ToolRepository, logger *zap.Logger) *toolService {
	return &toolService{
		toolRepo: toolRepo,
	}
}

func (s *toolService) ListTools() ([]*entities.Tool, error) {
	return s.toolRepo.ListTools()
}

func (s *toolService) ListToolData(ctx context.Context) ([]*entities.ToolData, error) {
	tools, err := s.toolRepo.ListToolData(ctx)
	if err != nil {
		return nil, err
	}

	return tools, nil
}

func (s *toolService) GetToolData(ctx context.Context, id string) (*entities.ToolData, error) {
	if id == "" {
		return nil, errors.ValidationErrorf("tool ID is required")
	}

	tool, err := s.toolRepo.GetToolData(ctx, id)
	if err != nil {
		return nil, err
	}

	return tool, nil
}

func (s *toolService) CreateToolData(ctx context.Context, tool *entities.ToolData) error {
	if tool.ID == "" {
		return errors.ValidationErrorf("tool id is required")
	}
	if tool.Name == "" {
		return errors.ValidationErrorf("tool name is required")
	}
	if tool.Description == "" {
		return errors.ValidationErrorf("tool description is required")
	}

	tool.CreatedAt = time.Now()
	tool.UpdatedAt = time.Now()

	if err := s.toolRepo.CreateToolData(ctx, tool); err != nil {
		return err
	}

	return nil
}

func (s *toolService) UpdateToolData(ctx context.Context, tool *entities.ToolData) error {
	if tool.ID == "" {
		return errors.ValidationErrorf("tool ID is required")
	}

	existing, err := s.toolRepo.GetToolData(ctx, tool.ID)
	if err != nil {
		return err
	}

	if tool.Name == "" {
		return errors.ValidationErrorf("tool name is required")
	}
	if tool.Description == "" {
		return errors.ValidationErrorf("tool description is required")
	}

	tool.CreatedAt = existing.CreatedAt
	tool.UpdatedAt = time.Now()

	if err := s.toolRepo.UpdateToolData(ctx, tool); err != nil {
		return err
	}

	return nil
}

func (s *toolService) DeleteToolData(ctx context.Context, id string) error {
	if id == "" {
		return errors.ValidationErrorf("tool ID is required")
	}

	_, err := s.toolRepo.GetToolData(ctx, id)
	if err != nil {
		return err
	}

	if err := s.toolRepo.DeleteToolData(ctx, id); err != nil {
		return err
	}

	return nil
}
