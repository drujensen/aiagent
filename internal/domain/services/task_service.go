package services

import (
	"context"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type TaskService interface {
	ListTasks(ctx context.Context) ([]*entities.Task, error)
	GetTask(ctx context.Context, id string) (*entities.Task, error)
	CreateTask(ctx context.Context, task *entities.Task) error
	UpdateTask(ctx context.Context, task *entities.Task) error
	DeleteTask(ctx context.Context, id string) error
}

type taskService struct {
	taskRepo interfaces.TaskRepository
	logger   *zap.Logger
}

func NewTaskService(taskRepo interfaces.TaskRepository, logger *zap.Logger) *taskService {
	return &taskService{
		taskRepo: taskRepo,
		logger:   logger,
	}
}

func (s *taskService) ListTasks(ctx context.Context) ([]*entities.Task, error) {
	tasks, err := s.taskRepo.ListTasks(ctx)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func (s *taskService) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	if id == "" {
		return nil, errors.ValidationErrorf("task ID is required")
	}

	task, err := s.taskRepo.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *taskService) CreateTask(ctx context.Context, task *entities.Task) error {
	if task.ID == "" {
		return errors.ValidationErrorf("task id is required")
	}
	if task.Name == "" {
		return errors.ValidationErrorf("task name is required")
	}

	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.CreateTask(ctx, task); err != nil {
		return err
	}

	return nil
}

func (s *taskService) UpdateTask(ctx context.Context, task *entities.Task) error {
	if task.ID == "" {
		return errors.ValidationErrorf("task ID is required")
	}

	existing, err := s.taskRepo.GetTask(ctx, task.ID)
	if err != nil {
		return err
	}

	if task.Name == "" {
		return errors.ValidationErrorf("task name is required")
	}

	task.CreatedAt = existing.CreatedAt
	task.UpdatedAt = time.Now()

	if err := s.taskRepo.UpdateTask(ctx, task); err != nil {
		return err
	}

	return nil
}

func (s *taskService) DeleteTask(ctx context.Context, id string) error {
	if id == "" {
		return errors.ValidationErrorf("task ID is required")
	}

	_, err := s.taskRepo.GetTask(ctx, id)
	if err != nil {
		return err
	}

	if err := s.taskRepo.DeleteTask(ctx, id); err != nil {
		return err
	}

	return nil
}

// verify interface implementation
var _ TaskService = &taskService{}
