package entities

import "time"

// TaskStatus represents the possible states of a task in the workflow automation platform.
// It is stored as a string in MongoDB for simplicity and readability.
type TaskStatus string

// Constants defining valid TaskStatus values
const (
	TaskPending    TaskStatus = "pending"     // Task is created but not yet started
	TaskInProgress TaskStatus = "in_progress" // Task is currently being processed
	TaskCompleted  TaskStatus = "completed"   // Task has finished successfully
	TaskFailed     TaskStatus = "failed"      // Task has encountered an unrecoverable error
)

// Task represents a unit of work assigned to an AI agent in the workflow automation platform.
// It is stored in MongoDB as a document in the 'tasks' collection and tracks the task's
// description, assignment, status, and human interaction requirements.
//
// Key features:
// - Hierarchical tasks: Supports subtasks via ParentTaskID.
// - Status tracking: Uses TaskStatus enum for consistent state management.
// - Human oversight: Allows flagging tasks for human input.
//
// Relationships:
// - One-to-one with AIAgent via AssignedTo.
// - Self-referential for subtasks via ParentTaskID.
// - One-to-one with Conversation via TaskID (when human interaction is required).
type Task struct {
	ID                       string     `bson:"_id,omitempty"`              // Unique identifier, auto-generated by MongoDB if omitted
	Description              string     `bson:"description"`                // Details of what the task entails, required
	AssignedTo               string     `bson:"assigned_to"`                // ID of the AIAgent responsible for the task, required
	ParentTaskID             string     `bson:"parent_task_id,omitempty"`   // ID of the parent task, optional for top-level tasks
	Status                   TaskStatus `bson:"status"`                     // Current state of the task, required
	Result                   string     `bson:"result,omitempty"`           // Outcome of the task, optional
	RequiresHumanInteraction bool       `bson:"requires_human_interaction"` // Flag indicating if human input is needed, defaults to false
	CreatedAt                time.Time  `bson:"created_at"`                 // Timestamp of creation, set on insert
	UpdatedAt                time.Time  `bson:"updated_at"`                 // Timestamp of last update, updated on modification
}

// Notes:
// - Status transitions are managed by the TaskService, ensuring valid changes (e.g., pending -> in_progress).
// - Result is optional and populated only on completion or failure.
// - Edge cases: Tasks with RequiresHumanInteraction=true must have an associated Conversation.
// - Assumption: AssignedTo references a valid AIAgent ID, validated in the service layer.
// - Limitation: No explicit timeout field; timeouts can be implemented in the service layer if needed.
