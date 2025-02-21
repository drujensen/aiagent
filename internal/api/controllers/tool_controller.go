package controllers

import (
	"encoding/json"
	"net/http"

	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
)

/**
 * @description
 * ToolController handles HTTP requests for tool-related endpoints in the AI Workflow Automation Platform.
 * It provides a RESTful API for listing available tools, ensuring proper authentication and error handling.
 *
 * Key features:
 * - Authentication: Validates requests using the X-API-Key header against the configured local API key.
 * - Tool Listing: Exposes a GET /tools endpoint to retrieve the list of available tools.
 *
 * @dependencies
 * - aiagent/internal/domain/services: Provides the ToolService for business logic.
 * - aiagent/internal/infrastructure/config: Provides access to application configuration.
 * - net/http: Standard Go package for HTTP handling.
 * - encoding/json: For JSON encoding of responses.
 *
 * @notes
 * - The controller assumes that the ToolService is properly initialized and injected.
 * - Error responses follow a consistent {"error": "message"} format with appropriate HTTP status codes.
 * - Edge cases include unauthorized requests, repository errors, and JSON encoding failures.
 * - Assumption: The ToolService handles any necessary validation or filtering of tools.
 */

type ToolController struct {
	ToolService services.ToolService // Service for tool-related business logic
	Config      *config.Config       // Configuration for API key validation
}

// ListTools handles GET requests to /tools, listing all available tools.
// It checks for authentication and uses the ToolService to retrieve the tools.
//
// Parameters:
// - w: HTTP response writer for sending responses.
// - r: HTTP request containing method, headers, and context.
//
// Returns:
// - None; writes directly to the response writer with status codes and JSON.
func (c *ToolController) ListTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	tools, err := c.ToolService.ListTools(r.Context())
	if err != nil {
		http.Error(w, `{"error": "failed to list tools"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tools); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}
