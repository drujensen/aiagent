package services

import (
	"aiagent/internal/domain/interfaces"
)

type ToolService interface {
	ListTools() ([]*interfaces.ToolIntegration, error)
}

type toolService struct {
	toolRepo interfaces.ToolRepository
}

func NewToolService(toolRepo interfaces.ToolRepository) *toolService {
	return &toolService{
		toolRepo: toolRepo,
	}
}

func (s *toolService) ListTools() ([]*interfaces.ToolIntegration, error) {
	return s.toolRepo.ListTools()
}
