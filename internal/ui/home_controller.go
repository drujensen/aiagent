/**
 * @description
 * This file defines the HomeController for the AI Workflow Automation Platform within the internal/ui package.
 * It handles rendering UI pages, including the home page, agent list, and agent form, with server-side HTML rendering
 * using Go's html/template package. It supports full page loads and HTMX-driven partial updates, now including
 * agent hierarchy data for the sidebar.
 *
 * Key features:
 * - Page Rendering: Renders home, agent list, and form pages with dynamic data.
 * - HTMX Support: Handles both full layout and partial template rendering.
 * - Hierarchy Integration: Fetches and builds agent hierarchy for sidebar display.
 *
 * @dependencies
 * - html/template: For server-side templating.
 * - net/http: For HTTP request handling.
 * - go.uber.org/zap: For structured logging.
 * - aiagent/internal/domain/entities: For AIAgent entity definitions.
 * - aiagent/internal/domain/services: For AgentService interface.
 *
 * @notes
 * - Template paths are relative to the project root (./internal/ui/templates/).
 * - Assumes layout.html includes header, sidebar, and content templates dynamically.
 * - Edge case: Handles missing agents or template errors with HTTP 500 and logs.
 * - Assumption: Static assets (htmx.min.js, styles.css) are served via main.go.
 */

package ui

import (
	"html/template"
	"net/http"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"

	"go.uber.org/zap"
)

// AgentNode represents a node in the agent hierarchy tree for UI rendering.
// It simplifies the AIAgent entity for template use, focusing on ID, Name, and Children.
type AgentNode struct {
	ID       string       // Unique identifier of the agent
	Name     string       // Display name of the agent
	Children []*AgentNode // Recursive list of child nodes
}

// HomeController manages UI-related requests for the web application.
type HomeController struct {
	logger       *zap.Logger           // Logger for error reporting
	tmpl         *template.Template    // Pre-parsed templates for rendering
	agentService services.AgentService // Service for fetching agent data
}

// NewHomeController creates a new HomeController instance with pre-parsed templates and agent service.
// It initializes the template set and ensures all dependencies are provided.
//
// Parameters:
// - logger: Zap logger for error logging.
// - agentService: Service interface for agent operations.
//
// Returns:
// - *HomeController: Initialized controller instance.
func NewHomeController(logger *zap.Logger, agentService services.AgentService) *HomeController {
	tmpl, err := template.ParseFiles(
		"./internal/ui/templates/layout.html",
		"./internal/ui/templates/header.html",
		"./internal/ui/templates/sidebar.html",
		"./internal/ui/templates/home.html",
		"./internal/ui/templates/agent_list.html",
		"./internal/ui/templates/agent_form.html",
	)
	if err != nil {
		logger.Fatal("Failed to parse templates", zap.Error(err))
	}
	return &HomeController{
		logger:       logger,
		tmpl:         tmpl,
		agentService: agentService,
	}
}

// buildHierarchy constructs a tree of AgentNodes from a flat list of AIAgents.
// It maps agents by ID and recursively builds parent-child relationships.
//
// Parameters:
// - agents: List of AIAgent entities from the service.
//
// Returns:
// - []*AgentNode: List of root nodes (agents with no parent).
func buildHierarchy(agents []*entities.AIAgent) []*AgentNode {
	agentMap := make(map[string]*entities.AIAgent)
	for _, agent := range agents {
		agentMap[agent.ID] = agent
	}
	rootAgents := []*AgentNode{}
	for _, agent := range agents {
		if agent.ParentID == "" {
			node := &AgentNode{ID: agent.ID, Name: agent.Name}
			rootAgents = append(rootAgents, node)
			buildChildren(node, agentMap)
		}
	}
	return rootAgents
}

// buildChildren recursively populates the Children field of an AgentNode.
// It looks up child agents in the map and builds their subtrees.
//
// Parameters:
// - node: Current node being processed.
// - agentMap: Map of agent IDs to AIAgent entities for quick lookup.
func buildChildren(node *AgentNode, agentMap map[string]*entities.AIAgent) {
	agent := agentMap[node.ID]
	for _, childID := range agent.ChildrenIDs {
		if childAgent, exists := agentMap[childID]; exists {
			childNode := &AgentNode{ID: childAgent.ID, Name: childAgent.Name}
			node.Children = append(node.Children, childNode)
			buildChildren(childNode, agentMap)
		}
	}
}

// HomeHandler handles requests to the root path ("/"), rendering the home page.
// It fetches the agent hierarchy and includes it in the template data.
//
// Parameters:
// - w: HTTP response writer for sending rendered HTML.
// - r: HTTP request with headers (e.g., HX-Request) and context.
func (c *HomeController) HomeHandler(w http.ResponseWriter, r *http.Request) {
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	rootAgents := buildHierarchy(agents)
	data := map[string]interface{}{
		"Title":           "AI Workflow Automation Platform",
		"ContentTemplate": "home_content",
		"RootAgents":      rootAgents,
	}
	isHtmxRequest := r.Header.Get("HX-Request") == "true"
	tmplName := "layout"
	if isHtmxRequest {
		tmplName = "home_content"
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, tmplName, data); err != nil {
		c.logger.Error("Failed to render template", zap.String("template", tmplName), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AgentListHandler handles requests to /agents, rendering the agent list page.
// It includes the hierarchy in the sidebar for consistency.
//
// Parameters:
// - w: HTTP response writer for sending rendered HTML.
// - r: HTTP request with context for fetching agents.
func (c *HomeController) AgentListHandler(w http.ResponseWriter, r *http.Request) {
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
// It includes the hierarchy in the sidebar.
//
// Parameters:
// - w: HTTP response writer for sending rendered HTML.
// - r: HTTP request with context for fetching agents.
func (c *HomeController) AgentFormHandler(w http.ResponseWriter, r *http.Request) {
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	rootAgents := buildHierarchy(agents)
	data := map[string]interface{}{
		"Title":           "Agent Form",
		"ContentTemplate": "agent_form_content",
		"Agent": struct {
			ID            string
			Name          string
			Prompt        string
			Configuration struct {
				APIKey, LocalURL string
				Temperature      float64
				ThinkingTime     int
			}
			HumanInteractionEnabled bool
		}{},
		"Tools": []struct {
			ID, Name string
			Selected bool
		}{{ID: "1", Name: "Tool1"}},
		"Providers": []struct {
			Value, Label string
			Selected     bool
		}{{Value: "openai", Label: "OpenAI"}},
		"RootAgents": rootAgents,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render agent_form", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
