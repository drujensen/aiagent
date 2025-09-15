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
	b.WriteString("## Required Parameters\n")
	b.WriteString("- **operation**: Must be 'write' or 'read'\n")
	b.WriteString("- **content**: Required when creating new tasks (operation='write' without id)\n")
	b.WriteString("- **id**: Optional, used to update existing tasks\n")
	b.WriteString("- **done**: Optional boolean to mark tasks complete/incomplete\n")
	b.WriteString("\n## Operations\n")
	b.WriteString("- **write**: Create new tasks or update existing ones\n")
	b.WriteString("- **read**: List all current tasks\n")
	b.WriteString("\n## Usage Pattern\n")
	b.WriteString("1. **Plan Agent**: Use 'write' with 'content' to create tasks\n")
	b.WriteString("2. **Build Agent**: Use 'read' to see tasks, then 'write' with 'id' and 'done' to complete them\n")
	b.WriteString("\n## Display Format\n")
	b.WriteString("- **☐ Task description**: Task not done\n")
	b.WriteString("- **☑ Task description**: Task completed\n")
	b.WriteString("\n## Examples\n")
	b.WriteString("**Create new task**: `{\"operation\": \"write\", \"content\": \"Implement user authentication\"}`\n")
	b.WriteString("**Mark task as done**: `{\"operation\": \"write\", \"id\": \"task-123\", \"done\": true}`\n")
	b.WriteString("**Mark task as not done**: `{\"operation\": \"write\", \"id\": \"task-123\", \"done\": false}`\n")
	b.WriteString("**Update task content**: `{\"operation\": \"write\", \"id\": \"task-123\", \"content\": \"New description\"}`\n")
	b.WriteString("**List all tasks**: `{\"operation\": \"read\"}`\n")
	b.WriteString("\n## Quick Reference\n")
	b.WriteString("- To create: `write` + `content`\n")
	b.WriteString("- To update: `write` + `id` + optional `content` or `done`\n")
	b.WriteString("- To list: `read` (no other parameters needed)\n")
	b.WriteString("\n## Important Notes\n")
	b.WriteString("- Always include 'content' when creating new tasks\n")
	b.WriteString("- Use 'id' to update existing tasks\n")
	b.WriteString("- 'done' is optional and defaults to false for new tasks\n")
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
			Description: "Task description - REQUIRED when creating new tasks (operation='write' without id). Example: 'Implement user authentication'",
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
		return "", fmt.Errorf("operation is required - must be 'write' or 'read'")
	}

	switch args.Operation {
	case "write":
		return t.writeTask(args.Content, args.Done, args.ID)
	case "read":
		return t.readTasks()
	default:
		return "", fmt.Errorf("unknown operation: %s - valid operations are 'write' and 'read'", args.Operation)
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
			// Provide helpful guidance in JSON format
			response := struct {
				Summary   string     `json:"summary"`
				FullTasks []struct{} `json:"full_tasks"`
				Error     string     `json:"error"`
				Examples  []string   `json:"examples"`
			}{
				Summary:   "❌ Missing content for new task",
				FullTasks: []struct{}{},
				Error:     "content parameter is required when creating new tasks",
				Examples: []string{
					`{"operation": "write", "content": "Implement user authentication"}`,
					`{"operation": "write", "id": "task-123", "done": true}`,
					`{"operation": "read"}`,
				},
			}

			jsonResult, err := json.Marshal(response)
			if err != nil {
				return "", fmt.Errorf("failed to marshal guidance response: %v", err)
			}
			return string(jsonResult), nil
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

		// Return JSON response
		response := struct {
			Summary   string `json:"summary"`
			FullTasks []struct {
				ID        string `json:"id"`
				Content   string `json:"content"`
				Status    string `json:"status"`
				CreatedAt string `json:"created_at"`
			} `json:"full_tasks"`
		}{
			Summary: fmt.Sprintf("✅ Task created: %s", content),
			FullTasks: []struct {
				ID        string `json:"id"`
				Content   string `json:"content"`
				Status    string `json:"status"`
				CreatedAt string `json:"created_at"`
			}{
				{
					ID:        task.ID,
					Content:   task.Content,
					Status:    doneStr,
					CreatedAt: task.CreatedAt.Format("2006-01-02 15:04:05"),
				},
			},
		}

		jsonResult, err := json.Marshal(response)
		if err != nil {
			return "", fmt.Errorf("failed to marshal task creation response: %v", err)
		}
		return string(jsonResult), nil

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

				// Return JSON response
				response := struct {
					Summary   string `json:"summary"`
					FullTasks []struct {
						ID        string `json:"id"`
						Content   string `json:"content"`
						Status    string `json:"status"`
						UpdatedAt string `json:"updated_at"`
					} `json:"full_tasks"`
				}{
					Summary: fmt.Sprintf("✅ Task updated: %s", task.Content),
					FullTasks: []struct {
						ID        string `json:"id"`
						Content   string `json:"content"`
						Status    string `json:"status"`
						UpdatedAt string `json:"updated_at"`
					}{
						{
							ID:        task.ID,
							Content:   task.Content,
							Status:    doneStr,
							UpdatedAt: task.UpdatedAt.Format("2006-01-02 15:04:05"),
						},
					},
				}

				jsonResult, err := json.Marshal(response)
				if err != nil {
					return "", fmt.Errorf("failed to marshal task update response: %v", err)
				}
				return string(jsonResult), nil
			}
		}
		return "", fmt.Errorf("task not found: %s", id)
	}
}

func (t *TaskTool) readTasks() (string, error) {
	if len(t.tasks) == 0 {
		response := struct {
			Summary   string     `json:"summary"`
			FullTasks []struct{} `json:"full_tasks"`
			Message   string     `json:"message"`
		}{
			Summary:   "📝 No tasks found",
			FullTasks: []struct{}{},
			Message:   "Use 'write' operation to create tasks",
		}

		jsonResult, err := json.Marshal(response)
		if err != nil {
			return "", fmt.Errorf("failed to marshal empty tasks response: %v", err)
		}
		return string(jsonResult), nil
	}

	// Sort tasks by creation time (newest first)
	sort.Slice(t.tasks, func(i, j int) bool {
		return t.tasks[i].CreatedAt.After(t.tasks[j].CreatedAt)
	})

	// Create TUI-friendly summary (checkboxes only, no IDs)
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("📋 Current Tasks (%d total):\n\n", len(t.tasks)))

	for _, task := range t.tasks {
		// Simple checkbox display
		checkbox := "☐"
		if task.Status == entities.TaskStatusCompleted {
			checkbox = "☑"
		}

		summary.WriteString(fmt.Sprintf("%s %s\n", checkbox, task.Content))
	}

	// Create full task data for AI (includes IDs and metadata)
	type TaskData struct {
		ID        string `json:"id"`
		Content   string `json:"content"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}

	fullTasks := make([]TaskData, len(t.tasks))
	for i, task := range t.tasks {
		status := "pending"
		if task.Status == entities.TaskStatusCompleted {
			status = "completed"
		}
		fullTasks[i] = TaskData{
			ID:        task.ID,
			Content:   task.Content,
			Status:    status,
			CreatedAt: task.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	// Create JSON response with summary for TUI and full data for AI
	response := struct {
		Summary   string     `json:"summary"`
		FullTasks []TaskData `json:"full_tasks"`
		Total     int        `json:"total"`
	}{
		Summary:   summary.String(),
		FullTasks: fullTasks,
		Total:     len(t.tasks),
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal task response", zap.Error(err))
		return summary.String(), nil // Fallback to summary only
	}

	t.logger.Info("Tasks listed", zap.Int("count", len(t.tasks)))
	return string(jsonResult), nil
}

var _ entities.Tool = (*TaskTool)(nil)
