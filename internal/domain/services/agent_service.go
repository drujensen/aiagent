package services

import (
	"context"
	"fmt"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type AgentService interface {
	CreateAgent(ctx context.Context, agent *entities.Agent) error
	UpdateAgent(ctx context.Context, agent *entities.Agent) error
	DeleteAgent(ctx context.Context, id string) error
	GetAgent(ctx context.Context, id string) (*entities.Agent, error)
	ListAgents(ctx context.Context) ([]*entities.Agent, error)
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

func (s *agentService) CreateAgent(ctx context.Context, agent *entities.Agent) error {
	if agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if agent.SystemPrompt == "" {
		return fmt.Errorf("agent prompt is required")
	}
	if agent.Endpoint == "" || agent.Model == "" || agent.APIKey == "" {
		return fmt.Errorf("agent endpoint, model, and API key are required")
	}

	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()

	if err := s.agentRepo.CreateAgent(ctx, agent); err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

func (s *agentService) UpdateAgent(ctx context.Context, agent *entities.Agent) error {
	if agent.ID.IsZero() {
		return fmt.Errorf("agent ID is required for update")
	}

	existing, err := s.agentRepo.GetAgent(ctx, agent.ID.Hex())
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("agent not found: %s", agent.ID.Hex())
		}
		return fmt.Errorf("failed to retrieve agent: %w", err)
	}

	if agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if agent.SystemPrompt == "" {
		return fmt.Errorf("agent prompt is required")
	}
	if agent.Endpoint == "" || agent.Model == "" || agent.APIKey == "" {
		return fmt.Errorf("agent endpoint, model, and API key are required")
	}

	agent.CreatedAt = existing.CreatedAt
	agent.UpdatedAt = time.Now()

	if err := s.agentRepo.UpdateAgent(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	return nil
}

func (s *agentService) DeleteAgent(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("agent ID is required for deletion")
	}

	_, err := s.agentRepo.GetAgent(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("agent not found: %s", id)
		}
		return fmt.Errorf("failed to retrieve agent: %w", err)
	}

	if err := s.agentRepo.DeleteAgent(ctx, id); err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

func (s *agentService) GetAgent(ctx context.Context, id string) (*entities.Agent, error) {
	if id == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	agent, err := s.agentRepo.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (s *agentService) ListAgents(ctx context.Context) ([]*entities.Agent, error) {
	agents, err := s.agentRepo.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}
