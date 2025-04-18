package uicontrollers

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	e.POST("/agents/repair-providers", c.RepairAgentProvidersHandler)
}

// RepairAgentProvidersHandler fixes agent provider IDs that are no longer valid
func (c *AgentController) RepairAgentProvidersHandler(eCtx echo.Context) error {
	agents, err := c.agentService.ListAgents(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to list agents")
	}

	repairedCount := 0
	agentIssues := []map[string]string{}

	for _, agent := range agents {
		if agent.ProviderID.IsZero() {
			c.logger.Warn("Agent has no provider ID", zap.String("agent_id", agent.ID.Hex()), zap.String("agent_name", agent.Name))
			agentIssues = append(agentIssues, map[string]string{
				"agent_id": agent.ID.Hex(),
				"name":     agent.Name,
				"issue":    "No provider ID",
			})
			continue
		}

		_, err := c.providerService.GetProvider(eCtx.Request().Context(), agent.ProviderID.Hex())
		if err != nil {
			c.logger.Warn("Agent has invalid provider ID",
				zap.String("agent_id", agent.ID.Hex()),
				zap.String("agent_name", agent.Name),
				zap.String("provider_id", agent.ProviderID.Hex()),
				zap.Error(err))

			agentIssue := map[string]string{
				"agent_id": agent.ID.Hex(),
				"name":     agent.Name,
				"issue":    "Invalid provider ID: " + agent.ProviderID.Hex(),
			}

			if agent.ProviderType != "" {
				providerByType, typeErr := c.providerService.GetProviderByType(eCtx.Request().Context(), agent.ProviderType)
				if typeErr == nil && providerByType != nil {
					c.logger.Info("Repairing agent provider ID",
						zap.String("agent_id", agent.ID.Hex()),
						zap.String("agent_name", agent.Name),
						zap.String("old_provider_id", agent.ProviderID.Hex()),
						zap.String("new_provider_id", providerByType.ID)) // ID is string, no .Hex()

					// Convert providerByType.ID (string UUID) to ObjectID
					newProviderID, err := uuidToObjectID(providerByType.ID)
					if err != nil {
						c.logger.Error("Failed to convert provider UUID to ObjectID",
							zap.String("provider_id", providerByType.ID),
							zap.Error(err))
						agentIssue["repair_status"] = "Failed: Invalid UUID format"
					} else {
						agent.ProviderID = newProviderID
						if updateErr := c.agentService.UpdateAgent(eCtx.Request().Context(), agent); updateErr != nil {
							c.logger.Error("Failed to update agent",
								zap.String("agent_id", agent.ID.Hex()),
								zap.Error(updateErr))
							agentIssue["repair_status"] = "Failed: " + updateErr.Error()
						} else {
							repairedCount++
							agentIssue["repair_status"] = "Fixed - Updated provider ID to " + newProviderID.Hex()
						}
					}
				} else {
					c.logger.Error("Failed to find provider by type",
						zap.String("agent_id", agent.ID.Hex()),
						zap.String("agent_name", agent.Name),
						zap.String("provider_type", string(agent.ProviderType)),
						zap.Error(typeErr))
					agentIssue["repair_status"] = "Failed - No provider found for type: " + string(agent.ProviderType)
				}
			} else {
				agentIssue["repair_status"] = "Failed - No provider type specified"
			}

			agentIssues = append(agentIssues, agentIssue)
		}
	}

	result := map[string]interface{}{
		"total_agents":   len(agents),
		"repaired_count": repairedCount,
		"agent_issues":   agentIssues,
	}

	return eCtx.JSON(http.StatusOK, result)
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

		if !agent.ProviderID.IsZero() {
			selectedProvider, err = c.providerService.GetProvider(eCtx.Request().Context(), agent.ProviderID.Hex())
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
		ID            string
		Name          string
		ProviderID    primitive.ObjectID
		ProviderType  entities.ProviderType
		Endpoint      string
		Model         string
		APIKey        string
		SystemPrompt  string
		Temperature   *float64
		MaxTokens     *int
		ContextWindow *int
		Tools         []string
	}{
		Tools: []string{},
	}
	if agent != nil {
		agentData.ID = agent.ID.Hex()
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
		for _, tool := range agent.Tools {
			agentData.Tools = append(agentData.Tools, tool)
		}
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

	c.logger.Info("Creating agent",
		zap.String("name", agent.Name),
		zap.String("endpoint", agent.Endpoint),
		zap.String("model", agent.Model),
		zap.String("api_key_length", fmt.Sprintf("%d chars", len(agent.APIKey))))

	providerId := eCtx.FormValue("provider_id")
	if providerId != "" {
		cleanProviderId := providerId
		if strings.Contains(cleanProviderId, "ObjectID") {
			start := strings.Index(cleanProviderId, "\"")
			end := strings.LastIndex(cleanProviderId, "\"")
			if start != -1 && end != -1 && end > start {
				cleanProviderId = cleanProviderId[start+1 : end]
				c.logger.Info("Cleaned provider ID",
					zap.String("original", providerId),
					zap.String("cleaned", cleanProviderId))
			}
		}

		oid, err := uuidToObjectID(cleanProviderId)
		if err != nil {
			c.logger.Error("Invalid provider ID",
				zap.String("provider_id", cleanProviderId),
				zap.Error(err))
			return eCtx.String(http.StatusBadRequest, "Invalid provider ID: "+err.Error())
		}
		agent.ProviderID = oid

		provider, err := c.providerService.GetProvider(eCtx.Request().Context(), cleanProviderId)
		if err == nil && provider != nil {
			agent.ProviderType = provider.Type
		} else {
			c.logger.Warn("Failed to get provider for setting provider type",
				zap.String("provider_id", cleanProviderId),
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

	if !agent.ProviderID.IsZero() {
		provider, err := c.providerService.GetProvider(eCtx.Request().Context(), agent.ProviderID.Hex())
		if err != nil {
			c.logger.Warn("Specified provider not found, attempting to find by type",
				zap.String("provider_id", agent.ProviderID.Hex()),
				zap.String("provider_type", string(agent.ProviderType)),
				zap.Error(err))

			if agent.ProviderType != "" {
				providerByType, typeErr := c.providerService.GetProviderByType(eCtx.Request().Context(), agent.ProviderType)
				if typeErr == nil && providerByType != nil {
					c.logger.Info("Found provider by type, updating agent's provider ID",
						zap.String("old_provider_id", agent.ProviderID.Hex()),
						zap.String("new_provider_id", providerByType.ID), // ID is string
						zap.String("provider_name", providerByType.Name))
					agent.ProviderID, _ = uuidToObjectID(providerByType.ID)
				} else {
					c.logger.Error("Failed to find provider by type",
						zap.String("provider_type", string(agent.ProviderType)),
						zap.Error(typeErr))
				}
			}
		} else {
			c.logger.Info("Provider verified",
				zap.String("provider_id", provider.ID), // ID is string
				zap.String("provider_name", provider.Name),
				zap.String("provider_type", string(provider.Type)))
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

	providerId := eCtx.FormValue("provider_id")
	if providerId != "" {
		cleanProviderId := providerId
		if strings.Contains(cleanProviderId, "ObjectID") {
			start := strings.Index(cleanProviderId, "\"")
			end := strings.LastIndex(cleanProviderId, "\"")
			if start != -1 && end != -1 && end > start {
				cleanProviderId = cleanProviderId[start+1 : end]
				c.logger.Info("Cleaned provider ID",
					zap.String("original", providerId),
					zap.String("cleaned", cleanProviderId))
			}
		}

		providerOid, err := uuidToObjectID(cleanProviderId)
		if err != nil {
			c.logger.Error("Invalid provider ID",
				zap.String("provider_id", cleanProviderId),
				zap.Error(err))
			return eCtx.String(http.StatusBadRequest, "Invalid provider ID: "+err.Error())
		}
		agent.ProviderID = providerOid

		provider, err := c.providerService.GetProvider(eCtx.Request().Context(), cleanProviderId)
		if err == nil && provider != nil {
			agent.ProviderType = provider.Type
		} else {
			c.logger.Warn("Failed to get provider for setting provider type",
				zap.String("provider_id", cleanProviderId),
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

	if !agent.ProviderID.IsZero() {
		provider, err := c.providerService.GetProvider(eCtx.Request().Context(), agent.ProviderID.Hex())
		if err != nil {
			c.logger.Warn("Specified provider not found, attempting to find by type",
				zap.String("provider_id", agent.ProviderID.Hex()),
				zap.String("provider_type", string(agent.ProviderType)),
				zap.Error(err))

			if agent.ProviderType != "" {
				providerByType, typeErr := c.providerService.GetProviderByType(eCtx.Request().Context(), agent.ProviderType)
				if typeErr == nil && providerByType != nil {
					c.logger.Info("Found provider by type, updating agent's provider ID",
						zap.String("old_provider_id", agent.ProviderID.Hex()),
						zap.String("new_provider_id", providerByType.ID), // ID is string
						zap.String("provider_name", providerByType.Name))
					agent.ProviderID, _ = uuidToObjectID(providerByType.ID)
				} else {
					c.logger.Error("Failed to find provider by type",
						zap.String("provider_type", string(agent.ProviderType)),
						zap.Error(typeErr))
				}
			}
		} else {
			c.logger.Info("Provider verified",
				zap.String("provider_id", provider.ID), // ID is string
				zap.String("provider_name", provider.Name),
				zap.String("provider_type", string(provider.Type)))
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

// uuidToObjectID converts a UUID string to a MongoDB ObjectID (12 bytes)
func uuidToObjectID(uuidStr string) (primitive.ObjectID, error) {
	u, err := uuid.Parse(uuidStr)
	if err != nil {
		return primitive.ObjectID{}, err
	}
	// Use the first 12 bytes of the UUID (UUID is 16 bytes, ObjectID is 12)
	bytes := u[0:12]
	return primitive.ObjectIDFromHex(fmt.Sprintf("%x", bytes))
}
