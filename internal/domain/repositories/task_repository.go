package repositories

import (
	"context"

	"aiagent/internal/domain/entities"
)

// TaskRepository defines the interface for managing Task entities in the domain layer.
// It provides methods to create, update, delete, and retrieve tasks, abstracting the data
// storage mechanism (e.g., MongoDB). This interface supports workflow execution and task delegation
// without tying the domain logic to a specific infrastructure.
//
// Key features:
// - CRUD Operations: Manages the full lifecycle of tasks.
// - Context Usage: Uses context.Context for timeout and cancellation support.
// - Error Handling: Returns ErrNotFound (from package scope) for non-existent tasks in GetTask and DeleteTask.
//
// Dependencies:
// - context: For managing request timeouts and cancellations.
// - aiagent/internal/domain/entities: Provides the Task entity definition.
//
// Notes:
// - CreateTask modifies the task's ID in place, assuming the storage generates it.
// - UpdateTask requires a valid ID in the Task struct.
// - ListTasks returns pointers to optimize memory usage for potentially large task lists.
type TaskRepository interface {
	// CreateTask inserts a new task into the repository and sets the task's ID.
	// The task's ID field is updated with the generated ID upon successful insertion.
	// Returns an error if the insertion fails (e.g., invalid AssignedTo, database error).
	CreateTask(ctx context.Context, task *entities.Task) error

	// UpdateTask updates an existing task in the repository.
	// The task must have a valid ID; returns an error if the task does not exist
	// or if the update fails (e.g., invalid status transition, database issue).
	UpdateTask(ctx context.Context, task *entities.Task) error

	// DeleteTask deletes a task by its ID.
	// Returns ErrNotFound if no task with the given ID exists; otherwise, returns
	// an error if the deletion fails (e.g., database connectivity issue).
	DeleteTask(ctx context.Context, id string) error

	// GetTask retrieves a task by its ID.
	// Returns a pointer to the Task and nil error on success, or nil and ErrNotFound
	// if the task does not exist, or nil and another error for other failures (e.g., database error).
	GetTask(ctx context.Context, id string) (*entities.Task, error)

	// ListTasks retrieves all tasks in the repository.
	// Returns a slice of pointers to Task entities; returns an empty slice if no tasks exist,
	// or an error if the retrieval fails (e.g., database connection lost).
	ListTasks(ctx context.Context) ([]*entities.Task, error)
}

// Notes:
// - Error handling assumes implementations will detail errors (e.g., "task not found: id=abc").
// - Edge case: Tasks requiring human interaction need associated conversations, enforced elsewhere.
// - Assumption: Task IDs are strings, aligning with MongoDB ObjectID usage in entities.
// - Limitation: No subtask-specific methods (e.g., ListSubtasks); could be added if required.
