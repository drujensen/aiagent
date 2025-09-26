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
type mockTaskRepository struct {
	mock.Mock
}

func (m *mockTaskRepository) CreateTask(ctx context.Context, task *entities.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *mockTaskRepository) UpdateTask(ctx context.Context, task *entities.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *mockTaskRepository) DeleteTask(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockTaskRepository) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Task), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockTaskRepository) ListTasks(ctx context.Context) ([]*entities.Task, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Task), args.Error(1)
}

func TestTaskService_ListTasks(t *testing.T) {
	mockRepo := new(mockTaskRepository)
	logger := zap.NewNop()
	service := NewTaskService(mockRepo, logger)

	ctx := context.Background()
	expectedTasks := []*entities.Task{
		entities.NewTask("Test Task", "Description", entities.TaskPriorityMedium),
	}

	mockRepo.On("ListTasks", ctx).Return(expectedTasks, nil)

	tasks, err := service.ListTasks(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expectedTasks, tasks)
	mockRepo.AssertExpectations(t)
}

func TestTaskService_GetTask(t *testing.T) {
	mockRepo := new(mockTaskRepository)
	logger := zap.NewNop()
	service := NewTaskService(mockRepo, logger)

	ctx := context.Background()
	task := entities.NewTask("Test Task", "Description", entities.TaskPriorityMedium)

	t.Run("valid task", func(t *testing.T) {
		mockRepo.On("GetTask", ctx, "valid-id").Return(task, nil).Once()

		result, err := service.GetTask(ctx, "valid-id")

		assert.NoError(t, err)
		assert.Equal(t, task, result)
	})

	t.Run("empty id", func(t *testing.T) {
		result, err := service.GetTask(ctx, "")

		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
		assert.Nil(t, result)
	})
}

func TestTaskService_CreateTask(t *testing.T) {
	mockRepo := new(mockTaskRepository)
	logger := zap.NewNop()
	service := NewTaskService(mockRepo, logger)

	ctx := context.Background()
	task := entities.NewTask("Test Task", "Description", entities.TaskPriorityMedium)
	task.ID = "test-id"

	t.Run("valid task", func(t *testing.T) {
		mockRepo.On("CreateTask", ctx, task).Return(nil).Once()

		err := service.CreateTask(ctx, task)

		assert.NoError(t, err)
		assert.NotZero(t, task.CreatedAt)
		assert.NotZero(t, task.UpdatedAt)
	})

	t.Run("missing id", func(t *testing.T) {
		invalidTask := &entities.Task{Name: "Test"}
		err := service.CreateTask(ctx, invalidTask)
		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
	})

	t.Run("missing name", func(t *testing.T) {
		invalidTask := &entities.Task{ID: "test-id"}
		err := service.CreateTask(ctx, invalidTask)
		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
	})
}

func TestTaskService_UpdateTask(t *testing.T) {
	mockRepo := new(mockTaskRepository)
	logger := zap.NewNop()
	service := NewTaskService(mockRepo, logger)

	ctx := context.Background()
	task := entities.NewTask("Test Task", "Description", entities.TaskPriorityMedium)
	task.ID = "test-id"
	existing := *task
	existing.CreatedAt = time.Now().Add(-time.Hour)

	t.Run("valid update", func(t *testing.T) {
		mockRepo.On("GetTask", ctx, task.ID).Return(&existing, nil).Once()
		mockRepo.On("UpdateTask", ctx, task).Return(nil).Once()

		err := service.UpdateTask(ctx, task)

		assert.NoError(t, err)
		assert.Equal(t, existing.CreatedAt, task.CreatedAt)
		assert.True(t, task.UpdatedAt.After(existing.CreatedAt))
	})

	t.Run("missing id", func(t *testing.T) {
		invalidTask := &entities.Task{Name: "Test"}
		err := service.UpdateTask(ctx, invalidTask)
		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
	})

	t.Run("task not found", func(t *testing.T) {
		mockRepo.On("GetTask", ctx, task.ID).Return(nil, &errs.NotFoundError{}).Once()
		err := service.UpdateTask(ctx, task)
		assert.Error(t, err)
		assert.IsType(t, &errs.NotFoundError{}, err)
	})
}

func TestTaskService_DeleteTask(t *testing.T) {
	mockRepo := new(mockTaskRepository)
	logger := zap.NewNop()
	service := NewTaskService(mockRepo, logger)

	ctx := context.Background()

	t.Run("valid delete", func(t *testing.T) {
		task := entities.NewTask("Test Task", "Description", entities.TaskPriorityMedium)
		task.ID = "test-id"
		mockRepo.On("GetTask", ctx, task.ID).Return(task, nil).Once()
		mockRepo.On("DeleteTask", ctx, task.ID).Return(nil).Once()

		err := service.DeleteTask(ctx, task.ID)

		assert.NoError(t, err)
	})

	t.Run("empty id", func(t *testing.T) {
		err := service.DeleteTask(ctx, "")
		assert.Error(t, err)
		assert.IsType(t, &errs.ValidationError{}, err)
	})

	t.Run("task not found", func(t *testing.T) {
		mockRepo.On("GetTask", ctx, "not-found").Return(nil, &errs.NotFoundError{}).Once()
		err := service.DeleteTask(ctx, "not-found")
		assert.Error(t, err)
		assert.IsType(t, &errs.NotFoundError{}, err)
	})
}