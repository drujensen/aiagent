package entities

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string
type TaskPriority string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

type Task struct {
	ID        string       `json:"id" bson:"_id"`
	Name      string       `json:"name" bson:"name"`
	Content   string       `json:"content" bson:"content"`
	Status    TaskStatus   `json:"status" bson:"status"`
	Priority  TaskPriority `json:"priority" bson:"priority"`
	CreatedAt time.Time    `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" bson:"updated_at"`
	DueDate   *time.Time   `json:"due_date,omitempty" bson:"due_date,omitempty"`
}

func NewTask(name, content string, priority TaskPriority) *Task {
	return &Task{
		ID:        uuid.New().String(),
		Name:      name,
		Content:   content,
		Status:    TaskStatusPending,
		Priority:  priority,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Implement the list.Item interface
func (t *Task) FilterValue() string {
	return t.Name
}

func (t *Task) Title() string {
	return t.Name
}

func (t *Task) Description() string {
	status := string(t.Status)
	if t.DueDate != nil {
		status += " | Due: " + t.DueDate.Format("2006-01-02")
	}
	return status
}
