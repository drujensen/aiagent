package uicontrollers

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
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
	agents, err := c.agentService.ListAgents(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

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

	providers, err := c.providerService.ListProviders(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list providers", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	var agent *entities.Agent
	var selectedProvider *entities.Provider
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

		if agent.ProviderID != "" {
			selectedProvider, err = c.providerService.GetProvider(eCtx.Request().Context(), agent.ProviderID)
			if err != nil {
				switch err.(type) {
				case *errors.NotFoundError:
					break
				default:
					return eCtx.String(http.StatusInternalServerError, "Failed to load agent")
				}
			}
		}
	}

	agentData := struct {
		ID              string
		Name            string
		ProviderID      string
		ProviderType    entities.ProviderType
		Endpoint        string
		Model           string
		APIKey          string
		SystemPrompt    string
		Temperature     *float64
		MaxTokens       *int
		ContextWindow   *int
		ReasoningEffort string
		Tools           []string
	}{
		Tools: []string{},
	}
	if agent != nil {
		agentData.ID = agent.ID
		agentData.Name = agent.Name
		agentData.ProviderID = agent.ProviderID
		agentData.ProviderType = agent.ProviderType
		agentData.Endpoint = agent.Endpoint
		agentData.Model = agent.Model
		agentData.APIKey = agent.APIKey
		agentData.SystemPrompt = agent.SystemPrompt
		agentData.Temperature = agent.Temperature
		agentData.MaxTokens = agent.MaxTokens
		agentData.ContextWindow = agent.ContextWindow
		agentData.ReasoningEffort = agent.ReasoningEffort
		for _, tool := range agent.Tools {
			agentData.Tools = append(agentData.Tools, tool)
		}
	} else {
		agentData.ID = uuid.New().String()
	}

	var selectedProviderModels []entities.ModelPricing
	if selectedProvider != nil {
		selectedProviderModels = selectedProvider.Models
	}

	data := map[string]interface{}{
		"Title":                  "AI Agents - Agent Form",
		"ContentTemplate":        "agent_form_content",
		"Agent":                  agentData,
		"Tools":                  toolNames,
		"Agents":                 agents,
		"Providers":              providers,
		"SelectedProvider":       selectedProvider,
		"SelectedProviderModels": selectedProviderModels,
		"IsEdit":                 isEdit,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *AgentController) CreateAgentHandler(eCtx echo.Context) error {
	agent := &entities.Agent{
		ID:              eCtx.FormValue("id"),
		Name:            eCtx.FormValue("name"),
		Endpoint:        eCtx.FormValue("endpoint"),
		Model:           eCtx.FormValue("model"),
		APIKey:          eCtx.FormValue("api_key"),
		SystemPrompt:    eCtx.FormValue("system_prompt"),
		ReasoningEffort: eCtx.FormValue("reasoning_effort"),
	}

	c.logger.Info("Creating agent",
		zap.String("id", agent.ID),
		zap.String("name", agent.Name),
		zap.String("endpoint", agent.Endpoint),
		zap.String("model", agent.Model),
		zap.String("api_key_length", fmt.Sprintf("%d chars", len(agent.APIKey))))

	providerId := eCtx.FormValue("provider_id")
	if providerId != "" {
		agent.ProviderID = providerId

		provider, err := c.providerService.GetProvider(eCtx.Request().Context(), providerId)
		if err == nil && provider != nil {
			agent.ProviderType = provider.Type
		} else {
			c.logger.Warn("Failed to get provider for setting provider type",
				zap.String("provider_id", providerId),
				zap.Error(err))
		}
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
	} else {
		agent.MaxTokens = nil
	}

	if contextWindowStr := eCtx.FormValue("context_window"); contextWindowStr != "" {
		if contextWindow, err := strconv.Atoi(contextWindowStr); err == nil {
			agent.ContextWindow = &contextWindow
		}
	}

	tools := eCtx.Request().Form["tools"]
	agent.Tools = make([]string, 0, len(tools))
	for _, tool := range tools {
		agent.Tools = append(agent.Tools, tool)
	}

	if agent.ProviderID != "" {
		provider, err := c.providerService.GetProvider(eCtx.Request().Context(), agent.ProviderID)
		if err != nil {
			c.logger.Error("Failed to get provider",
				zap.String("provider_id", agent.ProviderID),
				zap.Error(err))
			return eCtx.String(http.StatusBadRequest, "Failed to get provider: "+err.Error())
		} else {
			c.logger.Info("Provider verified",
				zap.String("provider_id", provider.ID), // ID is string
				zap.String("provider_name", provider.Name))
		}
	}

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

	agent := &entities.Agent{
		ID:              id,
		Name:            eCtx.FormValue("name"),
		Endpoint:        eCtx.FormValue("endpoint"),
		Model:           eCtx.FormValue("model"),
		APIKey:          eCtx.FormValue("api_key"),
		SystemPrompt:    eCtx.FormValue("system_prompt"),
		ReasoningEffort: eCtx.FormValue("reasoning_effort"),
	}

	providerId := eCtx.FormValue("provider_id")
	if providerId != "" {
		agent.ProviderID = providerId

		provider, err := c.providerService.GetProvider(eCtx.Request().Context(), providerId)
		if err == nil && provider != nil {
			agent.ProviderType = provider.Type
		} else {
			c.logger.Warn("Failed to get provider for setting provider type",
				zap.String("provider_id", providerId),
				zap.Error(err))
		}
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

	if contextWindowStr := eCtx.FormValue("context_window"); contextWindowStr != "" {
		if contextWindow, err := strconv.Atoi(contextWindowStr); err == nil {
			agent.ContextWindow = &contextWindow
		}
	} else {
		agent.ContextWindow = nil
	}

	tools := eCtx.Request().Form["tools"]
	agent.Tools = make([]string, 0, len(tools))
	for _, tool := range tools {
		agent.Tools = append(agent.Tools, tool)
	}

	if agent.ProviderID != "" {
		provider, err := c.providerService.GetProvider(eCtx.Request().Context(), agent.ProviderID)
		if err != nil {
			c.logger.Error("Failed to get provider",
				zap.String("provider_id", agent.ProviderID),
				zap.Error(err))
			return eCtx.String(http.StatusBadRequest, "Failed to get provider: "+err.Error())
		} else {
			c.logger.Info("Provider verified",
				zap.String("provider_id", provider.ID), // ID is string
				zap.String("provider_name", provider.Name))
		}
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
	data := map[string]interface{}{
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

	data := map[string]interface{}{
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
