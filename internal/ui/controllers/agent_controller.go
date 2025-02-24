package uicontrollers

import (
	"context"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type AgentController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	agentService services.AgentService
	toolService  services.ToolService
	config       *config.Config
}

func NewAgentController(logger *zap.Logger, tmpl *template.Template, agentService services.AgentService, toolService services.ToolService) *AgentController {
	cfg, err := config.InitConfig()
	if err != nil {
		logger.Fatal("Failed to initialize config in AgentController", zap.Error(err))
	}
	return &AgentController{
		logger:       logger,
		tmpl:         tmpl,
		agentService: agentService,
		toolService:  toolService,
		config:       cfg,
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
		"RootAgents":      agents,
		"APIKey":          c.config.LocalAPIKey,
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
		id := eCtx.Param("id")
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
		Tools        []string
	}{
		Tools: []string{},
	}
	if agent != nil {
		agentData.ID = agent.ID.Hex()
		agentData.Name = agent.Name
		agentData.Endpoint = agent.Endpoint
		agentData.Model = agent.Model
		agentData.APIKey = agent.APIKey
		agentData.SystemPrompt = agent.SystemPrompt
		agentData.Temperature = agent.Temperature
		agentData.MaxTokens = agent.MaxTokens
		for _, toolID := range agent.Tools {
			agentData.Tools = append(agentData.Tools, toolID.Hex())
		}
	}

	data := map[string]interface{}{
		"Title":           "Agent Form",
		"ContentTemplate": "agent_form_content",
		"Agent":           agentData,
		"Tools":           tools,
		"RootAgents":      agents,
		"APIKey":          c.config.LocalAPIKey,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *AgentController) CreateAgentHandler(eCtx echo.Context) error {
	agent := &entities.Agent{
		Name:         eCtx.FormValue("name"),
		Endpoint:     eCtx.FormValue("endpoint"),
		Model:        eCtx.FormValue("model"),
		APIKey:       eCtx.FormValue("api_key"),
		SystemPrompt: eCtx.FormValue("system_prompt"),
	}

	if tempStr := eCtx.FormValue("temperature"); tempStr != "" {
		if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
			agent.Temperature = &temp
		}
	}

	if maxTokensStr := eCtx.FormValue("max_tokens"); maxTokensStr != "" {
		if maxTokens, err := strconv.Atoi(maxTokensStr); err == nil {
			agent.MaxTokens = &maxTokens
		}
	}

	tools := eCtx.Request().Form["tools"]
	agent.Tools = make([]primitive.ObjectID, 0, len(tools))
	for _, toolID := range tools {
		if oid, err := primitive.ObjectIDFromHex(toolID); err == nil {
			agent.Tools = append(agent.Tools, oid)
		}
	}

	if err := c.agentService.CreateAgent(context.Background(), agent); err != nil {
		c.logger.Error("Failed to create agent", zap.Error(err))
		return eCtx.String(http.StatusBadRequest, "Failed to create agent: "+err.Error())
	}

	return eCtx.String(http.StatusOK, "Agent created successfully")
}

func (c *AgentController) UpdateAgentHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Agent ID is required")
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return eCtx.String(http.StatusBadRequest, "Invalid agent ID")
	}

	agent := &entities.Agent{
		ID:           oid,
		Name:         eCtx.FormValue("name"),
		Endpoint:     eCtx.FormValue("endpoint"),
		Model:        eCtx.FormValue("model"),
		APIKey:       eCtx.FormValue("api_key"),
		SystemPrompt: eCtx.FormValue("system_prompt"),
	}

	if tempStr := eCtx.FormValue("temperature"); tempStr != "" {
		if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
			agent.Temperature = &temp
		}
	} else {
		agent.Temperature = nil
	}

	if maxTokensStr := eCtx.FormValue("max_tokens"); maxTokensStr != "" {
		if maxTokens, err := strconv.Atoi(maxTokensStr); err == nil {
			agent.MaxTokens = &maxTokens
		}
	} else {
		agent.MaxTokens = nil
	}

	tools := eCtx.Request().Form["tools"]
	agent.Tools = make([]primitive.ObjectID, 0, len(tools))
	for _, toolID := range tools {
		if oid, err := primitive.ObjectIDFromHex(toolID); err == nil {
			agent.Tools = append(agent.Tools, oid)
		}
	}

	if err := c.agentService.UpdateAgent(context.Background(), agent); err != nil {
		c.logger.Error("Failed to update agent", zap.Error(err))
		return eCtx.String(http.StatusBadRequest, "Failed to update agent: "+err.Error())
	}

	return eCtx.String(http.StatusOK, "Agent updated successfully")
}
