package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TaskTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
	dataFile      string
	tasks         []*entities.Task
}

func NewTaskTool(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
	dataDir := configuration["data_dir"]
	if dataDir == "" {
		dataDir = "."
	}
	dataFile := filepath.Join(dataDir, ".aiagent", "tasks.json")

	tool := &TaskTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
		dataFile:      dataFile,
		tasks:         []*entities.Task{},
	}

	if err := tool.load(); err != nil {
		logger.Error("Failed to load tasks", zap.Error(err))
	}

	return tool
}

func (t *TaskTool) load() error {
	data, err := os.ReadFile(t.dataFile)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet, start with empty data
	}
	if err != nil {
		return fmt.Errorf("failed to read tasks.json: %v", err)
	}

	var tasks []*entities.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("failed to unmarshal tasks.json: %v", err)
	}

	t.tasks = tasks
	return nil
}

func (t *TaskTool) save() error {
	data, err := json.MarshalIndent(t.tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(t.dataFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(t.dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks.json: %v", err)
	}

	return nil
}

func (t *TaskTool) Name() string {
	return t.name
}

func (t *TaskTool) Description() string {
	return t.description
}

func (t *TaskTool) Configuration() map[string]string {
	return t.configuration
}

func (t *TaskTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *TaskTool) FullDescription() string {
	var b strings.Builder
	b.WriteString(t.Description())
	b.WriteString("\n\n")
	b.WriteString("## Operations\n")
	b.WriteString("- **create_task**: Create a new task. The name will be automatically derived from the content.\n")
	b.WriteString("- **list_tasks**: List all tasks.\n")
	b.WriteString("- **get_task**: Get a specific task by ID.\n")
	b.WriteString("- **update_task**: Update an existing task.\n")
	b.WriteString("- **delete_task**: Delete a task by ID.\n")
	b.WriteString("\n## Configuration\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *TaskTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"create_task", "list_tasks", "get_task", "update_task", "delete_task"},
			Description: "The task operation to perform",
			Required:    true,
		},
		{
			Name:        "task_id",
			Type:        "string",
			Description: "Task ID for get, update, delete operations",
			Required:    false,
		},
		{
			Name:        "name",
			Type:        "string",
			Description: "Task name for update operations (ignored for create, automatically derived from content)",
			Required:    false,
		},
		{
			Name:        "content",
			Type:        "string",
			Description: "Task content/description for create/update operations",
			Required:    false,
		},
		{
			Name:        "priority",
			Type:        "string",
			Enum:        []string{"low", "medium", "high"},
			Description: "Task priority for create/update operations",
			Required:    false,
		},
		{
			Name:        "status",
			Type:        "string",
			Enum:        []string{"pending", "in_progress", "completed", "cancelled"},
			Description: "Task status for update operations",
			Required:    false,
		},
	}
}

func (t *TaskTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing task operation", zap.String("arguments", arguments))

	var args struct {
		Operation string `json:"operation"`
		TaskID    string `json:"task_id"`
		Name      string `json:"name"`
		Content   string `json:"content"`
		Priority  string `json:"priority"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	if args.Operation == "" {
		t.logger.Error("Operation is required")
		return "", fmt.Errorf("operation is required")
	}

	switch args.Operation {
	case "create_task":
		return t.createTask(args.Content, args.Priority)
	case "list_tasks":
		return t.listTasks()
	case "get_task":
		return t.getTask(args.TaskID)
	case "update_task":
		return t.updateTask(args.TaskID, args.Name, args.Content, args.Priority, args.Status)
	case "delete_task":
		return t.deleteTask(args.TaskID)
	default:
		t.logger.Error("Unknown operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("unknown operation: %s", args.Operation)
	}
}

func (t *TaskTool) createTask(content, priorityStr string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("task content is required")
	}
	// Derive name from content
	name := content
	if len(content) > 50 {
		name = content[:47] + "..."
	}

	priority := entities.TaskPriorityMedium
	switch priorityStr {
	case "low":
		priority = entities.TaskPriorityLow
	case "high":
		priority = entities.TaskPriorityHigh
	}

	task := entities.NewTask(name, content, priority)
	task.ID = uuid.New().String()
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt

	t.tasks = append(t.tasks, task)
	if err := t.save(); err != nil {
		return "", err
	}

	result, _ := json.MarshalIndent(task, "", "  ")
	t.logger.Info("Task created", zap.String("id", task.ID))
	return string(result), nil
}

func (t *TaskTool) listTasks() (string, error) {
	result, _ := json.MarshalIndent(t.tasks, "", "  ")
	t.logger.Info("Tasks listed", zap.Int("count", len(t.tasks)))
	return string(result), nil
}

func (t *TaskTool) getTask(id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("task ID is required")
	}

	for _, task := range t.tasks {
		if task.ID == id {
			result, _ := json.MarshalIndent(task, "", "  ")
			t.logger.Info("Task retrieved", zap.String("id", id))
			return string(result), nil
		}
	}

	return "", fmt.Errorf("task not found: %s", id)
}

func (t *TaskTool) updateTask(id, name, content, priorityStr, statusStr string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("task ID is required")
	}

	for _, task := range t.tasks {
		if task.ID == id {
			if name != "" {
				task.Name = name
			}
			if content != "" {
				task.Content = content
			}
			if priorityStr != "" {
				switch priorityStr {
				case "low":
					task.Priority = entities.TaskPriorityLow
				case "medium":
					task.Priority = entities.TaskPriorityMedium
				case "high":
					task.Priority = entities.TaskPriorityHigh
				}
			}
			if statusStr != "" {
				switch statusStr {
				case "pending":
					task.Status = entities.TaskStatusPending
				case "in_progress":
					task.Status = entities.TaskStatusInProgress
				case "completed":
					task.Status = entities.TaskStatusCompleted
				case "cancelled":
					task.Status = entities.TaskStatusCancelled
				}
			}
			task.UpdatedAt = time.Now()

			if err := t.save(); err != nil {
				return "", err
			}

			result, _ := json.MarshalIndent(task, "", "  ")
			t.logger.Info("Task updated", zap.String("id", id))
			return string(result), nil
		}
	}

	return "", fmt.Errorf("task not found: %s", id)
}

func (t *TaskTool) deleteTask(id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("task ID is required")
	}

	for i, task := range t.tasks {
		if task.ID == id {
			t.tasks = append(t.tasks[:i], t.tasks[i+1:]...)
			if err := t.save(); err != nil {
				return "", err
			}
			t.logger.Info("Task deleted", zap.String("id", id))
			return "Task deleted successfully", nil
		}
	}

	return "", fmt.Errorf("task not found: %s", id)
}

var _ entities.Tool = (*TaskTool)(nil)
