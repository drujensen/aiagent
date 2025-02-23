package services

import (
	"context"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"
)

type ToolService interface {
	ListTools(ctx context.Context) ([]*entities.Tool, error)
}

type toolService struct {
	toolRepo interfaces.ToolRepository
}

func NewToolService(toolRepo interfaces.ToolRepository) *toolService {
	return &toolService{
		toolRepo: toolRepo,
	}
}

func (s *toolService) ListTools(ctx context.Context) ([]*entities.Tool, error) {
	return s.toolRepo.ListTools(ctx)
}
