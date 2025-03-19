package uicontrollers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"aiagent/internal/domain/entities"
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
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	chat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return eCtx.String(http.StatusNotFound, "Chat not found")
		}
		c.logger.Error("Failed to get chat", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
	}

	agent, err := c.agentService.GetAgent(eCtx.Request().Context(), chat.AgentID.Hex())
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return eCtx.String(http.StatusNotFound, "Agent not found")
		}
		c.logger.Error("Failed to get agent", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load agent")
	}

	data := map[string]interface{}{
		"Title":           "Chat",
		"ContentTemplate": "chat_content",
		"ChatID":          chatID,
		"ChatName":        chat.Name,
		"AgentName":       agent.Name,
		"AgentRole":       agent.Role,
		"Messages":        chat.Messages,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ChatController) ChatFormHandler(eCtx echo.Context) error {
	agents, err := c.agentService.ListAgents(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load agents")
	}

	data := map[string]interface{}{
		"Title":           "New Chat",
		"ContentTemplate": "chat_form_content",
		"Agents":          agents,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ChatController) CreateChatHandler(eCtx echo.Context) error {
	agentID := eCtx.FormValue("agent-select")
	name := eCtx.FormValue("chat-name")
	if agentID == "" || name == "" {
		return eCtx.String(http.StatusBadRequest, "Agent and name are required")
	}

	chat, err := c.chatService.CreateChat(eCtx.Request().Context(), agentID, name)
	if err != nil {
		c.logger.Error("Failed to create chat", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to create chat")
	}

	eCtx.Response().Header().Set("HX-Redirect", "/chats/"+chat.ID.Hex())
	return eCtx.String(http.StatusOK, "Chat created successfully")
}

func (c *ChatController) UpdateChatHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	name := eCtx.FormValue("chat-name")
	if chatID == "" || name == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID and name are required")
	}

	_, err := c.chatService.UpdateChat(eCtx.Request().Context(), chatID, name)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return eCtx.String(http.StatusNotFound, "Chat not found")
		}
		c.logger.Error("Failed to update chat", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to update chat")
	}

	eCtx.Response().Header().Set("HX-Redirect", "/chats/"+chatID)
	return eCtx.String(http.StatusOK, "Chat updated successfully")
}

func (c *ChatController) SendMessageHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	messageContent := eCtx.FormValue("message")
	if messageContent == "" {
		return eCtx.String(http.StatusBadRequest, "Message content is required")
	}

	userMessage := entities.NewMessage("user", messageContent)

	// Send the message and get the AI responses
	aiMessage, err := c.chatService.SendMessage(eCtx.Request().Context(), chatID, *userMessage)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return eCtx.String(http.StatusNotFound, "Chat not found")
		}
		c.logger.Error("Failed to send message", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to send message: "+err.Error())
	}

	// Get the chat to find all messages since the user's message
	chat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		c.logger.Error("Failed to get chat after sending message", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to retrieve updated messages: "+err.Error())
	}

	// Find the index of the user's message
	userMessageIndex := -1
	for i, msg := range chat.Messages {
		if msg.ID == userMessage.ID {
			userMessageIndex = i
			break
		}
	}

	// Extract all responses after the user message (AI assistant + tool messages)
	var aiMessages []entities.Message
	if userMessageIndex >= 0 && userMessageIndex < len(chat.Messages)-1 {
		aiMessages = chat.Messages[userMessageIndex+1:]
	} else if aiMessage != nil {
		// Fallback to just the direct AI response if we can't find all messages
		aiMessages = []entities.Message{*aiMessage}
	}

	// Create data for the template
	messageSessionID := fmt.Sprintf("message-session-%d", len(chat.Messages)/2) // Rough estimate of session count

	data := map[string]interface{}{
		"UserMessage": userMessage,
		"AIMessages":  aiMessages,
		"SessionID":   messageSessionID,
	}

	// Render complete message session using the template
	var buf bytes.Buffer
	if err := c.tmpl.ExecuteTemplate(&buf, "message_session_partial", data); err != nil {
		c.logger.Error("Failed to render message session", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to render messages: "+err.Error())
	}

	// Create a placeholder for the next message session
	responseHTML := fmt.Sprintf(`<div id="message-session-%s" class="message-session">%s</div><div id="next-message-session"></div>`,
		messageSessionID, buf.String())

	// Set the header to trigger scrolling to the response
	eCtx.Response().Header().Set("HX-Trigger", "messageReceived")

	return eCtx.HTML(http.StatusOK, responseHTML)
}

func (c *ChatController) DeleteChatHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	err := c.chatService.DeleteChat(eCtx.Request().Context(), id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return eCtx.String(http.StatusNotFound, "Chat not found")
		}
		c.logger.Error("Failed to delete chat", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to delete chat")
	}

	// After successful deletion, return the updated chats list
	chats, err := c.chatService.ListActiveChats(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list active chats", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load chats")
	}
	data := map[string]interface{}{
		"Chats": chats,
	}
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "sidebar_chats", data)
}
