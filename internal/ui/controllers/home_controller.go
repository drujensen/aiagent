package uicontrollers

import (
	"html/template"
	"net/http"

	"aiagent/internal/domain/services"

	"github.com/labstack/echo/v4"
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

func (c *HomeController) HomeHandler(eCtx echo.Context) error {
	conversations, err := c.chatService.ListActiveConversations(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list active conversations", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

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

	data := map[string]interface{}{
		"Title":           "AI Chat Application",
		"ContentTemplate": "home_content",
		"Conversations":   conversations,
		"Agents":          agents,
		"Tools":           tools,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}
