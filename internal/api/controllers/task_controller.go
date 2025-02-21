/**
 * @description
 * TaskController handles HTTP requests for task-related API endpoints in the AI Workflow Automation Platform.
 * It provides RESTful APIs for starting workflows, retrieving specific tasks, and listing all tasks,
 * ensuring proper authentication and JSON responses for programmatic clients.
 *
 * Key features:
 * - Authentication: Validates requests using the X-API-Key header against the configured local API key.
 * - Workflow Initiation: Exposes POST /api/workflows to start a new workflow, returning JSON responses.
 * - Task Status Retrieval: Exposes GET /api/tasks/{id} to fetch a specific task's status and details in JSON.
 * - Task Listing: Exposes GET /api/tasks to list all tasks with enriched agent names in JSON for API clients.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides the Task entity definition.
 * - aiagent/internal/domain/services: Provides TaskService and AgentService for business logic.
 * - aiagent/internal/infrastructure/config: Provides access to application configuration.
 * - encoding/json: For JSON decoding and encoding.
 * - fmt: For formatting strings in responses.
 * - net/http: Standard Go package for HTTP handling.
 * - strings: For parsing paths and content types.
 *
 * @notes
 * - The controller assumes TaskService and AgentService are properly initialized and injected.
 * - Error responses follow a consistent {"error": "message"} format with HTTP status codes.
 * - Edge cases include unauthorized requests, invalid bodies, non-existent tasks, and service errors.
 * - Assumption: API key validation is sufficient for local access control.
 */

package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/repositories"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
)

// TaskController manages task-related HTTP API endpoints.
// It interacts with TaskService for workflow logic and AgentService for agent data enrichment.
type TaskController struct {
	TaskService  services.TaskService  // Service for task-related business logic
	AgentService services.AgentService // Service for agent-related data, used for enriching task info
	Config       *config.Config        // Configuration for API key validation
}

// StartWorkflowRequest defines the structure of the request body for starting a workflow via API.
type StartWorkflowRequest struct {
	Description string `json:"description"` // Description of the task to be performed
	AssignedTo  string `json:"assigned_to"` // ID of the agent to assign the task to
}

// StartWorkflow handles POST requests to /api/workflows, creating and starting a new task workflow.
// It expects and returns JSON responses for API clients, without HTMX-specific logic.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing method, headers, and JSON body.
//
// Returns:
// - None; writes directly to the response writer with status codes and JSON.
func (c *TaskController) StartWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Only handle JSON requests for API
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, `{"error": "Content-Type must be application/json"}`, http.StatusUnsupportedMediaType)
		return
	}

	var req StartWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Description == "" || req.AssignedTo == "" {
		http.Error(w, `{"error": "description and assigned_to are required"}`, http.StatusBadRequest)
		return
	}

	// Create and start the task
	task := &entities.Task{
		Description: req.Description,
		AssignedTo:  req.AssignedTo,
		Status:      entities.TaskPending,
	}

	if err := c.TaskService.StartWorkflow(r.Context(), task); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Return JSON response for API clients
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

// TaskDetailHandler handles GET requests to /api/tasks/{id}, retrieving the status and details of a specific task.
// It returns JSON responses for API clients.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing method, headers, and path.
//
// Returns:
// - None; writes directly to the response writer with status codes and JSON.
func (c *TaskController) TaskDetailHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/tasks/") {
		http.NotFound(w, r)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	task, err := c.TaskService.GetTask(r.Context(), id)
	if err != nil {
		if err == repositories.ErrNotFound {
			http.Error(w, `{"error": "task not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "failed to get task"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

// ListTasks handles GET requests to /api/tasks, listing all tasks with agent names.
// It returns JSON responses for API clients.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing method, headers, and context.
//
// Returns:
// - None; writes directly to the response writer with status codes and JSON.
func (c *TaskController) ListTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Fetch all tasks
	tasks, err := c.TaskService.ListTasks(r.Context())
	if err != nil {
		http.Error(w, `{"error": "failed to list tasks"}`, http.StatusInternalServerError)
		return
	}

	// Fetch all agents to map IDs to names
	agents, err := c.AgentService.ListAgents(r.Context())
	if err != nil {
		http.Error(w, `{"error": "failed to list agents"}`, http.StatusInternalServerError)
		return
	}

	// Create a map for quick agent name lookup
	agentMap := make(map[string]string)
	for _, agent := range agents {
		agentMap[agent.ID] = agent.Name
	}

	// Define a view model for JSON response
	type TaskView struct {
		ID          string `json:"id"`          // Unique task identifier
		Description string `json:"description"` // Task description
		AssignedTo  string `json:"assigned_to"` // Agent ID assigned to the task
		AgentName   string `json:"agent_name"`  // Human-readable agent name
		Status      string `json:"status"`      // Current task status (e.g., "pending")
		Result      string `json:"result"`      // Task result, if completed or failed
	}

	// Enrich tasks with agent names
	var taskViews []TaskView
	for _, task := range tasks {
		agentName, ok := agentMap[task.AssignedTo]
		if !ok {
			agentName = "Unknown" // Fallback for missing agents
		}
		taskViews = append(taskViews, TaskView{
			ID:          task.ID,
			Description: task.Description,
			AssignedTo:  task.AssignedTo,
			AgentName:   agentName,
			Status:      string(task.Status),
			Result:      task.Result,
		})
	}

	// Return enriched task list in JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(taskViews); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}
