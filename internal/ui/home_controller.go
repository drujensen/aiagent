package ui

import (
	"html/template"
	"net/http"

	"aiagent/internal/domain/services"

	"go.uber.org/zap"
)

// HomeController manages the home page UI request.
type HomeController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	agentService services.AgentService
}

// NewHomeController creates a new HomeController instance.
func NewHomeController(logger *zap.Logger, tmpl *template.Template, agentService services.AgentService) *HomeController {
	return &HomeController{
		logger:       logger,
		tmpl:         tmpl,
		agentService: agentService,
	}
}

// HomeHandler handles requests to the root path ("/"), rendering the home page.
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
