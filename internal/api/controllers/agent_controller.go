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

// AgentController handles HTTP requests for agent management.
// It provides RESTful endpoints for CRUD operations on agents,
// enforcing authentication and interacting with AgentService.
//
// Key features:
// - RESTful Endpoints: Supports GET, POST for /agents and GET, PUT, DELETE for /agents/{id}.
// - Authentication: Validates requests using X-API-Key header against LOCAL_API_KEY.
// - JSON Handling: Parses request bodies and serializes responses in JSON format.
//
// Dependencies:
// - aiagent/internal/domain/entities: Provides the AIAgent entity definition.
// - aiagent/internal/domain/repositories: Provides ErrNotFound for error handling.
// - aiagent/internal/domain/services: Provides AgentService for business logic.
// - aiagent/internal/infrastructure/config: Provides access to application configuration.
// - net/http: Standard Go package for HTTP handling.
// - encoding/json: For JSON encoding and decoding.
// - strings: For path manipulation.
//
// Notes:
// - Authentication is checked in each handler method for simplicity.
// - Error responses follow {"error": "message"} format with appropriate HTTP status codes.
// - Edge cases include invalid JSON, missing fields, and unauthorized requests.
// - Assumption: Service layer handles validation and hierarchy checks.
// - Limitation: Manual path parsing used; gorilla/mux could enhance routing if needed.
type AgentController struct {
	Service services.AgentService // AgentService interface for business logic operations
	Config  *config.Config        // Configuration for API key validation
}

// AgentsHandler handles requests to /agents for listing (GET) and creating (POST) agents.
// It ensures the path is exactly "/agents" and delegates to specific methods based on HTTP method.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing method, headers, and body.
//
// Returns:
// - None; writes directly to response writer with status codes and JSON.
func (c *AgentController) AgentsHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure exact path match for /api/agents
	if r.URL.Path != "/api/agents" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		c.ListAgents(w, r)
	case http.MethodPost:
		c.CreateAgent(w, r)
	default:
		// Set allowed methods for clarity in error response
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// AgentDetailHandler handles requests to /agents/{id} for getting, updating, and deleting agents.
// It extracts the ID from the path and delegates based on HTTP method.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing method, headers, and body.
//
// Returns:
// - None; writes directly to response writer with status codes and JSON.
func (c *AgentController) AgentDetailHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure path starts with /api/agents/
	if !strings.HasPrefix(r.URL.Path, "/api/agents/") {
		http.NotFound(w, r)
		return
	}

	// Extract ID from path (e.g., /api/agents/123 -> 123)
	id := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		c.GetAgent(w, r, id)
	case http.MethodPut:
		c.UpdateAgent(w, r, id)
	case http.MethodDelete:
		c.DeleteAgent(w, r, id)
	default:
		// Set allowed methods for clarity in error response
		w.Header().Set("Allow", "GET, PUT, DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ListAgents lists all agents by calling AgentService.ListAgents.
// It checks authentication and returns a JSON array of agents.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing headers for authentication.
//
// Returns:
// - None; writes JSON array on success or error response on failure.
func (c *AgentController) ListAgents(w http.ResponseWriter, r *http.Request) {
	// Check API key authentication
	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Retrieve agents using AgentService
	agents, err := c.Service.ListAgents(r.Context())
	if err != nil {
		http.Error(w, `{"error": "failed to list agents"}`, http.StatusInternalServerError)
		return
	}

	// Set JSON content type and encode response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

// CreateAgent creates a new agent from the POST request body.
// It decodes JSON, resets ID for generation, and calls AgentService.CreateAgent.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing headers and body with agent data.
//
// Returns:
// - None; writes created agent JSON on success or error response on failure.
func (c *AgentController) CreateAgent(w http.ResponseWriter, r *http.Request) {
	// Check API key authentication
	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Decode request body into AIAgent
	var agent entities.AIAgent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Reset ID to ensure service generates new ID
	agent.ID = ""
	if err := c.Service.CreateAgent(r.Context(), &agent); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Set JSON content type and return created agent
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(agent); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

// GetAgent retrieves an agent by ID using AgentService.GetAgent.
// It returns the agent's JSON representation or an error response.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing headers for authentication.
// - id: The agent ID extracted from the URL path.
//
// Returns:
// - None; writes agent JSON on success or error response on failure.
func (c *AgentController) GetAgent(w http.ResponseWriter, r *http.Request, id string) {
	// Check API key authentication
	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Retrieve agent using AgentService
	agent, err := c.Service.GetAgent(r.Context(), id)
	if err != nil {
		if err == repositories.ErrNotFound {
			http.Error(w, `{"error": "agent not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "failed to get agent"}`, http.StatusInternalServerError)
		}
		return
	}

	// Set JSON content type and encode response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agent); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

// UpdateAgent updates an agent using the PUT request body and ID from path.
// It decodes JSON, sets ID, and calls AgentService.UpdateAgent.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing headers and body with updated agent data.
// - id: The agent ID extracted from the URL path.
//
// Returns:
// - None; writes updated agent JSON on success or error response on failure.
func (c *AgentController) UpdateAgent(w http.ResponseWriter, r *http.Request, id string) {
	// Check API key authentication
	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Decode request body into AIAgent
	var agent entities.AIAgent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Set ID from path to ensure correct agent is updated
	agent.ID = id
	if err := c.Service.UpdateAgent(r.Context(), &agent); err != nil {
		if err == repositories.ErrNotFound {
			http.Error(w, `{"error": "agent not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		}
		return
	}

	// Set JSON content type and encode response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agent); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

// DeleteAgent deletes an agent by ID using AgentService.DeleteAgent.
// It returns a 204 No Content on success or an error response on failure.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing headers for authentication.
// - id: The agent ID extracted from the URL path.
//
// Returns:
// - None; writes status code on success or error response on failure.
func (c *AgentController) DeleteAgent(w http.ResponseWriter, r *http.Request, id string) {
	// Check API key authentication
	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Delete agent using AgentService
	err := c.Service.DeleteAgent(r.Context(), id)
	if err != nil {
		if err == repositories.ErrNotFound {
			http.Error(w, `{"error": "agent not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		}
		return
	}

	// Return 204 No Content on successful deletion
	w.WriteHeader(http.StatusNoContent)
}
