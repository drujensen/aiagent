package ui

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/repositories"
	"aiagent/internal/domain/services"

	"go.uber.org/zap"
)

// AgentController manages agent-related UI requests.
type AgentController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	agentService services.AgentService
	toolService  services.ToolService
}

// NewAgentController creates a new AgentController instance.
func NewAgentController(logger *zap.Logger, tmpl *template.Template, agentService services.AgentService, toolService services.ToolService) *AgentController {
	return &AgentController{
		logger:       logger,
		tmpl:         tmpl,
		agentService: agentService,
		toolService:  toolService,
	}
}

// AgentListHandler handles GET requests to /agents, rendering the agent list page.
func (c *AgentController) AgentListHandler(w http.ResponseWriter, r *http.Request) {
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	rootAgents := buildHierarchy(agents)
	data := map[string]interface{}{
		"Title":           "Agents",
		"ContentTemplate": "agent_list_content",
		"Agents":          agents,
		"RootAgents":      rootAgents,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render agent_list", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AgentFormHandler handles requests to /agents/new or /agents/edit/{id}, rendering the agent form.
func (c *AgentController) AgentFormHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch agents for hierarchy
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	rootAgents := buildHierarchy(agents)

	// Fetch available tools from ToolService
	tools, err := c.toolService.ListTools(r.Context())
	if err != nil {
		c.logger.Error("Failed to list tools", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Determine if editing an existing agent
	var agent *entities.AIAgent
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
			if err == repositories.ErrNotFound {
				http.Error(w, "Agent not found", http.StatusNotFound)
			} else {
				c.logger.Error("Failed to get agent", zap.String("id", id), zap.Error(err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}

	// Prepare agent data for template
	agentData := struct {
		ID                      string
		Name                    string
		Prompt                  string
		Tools                   []string
		Configuration           map[string]interface{}
		HumanInteractionEnabled bool
	}{
		Configuration: make(map[string]interface{}),
	}
	if agent != nil {
		agentData.ID = agent.ID
		agentData.Name = agent.Name
		agentData.Prompt = agent.Prompt
		agentData.Tools = agent.Tools
		agentData.Configuration = agent.Configuration
		agentData.HumanInteractionEnabled = agent.HumanInteractionEnabled
	}

	// Define supported AI providers
	providers := []struct {
		Value    string
		Label    string
		Selected bool
	}{
		{Value: "openai", Label: "OpenAI", Selected: agentData.Configuration["provider"] == "openai"},
		{Value: "together", Label: "Together.ai", Selected: agentData.Configuration["provider"] == "together"},
		{Value: "grok", Label: "Grok/xAI", Selected: agentData.Configuration["provider"] == "grok"},
	}

	// Template data
	data := map[string]interface{}{
		"Title":           "Agent Form",
		"ContentTemplate": "agent_form_content",
		"Agent":           agentData,
		"Tools":           tools,
		"Providers":       providers,
		"RootAgents":      rootAgents,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render agent_form", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AgentSubmitHandler handles POST requests to /agents (HTMX form submission).
func (c *AgentController) AgentSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data for HTMX
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div id="response-message">Error: Failed to parse form data</div>`)
		return
	}

	// Extract required fields
	name := r.FormValue("name")
	prompt := r.FormValue("prompt")
	tools := r.Form["tools"] // Multi-select returns a slice
	provider := r.FormValue("provider")

	// Validate required fields
	if name == "" || prompt == "" || provider == "" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div id="response-message">Error: Name, Prompt, and Provider are required</div>`)
		return
	}

	// Extract configuration fields
	config := make(map[string]interface{})
	config["provider"] = provider
	if apiKey := r.FormValue("api_key"); apiKey != "" {
		config["api_key"] = apiKey
	}
	if localURL := r.FormValue("local_url"); localURL != "" {
		config["local_url"] = localURL
	}
	if temp := r.FormValue("temperature"); temp != "" {
		config["temperature"] = temp
	}
	if thinkTime := r.FormValue("thinking_time"); thinkTime != "" {
		config["thinking_time"] = thinkTime
	}

	// Handle human interaction checkbox
	humanInteraction := r.FormValue("human_interaction") == "on"

	// Create agent entity
	agent := &entities.AIAgent{
		Name:                    name,
		Prompt:                  prompt,
		Tools:                   tools,
		Configuration:           config,
		HumanInteractionEnabled: humanInteraction,
	}

	// Create agent using AgentService
	if err := c.agentService.CreateAgent(r.Context(), agent); err != nil {
		c.logger.Error("Failed to create agent", zap.Error(err))
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `<div id="response-message">Error: %s</div>`, err.Error())
		return
	}

	// Return success message for HTMX
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `<div id="response-message">Agent "%s" created successfully. ID: %s</div>`, name, agent.ID)
}
