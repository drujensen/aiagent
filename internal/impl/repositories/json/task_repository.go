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

type JsonTaskRepository struct {
	mu       sync.RWMutex
	filePath string
	data     []*entities.Task
}

func NewJSONTaskRepository(storageDir string) (interfaces.TaskRepository, error) {
	filePath := filepath.Join(storageDir, "tasks.json")
	repo := &JsonTaskRepository{
		filePath: filePath,
		data:     []*entities.Task{},
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

// copyTask performs a full copy including DueDate *time.Time, which a bare
// struct copy (taskCopy := *t) would otherwise still alias.
func copyTask(t *entities.Task) *entities.Task {
	var dueDateCopy *time.Time
	if t.DueDate != nil {
		d := *t.DueDate
		dueDateCopy = &d
	}
	return &entities.Task{
		ID:              t.ID,
		Name:            t.Name,
		Content:         t.Content,
		Status:          t.Status,
		Priority:        t.Priority,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
		DueDate:         dueDateCopy,
		PlanID:          t.PlanID,
		AssignedAgentID: t.AssignedAgentID,
	}
}

// load and save are only ever called by exported methods that already hold
// r.mu, so neither acquires the lock itself.
func (r *JsonTaskRepository) load() error {
	data, err := os.ReadFile(r.filePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet, start with empty data
	}
	if err != nil {
		return errors.InternalErrorf("failed to read tasks.json: %v", err)
	}

	var tasks []*entities.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return errors.InternalErrorf("failed to unmarshal tasks.json: %v", err)
	}

	// Validate UUIDs
	for _, task := range tasks {
		if task.ID == "" {
			return errors.InternalErrorf("task is missing an ID")
		}
		if _, err := uuid.Parse(task.ID); err != nil {
			return errors.InternalErrorf("task has an invalid UUID: %v", err)
		}
	}

	r.data = tasks
	return nil
}

func (r *JsonTaskRepository) save() error {
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return errors.InternalErrorf("failed to marshal tasks: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return errors.InternalErrorf("failed to create directory: %v", err)
	}

	return atomicWriteFile(r.filePath, data)
}

func (r *JsonTaskRepository) ListTasks(ctx context.Context) ([]*entities.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasksCopy := make([]*entities.Task, len(r.data))
	for i, t := range r.data {
		tasksCopy[i] = copyTask(t)
	}
	return tasksCopy, nil
}

func (r *JsonTaskRepository) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, task := range r.data {
		if task.ID == id {
			return copyTask(task), nil
		}
	}
	return nil, errors.NotFoundErrorf("task not found: %s", id)
}

func (r *JsonTaskRepository) CreateTask(ctx context.Context, task *entities.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt

	r.data = append(r.data, copyTask(task))
	return r.save()
}

func (r *JsonTaskRepository) UpdateTask(ctx context.Context, task *entities.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, t := range r.data {
		if t.ID == task.ID {
			task.UpdatedAt = time.Now()
			r.data[i] = copyTask(task)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("task not found: %s", task.ID)
}

func (r *JsonTaskRepository) DeleteTask(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, t := range r.data {
		if t.ID == id {
			r.data = slices.Delete(r.data, i, i+1)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("task not found: %s", id)
}

var _ interfaces.TaskRepository = (*JsonTaskRepository)(nil)
