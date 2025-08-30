package interfaces

import (
	"context"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

type TaskRepository interface {
	CreateTask(ctx context.Context, task *entities.Task) error
	UpdateTask(ctx context.Context, task *entities.Task) error
	DeleteTask(ctx context.Context, id string) error
	GetTask(ctx context.Context, id string) (*entities.Task, error)
	ListTasks(ctx context.Context) ([]*entities.Task, error)
}
