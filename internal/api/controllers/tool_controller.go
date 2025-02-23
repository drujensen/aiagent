package controllers

import (
	"encoding/json"
	"net/http"

	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
)

type ToolController struct {
	ToolService services.ToolService
	Config      *config.Config
}

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
