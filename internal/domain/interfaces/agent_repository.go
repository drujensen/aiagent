package interfaces

import (
	"context"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

type AgentRepository interface {
	CreateAgent(ctx context.Context, agent *entities.Agent) error
	UpdateAgent(ctx context.Context, agent *entities.Agent) error
	DeleteAgent(ctx context.Context, id string) error
	GetAgent(ctx context.Context, id string) (*entities.Agent, error)
	ListAgents(ctx context.Context) ([]*entities.Agent, error)
}
