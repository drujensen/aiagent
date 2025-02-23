package uicontrollers

import (
	"html/template"
	"net/http"

	"aiagent/internal/domain/services"

	"go.uber.org/zap"
)

type HomeController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	chatService  services.ChatService
	agentService services.AgentService
	toolService  services.ToolService
}

func NewHomeController(logger *zap.Logger, tmpl *template.Template, chatService services.ChatService, agentService services.AgentService, toolService services.ToolService) *HomeController {
	return &HomeController{
		logger:       logger,
		tmpl:         tmpl,
		chatService:  chatService,
		agentService: agentService,
		toolService:  toolService,
	}
}

func (c *HomeController) HomeHandler(w http.ResponseWriter, r *http.Request) {
	conversations, err := c.chatService.ListActiveConversations(r.Context())
	if err != nil {
		c.logger.Error("Failed to list active conversations", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

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

	data := map[string]interface{}{
		"Title":           "AI Chat Application",
		"ContentTemplate": "home_content",
		"Conversations":   conversations,
		"Agents":          agents,
		"Tools":           tools,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
