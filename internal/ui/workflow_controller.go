package ui

import (
	"fmt"
	"html/template"
	"net/http"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"

	"go.uber.org/zap"
)

// WorkflowController manages workflow-related UI requests.
type WorkflowController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	agentService services.AgentService
	taskService  services.TaskService
}

// NewWorkflowController creates a new WorkflowController instance.
func NewWorkflowController(logger *zap.Logger, tmpl *template.Template, agentService services.AgentService, taskService services.TaskService) *WorkflowController {
	return &WorkflowController{
		logger:       logger,
		tmpl:         tmpl,
		agentService: agentService,
		taskService:  taskService,
	}
}

// WorkflowHandler handles GET requests to /workflows, rendering the workflow page.
func (c *WorkflowController) WorkflowHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch agents for the dropdown
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Fetch tasks for initial display
	tasks, err := c.taskService.ListTasks(r.Context())
	if err != nil {
		c.logger.Error("Failed to list tasks", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create agent name map for enrichment
	agentMap := make(map[string]string)
	for _, agent := range agents {
		agentMap[agent.ID] = agent.Name
	}

	// Prepare task views as a slice of maps for template rendering
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

	// Template data
	data := map[string]interface{}{
		"Title":           "Workflows",
		"ContentTemplate": "workflow_content",
		"Agents":          agents,
		"Tasks":           taskViews,
		"RootAgents":      buildHierarchy(agents),
	}

	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render workflow", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// WorkflowSubmitHandler handles POST requests to /workflows (HTMX form submission).
func (c *WorkflowController) WorkflowSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data for HTMX
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	description := r.FormValue("description")
	assignedTo := r.FormValue("assigned_to")

	// Validate required fields
	if description == "" || assignedTo == "" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `<div id="response-message">Error: description and assigned_to are required</div>`)
		return
	}

	// Create and start the task using TaskService
	task := &entities.Task{
		Description: description,
		AssignedTo:  assignedTo,
		Status:      entities.TaskPending,
	}

	if err := c.taskService.StartWorkflow(r.Context(), task); err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div id="response-message">Error: %s</div>`, err.Error())
		return
	}

	// Return HTML for HTMX update
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `<div id="response-message">Workflow started successfully. Task ID: %s</div>`, task.ID)
}
