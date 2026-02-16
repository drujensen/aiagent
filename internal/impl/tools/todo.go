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

type TodoItem struct {
	Content    string    `json:"content"`
	Status     string    `json:"status"` // pending, in_progress, completed, cancelled
	ID         string    `json:"id"`
	WorkflowID string    `json:"workflow_id,omitempty"` // Optional grouping for multi-step workflows
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
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
				"description": "Action to perform: write, read, update_status, clear",
				"enum":        []string{"write", "read", "update_status", "clear"},
			},
			"todos": map[string]any{
				"type":        "array",
				"description": "For write action: array of todo objects with content and optional workflow_id",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{
							"type":        "string",
							"description": "Task description",
						},
						"workflow_id": map[string]any{
							"type":        "string",
							"description": "Optional workflow identifier for grouping related tasks",
						},
					},
					"required": []string{"content"},
				},
			},
			"id": map[string]any{
				"type":        "string",
				"description": "For update_status action: the ID of the todo to update",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "For update_status action: new status (pending, in_progress, completed, cancelled)",
				"enum":        []string{"pending", "in_progress", "completed", "cancelled"},
			},
			"session_id": map[string]any{
				"type":        "string",
				"description": "Required chat session ID (auto-injected by service).",
			},
		},
		"required": []string{"action", "session_id"},
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
		Action string `json:"action"`
		Todos  []struct {
			Content    string `json:"content"`
			WorkflowID string `json:"workflow_id,omitempty"`
		} `json:"todos,omitempty"`
		ID        string `json:"id,omitempty"`
		Status    string `json:"status,omitempty"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	sessionID := args.SessionID
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required")
	}

	switch args.Action {
	case "write":
		return t.writeTodos(sessionID, args.Todos)
	case "read":
		return t.readTodos(sessionID)
	case "update_status":
		return t.updateStatus(sessionID, args.ID, args.Status)
	case "clear":
		return t.clearTodos(sessionID)
	default:
		return "", fmt.Errorf("unknown action: %s", args.Action)
	}
}

func (t *TodoTool) writeTodos(sessionID string, todos []struct {
	Content    string `json:"content"`
	WorkflowID string `json:"workflow_id,omitempty"`
}) (string, error) {
	todoList, err := t.loadTodos(sessionID)
	if err != nil {
		return "", err
	}

	for _, todo := range todos {
		newTodo := TodoItem{
			Content:    todo.Content,
			Status:     "pending",
			ID:         uuid.New().String(),
			WorkflowID: todo.WorkflowID,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		todoList.Todos = append(todoList.Todos, newTodo)
	}

	if err := t.saveTodos(sessionID, todoList); err != nil {
		return "", err
	}

	// Build the response with action message and full task plan
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Added %d todos to the list\n\n", len(todos)))

	// Add the full task plan
	if len(todoList.Todos) > 0 {
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

			result.WriteString(fmt.Sprintf("%s %s\n", checkbox, todo.Content))
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

	var result strings.Builder
	result.WriteString("📋 Task Plan:\n\n")

	// Group todos by workflow
	workflows := make(map[string][]TodoItem)
	orphaned := []TodoItem{}

	for _, todo := range todoList.Todos {
		if todo.WorkflowID != "" {
			workflows[todo.WorkflowID] = append(workflows[todo.WorkflowID], todo)
		} else {
			orphaned = append(orphaned, todo)
		}
	}

	// Display workflows
	for workflowID, todos := range workflows {
		result.WriteString(fmt.Sprintf("## %s\n", workflowID))
		for _, todo := range todos {
			result.WriteString(t.formatTodo(todo))
		}
		result.WriteString("\n")
	}

	// Display orphaned todos
	if len(orphaned) > 0 {
		if len(workflows) > 0 {
			result.WriteString("## General Tasks\n")
		}
		for _, todo := range orphaned {
			result.WriteString(t.formatTodo(todo))
		}
	}

	// JSON for AI processing
	jsonResult, _ := json.Marshal(map[string]interface{}{
		"summary": result.String(),
		"todos":   todoList.Todos,
	})

	return string(jsonResult), nil
}

func (t *TodoTool) formatTodo(todo TodoItem) string {
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

	return fmt.Sprintf("%s %s\n", checkbox, todo.Content)
}

func (t *TodoTool) updateStatus(sessionID, id, status string) (string, error) {
	todoList, err := t.loadTodos(sessionID)
	if err != nil {
		return "", err
	}

	if len(todoList.Todos) == 0 {
		// Log potential session_id mismatch for debugging
		fmt.Printf("Warning: Attempting to update todo %s in session %s, but no todos found. Possible session_id mismatch.\n", id, sessionID)
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

	if err := t.saveTodos(sessionID, todoList); err != nil {
		return "", err
	}

	// Build the response with action message and full task plan
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Updated todo %s to status %s\n\n", id, status))

	// Add the full task plan
	if len(todoList.Todos) > 0 {
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

			result.WriteString(fmt.Sprintf("%s %s\n", checkbox, todo.Content))
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

var _ entities.Tool = (*TodoTool)(nil)
