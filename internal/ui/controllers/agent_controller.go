package uicontrollers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/services"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"html/template"
)

type AgentController struct {
	logger          *zap.Logger
	tmpl            *template.Template
	agentService    services.AgentService
	toolService     services.ToolService
	providerService services.ProviderService
}

func NewAgentController(logger *zap.Logger, tmpl *template.Template, agentService services.AgentService, toolService services.ToolService, providerService services.ProviderService) *AgentController {
	return &AgentController{
		logger:          logger,
		tmpl:            tmpl,
		agentService:    agentService,
		toolService:     toolService,
		providerService: providerService,
	}
}

func (c *AgentController) RegisterRoutes(e *echo.Echo) {
	e.GET("/agents/new", c.AgentFormHandler)
	e.POST("/agents", c.CreateAgentHandler)
	e.GET("/agents/:id/edit", c.AgentFormHandler)
	e.PUT("/agents/:id", c.UpdateAgentHandler)
	e.DELETE("/agents/:id", c.DeleteAgentHandler)

	e.GET("/agents/provider-models", c.GetProviderModelsHandler)
}

func (c *AgentController) AgentFormHandler(eCtx echo.Context) error {
	tools, err := c.toolService.ListTools()
	if err != nil {
		c.logger.Error("Failed to list tools", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	toolNames := []string{}
	for _, tool := range tools {
		toolNames = append(toolNames, (*tool).Name())
	}
	sort.Strings(toolNames)

	var agent *entities.Agent
	path := eCtx.Request().URL.Path
	isEdit := strings.HasSuffix(path, "/edit")
	if isEdit {
		id := eCtx.Param("id")
		if id == "" {
			return eCtx.String(http.StatusBadRequest, "Agent ID is required for editing")
		}
		agent, err = c.agentService.GetAgent(eCtx.Request().Context(), id)
		if err != nil {
			switch err.(type) {
			case *errors.NotFoundError:
				return eCtx.Redirect(http.StatusFound, "/")
			default:
				return eCtx.String(http.StatusInternalServerError, "Failed to load agent")
			}
		}
	}

	agentData := struct {
		ID           string
		Name         string
		SystemPrompt string
		Tools        []string
	}{
		Tools: []string{},
	}
	if agent != nil {
		agentData.ID = agent.ID
		agentData.Name = agent.Name
		agentData.SystemPrompt = agent.SystemPrompt
		for _, tool := range agent.Tools {
			agentData.Tools = append(agentData.Tools, tool)
		}
	} else {
		agentData.ID = uuid.New().String()
	}

	data := map[string]any{
		"Title":           "AI Agents - Agent Form",
		"ContentTemplate": "agent_form_content",
		"Agent":           agentData,
		"Tools":           toolNames,
		"IsEdit":          isEdit,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *AgentController) CreateAgentHandler(eCtx echo.Context) error {
	name := eCtx.FormValue("name")
	systemPrompt := eCtx.FormValue("system_prompt")

	// Get available tools
	toolsList, err := c.toolService.ListTools()
	if err != nil {
		c.logger.Error("Failed to list tools", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	toolNames := []string{}
	for _, tool := range toolsList {
		toolNames = append(toolNames, (*tool).Name())
	}

	// Parse tools from form
	tools := []string{}
	for _, toolName := range toolNames {
		if eCtx.FormValue("tool_"+toolName) == "on" {
			tools = append(tools, toolName)
		}
	}

	agent := entities.NewAgent(name, systemPrompt, tools)

	if err := c.agentService.CreateAgent(context.Background(), agent); err != nil {
		c.logger.Error("Failed to create agent", zap.Error(err))
		return eCtx.String(http.StatusBadRequest, "Failed to create agent: "+err.Error())
	}

	eCtx.Response().Header().Set("HX-Trigger", `{"refreshAgents": true}`)
	return eCtx.String(http.StatusOK, "Agent created successfully")
}

func (c *AgentController) UpdateAgentHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Agent ID is required")
	}

	// Get existing agent
	existing, err := c.agentService.GetAgent(eCtx.Request().Context(), id)
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to get agent")
	}

	name := eCtx.FormValue("name")
	systemPrompt := eCtx.FormValue("system_prompt")

	// Get available tools
	toolsList, err := c.toolService.ListTools()
	if err != nil {
		c.logger.Error("Failed to list tools", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	toolNames := []string{}
	for _, tool := range toolsList {
		toolNames = append(toolNames, (*tool).Name())
	}

	// Parse tools from form
	tools := []string{}
	for _, toolName := range toolNames {
		if eCtx.FormValue("tool_"+toolName) == "on" {
			tools = append(tools, toolName)
		}
	}

	agent := &entities.Agent{
		ID:           id,
		Name:         name,
		SystemPrompt: systemPrompt,
		Tools:        tools,
		CreatedAt:    existing.CreatedAt,
		UpdatedAt:    existing.UpdatedAt,
	}

	if err := c.agentService.UpdateAgent(context.Background(), agent); err != nil {
		c.logger.Error("Failed to update agent", zap.Error(err))
		return eCtx.String(http.StatusBadRequest, "Failed to update agent: "+err.Error())
	}

	eCtx.Response().Header().Set("HX-Trigger", `{"refreshAgents": true}`)
	return eCtx.String(http.StatusOK, "Agent updated successfully")
}

func (c *AgentController) DeleteAgentHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Agent ID is required")
	}

	err := c.agentService.DeleteAgent(eCtx.Request().Context(), id)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Agent not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load agent")
		}
	}

	agents, err := c.agentService.ListAgents(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load agents")
	}
	data := map[string]any{
		"Agents": agents,
	}
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "sidebar_agents", data)
}

func (c *AgentController) GetProviderModelsHandler(eCtx echo.Context) error {
	providerID := eCtx.QueryParam("provider_id")
	if providerID == "" {
		return eCtx.String(http.StatusBadRequest, "Provider ID is required")
	}

	c.logger.Info("Fetching provider models", zap.String("provider_id", providerID))

	cleanProviderID := providerID
	if strings.Contains(cleanProviderID, "ObjectID") {
		start := strings.Index(cleanProviderID, "\"")
		end := strings.LastIndex(cleanProviderID, "\"")
		if start != -1 && end != -1 && end > start {
			cleanProviderID = cleanProviderID[start+1 : end]
			c.logger.Info("Cleaned provider ID",
				zap.String("original", providerID),
				zap.String("cleaned", cleanProviderID))
		}
	}

	provider, err := c.providerService.GetProvider(eCtx.Request().Context(), cleanProviderID)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Provider not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load provider")
		}
	}

	c.logger.Info("Provider found",
		zap.String("id", provider.ID), // ID is string
		zap.String("name", provider.Name),
		zap.String("type", string(provider.Type)),
		zap.Int("models_count", len(provider.Models)))

	for i, model := range provider.Models {
		c.logger.Info("Provider model",
			zap.Int("index", i),
			zap.String("name", model.Name))
	}

	data := map[string]any{
		"Models":   provider.Models,
		"Provider": provider,
	}

	var buf bytes.Buffer
	if err := c.tmpl.ExecuteTemplate(&buf, "provider_models_partial", data); err != nil {
		c.logger.Error("Failed to render provider models", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to render provider models")
	}

	eCtx.Response().Header().Set("X-Provider-Key-Name", provider.APIKeyName)
	eCtx.Response().Header().Set("X-Provider-Type", string(provider.Type))
	eCtx.Response().Header().Set("X-Provider-URL", provider.BaseURL)

	c.logger.Info("Returning provider models template",
		zap.String("html_length", fmt.Sprintf("%d bytes", buf.Len())),
		zap.String("provider_url", provider.BaseURL))
	return eCtx.HTML(http.StatusOK, buf.String())
}
