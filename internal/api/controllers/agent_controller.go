package apicontrollers

import (
	"net/http"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AgentController struct {
	Service services.AgentService
	Config  *config.Config
}

func NewAgentController(agentService services.AgentService, cfg *config.Config) *AgentController {
	return &AgentController{
		Service: agentService,
		Config:  cfg,
	}
}

func (c *AgentController) AgentsHandler(eCtx echo.Context) error {
	if !strings.HasPrefix(eCtx.Request().URL.Path, "/api/agents") {
		return eCtx.NoContent(http.StatusNotFound)
	}

	switch eCtx.Request().Method {
	case http.MethodGet:
		return c.ListAgents(eCtx)
	case http.MethodPost:
		return c.CreateAgent(eCtx)
	default:
		eCtx.Response().Header().Set("Allow", "GET, POST")
		return eCtx.String(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (c *AgentController) AgentDetailHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.NoContent(http.StatusNotFound)
	}

	switch eCtx.Request().Method {
	case http.MethodGet:
		return c.GetAgent(eCtx, id)
	case http.MethodPut:
		return c.UpdateAgent(eCtx, id)
	case http.MethodDelete:
		return c.DeleteAgent(eCtx, id)
	default:
		eCtx.Response().Header().Set("Allow", "GET, PUT, DELETE")
		return eCtx.String(http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (c *AgentController) ListAgents(eCtx echo.Context) error {
	if eCtx.Request().Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		return eCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	agents, err := c.Service.ListAgents(eCtx.Request().Context())
	if err != nil {
		return eCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list agents"})
	}

	return eCtx.JSON(http.StatusOK, agents)
}

func (c *AgentController) CreateAgent(eCtx echo.Context) error {
	if eCtx.Request().Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		return eCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var agent entities.Agent
	if err := eCtx.Bind(&agent); err != nil {
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Convert tool IDs from strings to ObjectIDs
	if len(agent.Tools) > 0 {
		toolIDs := make([]primitive.ObjectID, 0, len(agent.Tools))
		for _, toolID := range agent.Tools {
			if oid, err := primitive.ObjectIDFromHex(toolID.Hex()); err == nil {
				toolIDs = append(toolIDs, oid)
			}
		}
		agent.Tools = toolIDs
	}

	if err := c.Service.CreateAgent(eCtx.Request().Context(), &agent); err != nil {
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	eCtx.Response().Header().Set("Content-Type", "application/json")
	eCtx.Response().WriteHeader(http.StatusCreated)
	return eCtx.JSON(http.StatusCreated, agent)
}

func (c *AgentController) GetAgent(eCtx echo.Context, id string) error {
	if eCtx.Request().Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		return eCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	agent, err := c.Service.GetAgent(eCtx.Request().Context(), id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return eCtx.JSON(http.StatusNotFound, map[string]string{"error": "agent not found"})
		}
		return eCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get agent"})
	}

	return eCtx.JSON(http.StatusOK, agent)
}

func (c *AgentController) UpdateAgent(eCtx echo.Context, id string) error {
	if eCtx.Request().Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		return eCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	var agent entities.Agent
	if err := eCtx.Bind(&agent); err != nil {
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid agent ID"})
	}
	agent.ID = oid

	// Convert tool IDs from strings to ObjectIDs
	if len(agent.Tools) > 0 {
		toolIDs := make([]primitive.ObjectID, 0, len(agent.Tools))
		for _, toolID := range agent.Tools {
			if oid, err := primitive.ObjectIDFromHex(toolID.Hex()); err == nil {
				toolIDs = append(toolIDs, oid)
			}
		}
		agent.Tools = toolIDs
	}

	if err := c.Service.UpdateAgent(eCtx.Request().Context(), &agent); err != nil {
		if err == mongo.ErrNoDocuments {
			return eCtx.JSON(http.StatusNotFound, map[string]string{"error": "agent not found"})
		}
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return eCtx.JSON(http.StatusOK, agent)
}

func (c *AgentController) DeleteAgent(eCtx echo.Context, id string) error {
	if eCtx.Request().Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		return eCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	err := c.Service.DeleteAgent(eCtx.Request().Context(), id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return eCtx.JSON(http.StatusNotFound, map[string]string{"error": "agent not found"})
		}
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return eCtx.NoContent(http.StatusNoContent)
}
