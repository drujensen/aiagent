package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AgentController struct {
	Service services.AgentService
	Config  *config.Config
}

func (c *AgentController) AgentsHandler(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *AgentController) AgentDetailHandler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/agents/") {
		http.NotFound(w, r)
		return
	}

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
		w.Header().Set("Allow", "GET, PUT, DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *AgentController) ListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	agents, err := c.Service.ListAgents(r.Context())
	if err != nil {
		http.Error(w, `{"error": "failed to list agents"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agents); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

func (c *AgentController) CreateAgent(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var agent entities.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := c.Service.CreateAgent(r.Context(), &agent); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(agent); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

func (c *AgentController) GetAgent(w http.ResponseWriter, r *http.Request, id string) {
	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	agent, err := c.Service.GetAgent(r.Context(), id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, `{"error": "agent not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "failed to get agent"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agent); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

func (c *AgentController) UpdateAgent(w http.ResponseWriter, r *http.Request, id string) {
	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var agent entities.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, `{"error": "invalid agent ID"}`, http.StatusBadRequest)
		return
	}
	agent.ID = oid

	if err := c.Service.UpdateAgent(r.Context(), &agent); err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, `{"error": "agent not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(agent); err != nil {
		http.Error(w, `{"error": "failed to encode response"}`, http.StatusInternalServerError)
	}
}

func (c *AgentController) DeleteAgent(w http.ResponseWriter, r *http.Request, id string) {
	if r.Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	err := c.Service.DeleteAgent(r.Context(), id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, `{"error": "agent not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
