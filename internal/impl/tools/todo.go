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

	"go.uber.org/zap"
)

type TodoItem struct {
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
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
	b.WriteString("This tool manages a structured task list for complex tasks. It supports creating, reading, updating, and managing tasks with statuses and workflow grouping.\n")
	b.WriteString("Tasks can have statuses: pending, in_progress, completed, cancelled\n")
	b.WriteString("Workflows: Use workflow_id to group related tasks (e.g., 'auth-feature', 'user-registration')\n")
	b.WriteString("\nPer-session support: session_id is required and auto-injected by the chat service using current chat.ID.\n")
	b.WriteString("LLMs do not need to provide it. Clear: Use action='clear' to delete all todos for the session.\n")
	b.WriteString("\n## Configuration\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *TodoTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "The action to perform: write (create todos), read (list todos), update_status (update a todo status), clear (delete all todos)",
				"enum":        []string{"write", "read", "update_status", "clear"},
			},
			"todos": map[string]any{
				"type":        "array",
				"description": "The updated todo list (required for write action)",
				"items": map[string]any{
					"type":        "string",
					"description": "Brief description of the task",
				},
			},
			"index": map[string]any{
				"type":        "integer",
				"description": "For update_status: the 1-based index of the todo to update",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "For update_status: new status (pending, in_progress, completed, cancelled)",
				"enum":        []string{"pending", "in_progress", "completed", "cancelled"},
			},
		},
		"required": []string{"action"},
	}
}

func (t *TodoTool) getTodoFilePath(sessionID string) string {
	workspace := t.configuration["workspace"]
	if workspace == "" {
		workspace, _ = os.Getwd()
	}
	dir := filepath.Join(workspace, ".aiagent")
	return filepath.Join(dir, fmt.Sprintf("todos_%s.json", sessionID))
}

func (t *TodoTool) loadTodos(sessionID string) (*TodoList, error) {
	path := t.getTodoFilePath(sessionID)
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

func (t *TodoTool) saveTodos(sessionID string, todoList *TodoList) error {
	path := t.getTodoFilePath(sessionID)
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
		Action    string   `json:"action,omitempty"`
		Todos     []string `json:"todos"`
		Index     int      `json:"index,omitempty"`
		Status    string   `json:"status,omitempty"`
		SessionID string   `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	sessionID := args.SessionID
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required")
	}

	// Use action if provided, otherwise infer
	action := args.Action
	if action == "" {
		if len(args.Todos) > 0 {
			action = "write"
		} else if args.Index != 0 && args.Status != "" {
			action = "update_status"
		} else {
			action = "read"
		}
	}

	switch action {
	case "write":
		return t.writeTodos(sessionID, args.Todos)
	case "read":
		return t.readTodos(sessionID)
	case "update_status":
		return t.updateStatus(sessionID, args.Index, args.Status)
	case "clear":
		return t.clearTodos(sessionID)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (t *TodoTool) writeTodos(sessionID string, todos []string) (string, error) {
	todoList, err := t.loadTodos(sessionID)
	if err != nil {
		return "", err
	}

	for _, content := range todos {
		newTodo := TodoItem{
			Content:   content,
			Status:    "pending",
			Priority:  "medium",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		todoList.Todos = append(todoList.Todos, newTodo)
	}

	if err := t.saveTodos(sessionID, todoList); err != nil {
		return "", err
	}

	// Sort todos by CreatedAt ascending
	sort.Slice(todoList.Todos, func(i, j int) bool {
		return todoList.Todos[i].CreatedAt.Before(todoList.Todos[j].CreatedAt)
	})

	// Build the response with action message and full task plan
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Added %d todos to the list\n\n", len(todos)))

	// Add the full task plan
	if len(todoList.Todos) > 0 {
		result.WriteString("📋 Task Plan:\n\n")
		for i, todo := range todoList.Todos {
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

			result.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, checkbox, todo.Content))
		}
	}

	// Return JSON with summary and todos
	jsonResult, _ := json.Marshal(map[string]interface{}{
		"summary": result.String(),
		"todos":   todoList.Todos,
	})

	return string(jsonResult), nil
}

func (t *TodoTool) readTodos(sessionID string) (string, error) {
	todoList, err := t.loadTodos(sessionID)
	if err != nil {
		return "", err
	}

	if len(todoList.Todos) == 0 {
		jsonResult, _ := json.Marshal(map[string]interface{}{
			"summary": "No todos found",
			"todos":   []TodoItem{},
		})
		return string(jsonResult), nil
	}

	// Sort todos by CreatedAt ascending
	sort.Slice(todoList.Todos, func(i, j int) bool {
		return todoList.Todos[i].CreatedAt.Before(todoList.Todos[j].CreatedAt)
	})

	var result strings.Builder
	result.WriteString("📋 Task Plan:\n\n")

	// Display todos without workflow grouping
	for i, todo := range todoList.Todos {
		result.WriteString(t.formatTodo(i+1, todo))
	}

	// JSON for AI processing
	jsonResult, _ := json.Marshal(map[string]interface{}{
		"summary": result.String(),
		"todos":   todoList.Todos,
	})

	return string(jsonResult), nil
}

func (t *TodoTool) formatTodo(index int, todo TodoItem) string {
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

	return fmt.Sprintf("%d. %s %s\n", index, checkbox, todo.Content)
}

func (t *TodoTool) updateStatus(sessionID string, index int, status string) (string, error) {
	todoList, err := t.loadTodos(sessionID)
	if err != nil {
		return "", err
	}

	// Sort todos by CreatedAt ascending
	sort.Slice(todoList.Todos, func(i, j int) bool {
		return todoList.Todos[i].CreatedAt.Before(todoList.Todos[j].CreatedAt)
	})

	if index < 1 || index > len(todoList.Todos) {
		return "", fmt.Errorf("invalid index %d, must be between 1 and %d", index, len(todoList.Todos))
	}

	todoList.Todos[index-1].Status = status
	todoList.Todos[index-1].UpdatedAt = time.Now()

	if err := t.saveTodos(sessionID, todoList); err != nil {
		return "", err
	}

	// Build the response
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Updated todo at index %d to status %s\n\n", index, status))

	// Add the full task plan
	if len(todoList.Todos) > 0 {
		result.WriteString("📋 Task Plan:\n\n")
		for i, todo := range todoList.Todos {
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

			result.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, checkbox, todo.Content))
		}
	}

	// Return JSON with summary and todos
	jsonResult, _ := json.Marshal(map[string]interface{}{
		"summary": result.String(),
		"todos":   todoList.Todos,
	})

	return string(jsonResult), nil
}

func (t *TodoTool) clearTodos(sessionID string) (string, error) {
	path := t.getTodoFilePath(sessionID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return `{"summary": "No todos found for this session."}`, nil
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return `{"summary": "Cleared all todos for this session."}`, nil
}

func (t *TodoTool) DisplayName(ui string, arguments string) (string, string) {
	return t.Name(), ""
}

func (t *TodoTool) FormatResult(ui string, result string, diff string, arguments string) string {
	var response struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(result), &response); err == nil && response.Summary != "" {
		return response.Summary
	}
	return result
}

var _ entities.Tool = (*TodoTool)(nil)
