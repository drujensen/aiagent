package apicontrollers

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"
	"aiagent/internal/impl/config"
	"encoding/json"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type AgentController struct {
	logger       *zap.Logger
	agentService services.AgentService
	config       *config.Config
}

// NewAgentController is a constructor that returns a new instance of the AgentController
func NewAgentController(logger *zap.Logger, agentService services.AgentService, config *config.Config) *AgentController {
	return &AgentController{
		logger:       logger,
		agentService: agentService,
		config:       config,
	}
}

// ListAgents handles the GET request to list all agents
func (c *AgentController) ListAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agents, err := c.agentService.ListAgents(ctx)
	if err != nil {
		c.handleError(w, err, http.StatusInternalServerError)
		return
	}

	c.respondWithJSON(w, http.StatusOK, agents)
}

// GetAgent handles the GET request to retrieve a specific agent
func (c *AgentController) GetAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.URL.Query().Get("id")
	if id == "" {
		c.handleError(w, "Missing agent ID", http.StatusBadRequest)
		return
	}

	agent, err := c.agentService.GetAgent(ctx, id)
	if err != nil {
		if err == entities.ErrAgentNotFound {
			c.handleError(w, "Agent not found", http.StatusNotFound)
			return
		}
		c.handleError(w, err, http.StatusInternalServerError)
		return
	}

	c.respondWithJSON(w, http.StatusOK, agent)
}

// CreateAgent handles the POST request to create a new agent
func (c *AgentController) CreateAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var agent entities.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		c.handleError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	agent.ID = primitive.NewObjectID()
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()

	if err := c.agentService.CreateAgent(ctx, &agent); err != nil {
		c.handleError(w, err, http.StatusInternalServerError)
		return
	}

	c.respondWithJSON(w, http.StatusCreated, agent)
}

// UpdateAgent handles the PUT request to update an existing agent
func (c *AgentController) UpdateAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.URL.Query().Get("id")
	if id == "" {
		c.handleError(w, "Missing agent ID", http.StatusBadRequest)
		return
	}

	var agent entities.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		c.handleError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	agent.ID, _ = primitive.ObjectIDFromHex(id)
	agent.UpdatedAt = time.Now()

	if err := c.agentService.UpdateAgent(ctx, &agent); err != nil {
		if err == entities.ErrAgentNotFound {
			c.handleError(w, "Agent not found", http.StatusNotFound)
			return
		}
		c.handleError(w, err, http.StatusInternalServerError)
		return
	}

	c.respondWithJSON(w, http.StatusOK, agent)
}

// DeleteAgent handles the DELETE request to delete an agent
func (c *AgentController) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.URL.Query().Get("id")
	if id == "" {
		c.handleError(w, "Missing agent ID", http.StatusBadRequest)
		return
	}

	if err := c.agentService.DeleteAgent(ctx, id); err != nil {
		if err == entities.ErrAgentNotFound {
			c.handleError(w, "Agent not found", http.StatusNotFound)
			return
		}
		c.handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper function to handle errors
func (c *AgentController) handleError(w http.ResponseWriter, err interface{}, statusCode int) {
	c.logger.Error("Error occurred", zap.Any("error", err))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err,
	})
}

// Helper function to respond with JSON
func (c *AgentController) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
