package services

import (
	"context"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type AgentService interface {
	ListAgents(ctx context.Context) ([]*entities.Agent, error)
	GetAgent(ctx context.Context, id string) (*entities.Agent, error)
	CreateAgent(ctx context.Context, agent *entities.Agent) error
	UpdateAgent(ctx context.Context, agent *entities.Agent) error
	DeleteAgent(ctx context.Context, id string) error
}

type agentService struct {
	agentRepo interfaces.AgentRepository
	logger    *zap.Logger
}

func NewAgentService(agentRepo interfaces.AgentRepository, logger *zap.Logger) *agentService {
	return &agentService{
		agentRepo: agentRepo,
		logger:    logger,
	}
}

func (s *agentService) ListAgents(ctx context.Context) ([]*entities.Agent, error) {
	agents, err := s.agentRepo.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	return agents, nil
}

func (s *agentService) GetAgent(ctx context.Context, id string) (*entities.Agent, error) {
	if id == "" {
		return nil, errors.ValidationErrorf("agent ID is required")
	}

	agent, err := s.agentRepo.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (s *agentService) CreateAgent(ctx context.Context, agent *entities.Agent) error {
	if agent.ID == "" {
		return errors.ValidationErrorf("agent id is required")
	}
	if agent.Name == "" {
		return errors.ValidationErrorf("agent name is required")
	}
	if agent.SystemPrompt == "" {
		return errors.ValidationErrorf("agent prompt is required")
	}
	if agent.Model == "" || agent.APIKey == "" {
		return errors.ValidationErrorf("agent endpoint, model, and API key are required")
	}

	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()

	if err := s.agentRepo.CreateAgent(ctx, agent); err != nil {
		return err
	}

	return nil
}

func (s *agentService) UpdateAgent(ctx context.Context, agent *entities.Agent) error {
	if agent.ID == "" {
		return errors.ValidationErrorf("agent ID is required")
	}

	existing, err := s.agentRepo.GetAgent(ctx, agent.ID)
	if err != nil {
		return err
	}

	if agent.Name == "" {
		return errors.ValidationErrorf("agent name is required")
	}
	if agent.SystemPrompt == "" {
		return errors.ValidationErrorf("agent prompt is required")
	}
	if agent.Model == "" || agent.APIKey == "" {
		return errors.ValidationErrorf("agent endpoint, model, and API key are required")
	}

	agent.CreatedAt = existing.CreatedAt
	agent.UpdatedAt = time.Now()

	if err := s.agentRepo.UpdateAgent(ctx, agent); err != nil {
		return err
	}

	return nil
}

func (s *agentService) DeleteAgent(ctx context.Context, id string) error {
	if id == "" {
		return errors.ValidationErrorf("agent ID is required")
	}

	_, err := s.agentRepo.GetAgent(ctx, id)
	if err != nil {
		return err
	}

	if err := s.agentRepo.DeleteAgent(ctx, id); err != nil {
		return err
	}

	return nil
}

// verify interface implementation
var _ AgentService = &agentService{}
