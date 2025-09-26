package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

type TodoItem struct {
	Content   string    `json:"content"`
	Status    string    `json:"status"`   // pending, in_progress, completed, cancelled
	Priority  string    `json:"priority"` // high, medium, low
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TodoList struct {
	Todos []TodoItem `json:"todos"`
}

type TodoTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

func NewTodoTool(name, description string, configuration map[string]string, logger *zap.Logger) *TodoTool {
	return &TodoTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *TodoTool) Name() string {
	return t.name
}

func (t *TodoTool) Description() string {
	return t.description
}

func (t *TodoTool) Configuration() map[string]string {
	return t.configuration
}

func (t *TodoTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *TodoTool) FullDescription() string {
	var b strings.Builder
	b.WriteString(t.Description())
	b.WriteString("\n\n")
	b.WriteString("## Usage Instructions\n")
	b.WriteString("This tool manages a structured task list for complex tasks. It supports creating, reading, updating, and managing tasks with priorities and statuses.\n")
	b.WriteString("Tasks can have statuses: pending, in_progress, completed, cancelled\n")
	b.WriteString("Priorities: high, medium, low\n")
	b.WriteString("\n## Configuration\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *TodoTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "action",
			Type:        "string",
			Description: "Action to perform: write, read, update_status",
			Required:    true,
			Enum:        []string{"write", "read", "update_status"},
		},
		{
			Name:        "todos",
			Type:        "array",
			Description: "For write action: array of todo objects with content, priority",
			Required:    false,
			Items: []entities.Item{
				{Type: "object"},
			},
		},
		{
			Name:        "id",
			Type:        "string",
			Description: "For update_status action: the ID of the todo to update",
			Required:    false,
		},
		{
			Name:        "status",
			Type:        "string",
			Description: "For update_status action: new status (pending, in_progress, completed, cancelled)",
			Required:    false,
			Enum:        []string{"pending", "in_progress", "completed", "cancelled"},
		},
	}
}

func (t *TodoTool) getTodoFilePath() string {
	workspace := t.configuration["workspace"]
	if workspace == "" {
		workspace, _ = os.Getwd()
	}
	return filepath.Join(workspace, ".aiagent", "todos.json")
}

func (t *TodoTool) loadTodos() (*TodoList, error) {
	path := t.getTodoFilePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &TodoList{Todos: []TodoItem{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var todoList TodoList
	if err := json.Unmarshal(data, &todoList); err != nil {
		return nil, err
	}

	return &todoList, nil
}

func (t *TodoTool) saveTodos(todoList *TodoList) error {
	path := t.getTodoFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(todoList, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (t *TodoTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing todo command", zap.String("arguments", arguments))
	var args struct {
		Action string `json:"action"`
		Todos  []struct {
			Content  string `json:"content"`
			Priority string `json:"priority"`
		} `json:"todos,omitempty"`
		ID     string `json:"id,omitempty"`
		Status string `json:"status,omitempty"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	switch args.Action {
	case "write":
		return t.writeTodos(args.Todos)
	case "read":
		return t.readTodos()
	case "update_status":
		return t.updateStatus(args.ID, args.Status)
	default:
		return "", fmt.Errorf("unknown action: %s", args.Action)
	}
}

func (t *TodoTool) writeTodos(todos []struct {
	Content  string `json:"content"`
	Priority string `json:"priority"`
}) (string, error) {
	todoList, err := t.loadTodos()
	if err != nil {
		return "", err
	}

	for _, todo := range todos {
		if todo.Priority == "" {
			todo.Priority = "medium"
		}
		newTodo := TodoItem{
			Content:   todo.Content,
			Status:    "pending",
			Priority:  todo.Priority,
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		todoList.Todos = append(todoList.Todos, newTodo)
	}

	if err := t.saveTodos(todoList); err != nil {
		return "", err
	}

	return fmt.Sprintf("Added %d todos to the list", len(todos)), nil
}

func (t *TodoTool) readTodos() (string, error) {
	todoList, err := t.loadTodos()
	if err != nil {
		return "", err
	}

	if len(todoList.Todos) == 0 {
		return "No todos found", nil
	}

	var result strings.Builder
	result.WriteString("📋 Task Plan:\n\n")

	for _, todo := range todoList.Todos {
		checkbox := ""
		switch todo.Status {
		case "pending":
			checkbox = "- [ ]"
		case "in_progress":
			checkbox = "- [🔄]"
		case "completed":
			checkbox = "- [x]"
		case "cancelled":
			checkbox = "- [❌]"
		}

		priorityIcon := ""
		switch todo.Priority {
		case "high":
			priorityIcon = "🔴"
		case "medium":
			priorityIcon = "🟡"
		case "low":
			priorityIcon = "🟢"
		}

		result.WriteString(fmt.Sprintf("%s %s %s\n", checkbox, priorityIcon, todo.Content))
	}

	// JSON for AI processing
	jsonResult, _ := json.Marshal(map[string]interface{}{
		"summary": result.String(),
		"todos":   todoList.Todos,
	})

	return string(jsonResult), nil
}

func (t *TodoTool) updateStatus(id, status string) (string, error) {
	todoList, err := t.loadTodos()
	if err != nil {
		return "", err
	}

	found := false
	for i := range todoList.Todos {
		if todoList.Todos[i].ID == id {
			todoList.Todos[i].Status = status
			todoList.Todos[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("todo with id %s not found", id)
	}

	if err := t.saveTodos(todoList); err != nil {
		return "", err
	}

	return fmt.Sprintf("Updated todo %s to status %s", id, status), nil
}

var _ entities.Tool = (*TodoTool)(nil)
