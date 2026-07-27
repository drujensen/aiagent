package services

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// mockChatServiceForPlan is a from-scratch mock of the 13-method ChatService
// interface, needed here because PlanService.DispatchTask/DispatchPlan only
// depend on CreateSubChat and SendMessage but must satisfy the full interface
// to type-check as a ChatService dependency.
type mockChatServiceForPlan struct {
	mock.Mock
}

func (m *mockChatServiceForPlan) ListChats(ctx context.Context) ([]*entities.Chat, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Chat), args.Error(1)
}

func (m *mockChatServiceForPlan) GetChat(ctx context.Context, id string) (*entities.Chat, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForPlan) GetActiveChat(ctx context.Context) (*entities.Chat, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForPlan) SetActiveChat(ctx context.Context, chatID string) error {
	args := m.Called(ctx, chatID)
	return args.Error(0)
}

func (m *mockChatServiceForPlan) CreateChat(ctx context.Context, agentID, modelID, name string) (*entities.Chat, error) {
	args := m.Called(ctx, agentID, modelID, name)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForPlan) CreateSubChat(ctx context.Context, agentID, modelID, name, parentChatID string) (*entities.Chat, error) {
	args := m.Called(ctx, agentID, modelID, name, parentChatID)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForPlan) UpdateChat(ctx context.Context, id, agentID, modelID, name string) (*entities.Chat, error) {
	args := m.Called(ctx, id, agentID, modelID, name)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForPlan) DeleteChat(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockChatServiceForPlan) SendMessage(ctx context.Context, id string, message *entities.Message) (*entities.Message, error) {
	args := m.Called(ctx, id, message)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Message), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForPlan) SaveMessagesIncrementally(ctx context.Context, chatID string, messages []*entities.Message) error {
	args := m.Called(ctx, chatID, messages)
	return args.Error(0)
}

func (m *mockChatServiceForPlan) CalculateTotalChatCost(ctx context.Context, chatID string) (float64, error) {
	args := m.Called(ctx, chatID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockChatServiceForPlan) GenerateAndUpdateTitle(ctx context.Context, chatID string) (*entities.Chat, error) {
	args := m.Called(ctx, chatID)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForPlan) ExecuteSkill(ctx context.Context, chatID, skillName string) error {
	args := m.Called(ctx, chatID, skillName)
	return args.Error(0)
}

var _ ChatService = (*mockChatServiceForPlan)(nil)

// mockAgentServiceForPlan mocks AgentService; only GetAgent is exercised by
// PlanService, but the full interface must be implemented to type-check.
type mockAgentServiceForPlan struct {
	mock.Mock
}

func (m *mockAgentServiceForPlan) ListAgents(ctx context.Context) ([]*entities.Agent, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Agent), args.Error(1)
}

func (m *mockAgentServiceForPlan) GetAgent(ctx context.Context, id string) (*entities.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Agent), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAgentServiceForPlan) CreateAgent(ctx context.Context, agent *entities.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *mockAgentServiceForPlan) UpdateAgent(ctx context.Context, agent *entities.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *mockAgentServiceForPlan) DeleteAgent(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

var _ AgentService = (*mockAgentServiceForPlan)(nil)

// mockModelServiceForPlan mocks ModelService; only ListModels is exercised by
// PlanService.resolveDispatchModel.
type mockModelServiceForPlan struct {
	mock.Mock
}

func (m *mockModelServiceForPlan) ListModels(ctx context.Context) ([]*entities.Model, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Model), args.Error(1)
}

func (m *mockModelServiceForPlan) GetModel(ctx context.Context, id string) (*entities.Model, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Model), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockModelServiceForPlan) CreateModel(ctx context.Context, model *entities.Model) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *mockModelServiceForPlan) UpdateModel(ctx context.Context, model *entities.Model) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *mockModelServiceForPlan) DeleteModel(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockModelServiceForPlan) GetModelsByProvider(ctx context.Context, providerID string) ([]*entities.Model, error) {
	args := m.Called(ctx, providerID)
	return args.Get(0).([]*entities.Model), args.Error(1)
}

var _ ModelService = (*mockModelServiceForPlan)(nil)

// fakePlanRepository is an in-memory interfaces.PlanRepository used instead
// of the real JSON repository, since domain/services must never import
// impl/repositories/json (that would be a layer-violation import cycle:
// impl already imports domain/services via tool_factory.go).
type fakePlanRepository struct {
	mu   sync.Mutex
	data map[string]*entities.Plan
}

func newFakePlanRepository() *fakePlanRepository {
	return &fakePlanRepository{data: make(map[string]*entities.Plan)}
}

func (r *fakePlanRepository) CreatePlan(ctx context.Context, plan *entities.Plan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[plan.ID] = plan
	return nil
}

func (r *fakePlanRepository) UpdatePlan(ctx context.Context, plan *entities.Plan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[plan.ID]; !ok {
		return errors.NotFoundErrorf("plan not found: %s", plan.ID)
	}
	r.data[plan.ID] = plan
	return nil
}

func (r *fakePlanRepository) DeletePlan(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[id]; !ok {
		return errors.NotFoundErrorf("plan not found: %s", id)
	}
	delete(r.data, id)
	return nil
}

func (r *fakePlanRepository) GetPlan(ctx context.Context, id string) (*entities.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	plan, ok := r.data[id]
	if !ok {
		return nil, errors.NotFoundErrorf("plan not found: %s", id)
	}
	return plan, nil
}

func (r *fakePlanRepository) ListPlans(ctx context.Context) ([]*entities.Plan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	plans := make([]*entities.Plan, 0, len(r.data))
	for _, p := range r.data {
		plans = append(plans, p)
	}
	return plans, nil
}

var _ interfaces.PlanRepository = (*fakePlanRepository)(nil)

// fakeTaskRepository is an in-memory interfaces.TaskRepository, used for the
// same import-cycle-avoidance reason as fakePlanRepository above. It uses a
// mutex because DispatchPlan fans out concurrent goroutines that each read
// and write through this repository.
type fakeTaskRepository struct {
	mu   sync.Mutex
	data map[string]*entities.Task
}

func newFakeTaskRepository() *fakeTaskRepository {
	return &fakeTaskRepository{data: make(map[string]*entities.Task)}
}

func (r *fakeTaskRepository) CreateTask(ctx context.Context, task *entities.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[task.ID] = task
	return nil
}

func (r *fakeTaskRepository) UpdateTask(ctx context.Context, task *entities.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[task.ID]; !ok {
		return errors.NotFoundErrorf("task not found: %s", task.ID)
	}
	r.data[task.ID] = task
	return nil
}

func (r *fakeTaskRepository) DeleteTask(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[id]; !ok {
		return errors.NotFoundErrorf("task not found: %s", id)
	}
	delete(r.data, id)
	return nil
}

func (r *fakeTaskRepository) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.data[id]
	if !ok {
		return nil, errors.NotFoundErrorf("task not found: %s", id)
	}
	return task, nil
}

func (r *fakeTaskRepository) ListTasks(ctx context.Context) ([]*entities.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tasks := make([]*entities.Task, 0, len(r.data))
	for _, t := range r.data {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

var _ interfaces.TaskRepository = (*fakeTaskRepository)(nil)

func testModel() *entities.Model {
	temp := 0.7
	maxTokens := 4096
	contextWindow := 128000
	return entities.NewModel("Test Model", "provider-1", entities.ProviderAnthropic, "claude-x", "key", &temp, &maxTokens, &contextWindow, "", "", false, true, false, false, false)
}

func newTestPlanService(planRepo interfaces.PlanRepository, taskRepo interfaces.TaskRepository, agentService AgentService, modelService ModelService, chatService ChatService) PlanService {
	return NewPlanService(planRepo, taskRepo, agentService, modelService, chatService, zap.NewNop())
}

// --- PlanService orchestration (AC5, AC5b) ---

func TestPlanService_CreateAndDecompose(t *testing.T) {
	planRepo := newFakePlanRepository()
	taskRepo := newFakeTaskRepository()
	mockAgent := new(mockAgentServiceForPlan)
	mockModel := new(mockModelServiceForPlan)
	mockChat := new(mockChatServiceForPlan)

	service := newTestPlanService(planRepo, taskRepo, mockAgent, mockModel, mockChat)
	ctx := context.Background()

	plan, err := service.CreatePlan(ctx, "Ship feature X", []string{"no downtime"}, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, plan.ID)

	tasks, err := service.DecomposeIntoTasks(ctx, plan.ID, []TaskSpec{
		{Name: "Task A", Content: "Do A", Priority: entities.TaskPriorityHigh, AssignedAgentID: "agent-a"},
		{Name: "Task B", Content: "Do B", Priority: entities.TaskPriorityMedium, AssignedAgentID: "agent-b"},
	})
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
	for _, task := range tasks {
		assert.Equal(t, plan.ID, task.PlanID)
		assert.Equal(t, entities.TaskStatusPending, task.Status)
	}

	// Unknown plan ID must fail decomposition.
	_, err = service.DecomposeIntoTasks(ctx, "does-not-exist", []TaskSpec{{Name: "x", Content: "y"}})
	assert.Error(t, err)
}

func TestPlanService_DispatchTask(t *testing.T) {
	planRepo := newFakePlanRepository()
	taskRepo := newFakeTaskRepository()
	mockAgent := new(mockAgentServiceForPlan)
	mockModel := new(mockModelServiceForPlan)
	mockChat := new(mockChatServiceForPlan)

	service := newTestPlanService(planRepo, taskRepo, mockAgent, mockModel, mockChat)
	ctx := context.Background()

	plan, err := service.CreatePlan(ctx, "Ship feature X", nil, nil)
	assert.NoError(t, err)
	tasks, err := service.DecomposeIntoTasks(ctx, plan.ID, []TaskSpec{
		{Name: "Task A", Content: "Do A", Priority: entities.TaskPriorityHigh, AssignedAgentID: "agent-a"},
	})
	assert.NoError(t, err)
	task := tasks[0]

	agent := entities.NewAgent("Agent A", "You are helpful", nil)
	agent.ID = "agent-a"
	model := testModel()

	chat := entities.NewChat(agent.ID, model.ID, "Task: Task A")
	response := entities.NewMessage("assistant", "done")

	mockAgent.On("GetAgent", ctx, "agent-a").Return(agent, nil)
	mockModel.On("ListModels", ctx).Return([]*entities.Model{model}, nil)
	mockChat.On("CreateSubChat", ctx, agent.ID, model.ID, "Task: Task A", "").Return(chat, nil)
	mockChat.On("SendMessage", ctx, chat.ID, mock.MatchedBy(func(m *entities.Message) bool {
		return m.Content == "Do A"
	})).Return(response, nil)

	result, err := service.DispatchTask(ctx, task.ID)
	assert.NoError(t, err)
	assert.Equal(t, task.ID, result.TaskID)
	assert.Equal(t, chat.ID, result.ChatID)
	assert.Equal(t, "done", result.Result)
	assert.Nil(t, result.Err)

	completed, err := taskRepo.GetTask(ctx, task.ID)
	assert.NoError(t, err)
	assert.Equal(t, entities.TaskStatusCompleted, completed.Status)

	mockAgent.AssertExpectations(t)
	mockModel.AssertExpectations(t)
	mockChat.AssertExpectations(t)
}

func TestPlanService_DispatchTask_NoAssignedAgent(t *testing.T) {
	planRepo := newFakePlanRepository()
	taskRepo := newFakeTaskRepository()
	service := newTestPlanService(planRepo, taskRepo, new(mockAgentServiceForPlan), new(mockModelServiceForPlan), new(mockChatServiceForPlan))
	ctx := context.Background()

	plan, err := service.CreatePlan(ctx, "Goal", nil, nil)
	assert.NoError(t, err)
	tasks, err := service.DecomposeIntoTasks(ctx, plan.ID, []TaskSpec{{Name: "Unassigned", Content: "x"}})
	assert.NoError(t, err)

	_, err = service.DispatchTask(ctx, tasks[0].ID)
	assert.Error(t, err)
}

// TestPlanService_DispatchPlan_ConcurrentAttribution dispatches multiple
// tasks under one plan concurrently and verifies each DispatchResult is
// attributed to the correct originating task, not cross-contaminated by the
// goroutine fan-out in DispatchPlan.
func TestPlanService_DispatchPlan_ConcurrentAttribution(t *testing.T) {
	planRepo := newFakePlanRepository()
	taskRepo := newFakeTaskRepository()
	mockAgent := new(mockAgentServiceForPlan)
	mockModel := new(mockModelServiceForPlan)
	mockChat := new(mockChatServiceForPlan)

	service := newTestPlanService(planRepo, taskRepo, mockAgent, mockModel, mockChat)
	ctx := context.Background()

	plan, err := service.CreatePlan(ctx, "Fan-out goal", nil, nil)
	assert.NoError(t, err)

	const n = 8
	specs := make([]TaskSpec, n)
	for i := 0; i < n; i++ {
		specs[i] = TaskSpec{
			Name:            fmt.Sprintf("Task-%d", i),
			Content:         fmt.Sprintf("content-%d", i),
			Priority:        entities.TaskPriorityMedium,
			AssignedAgentID: "agent-shared",
		}
	}
	tasks, err := service.DecomposeIntoTasks(ctx, plan.ID, specs)
	assert.NoError(t, err)

	agent := entities.NewAgent("Shared Agent", "prompt", nil)
	agent.ID = "agent-shared"
	model := testModel()

	mockAgent.On("GetAgent", ctx, "agent-shared").Return(agent, nil)
	mockModel.On("ListModels", ctx).Return([]*entities.Model{model}, nil)

	// Each task's Name/Content is unique (Task-N/content-N), so each mock
	// expectation below is keyed by distinct argument values rather than
	// testify's FIFO .Once() registration order - the goroutines in
	// DispatchPlan invoke these concurrently in a nondeterministic order,
	// so argument-based matching (not call order) is what proves correct
	// per-task attribution.
	for i, task := range tasks {
		chatName := fmt.Sprintf("Task: Task-%d", i)
		chat := entities.NewChat(agent.ID, model.ID, chatName)
		response := entities.NewMessage("assistant", "result-for-"+task.ID)
		mockChat.On("CreateSubChat", ctx, agent.ID, model.ID, chatName, "").Return(chat, nil)
		mockChat.On("SendMessage", ctx, chat.ID, mock.MatchedBy(func(m *entities.Message) bool {
			return m.Content == task.Content
		})).Return(response, nil)
	}

	results, err := service.DispatchPlan(ctx, plan.ID)
	assert.NoError(t, err)
	assert.Len(t, results, n)

	seen := make(map[string]bool, n)
	for i, result := range results {
		assert.NotNil(t, result, "result at index %d must not be nil", i)
		assert.NoError(t, result.Err)
		assert.Equal(t, "result-for-"+result.TaskID, result.Result)
		assert.False(t, seen[result.TaskID], "task ID %s attributed to more than one result", result.TaskID)
		seen[result.TaskID] = true
	}
	for _, task := range tasks {
		assert.True(t, seen[task.ID], "task %s missing from dispatch results", task.ID)
	}
}

func TestPlanService_DispatchPlan_UnknownPlan(t *testing.T) {
	planRepo := newFakePlanRepository()
	taskRepo := newFakeTaskRepository()
	service := newTestPlanService(planRepo, taskRepo, new(mockAgentServiceForPlan), new(mockModelServiceForPlan), new(mockChatServiceForPlan))
	_, err := service.DispatchPlan(context.Background(), "no-such-plan")
	assert.Error(t, err)
}
