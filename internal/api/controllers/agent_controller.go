package apicontrollers

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/services"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type AgentController struct {
	logger       *zap.Logger
	agentService services.AgentService
}

func NewAgentController(logger *zap.Logger, agentService services.AgentService) *AgentController {
	return &AgentController{
		logger:       logger,
		agentService: agentService,
	}
}

// RegisterRoutes registers all agent-related routes with Echo
func (c *AgentController) RegisterRoutes(e *echo.Group) {
	e.GET("/agents", c.ListAgents)
	e.GET("/agents/:id", c.GetAgent)
	e.POST("/agents", c.CreateAgent)
	e.PUT("/agents/:id", c.UpdateAgent)
	e.DELETE("/agents/:id", c.DeleteAgent)
}

// ListAgents godoc
// @Summary List all agents
// @Description Retrieves a list of all agents.
// @Tags agents
// @Produce json
// @Success 200 {array} entities.Agent "Successfully retrieved list of agents"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/agents [get]
func (c *AgentController) ListAgents(ctx echo.Context) error {
	agents, err := c.agentService.ListAgents(ctx.Request().Context())
	if err != nil {
		return c.handleError(ctx, err, http.StatusInternalServerError)
	}
	return ctx.JSON(http.StatusOK, agents)
}

// GetAgent godoc
// @Summary Get an agent by ID
// @Description Retrieves an agent's information by their ID.
// @Tags agents
// @Produce json
// @Param id path string true "Agent ID"
// @Success 200 {object} entities.Agent "Successfully retrieved agent"
// @Failure 400 {object} map[string]interface{} "Invalid agent ID"
// @Failure 404 {object} map[string]interface{} "Agent not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/agents/{id} [get]
func (c *AgentController) GetAgent(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return c.handleError(ctx, "Missing agent ID", http.StatusBadRequest)
	}

	agent, err := c.agentService.GetAgent(ctx.Request().Context(), id)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return c.handleError(ctx, "Agent not found", http.StatusNotFound)
		default:
			return c.handleError(ctx, err, http.StatusInternalServerError)
		}
	}

	return ctx.JSON(http.StatusOK, agent)
}

// CreateAgent godoc
// @Summary Create a new agent
// @Description Creates a new agent with the provided information.
// @Tags agents
// @Accept json
// @Produce json
// @Param agent body entities.Agent true "Agent information to create"
// @Success 201 {object} entities.Agent "Successfully created agent"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/agents [post]
func (c *AgentController) CreateAgent(ctx echo.Context) error {
	var agent entities.Agent
	if err := ctx.Bind(&agent); err != nil {
		return c.handleError(ctx, "Invalid request body", http.StatusBadRequest)
	}

	agent.ID = primitive.NewObjectID()
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()

	if err := c.agentService.CreateAgent(ctx.Request().Context(), &agent); err != nil {
		return c.handleError(ctx, err, http.StatusInternalServerError)
	}

	return ctx.JSON(http.StatusCreated, agent)
}

// UpdateAgent godoc
// @Summary Update an existing agent
// @Description Updates an existing agent with the provided information.
// @Tags agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID"
// @Param agent body entities.Agent true "Agent information to update"
// @Success 200 {object} entities.Agent "Successfully updated agent"
// @Failure 400 {object} map[string]interface{} "Invalid request body or agent ID"
// @Failure 404 {object} map[string]interface{} "Agent not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/agents/{id} [put]
func (c *AgentController) UpdateAgent(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return c.handleError(ctx, "Missing agent ID", http.StatusBadRequest)
	}

	var agent entities.Agent
	if err := ctx.Bind(&agent); err != nil {
		return c.handleError(ctx, "Invalid request body", http.StatusBadRequest)
	}

	var err error
	agent.ID, err = primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.handleError(ctx, "Invalid agent ID", http.StatusBadRequest)
	}
	agent.UpdatedAt = time.Now()

	if err := c.agentService.UpdateAgent(ctx.Request().Context(), &agent); err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return c.handleError(ctx, "Agent not found", http.StatusNotFound)
		default:
			return c.handleError(ctx, err, http.StatusInternalServerError)
		}
	}

	return ctx.JSON(http.StatusOK, agent)
}

// DeleteAgent godoc
// @Summary Delete an agent
// @Description Deletes an agent by their ID.
// @Tags agents
// @Produce json
// @Param id path string true "Agent ID"
// @Success 204 "Successfully deleted agent"
// @Failure 400 {object} map[string]interface{} "Invalid agent ID"
// @Failure 404 {object} map[string]interface{} "Agent not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/agents/{id} [delete]
func (c *AgentController) DeleteAgent(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return c.handleError(ctx, "Missing agent ID", http.StatusBadRequest)
	}

	if err := c.agentService.DeleteAgent(ctx.Request().Context(), id); err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return c.handleError(ctx, "Agent not found", http.StatusNotFound)
		default:
			return c.handleError(ctx, err, http.StatusInternalServerError)
		}
	}

	return ctx.NoContent(http.StatusNoContent)
}

// handleError handles errors and returns them in a consistent format
func (c *AgentController) handleError(ctx echo.Context, err interface{}, statusCode int) error {
	c.logger.Error("Error occurred", zap.Any("error", err))
	return ctx.JSON(statusCode, map[string]interface{}{
		"error": err,
	})
}
