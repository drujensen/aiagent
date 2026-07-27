package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// TaskSpec describes one task to create as part of decomposing a Plan.
type TaskSpec struct {
	Name            string
	Content         string
	Priority        entities.TaskPriority
	AssignedAgentID string
}

// DispatchResult records the outcome of dispatching one Task: the sub-chat
// it ran in, its result content on success, or the error on failure. TaskID
// is always set, so callers can match a result back to the Task that
// produced it even when dispatching many Tasks concurrently.
type DispatchResult struct {
	TaskID string
	ChatID string
	Result string
	Err    error
}

// PlanService creates Plans, decomposes them into Tasks, and dispatches
// those Tasks to sub-agents - wiring the Task entity/repository/service
// stack (previously present but never constructed in main.go) into actual
// use for the first time.
type PlanService interface {
	CreatePlan(ctx context.Context, goal string, constraints []string, features []entities.Feature) (*entities.Plan, error)
	DecomposeIntoTasks(ctx context.Context, planID string, specs []TaskSpec) ([]*entities.Task, error)
	DispatchTask(ctx context.Context, taskID string) (*DispatchResult, error)
	DispatchPlan(ctx context.Context, planID string) ([]*DispatchResult, error)
}

type planService struct {
	planRepo     interfaces.PlanRepository
	taskRepo     interfaces.TaskRepository
	agentService AgentService
	modelService ModelService
	chatService  ChatService
	logger       *zap.Logger
}

func NewPlanService(
	planRepo interfaces.PlanRepository,
	taskRepo interfaces.TaskRepository,
	agentService AgentService,
	modelService ModelService,
	chatService ChatService,
	logger *zap.Logger,
) PlanService {
	return &planService{
		planRepo:     planRepo,
		taskRepo:     taskRepo,
		agentService: agentService,
		modelService: modelService,
		chatService:  chatService,
		logger:       logger,
	}
}

func (s *planService) CreatePlan(ctx context.Context, goal string, constraints []string, features []entities.Feature) (*entities.Plan, error) {
	if goal == "" {
		return nil, errors.ValidationErrorf("plan goal is required")
	}

	plan := entities.NewPlan(goal, constraints, features)
	if err := s.planRepo.CreatePlan(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *planService) DecomposeIntoTasks(ctx context.Context, planID string, specs []TaskSpec) ([]*entities.Task, error) {
	if _, err := s.planRepo.GetPlan(ctx, planID); err != nil {
		return nil, fmt.Errorf("failed to decompose unknown plan %s: %w", planID, err)
	}
	if len(specs) == 0 {
		return nil, errors.ValidationErrorf("at least one task spec is required")
	}

	tasks := make([]*entities.Task, 0, len(specs))
	for _, spec := range specs {
		if spec.Name == "" || spec.Content == "" {
			return nil, errors.ValidationErrorf("task name and content are required")
		}
		task := entities.NewTask(spec.Name, spec.Content, spec.Priority)
		task.PlanID = planID
		task.AssignedAgentID = spec.AssignedAgentID
		if err := s.taskRepo.CreateTask(ctx, task); err != nil {
			return nil, fmt.Errorf("failed to create task %q: %w", spec.Name, err)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// resolveDispatchModel picks a deterministic model for a dispatched task -
// the first available model - rather than inheriting whatever chat happens
// to be active for the human user (as tools.AgentTool.Execute does when no
// model_name argument is given). Inheriting the active chat's model is the
// right default for an LLM directly invoking the Agent tool mid-conversation,
// but wrong for Go-orchestrated fan-out: a background dispatch run has no
// relationship to whatever the human happens to have open at the time, and
// using it would make dispatch behavior depend on unrelated UI state.
func (s *planService) resolveDispatchModel(ctx context.Context) (*entities.Model, error) {
	models, err := s.modelService.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}
	return models[0], nil
}

func (s *planService) DispatchTask(ctx context.Context, taskID string) (*DispatchResult, error) {
	task, err := s.taskRepo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task.AssignedAgentID == "" {
		return nil, errors.ValidationErrorf("task %s has no assigned agent", taskID)
	}

	agent, err := s.agentService.GetAgent(ctx, task.AssignedAgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned agent for task %s: %w", taskID, err)
	}

	model, err := s.resolveDispatchModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve model for task %s: %w", taskID, err)
	}

	task.Status = entities.TaskStatusInProgress
	if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to mark task %s in progress: %w", taskID, err)
	}

	chat, err := s.chatService.CreateSubChat(ctx, agent.ID, model.ID, fmt.Sprintf("Task: %s", task.Name), "")
	if err != nil {
		task.Status = entities.TaskStatusCancelled
		if updateErr := s.taskRepo.UpdateTask(ctx, task); updateErr != nil {
			s.logger.Warn("failed to mark task cancelled after dispatch failure", zap.String("task_id", taskID), zap.Error(updateErr))
		}
		return &DispatchResult{TaskID: taskID, Err: err}, err
	}

	msg := entities.NewMessage("user", task.Content)
	response, err := s.chatService.SendMessage(ctx, chat.ID, msg)
	if err != nil {
		task.Status = entities.TaskStatusCancelled
		if updateErr := s.taskRepo.UpdateTask(ctx, task); updateErr != nil {
			s.logger.Warn("failed to mark task cancelled after dispatch failure", zap.String("task_id", taskID), zap.Error(updateErr))
		}
		return &DispatchResult{TaskID: taskID, ChatID: chat.ID, Err: err}, err
	}

	task.Status = entities.TaskStatusCompleted
	if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
		s.logger.Warn("failed to mark task completed", zap.String("task_id", taskID), zap.Error(err))
	}

	return &DispatchResult{TaskID: taskID, ChatID: chat.ID, Result: response.Content}, nil
}

// DispatchPlan dispatches every Pending Task under the given Plan
// concurrently, mirroring the goroutine fan-out pattern already used by
// executeToolsParallel (internal/impl/integrations/aimodel.go) for
// concurrent tool calls within one chat turn.
func (s *planService) DispatchPlan(ctx context.Context, planID string) ([]*DispatchResult, error) {
	if _, err := s.planRepo.GetPlan(ctx, planID); err != nil {
		return nil, fmt.Errorf("failed to dispatch unknown plan %s: %w", planID, err)
	}

	allTasks, err := s.taskRepo.ListTasks(ctx)
	if err != nil {
		return nil, err
	}

	var pending []*entities.Task
	for _, t := range allTasks {
		if t.PlanID == planID && t.Status == entities.TaskStatusPending {
			pending = append(pending, t)
		}
	}

	results := make([]*DispatchResult, len(pending))
	var wg sync.WaitGroup
	for i, task := range pending {
		wg.Add(1)
		go func(i int, taskID string) {
			defer wg.Done()
			result, err := s.DispatchTask(ctx, taskID)
			if err != nil && result == nil {
				result = &DispatchResult{TaskID: taskID, Err: err}
			}
			results[i] = result
		}(i, task.ID)
	}
	wg.Wait()

	return results, nil
}

var _ PlanService = &planService{}
