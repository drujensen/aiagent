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

/**
 * @description
 * TaskController handles HTTP requests for task-related endpoints in the AI Workflow Automation Platform.
 * It provides RESTful APIs for starting workflows and retrieving task statuses, ensuring proper authentication and error handling.
 *
 * Key features:
 * - Authentication: Validates requests using the X-API-Key header against the configured local API key.
 * - Workflow Initiation: Exposes a POST /workflows endpoint to start a new workflow by creating and assigning a task.
 * - Task Status Retrieval: Exposes a GET /tasks/{id} endpoint to fetch the status and details of a specific task.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides the Task entity definition.
 * - aiagent/internal/domain/services: Provides the TaskService for business logic.
 * - aiagent/internal/infrastructure/config: Provides access to application configuration.
 * - net/http: Standard Go package for HTTP handling.
 * - encoding/json: For JSON decoding of requests and encoding of responses.
 * - strings: For path manipulation in TaskDetailHandler.
 *
 * @notes
 * - The controller assumes that the TaskService is properly initialized and injected.
 * - Error responses follow a consistent {"error": "message"} format with appropriate HTTP status codes.
 * - Edge cases include unauthorized requests, invalid request bodies, non-existent tasks, and service errors.
 * - Assumption: The TaskService handles task validation and workflow initiation logic.
 */

type TaskController struct {
	TaskService services.TaskService // Service for task-related business logic
	Config      *config.Config       // Configuration for API key validation
}

// StartWorkflowRequest defines the structure of the request body for starting a workflow.
// It expects a JSON object with description and assigned_to fields.
type StartWorkflowRequest struct {
	Description string `json:"description"` // Description of the task to be performed
	AssignedTo  string `json:"assigned_to"` // ID of the agent to assign the task to
}

// StartWorkflow handles POST requests to /workflows, creating and starting a new task workflow.
// It decodes the request body, validates required fields, and initiates the workflow via the TaskService.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing method, headers, and body.
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

	var req StartWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Description == "" || req.AssignedTo == "" {
		http.Error(w, `{"error": "description and assigned_to are required"}`, http.StatusBadRequest)
		return
	}

	task := &entities.Task{
		Description: req.Description,
		AssignedTo:  req.AssignedTo,
		Status:      entities.TaskPending,
	}

	if err := c.TaskService.StartWorkflow(r.Context(), task); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

// TaskDetailHandler handles GET requests to /tasks/{id}, retrieving the status and details of a specific task.
// It extracts the task ID from the URL path and uses the TaskService to fetch the task.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing method, headers, and path.
//
// Returns:
// - None; writes directly to the response writer with status codes and JSON.
func (c *TaskController) TaskDetailHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/tasks/") {
		http.NotFound(w, r)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/tasks/")
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
