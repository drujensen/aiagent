package uicontrollers

import (
	"html/template"
	"net/http"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"

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

func (c *AgentController) AgentListHandler(w http.ResponseWriter, r *http.Request) {
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Title":           "Agents",
		"ContentTemplate": "agent_list_content",
		"Agents":          agents,
		"RootAgents":      agents, // Simplified for chat app
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render agent_list", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (c *AgentController) AgentFormHandler(w http.ResponseWriter, r *http.Request) {
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tools, err := c.toolService.ListTools(r.Context())
	if err != nil {
		c.logger.Error("Failed to list tools", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var agent *entities.Agent
	path := r.URL.Path
	isEdit := strings.HasPrefix(path, "/agents/edit/")
	if isEdit {
		id := strings.TrimPrefix(path, "/agents/edit/")
		if id == "" {
			http.Error(w, "Agent ID is required for editing", http.StatusBadRequest)
			return
		}
		agent, err = c.agentService.GetAgent(r.Context(), id)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(w, "Agent not found", http.StatusNotFound)
			} else {
				c.logger.Error("Failed to get agent", zap.String("id", id), zap.Error(err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
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

	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render agent_form", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
