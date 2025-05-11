package apicontrollers

import (
	"aiagent/internal/domain/services"
	"net/http"

	"github.com/labstack/echo/v4"
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
}

// ListAgents godoc
//	@Summary		List all agents
//	@Description	Retrieves a list of all agents.
//	@Tags			agents
//	@Produce		json
//	@Success		200	{array}		entities.Agent			"Successfully retrieved list of agents"
//	@Failure		500	{object}	map[string]any	"Internal server error"
//	@Router			/api/agents [get]
func (c *AgentController) ListAgents(ctx echo.Context) error {
	agents, err := c.agentService.ListAgents(ctx.Request().Context())
	if err != nil {
		return c.handleError(ctx, err, http.StatusInternalServerError)
	}
	return ctx.JSON(http.StatusOK, agents)
}

// handleError handles errors and returns them in a consistent format
func (c *AgentController) handleError(ctx echo.Context, err any, statusCode int) error {
	c.logger.Error("Error occurred", zap.Any("error", err))
	return ctx.JSON(statusCode, map[string]any{
		"error": err,
	})
}
