package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
)

type jsonAgentRepository struct {
	filePath string
	data     []*entities.Agent
	mu       sync.RWMutex
}

func NewJSONAgentRepository(dataDir string) (interfaces.AgentRepository, error) {
	filePath := filepath.Join(dataDir, ".aiagent", "agents.json")
	repo := &jsonAgentRepository{
		filePath: filePath,
		data:     []*entities.Agent{},
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *jsonAgentRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet, start with empty data
	}
	if err != nil {
		return errors.InternalErrorf("failed to read agents.json: %v", err)
	}

	var agents []*entities.Agent
	if err := json.Unmarshal(data, &agents); err != nil {
		return errors.InternalErrorf("failed to unmarshal agents.json: %v", err)
	}

	// Validate UUIDs
	for _, agent := range agents {
		if agent.ID == "" {
			return errors.InternalErrorf("agent is missing an ID")
		}
		if _, err := uuid.Parse(agent.ID); err != nil {
			return errors.InternalErrorf("agent has an invalid UUID: %v", err)
		}
	}

	r.data = agents
	return nil
}

func (r *jsonAgentRepository) save() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return errors.InternalErrorf("failed to marshal agents: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return errors.InternalErrorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return errors.InternalErrorf("failed to write agents.json: %v", err)
	}

	return nil
}

func (r *jsonAgentRepository) ListAgents(ctx context.Context) ([]*entities.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agentsCopy := make([]*entities.Agent, len(r.data))
	for i, a := range r.data {
		agentsCopy[i] = &entities.Agent{
			ID:              a.ID,
			Name:            a.Name,
			ProviderID:      a.ProviderID,
			ProviderType:    a.ProviderType,
			Endpoint:        a.Endpoint,
			Model:           a.Model,
			APIKey:          a.APIKey,
			SystemPrompt:    a.SystemPrompt,
			Temperature:     a.Temperature,
			MaxTokens:       a.MaxTokens,
			ContextWindow:   a.ContextWindow,
			ReasoningEffort: a.ReasoningEffort,
			Tools:           slices.Clone(a.Tools),
			CreatedAt:       a.CreatedAt,
			UpdatedAt:       a.UpdatedAt,
		}
	}
	return agentsCopy, nil
}

func (r *jsonAgentRepository) GetAgent(ctx context.Context, id string) (*entities.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, agent := range r.data {
		if agent.ID == id {
			return &entities.Agent{
				ID:              agent.ID,
				Name:            agent.Name,
				ProviderID:      agent.ProviderID,
				ProviderType:    agent.ProviderType,
				Endpoint:        agent.Endpoint,
				Model:           agent.Model,
				APIKey:          agent.APIKey,
				SystemPrompt:    agent.SystemPrompt,
				Temperature:     agent.Temperature,
				MaxTokens:       agent.MaxTokens,
				ContextWindow:   agent.ContextWindow,
				ReasoningEffort: agent.ReasoningEffort,
				Tools:           slices.Clone(agent.Tools),
				CreatedAt:       agent.CreatedAt,
				UpdatedAt:       agent.UpdatedAt,
			}, nil
		}
	}
	return nil, errors.NotFoundErrorf("agent not found: %s", id)
}

func (r *jsonAgentRepository) CreateAgent(ctx context.Context, agent *entities.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = agent.CreatedAt

	r.data = append(r.data, agent)
	return r.save()
}

func (r *jsonAgentRepository) UpdateAgent(ctx context.Context, agent *entities.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, a := range r.data {
		if a.ID == agent.ID {
			agent.UpdatedAt = time.Now()
			r.data[i] = agent
			return r.save()
		}
	}
	return errors.NotFoundErrorf("agent not found: %s", agent.ID)
}

func (r *jsonAgentRepository) DeleteAgent(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, a := range r.data {
		if a.ID == id {
			r.data = slices.Delete(r.data, i, i+1)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("agent not found: %s", id)
}

var _ interfaces.AgentRepository = (*jsonAgentRepository)(nil)
