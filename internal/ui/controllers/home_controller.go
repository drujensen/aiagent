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

func (c *HomeController) RegisterRoutes(e *echo.Echo) {
	e.GET("/", c.HomeHandler)
	e.GET("/sidebar/chats", c.ChatsPartialHandler)
	e.GET("/sidebar/agents", c.AgentsPartialHandler)
}

func (c *HomeController) HomeHandler(eCtx echo.Context) error {
	// No longer fetch data here; let HTMX handle it
	data := map[string]interface{}{
		"Title":           "AI Chat Application",
		"ContentTemplate": "home_content",
	}
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *HomeController) ChatsPartialHandler(eCtx echo.Context) error {
	chats, err := c.chatService.ListChats(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list active chats", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load chats")
	}

	// Create a new slice to hold the processed chat data
	processedChats := make([]map[string]string, 0, len(chats))

	for _, chat := range chats {
		agent, err := c.agentService.GetAgent(eCtx.Request().Context(), chat.AgentID.Hex())
		if err != nil {
			c.logger.Error("Failed to get agent for chat", zap.String("chatID", chat.ID.Hex()), zap.Error(err))
			continue // Skip this chat if we can't find the agent
		}

		// Create a new map for each chat with the required fields
		chatData := map[string]string{
			"ID":        chat.ID.Hex(),
			"ChatName":  chat.Name,
			"AgentName": agent.Name,
			"AgentRole": agent.Role,
		}

		// Append the processed chat data to the slice
		processedChats = append(processedChats, chatData)
	}

	data := map[string]interface{}{
		"Chats": processedChats,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "sidebar_chats", data)
}

func (c *HomeController) AgentsPartialHandler(eCtx echo.Context) error {
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
