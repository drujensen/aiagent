package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"go.uber.org/zap"
)

type AgentSessionService interface {
	CreateSession(ctx context.Context, parentAgentID, subagentID, taskID string) (string, error)
	GetSession(ctx context.Context, sessionID string) (*entities.AgentSession, error)
	UpdateSessionStatus(ctx context.Context, sessionID, status string) error
	CompleteSession(ctx context.Context, sessionID string, result interface{}) error
	ListActiveSessions(ctx context.Context, agentID string) ([]*entities.AgentSession, error)
	CleanupExpiredSessions(ctx context.Context) error
}

type agentSessionService struct {
	sessionRepo interfaces.AgentSessionRepository
	logger      *zap.Logger
	sessions    map[string]*entities.AgentSession
	mutex       sync.RWMutex
}

func NewAgentSessionService(sessionRepo interfaces.AgentSessionRepository, logger *zap.Logger) AgentSessionService {
	return &agentSessionService{
		sessionRepo: sessionRepo,
		logger:      logger,
		sessions:    make(map[string]*entities.AgentSession),
	}
}

func (s *agentSessionService) CreateSession(ctx context.Context, parentAgentID, subagentID, taskID string) (string, error) {
	session := &entities.AgentSession{
		ID:          fmt.Sprintf("session_%d", time.Now().UnixNano()),
		ParentAgent: parentAgentID,
		Subagent:    subagentID,
		TaskID:      taskID,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	// Store in memory for fast access
	s.mutex.Lock()
	s.sessions[session.ID] = session
	s.mutex.Unlock()

	// Persist to repository
	if err := s.sessionRepo.CreateSession(ctx, session); err != nil {
		s.logger.Error("Failed to persist session", zap.String("session_id", session.ID), zap.Error(err))
		// Remove from memory if persistence fails
		s.mutex.Lock()
		delete(s.sessions, session.ID)
		s.mutex.Unlock()
		return "", fmt.Errorf("failed to create session: %v", err)
	}

	s.logger.Info("Created agent session",
		zap.String("session_id", session.ID),
		zap.String("parent_agent", parentAgentID),
		zap.String("subagent", subagentID))

	return session.ID, nil
}

func (s *agentSessionService) GetSession(ctx context.Context, sessionID string) (*entities.AgentSession, error) {
	// Check memory first for fast access
	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	s.mutex.RUnlock()

	if exists {
		return session, nil
	}

	// Fallback to repository
	session, err := s.sessionRepo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %v", err)
	}

	// Cache in memory
	s.mutex.Lock()
	s.sessions[sessionID] = session
	s.mutex.Unlock()

	return session, nil
}

func (s *agentSessionService) UpdateSessionStatus(ctx context.Context, sessionID, status string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Status = status
	if status == "completed" || status == "failed" {
		now := time.Now()
		session.CompletedAt = &now
	}

	// Update in repository
	if err := s.sessionRepo.UpdateSession(ctx, session); err != nil {
		s.logger.Error("Failed to update session in repository",
			zap.String("session_id", sessionID),
			zap.Error(err))
		return fmt.Errorf("failed to update session: %v", err)
	}

	s.logger.Debug("Updated session status",
		zap.String("session_id", sessionID),
		zap.String("status", status))

	return nil
}

func (s *agentSessionService) CompleteSession(ctx context.Context, sessionID string, result interface{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Status = "completed"
	session.Result = result
	now := time.Now()
	session.CompletedAt = &now

	// Update in repository
	if err := s.sessionRepo.UpdateSession(ctx, session); err != nil {
		s.logger.Error("Failed to complete session in repository",
			zap.String("session_id", sessionID),
			zap.Error(err))
		return fmt.Errorf("failed to complete session: %v", err)
	}

	s.logger.Info("Completed agent session",
		zap.String("session_id", sessionID),
		zap.Any("result", result))

	return nil
}

func (s *agentSessionService) ListActiveSessions(ctx context.Context, agentID string) ([]*entities.AgentSession, error) {
	s.mutex.RLock()
	var activeSessions []*entities.AgentSession
	for _, session := range s.sessions {
		if (session.ParentAgent == agentID || session.Subagent == agentID) &&
			(session.Status == "pending" || session.Status == "active") {
			activeSessions = append(activeSessions, session)
		}
	}
	s.mutex.RUnlock()

	// If we have sessions in memory, return them
	if len(activeSessions) > 0 {
		return activeSessions, nil
	}

	// Fallback to repository
	sessions, err := s.sessionRepo.ListActiveSessions(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active sessions: %v", err)
	}

	// Cache in memory
	s.mutex.Lock()
	for _, session := range sessions {
		s.sessions[session.ID] = session
	}
	s.mutex.Unlock()

	return sessions, nil
}

func (s *agentSessionService) CleanupExpiredSessions(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Remove sessions older than 1 hour
	cutoff := time.Now().Add(-1 * time.Hour)
	var expiredSessions []string

	for id, session := range s.sessions {
		if session.CompletedAt != nil && session.CompletedAt.Before(cutoff) {
			expiredSessions = append(expiredSessions, id)
		}
	}

	// Remove from memory
	for _, id := range expiredSessions {
		delete(s.sessions, id)
	}

	// Clean up repository
	if err := s.sessionRepo.CleanupExpiredSessions(ctx, cutoff); err != nil {
		s.logger.Error("Failed to cleanup expired sessions in repository", zap.Error(err))
		return fmt.Errorf("failed to cleanup expired sessions: %v", err)
	}

	if len(expiredSessions) > 0 {
		s.logger.Info("Cleaned up expired sessions", zap.Int("count", len(expiredSessions)))
	}

	return nil
}

// verify that agentSessionService implements AgentSessionService
var _ AgentSessionService = &agentSessionService{}
