package services

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type ToolService interface {
	ListTools() ([]*entities.Tool, error)
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
