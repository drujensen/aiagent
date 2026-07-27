package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
)

type JsonAgentRepository struct {
	mu       sync.RWMutex
	filePath string
	data     []*entities.Agent
}

func NewJSONAgentRepository(storageDir string) (interfaces.AgentRepository, error) {
	filePath := filepath.Join(storageDir, "agents.json")
	repo := &JsonAgentRepository{
		filePath: filePath,
		data:     []*entities.Agent{},
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func copyAgent(a *entities.Agent) *entities.Agent {
	return &entities.Agent{
		ID:           a.ID,
		Name:         a.Name,
		SystemPrompt: a.SystemPrompt,
		Tools:        slices.Clone(a.Tools),
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

// load and save are only ever called by exported methods that already hold
// r.mu, so neither acquires the lock itself.
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

	return atomicWriteFile(r.filePath, data)
}

func (r *JsonAgentRepository) ListAgents(ctx context.Context) ([]*entities.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agentsCopy := make([]*entities.Agent, len(r.data))
	for i, a := range r.data {
		agentsCopy[i] = copyAgent(a)
	}
	return agentsCopy, nil
}

func (r *JsonAgentRepository) GetAgent(ctx context.Context, id string) (*entities.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, agent := range r.data {
		if agent.ID == id {
			return copyAgent(agent), nil
		}
	}
	return nil, errors.NotFoundErrorf("agent not found: %s", id)
}

func (r *JsonAgentRepository) CreateAgent(ctx context.Context, agent *entities.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = agent.CreatedAt

	r.data = append(r.data, copyAgent(agent))
	return r.save()
}

func (r *JsonAgentRepository) UpdateAgent(ctx context.Context, agent *entities.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, a := range r.data {
		if a.ID == agent.ID {
			agent.UpdatedAt = time.Now()
			r.data[i] = copyAgent(agent)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("agent not found: %s", agent.ID)
}

func (r *JsonAgentRepository) DeleteAgent(ctx context.Context, id string) error {
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

var _ interfaces.AgentRepository = (*JsonAgentRepository)(nil)
