package uicontrollers

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ChatController struct {
	logger          *zap.Logger
	tmpl            *template.Template
	chatService     services.ChatService
	agentService    services.AgentService
	activeCancelers sync.Map // Maps chatID to context cancelFunc
}

func NewChatController(logger *zap.Logger, tmpl *template.Template, chatService services.ChatService, agentService services.AgentService) *ChatController {
	return &ChatController{
		logger:          logger,
		tmpl:            tmpl,
		chatService:     chatService,
		agentService:    agentService,
		activeCancelers: sync.Map{},
	}
}

func (c *ChatController) RegisterRoutes(e *echo.Echo) {
	e.GET("/chats/new", c.ChatFormHandler)
	e.POST("/chats", c.CreateChatHandler)
	e.GET("/chats/:id", c.ChatHandler)
	e.GET("/chats/:id/edit", c.ChatFormHandler)
	e.PUT("/chats/:id", c.UpdateChatHandler)
	e.DELETE("/chats/:id", c.DeleteChatHandler)

	e.POST("/chats/:id/messages", c.SendMessageHandler)
	e.POST("/chats/:id/cancel", c.CancelMessageHandler)
	e.GET("/chat-cost", c.ChatCostHandler)
}

func (c *ChatController) ChatHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.Redirect(http.StatusFound, "/")
	}

	chat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.Redirect(http.StatusFound, "/")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
		}
	}

	agent, err := c.agentService.GetAgent(eCtx.Request().Context(), chat.AgentID)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Agent not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load agent")
		}
	}

	data := map[string]interface{}{
		"Title":           "AI Agents - Chat",
		"ContentTemplate": "chat_content",
		"ChatID":          chatID,
		"ChatName":        chat.Name,
		"AgentName":       agent.Name,
		"ChatCost":        chat.Usage.TotalCost,
		"TotalTokens":     chat.Usage.TotalTokens,
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

	var chat *entities.Chat
	path := eCtx.Request().URL.Path
	isEdit := strings.HasSuffix(path, "/edit")
	if isEdit {
		id := eCtx.Param("id")
		if id == "" {
			return eCtx.String(http.StatusBadRequest, "Chat ID is required for editing")
		}
		chat, err = c.chatService.GetChat(eCtx.Request().Context(), id)
		if err != nil {
			switch err.(type) {
			case *errors.NotFoundError:
				return eCtx.Redirect(http.StatusFound, "/")
			default:
				return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
			}
		}
	}

	chatData := struct {
		ID      string
		Name    string
		AgentID string
	}{}

	if chat != nil {
		chatData.ID = chat.ID
		chatData.Name = chat.Name
		chatData.AgentID = chat.AgentID
	} else {
		chatData.ID = uuid.New().String()
	}

	data := map[string]interface{}{
		"Title":           "AI Agents - New Chat",
		"ContentTemplate": "chat_form_content",
		"Chat":            chatData,
		"Agents":          agents,
		"IsEdit":          isEdit,
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

	eCtx.Response().Header().Set("HX-Redirect", "/chats/"+chat.ID)
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
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Chat not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
		}
	}

	eCtx.Response().Header().Set("HX-Redirect", "/chats/"+chatID)
	return eCtx.String(http.StatusOK, "Chat updated successfully")
}

func (c *ChatController) DeleteChatHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	err := c.chatService.DeleteChat(eCtx.Request().Context(), id)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Chat not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
		}
	}

	// After successful deletion, return the updated chats list
	eCtx.Response().Header().Set("HX-Trigger", `{"refreshChats": true}`)
	return eCtx.String(http.StatusOK, "Chat deleted successfully")
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

	// Create a cancellable context
	ctx, cancel := context.WithCancel(eCtx.Request().Context())

	// Store the cancellation function
	c.activeCancelers.Store(chatID, cancel)

	// Ensure cancellation when the operation completes (success or failure)
	defer func() {
		cancel()
		c.activeCancelers.Delete(chatID)
	}()

	// Send the message and get the AI responses
	aiMessage, err := c.chatService.SendMessage(ctx, chatID, *userMessage)
	if err != nil {
		switch err.(type) {
		case *errors.CanceledError:
			c.logger.Info("Message processing was canceled", zap.String("chatID", chatID))
			return eCtx.String(http.StatusRequestTimeout, "Request was canceled")
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Chat not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
		}
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

	// Set the header to trigger scrolling to the response and refresh chat cost
	eCtx.Response().Header().Set("HX-Trigger", "messageReceived, refreshChatCost")

	return eCtx.HTML(http.StatusOK, responseHTML)
}

// CancelMessageHandler cancels an ongoing message processing operation
// ChatCostHandler handles the request to update the token and cost display
func (c *ChatController) ChatCostHandler(eCtx echo.Context) error {
	chatID := eCtx.QueryParam("chat_id")
	if chatID == "" {
		return eCtx.HTML(http.StatusBadRequest, "<div class=\"chat-cost\">Tokens: 0 Cost: $0.00</div>")
	}

	chat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		c.logger.Error("Failed to get chat for cost update", zap.Error(err))
		return eCtx.HTML(http.StatusInternalServerError, "<div class=\"chat-cost\">Tokens: 0 Cost: $0.00</div>")
	}

	data := map[string]interface{}{
		"TotalTokens": chat.Usage.TotalTokens,
		"ChatCost":    chat.Usage.TotalCost,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "chat_cost_partial", data)
}

func (c *ChatController) CancelMessageHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	// Check if there's an active cancellation function for this chat
	cancelValue, exists := c.activeCancelers.Load(chatID)
	if !exists {
		c.logger.Warn("No active request to cancel for this chat", zap.String("chatID", chatID))
		return eCtx.String(http.StatusOK, "No active request to cancel")
	}

	// Execute the cancellation
	if cancelFunc, ok := cancelValue.(context.CancelFunc); ok {
		cancelFunc()
		c.logger.Info("Request canceled successfully", zap.String("chatID", chatID))
		return eCtx.String(http.StatusOK, "Request canceled")
	}

	c.logger.Error("Invalid cancellation function", zap.String("chatID", chatID))
	return eCtx.String(http.StatusInternalServerError, "Failed to cancel request")
}
