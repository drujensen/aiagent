package uicontrollers

import (
	"html/template"
	"net/http"

	"aiagent/internal/domain/services"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type ChatController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	chatService  services.ChatService
	agentService services.AgentService
}

func NewChatController(logger *zap.Logger, tmpl *template.Template, chatService services.ChatService, agentService services.AgentService) *ChatController {
	return &ChatController{
		logger:       logger,
		tmpl:         tmpl,
		chatService:  chatService,
		agentService: agentService,
	}
}

func (c *ChatController) ChatHandler(eCtx echo.Context) error {
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

	data := map[string]interface{}{
		"Title":           "Chat",
		"ContentTemplate": "chat_content",
		"Conversations":   conversations,
		"RootAgents":      agents,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ChatController) ChatConversationHandler(eCtx echo.Context) error {
	id := eCtx.Param("id") // Using Echo's param instead of manual path parsing
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Conversation ID is required")
	}

	conversation, err := c.chatService.GetConversation(eCtx.Request().Context(), id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return eCtx.String(http.StatusNotFound, "Conversation not found")
		}
		c.logger.Error("Failed to get conversation", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	data := map[string]interface{}{
		"ConversationID": id,
		"Messages":       conversation.Messages,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "messages", data)
}
