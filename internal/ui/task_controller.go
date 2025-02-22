package ui

import (
	"html/template"
	"net/http"

	"aiagent/internal/domain/services"

	"go.uber.org/zap"
)

// TaskController manages task-related UI requests.
type TaskController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	taskService  services.TaskService
	agentService services.AgentService
}

// NewTaskController creates a new TaskController instance.
func NewTaskController(logger *zap.Logger, tmpl *template.Template, taskService services.TaskService, agentService services.AgentService) *TaskController {
	return &TaskController{
		logger:       logger,
		tmpl:         tmpl,
		taskService:  taskService,
		agentService: agentService,
	}
}

// TaskListHandler handles GET requests to /tasks, rendering the task list partial.
func (c *TaskController) TaskListHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch all tasks
	tasks, err := c.taskService.ListTasks(r.Context())
	if err != nil {
		c.logger.Error("Failed to list tasks", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Fetch all agents for name enrichment
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create agent name map
	agentMap := make(map[string]string)
	for _, agent := range agents {
		agentMap[agent.ID] = agent.Name
	}

	// Prepare task views for template
	var taskViews []map[string]interface{}
	for _, task := range tasks {
		agentName, ok := agentMap[task.AssignedTo]
		if !ok {
			agentName = "Unknown" // Fallback for missing agents
		}
		taskViews = append(taskViews, map[string]interface{}{
			"ID":          task.ID,
			"Description": task.Description,
			"AssignedTo":  task.AssignedTo,
			"AgentName":   agentName,
			"Status":      string(task.Status),
			"Result":      task.Result,
		})
	}

	// Render the task_list partial
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "task_list", taskViews); err != nil {
		c.logger.Error("Failed to render task_list", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
