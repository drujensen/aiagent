/**
 * @description
 * HomeController manages UI-related requests for the AI Workflow Automation Platform within the internal/ui package.
 * It handles rendering pages like home, agent list, agent form, workflows, and task lists with server-side HTML rendering
 * using Go's html/template package. It supports full page loads and HTMX-driven partial updates, including agent hierarchy
 * data for the sidebar, workflow initiation, and task monitoring for workflows.
 *
 * Key features:
 * - Page Rendering: Renders multiple UI pages with dynamic data.
 * - HTMX Support: Handles both full layout and partial template rendering for HTMX requests.
 * - Workflow UI: Provides workflow initiation forms and task monitoring interfaces, including HTMX form submissions.
 * - Hierarchy Integration: Fetches and builds agent hierarchy for sidebar display.
 *
 * @dependencies
 * - html/template: For server-side templating.
 * - net/http: For HTTP request handling.
 * - go.uber.org/zap: For structured logging.
 * - aiagent/internal/domain/entities: For AIAgent and Task entity definitions.
 * - aiagent/internal/domain/services: For AgentService and TaskService interfaces.
 *
 * @notes
 * - Template paths are relative to the project root (./internal/ui/templates/).
 * - Assumes layout.html includes header, sidebar, and content templates dynamically.
 * - Edge cases: Handles missing agents, tasks, or template errors with HTTP 500 and logs.
 * - Assumption: Static assets (htmx.min.js, styles.css) are served via main.go.
 */

package ui

import (
	"fmt"
	"html/template"
	"net/http"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"

	"go.uber.org/zap"
)

// AgentNode represents a node in the agent hierarchy tree for UI rendering.
// It simplifies the AIAgent entity for template use, focusing on ID, Name, and Children.
type AgentNode struct {
	ID       string       // Unique identifier of the agent
	Name     string       // Display name of the agent
	Children []*AgentNode // Recursive list of child nodes
}

// HomeController manages UI-related requests for the web application.
type HomeController struct {
	logger       *zap.Logger           // Logger for error reporting
	tmpl         *template.Template    // Pre-parsed templates for rendering
	agentService services.AgentService // Service for fetching agent data
	taskService  services.TaskService  // Service for fetching task data
}

// NewHomeController creates a new HomeController instance with pre-parsed templates and services.
// It initializes the template set and ensures all dependencies are provided.
//
// Parameters:
// - logger: Zap logger for error logging.
// - agentService: Service interface for agent operations.
// - taskService: Service interface for task operations.
//
// Returns:
// - *HomeController: Initialized controller instance.
func NewHomeController(logger *zap.Logger, agentService services.AgentService, taskService services.TaskService) *HomeController {
	tmpl, err := template.ParseFiles(
		"./internal/ui/templates/layout.html",
		"./internal/ui/templates/header.html",
		"./internal/ui/templates/sidebar.html",
		"./internal/ui/templates/home.html",
		"./internal/ui/templates/agent_list.html",
		"./internal/ui/templates/agent_form.html",
		"./internal/ui/templates/workflow.html", // Added for workflow UI
	)
	if err != nil {
		logger.Fatal("Failed to parse templates", zap.Error(err))
	}
	return &HomeController{
		logger:       logger,
		tmpl:         tmpl,
		agentService: agentService,
		taskService:  taskService,
	}
}

// buildHierarchy constructs a tree of AgentNodes from a flat list of AIAgents.
// It maps agents by ID and recursively builds parent-child relationships.
//
// Parameters:
// - agents: List of AIAgent entities from the service.
//
// Returns:
// - []*AgentNode: List of root nodes (agents with no parent).
func buildHierarchy(agents []*entities.AIAgent) []*AgentNode {
	agentMap := make(map[string]*entities.AIAgent)
	for _, agent := range agents {
		agentMap[agent.ID] = agent
	}
	rootAgents := []*AgentNode{}
	for _, agent := range agents {
		if agent.ParentID == "" {
			node := &AgentNode{ID: agent.ID, Name: agent.Name}
			rootAgents = append(rootAgents, node)
			buildChildren(node, agentMap)
		}
	}
	return rootAgents
}

// buildChildren recursively populates the Children field of an AgentNode.
// It looks up child agents in the map and builds their subtrees.
//
// Parameters:
// - node: Current node being processed.
// - agentMap: Map of agent IDs to AIAgent entities for quick lookup.
func buildChildren(node *AgentNode, agentMap map[string]*entities.AIAgent) {
	agent := agentMap[node.ID]
	for _, childID := range agent.ChildrenIDs {
		if childAgent, exists := agentMap[childID]; exists {
			childNode := &AgentNode{ID: childAgent.ID, Name: childAgent.Name}
			node.Children = append(node.Children, childNode)
			buildChildren(childNode, agentMap)
		}
	}
}

// HomeHandler handles requests to the root path ("/"), rendering the home page.
// It fetches the agent hierarchy and includes it in the template data.
//
// Parameters:
// - w: HTTP response writer for sending rendered HTML.
// - r: HTTP request with headers (e.g., HX-Request) and context.
func (c *HomeController) HomeHandler(w http.ResponseWriter, r *http.Request) {
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	rootAgents := buildHierarchy(agents)
	data := map[string]interface{}{
		"Title":           "AI Workflow Automation Platform",
		"ContentTemplate": "home_content",
		"RootAgents":      rootAgents,
	}
	isHtmxRequest := r.Header.Get("HX-Request") == "true"
	tmplName := "layout"
	if isHtmxRequest {
		tmplName = "home_content"
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, tmplName, data); err != nil {
		c.logger.Error("Failed to render template", zap.String("template", tmplName), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AgentListHandler handles requests to /agents, rendering the agent list page.
// It includes the hierarchy in the sidebar for consistency.
//
// Parameters:
// - w: HTTP response writer for sending rendered HTML.
// - r: HTTP request with context for fetching agents.
func (c *HomeController) AgentListHandler(w http.ResponseWriter, r *http.Request) {
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	rootAgents := buildHierarchy(agents)
	data := map[string]interface{}{
		"Title":           "Agents",
		"ContentTemplate": "agent_list_content",
		"Agents":          agents,
		"RootAgents":      rootAgents,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render agent_list", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AgentFormHandler handles requests to /agents/new or /agents/edit/{id}, rendering the agent form.
// It includes the hierarchy in the sidebar.
//
// Parameters:
// - w: HTTP response writer for sending rendered HTML.
// - r: HTTP request with context for fetching agents.
func (c *HomeController) AgentFormHandler(w http.ResponseWriter, r *http.Request) {
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	rootAgents := buildHierarchy(agents)
	data := map[string]interface{}{
		"Title":           "Agent Form",
		"ContentTemplate": "agent_form_content",
		"Agent": struct {
			ID            string
			Name          string
			Prompt        string
			Configuration struct {
				APIKey, LocalURL string
				Temperature      float64
				ThinkingTime     int
			}
			HumanInteractionEnabled bool
		}{},
		"Tools": []struct {
			ID, Name string
			Selected bool
		}{{ID: "1", Name: "Tool1"}},
		"Providers": []struct {
			Value, Label string
			Selected     bool
		}{{Value: "openai", Label: "OpenAI"}},
		"RootAgents": rootAgents,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render agent_form", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// WorkflowHandler handles GET requests to /workflows, rendering the workflow page.
// It fetches agents for the dropdown and tasks for the initial list, enriching tasks with agent names.
//
// Parameters:
// - w: HTTP response writer for sending rendered HTML.
// - r: HTTP request with context for fetching data.
//
// Returns:
// - None; writes directly to the response writer with HTML or error responses.
func (c *HomeController) WorkflowHandler(w http.ResponseWriter, r *http.Request) {
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
// It processes the form data, starts a workflow, and returns HTML for HTMX updates.
//
// Parameters:
// - w: HTTP response writer for sending rendered HTML.
// - r: HTTP request containing method, headers, and form data.
//
// Returns:
// - None; writes directly to the response writer with HTML or error responses.
func (c *HomeController) WorkflowSubmitHandler(w http.ResponseWriter, r *http.Request) {
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
	fmt.Fprintf(w, `<div id="response-message">Workflow started successfully. Task ID: %s</div>`)
}

// TaskListHandler handles GET requests to /tasks, rendering the task list partial.
// It fetches tasks, enriches with agent names, and renders the "task_list" template for HTMX updates.
//
// Parameters:
// - w: HTTP response writer for sending rendered HTML.
// - r: HTTP request with context for fetching data.
//
// Returns:
// - None; writes directly to the response writer with HTML or error responses.
func (c *HomeController) TaskListHandler(w http.ResponseWriter, r *http.Request) {
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
