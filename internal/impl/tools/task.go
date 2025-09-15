package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
		// File doesn't exist, start with empty tasks
		return nil
	}
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &t.tasks); err != nil {
		t.logger.Warn("Failed to parse tasks file, starting fresh", zap.Error(err))
		t.tasks = []*entities.Task{}
	}

	return nil
}

func (t *TaskTool) save() error {
	if err := os.MkdirAll(filepath.Dir(t.dataFile), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(t.tasks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(t.dataFile, data, 0644)
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
	b.WriteString("## Ultra-Simple Task Management\n")
	b.WriteString("Brain-dead easy task tracking for Build and Plan agents.\n\n")
	b.WriteString("## Operations\n")
	b.WriteString("- **write**: Create new tasks or mark existing ones as done\n")
	b.WriteString("- **read**: List all current tasks\n")
	b.WriteString("\n## Usage Pattern\n")
	b.WriteString("1. **Plan Agent**: Use 'write' to create tasks\n")
	b.WriteString("2. **Build Agent**: Use 'read' to see tasks, then 'write' with done=true to complete them\n")
	b.WriteString("\n## Display Format\n")
	b.WriteString("- **☐ Task description**: Task not done\n")
	b.WriteString("- **☑ Task description**: Task completed\n")
	b.WriteString("\n## Examples\n")
	b.WriteString("**Create task**: `{\"operation\": \"write\", \"content\": \"Implement user authentication\"}`\n")
	b.WriteString("**Mark as done**: `{\"operation\": \"write\", \"id\": \"task-123\", \"done\": true}`\n")
	b.WriteString("**Mark as not done**: `{\"operation\": \"write\", \"id\": \"task-123\", \"done\": false}`\n")
	b.WriteString("**Update content**: `{\"operation\": \"write\", \"id\": \"task-123\", \"content\": \"New description\"}`\n")
	b.WriteString("**List tasks**: `{\"operation\": \"read\"}`\n")
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
			Enum:        []string{"write", "read"},
			Description: "Operation: 'write' to create/update tasks, 'read' to list all tasks",
			Required:    true,
		},
		{
			Name:        "content",
			Type:        "string",
			Description: "Task description (required for new tasks)",
			Required:    false,
		},
		{
			Name:        "done",
			Type:        "boolean",
			Description: "Mark task as completed (true) or not done (false)",
			Required:    false,
		},
		{
			Name:        "id",
			Type:        "string",
			Description: "Task ID for updates (omit to create new task)",
			Required:    false,
		},
	}
}

func (t *TaskTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing ultra-simple task operation", zap.String("arguments", arguments))

	var args struct {
		Operation string `json:"operation"`
		Content   string `json:"content"`
		Done      *bool  `json:"done"` // Use pointer to distinguish false from not set
		ID        string `json:"id"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Operation == "" {
		return "", fmt.Errorf("operation is required")
	}

	switch args.Operation {
	case "write":
		return t.writeTask(args.Content, args.Done, args.ID)
	case "read":
		return t.readTasks()
	default:
		return "", fmt.Errorf("unknown operation: %s (use 'write' or 'read')", args.Operation)
	}
}

func (t *TaskTool) writeTask(content string, done *bool, id string) (string, error) {
	// Determine status from done boolean
	status := entities.TaskStatusPending
	if done != nil {
		if *done {
			status = entities.TaskStatusCompleted
		} else {
			status = entities.TaskStatusPending
		}
	}

	if id == "" {
		// Create new task
		if content == "" {
			return "", fmt.Errorf("content is required for new tasks")
		}

		task := &entities.Task{
			ID:        uuid.New().String(),
			Name:      content,
			Content:   content,
			Status:    status,
			Priority:  entities.TaskPriorityMedium, // Default, but ignored in display
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		t.tasks = append(t.tasks, task)

		if err := t.save(); err != nil {
			return "", fmt.Errorf("failed to save task: %v", err)
		}

		doneStr := "not done"
		if done != nil && *done {
			doneStr = "done"
		}
		t.logger.Info("Task created", zap.String("id", task.ID), zap.String("content", content), zap.String("status", doneStr))
		return fmt.Sprintf("✅ Task created: %s", content), nil

	} else {
		// Update existing task
		for _, task := range t.tasks {
			if task.ID == id {
				if content != "" {
					task.Content = content
					task.Name = content
				}
				task.Status = status
				task.UpdatedAt = time.Now()

				if err := t.save(); err != nil {
					return "", fmt.Errorf("failed to save task: %v", err)
				}

				doneStr := "not done"
				if done != nil && *done {
					doneStr = "done"
				}
				t.logger.Info("Task updated", zap.String("id", id), zap.String("status", doneStr))
				return fmt.Sprintf("✅ Task updated: %s", task.Content), nil
			}
		}
		return "", fmt.Errorf("task not found: %s", id)
	}
}

func (t *TaskTool) readTasks() (string, error) {
	if len(t.tasks) == 0 {
		return "📝 No tasks found. Use 'write' operation to create tasks.", nil
	}

	// Sort tasks by creation time (newest first)
	sort.Slice(t.tasks, func(i, j int) bool {
		return t.tasks[i].CreatedAt.After(t.tasks[j].CreatedAt)
	})

	var result strings.Builder
	result.WriteString(fmt.Sprintf("📋 Current Tasks (%d total):\n\n", len(t.tasks)))

	for _, task := range t.tasks {
		// Simple checkbox display
		checkbox := "☐"
		if task.Status == entities.TaskStatusCompleted {
			checkbox = "☑"
		}

		result.WriteString(fmt.Sprintf("%s %s\n", checkbox, task.Content))
		result.WriteString(fmt.Sprintf("   ID: %s\n", task.ID))
	}

	t.logger.Info("Tasks listed", zap.Int("count", len(t.tasks)))
	return result.String(), nil
}

var _ entities.Tool = (*TaskTool)(nil)
