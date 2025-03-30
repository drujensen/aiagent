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

// ListAgents handles the GET request to list all agents
func (c *AgentController) ListAgents(ctx echo.Context) error {
	agents, err := c.agentService.ListAgents(ctx.Request().Context())
	if err != nil {
		return c.handleError(ctx, err, http.StatusInternalServerError)
	}
	return ctx.JSON(http.StatusOK, agents)
}

// GetAgent handles the GET request to retrieve a specific agent
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

// CreateAgent handles the POST request to create a new agent
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

// UpdateAgent handles the PUT request to update an existing agent
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

// DeleteAgent handles the DELETE request to delete an agent
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
