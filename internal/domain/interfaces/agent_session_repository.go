package interfaces

import (
	"context"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

type AgentSessionRepository interface {
	CreateSession(ctx context.Context, session *entities.AgentSession) error
	UpdateSession(ctx context.Context, session *entities.AgentSession) error
	GetSession(ctx context.Context, sessionID string) (*entities.AgentSession, error)
	ListActiveSessions(ctx context.Context, agentID string) ([]*entities.AgentSession, error)
	CleanupExpiredSessions(ctx context.Context, cutoff time.Time) error
}
