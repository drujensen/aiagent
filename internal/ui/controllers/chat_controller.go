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

func (c *ChatController) ChatFormHandler(eCtx echo.Context) error {
	agents, err := c.agentService.ListAgents(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	data := map[string]interface{}{
		"Title":           "New Chat",
		"ContentTemplate": "chat_form_content",
		"Agents":          agents,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ChatController) CreateChatHandler(eCtx echo.Context) error {
	name := eCtx.FormValue("chat-name")
	agentID := eCtx.FormValue("agent-select")
	if name == "" || agentID == "" {
		return eCtx.String(http.StatusBadRequest, "Chat name and agent are required")
	}

	chat, err := c.chatService.CreateChat(eCtx.Request().Context(), agentID, name)
	if err != nil {
		c.logger.Error("Failed to create chat", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to create chat")
	}

	// Render messages template for the new chat
	messagesData := map[string]interface{}{
		"Title":           "Chat",
		"ContentTemplate": "chat_form_content",
		"ChatID":          chat.ID.Hex(),
		"ChatName":        chat.Name,
		"Messages":        chat.Messages,
	}
	var messagesBuf bytes.Buffer
	if err := c.tmpl.ExecuteTemplate(&messagesBuf, "layout", messagesData); err != nil {
		c.logger.Error("Failed to render messages template", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to render template")
	}

	// Render updated sidebar_chats template
	chats, err := c.chatService.ListActiveChats(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list chats", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to list chats")
	}
	sidebarData := map[string]interface{}{
		"Chats": chats,
	}
	var sidebarBuf bytes.Buffer
	if err := c.tmpl.ExecuteTemplate(&sidebarBuf, "sidebar_chats", sidebarData); err != nil {
		c.logger.Error("Failed to render sidebar template", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to render template")
	}

	// Combine the response: main content for #message-history and OOB for #sidebar-chats
	response := fmt.Sprintf(`%s<div id="sidebar-chats" hx-swap-oob="innerHTML">%s</div>`, messagesBuf.String(), sidebarBuf.String())

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return eCtx.String(http.StatusOK, response)
}

func (c *ChatController) UpdateChatHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	name := eCtx.FormValue("chat-name")
	if name == "" {
		return eCtx.String(http.StatusBadRequest, "Chat name is required")
	}

	// Render updated sidebar_chats template after updating chat
	chats, err := c.chatService.ListActiveChats(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list chats", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to list chats")
	}

	sidebarData := map[string]interface{}{
		"Chats": chats,
	}
	var sidebarBuf bytes.Buffer
	if err := c.tmpl.ExecuteTemplate(&sidebarBuf, "sidebar_chats", sidebarData); err != nil {
		c.logger.Error("Failed to render sidebar template", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to render template")
	}

	// Return the updated sidebar HTML with OOB swap
	response := fmt.Sprintf(`<div id="sidebar-chats" hx-swap-oob="innerHTML">%s</div>`, sidebarBuf.String())

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return eCtx.String(http.StatusOK, response)
}

func (c *ChatController) ChatHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	chat, err := c.chatService.GetChat(eCtx.Request().Context(), id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return eCtx.String(http.StatusNotFound, "Chat not found")
		}
		c.logger.Error("Failed to get chat", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	data := map[string]interface{}{
		"Title":           "Chat",
		"ContentTemplate": "chat_content",
		"ChatID":          id,
		"ChatName":        chat.Name,
		"Messages":        chat.Messages,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}
