package repositories_json

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"github.com/google/uuid"
)

type agentSessionRepository struct {
	filePath string
	sessions map[string]*entities.AgentSession
	mutex    sync.RWMutex
}

func NewAgentSessionRepository(dataDir string) interfaces.AgentSessionRepository {
	repo := &agentSessionRepository{
		filePath: filepath.Join(dataDir, "agent_sessions.json"),
		sessions: make(map[string]*entities.AgentSession),
	}

	// Load existing sessions
	repo.loadSessions()

	return repo
}

func (r *agentSessionRepository) CreateSession(ctx context.Context, session *entities.AgentSession) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	r.sessions[session.ID] = session
	return r.saveSessions()
}

func (r *agentSessionRepository) UpdateSession(ctx context.Context, session *entities.AgentSession) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.sessions[session.ID]; !exists {
		return fmt.Errorf("session not found: %s", session.ID)
	}

	r.sessions[session.ID] = session
	return r.saveSessions()
}

func (r *agentSessionRepository) GetSession(ctx context.Context, sessionID string) (*entities.AgentSession, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	session, exists := r.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Return a copy to prevent external modification
	sessionCopy := *session
	return &sessionCopy, nil
}

func (r *agentSessionRepository) ListActiveSessions(ctx context.Context, agentID string) ([]*entities.AgentSession, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var activeSessions []*entities.AgentSession
	for _, session := range r.sessions {
		if (session.ParentAgent == agentID || session.Subagent == agentID) &&
			(session.Status == "pending" || session.Status == "active") {
			// Return a copy to prevent external modification
			sessionCopy := *session
			activeSessions = append(activeSessions, &sessionCopy)
		}
	}

	return activeSessions, nil
}

func (r *agentSessionRepository) CleanupExpiredSessions(ctx context.Context, cutoff time.Time) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var expiredIDs []string
	for id, session := range r.sessions {
		if session.CompletedAt != nil && session.CompletedAt.Before(cutoff) {
			expiredIDs = append(expiredIDs, id)
		}
	}

	for _, id := range expiredIDs {
		delete(r.sessions, id)
	}

	return r.saveSessions()
}

func (r *agentSessionRepository) loadSessions() error {
	if _, err := os.Stat(r.filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty sessions
		return nil
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to read sessions file: %v", err)
	}

	if len(data) == 0 {
		return nil
	}

	var sessions map[string]*entities.AgentSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return fmt.Errorf("failed to unmarshal sessions: %v", err)
	}

	r.sessions = sessions
	return nil
}

func (r *agentSessionRepository) saveSessions() error {
	data, err := json.MarshalIndent(r.sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sessions: %v", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write sessions file: %v", err)
	}

	return nil
}
