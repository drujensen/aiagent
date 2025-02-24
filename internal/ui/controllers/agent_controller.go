package uicontrollers

import (
	"html/template"
	"net/http"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type AgentController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	agentService services.AgentService
	toolService  services.ToolService
}

func NewAgentController(logger *zap.Logger, tmpl *template.Template, agentService services.AgentService, toolService services.ToolService) *AgentController {
	return &AgentController{
		logger:       logger,
		tmpl:         tmpl,
		agentService: agentService,
		toolService:  toolService,
	}
}

func (c *AgentController) AgentListHandler(eCtx echo.Context) error {
	agents, err := c.agentService.ListAgents(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}
	data := map[string]interface{}{
		"Title":           "Agents",
		"ContentTemplate": "agent_list_content",
		"Agents":          agents,
		"RootAgents":      agents, // Simplified for chat app
	}
	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *AgentController) AgentFormHandler(eCtx echo.Context) error {
	agents, err := c.agentService.ListAgents(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	tools, err := c.toolService.ListTools(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list tools", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	var agent *entities.Agent
	path := eCtx.Request().URL.Path
	isEdit := strings.HasPrefix(path, "/agents/edit/")
	if isEdit {
		id := eCtx.Param("id") // Using Echo's param instead of manual path parsing
		if id == "" {
			return eCtx.String(http.StatusBadRequest, "Agent ID is required for editing")
		}
		agent, err = c.agentService.GetAgent(eCtx.Request().Context(), id)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return eCtx.String(http.StatusNotFound, "Agent not found")
			}
			c.logger.Error("Failed to get agent", zap.String("id", id), zap.Error(err))
			return eCtx.String(http.StatusInternalServerError, "Internal server error")
		}
	}

	agentData := struct {
		ID           string
		Name         string
		Endpoint     string
		Model        string
		APIKey       string
		SystemPrompt string
		Temperature  *float64
		MaxTokens    *int
	}{}
	if agent != nil {
		agentData.ID = agent.ID.Hex()
		agentData.Name = agent.Name
		agentData.Endpoint = agent.Endpoint
		agentData.Model = agent.Model
		agentData.APIKey = agent.APIKey
		agentData.SystemPrompt = agent.SystemPrompt
		agentData.Temperature = agent.Temperature
		agentData.MaxTokens = agent.MaxTokens
	}

	data := map[string]interface{}{
		"Title":           "Agent Form",
		"ContentTemplate": "agent_form_content",
		"Agent":           agentData,
		"Tools":           tools,
		"RootAgents":      agents,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}
