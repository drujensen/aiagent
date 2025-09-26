package services

import (
	"context"
	"testing"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Mock repository for testing
type mockAgentRepository struct {
	mock.Mock
	agents []*entities.Agent
}

func (m *mockAgentRepository) ListAgents(ctx context.Context) ([]*entities.Agent, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Agent), args.Error(1)
}

func (m *mockAgentRepository) GetAgent(ctx context.Context, id string) (*entities.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Agent), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAgentRepository) CreateAgent(ctx context.Context, agent *entities.Agent) error {
	args := m.Called(ctx, agent)
	m.agents = append(m.agents, agent)
	return args.Error(0)
}

func (m *mockAgentRepository) UpdateAgent(ctx context.Context, agent *entities.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *mockAgentRepository) DeleteAgent(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestAgentService_ListAgents(t *testing.T) {
	mockRepo := new(mockAgentRepository)
	logger := zap.NewNop()
	service := NewAgentService(mockRepo, logger)

	ctx := context.Background()
	expectedAgents := []*entities.Agent{
		entities.NewAgent("TestAgent", "test", "prov1", entities.ProviderOpenAI, "url", "model", "key", "prompt", []string{"tool1"}),
	}

	mockRepo.On("ListAgents", ctx).Return(expectedAgents, nil)

	agents, err := service.ListAgents(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expectedAgents, agents)
	mockRepo.AssertExpectations(t)
}

func TestAgentService_GetAgent(t *testing.T) {
	mockRepo := new(mockAgentRepository)
	logger := zap.NewNop()
	service := NewAgentService(mockRepo, logger)

	ctx := context.Background()
	agent := entities.NewAgent("TestAgent", "test", "prov1", entities.ProviderOpenAI, "url", "model", "key", "prompt", []string{"tool1"})

	t.Run("valid agent", func(t *testing.T) {
		mockRepo.On("GetAgent", ctx, "valid-id").Return(agent, nil).Once()

		result, err := service.GetAgent(ctx, "valid-id")

		assert.NoError(t, err)
		assert.Equal(t, agent, result)
	})

	t.Run("empty id", func(t *testing.T) {
		result, err := service.GetAgent(ctx, "")

		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
		assert.Nil(t, result)
	})
}

func TestAgentService_CreateAgent(t *testing.T) {
	mockRepo := new(mockAgentRepository)
	logger := zap.NewNop()
	service := NewAgentService(mockRepo, logger)

	ctx := context.Background()
	agent := entities.NewAgent("TestAgent", "test", "prov1", entities.ProviderOpenAI, "url", "model", "key", "prompt", []string{"tool1"})

	t.Run("valid agent", func(t *testing.T) {
		mockRepo.On("CreateAgent", ctx, agent).Return(nil).Once()

		err := service.CreateAgent(ctx, agent)

		assert.NoError(t, err)
		assert.NotZero(t, agent.CreatedAt)
		assert.NotZero(t, agent.UpdatedAt)
	})

	t.Run("missing id", func(t *testing.T) {
		invalidAgent := &entities.Agent{Name: "Test"}
		err := service.CreateAgent(ctx, invalidAgent)
		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
	})

	t.Run("missing name", func(t *testing.T) {
		invalidAgent := &entities.Agent{ID: "test-id"}
		err := service.CreateAgent(ctx, invalidAgent)
		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
	})

	t.Run("missing prompt", func(t *testing.T) {
		invalidAgent := &entities.Agent{ID: "test-id", Name: "Test"}
		err := service.CreateAgent(ctx, invalidAgent)
		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
	})

	t.Run("missing model or api key", func(t *testing.T) {
		invalidAgent := &entities.Agent{ID: "test-id", Name: "Test", SystemPrompt: "prompt"}
		err := service.CreateAgent(ctx, invalidAgent)
		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
	})
}

func TestAgentService_UpdateAgent(t *testing.T) {
	mockRepo := new(mockAgentRepository)
	logger := zap.NewNop()
	service := NewAgentService(mockRepo, logger)

	ctx := context.Background()
	agent := entities.NewAgent("TestAgent", "test", "prov1", entities.ProviderOpenAI, "url", "model", "key", "prompt", []string{"tool1"})
	existing := *agent
	existing.CreatedAt = time.Now().Add(-time.Hour)

	t.Run("valid update", func(t *testing.T) {
		mockRepo.On("GetAgent", ctx, agent.ID).Return(&existing, nil).Once()
		mockRepo.On("UpdateAgent", ctx, agent).Return(nil).Once()

		err := service.UpdateAgent(ctx, agent)

		assert.NoError(t, err)
		assert.Equal(t, existing.CreatedAt, agent.CreatedAt)
		assert.True(t, agent.UpdatedAt.After(existing.CreatedAt))
	})

	t.Run("missing id", func(t *testing.T) {
		invalidAgent := &entities.Agent{Name: "Test"}
		err := service.UpdateAgent(ctx, invalidAgent)
		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
	})

	t.Run("agent not found", func(t *testing.T) {
		mockRepo.On("GetAgent", ctx, agent.ID).Return(nil, &errs.NotFoundError{}).Once()
		err := service.UpdateAgent(ctx, agent)
		assert.Error(t, err)
		assert.IsType(t, &errs.NotFoundError{}, err)
	})
}

func TestAgentService_DeleteAgent(t *testing.T) {
	mockRepo := new(mockAgentRepository)
	logger := zap.NewNop()
	service := NewAgentService(mockRepo, logger)

	ctx := context.Background()

	t.Run("valid delete", func(t *testing.T) {
		agent := entities.NewAgent("TestAgent", "test", "prov1", entities.ProviderOpenAI, "url", "model", "key", "prompt", []string{"tool1"})
		mockRepo.On("GetAgent", ctx, agent.ID).Return(agent, nil).Once()
		mockRepo.On("DeleteAgent", ctx, agent.ID).Return(nil).Once()

		err := service.DeleteAgent(ctx, agent.ID)

		assert.NoError(t, err)
	})

	t.Run("empty id", func(t *testing.T) {
		err := service.DeleteAgent(ctx, "")
		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
	})

	t.Run("agent not found", func(t *testing.T) {
		mockRepo.On("GetAgent", ctx, "not-found").Return(nil, &errs.NotFoundError{}).Once()
		err := service.DeleteAgent(ctx, "not-found")
		assert.Error(t, err)
		assert.IsType(t, &errs.NotFoundError{}, err)
	})
}
