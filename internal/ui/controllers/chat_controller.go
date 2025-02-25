package uicontrollers

import (
	"bytes"
	"fmt"
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

func (c *ChatController) ConversationFormHandler(eCtx echo.Context) error {
	agents, err := c.agentService.ListAgents(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	data := map[string]interface{}{
		"Title":           "New Conversation",
		"ContentTemplate": "chat_form_content",
		"Agents":          agents,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ChatController) CreateConversationHandler(eCtx echo.Context) error {
	name := eCtx.FormValue("conversation-name")
	agentID := eCtx.FormValue("agent-select")
	if name == "" || agentID == "" {
		return eCtx.String(http.StatusBadRequest, "Conversation name and agent are required")
	}

	conversation, err := c.chatService.CreateConversation(eCtx.Request().Context(), agentID, name)
	if err != nil {
		c.logger.Error("Failed to create conversation", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to create conversation")
	}

	// Render messages template for the new conversation
	messagesData := map[string]interface{}{
		"Title":            "Chat",
		"ContentTemplate":  "chat_content",
		"ConversationID":   conversation.ID.Hex(),
		"ConversationName": conversation.Name,
		"Messages":         conversation.Messages,
	}
	var messagesBuf bytes.Buffer
	if err := c.tmpl.ExecuteTemplate(&messagesBuf, "messages", messagesData); err != nil {
		c.logger.Error("Failed to render messages template", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to render template")
	}

	// Render updated sidebar_conversations template
	conversations, err := c.chatService.ListActiveConversations(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list conversations", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to list conversations")
	}
	sidebarData := map[string]interface{}{
		"Conversations": conversations,
	}
	var sidebarBuf bytes.Buffer
	if err := c.tmpl.ExecuteTemplate(&sidebarBuf, "sidebar_conversations", sidebarData); err != nil {
		c.logger.Error("Failed to render sidebar template", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to render template")
	}

	// Combine the response: main content for #message-history and OOB for #sidebar-conversations
	response := fmt.Sprintf(`%s<div id="sidebar-conversations" hx-swap-oob="innerHTML">%s</div>`, messagesBuf.String(), sidebarBuf.String())

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return eCtx.String(http.StatusOK, response)
}

func (c *ChatController) ChatConversationHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
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
		"Title":            "Chat",
		"ContentTemplate":  "chat_content",
		"ConversationID":   id,
		"ConversationName": conversation.Name,
		"Messages":         conversation.Messages,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}
