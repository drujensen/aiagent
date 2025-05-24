package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
)

type JsonAgentRepository struct {
	filePath string
	data     []*entities.Agent
}

func NewJSONAgentRepository(dataDir string) (interfaces.AgentRepository, error) {
	filePath := filepath.Join(dataDir, ".aiagent", "agents.json")
	repo := &JsonAgentRepository{
		filePath: filePath,
		data:     []*entities.Agent{},
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *JsonAgentRepository) load() error {
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

func (r *JsonAgentRepository) save() error {
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

func (r *JsonAgentRepository) ListAgents(ctx context.Context) ([]*entities.Agent, error) {
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

func (r *JsonAgentRepository) GetAgent(ctx context.Context, id string) (*entities.Agent, error) {
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

func (r *JsonAgentRepository) CreateAgent(ctx context.Context, agent *entities.Agent) error {
	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = agent.CreatedAt

	r.data = append(r.data, agent)
	return r.save()
}

func (r *JsonAgentRepository) UpdateAgent(ctx context.Context, agent *entities.Agent) error {
	for i, a := range r.data {
		if a.ID == agent.ID {
			agent.UpdatedAt = time.Now()
			r.data[i] = agent
			return r.save()
		}
	}
	return errors.NotFoundErrorf("agent not found: %s", agent.ID)
}

func (r *JsonAgentRepository) DeleteAgent(ctx context.Context, id string) error {
	for i, a := range r.data {
		if a.ID == id {
			r.data = slices.Delete(r.data, i, i+1)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("agent not found: %s", id)
}

var _ interfaces.AgentRepository = (*JsonAgentRepository)(nil)
