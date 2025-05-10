package apicontrollers

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/services"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ChatController struct {
	logger      *zap.Logger
	chatService services.ChatService
}

func NewChatController(logger *zap.Logger, chatService services.ChatService) *ChatController {
	return &ChatController{
		logger:      logger,
		chatService: chatService,
	}
}

// RegisterRoutes registers all chat-related routes with Echo
func (c *ChatController) RegisterRoutes(e *echo.Group) {
	e.GET("/chats", c.ListChats)
	e.POST("/chats", c.CreateChat)
	e.GET("/chats/:id", c.GetChat)
	e.POST("/chats/:id/messages", c.SendMessage)
}

// ListChats godoc
//
//	@Summary		List all chats
//	@Description	Retrieves a list of all chats.
//	@Tags			chats
//	@Produce		json
//	@Success		200	{array}		entities.Chat			"Successfully retrieved list of chats"
//	@Failure		500	{object}	map[string]interface{}	"Internal server error"
//	@Router			/api/chats [get]
func (c *ChatController) ListChats(ctx echo.Context) error {
	chats, err := c.chatService.ListChats(ctx.Request().Context())
	if err != nil {
		return c.handleError(ctx, err, http.StatusInternalServerError)
	}
	return ctx.JSON(http.StatusOK, chats)
}

// GetChat godoc
//
//	@Summary		Get a chat by ID
//	@Description	Retrieves a chat's information by its ID.
//	@Tags			chats
//	@Produce		json
//	@Param			id	path		string					true	"Chat ID"
//	@Success		200	{object}	entities.Chat			"Successfully retrieved chat"
//	@Failure		400	{object}	map[string]interface{}	"Invalid chat ID"
//	@Failure		404	{object}	map[string]interface{}	"Chat not found"
//	@Failure		500	{object}	map[string]interface{}	"Internal server error"
//	@Router			/api/chats/{id} [get]
func (c *ChatController) GetChat(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return c.handleError(ctx, "Missing chat ID", http.StatusBadRequest)
	}

	chat, err := c.chatService.GetChat(ctx.Request().Context(), id)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return c.handleError(ctx, "Chat not found", http.StatusNotFound)
		default:
			return c.handleError(ctx, err, http.StatusInternalServerError)
		}
	}

	return ctx.JSON(http.StatusOK, chat)
}

// CreateChat godoc
//
//	@Summary		Create a new chat
//	@Description	Creates a new chat with the provided information.
//	@Tags			chats
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateChatRequest		true	"Chat information to create"
//	@Success		201		{object}	entities.Chat			"Successfully created chat"
//	@Failure		400		{object}	map[string]interface{}	"Invalid request body"
//	@Failure		500		{object}	map[string]interface{}	"Internal server error"
//	@Router			/api/chats [post]
func (c *ChatController) CreateChat(ctx echo.Context) error {
	var input struct {
		AgentID string `json:"agent_id"`
		Name    string `json:"name"`
	}
	if err := ctx.Bind(&input); err != nil {
		return c.handleError(ctx, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
	}

	chat, err := c.chatService.CreateChat(ctx.Request().Context(), input.AgentID, input.Name)
	if err != nil {
		return c.handleError(ctx, err, http.StatusInternalServerError)
	}

	return ctx.JSON(http.StatusCreated, chat)
}

// SendMessage godoc
//
//	@Summary		Send a message to a chat
//	@Description	Sends a new message to a chat.
//	@Tags			chats
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Chat ID"
//	@Param			request	body		SendMessageRequest		true	"Message to send"
//	@Success		200		{object}	SendMessageResponse		"Successfully sent message"
//	@Failure		400		{object}	map[string]interface{}	"Invalid request body or chat ID"
//	@Failure		404		{object}	map[string]interface{}	"Chat not found"
//	@Failure		500		{object}	map[string]interface{}	"Internal server error"
//	@Router			/api/chats/{id}/messages [post]
func (c *ChatController) SendMessage(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return c.handleError(ctx, "Missing chat ID", http.StatusBadRequest)
	}

	var messageRequest SendMessageRequest
	if err := ctx.Bind(&messageRequest); err != nil {
		return c.handleError(ctx, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
	}

	var message = entities.NewMessage(messageRequest.Role, messageRequest.Message)

	newMessage, err := c.chatService.SendMessage(ctx.Request().Context(), id, message)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return c.handleError(ctx, "Chat not found", http.StatusNotFound)
		default:
			return c.handleError(ctx, err, http.StatusInternalServerError)
		}
	}

	var response = SendMessageResponse{
		Role:    newMessage.Role,
		Message: newMessage.Content,
	}

	return ctx.JSON(http.StatusOK, response)
}

// handleError handles errors and returns them in a consistent format
func (c *ChatController) handleError(ctx echo.Context, err interface{}, statusCode int) error {
	var errorMessage string
	switch e := err.(type) {
	case error:
		errorMessage = e.Error()
	case string:
		errorMessage = e
	default:
		errorMessage = fmt.Sprintf("%v", e)
	}
	c.logger.Error("Error occurred", zap.String("error", errorMessage))
	return ctx.JSON(statusCode, map[string]interface{}{
		"error": errorMessage,
	})
}

// CreateChatRequest represents the request body for creating a new chat.
type CreateChatRequest struct {
	AgentID string `json:"agent_id" example:"60d0ddb0f0a4a729c0a8e9b1"`
	Name    string `json:"name" example:"My Chat"`
}

// SendMessageRequest represents the request body for sending a message.
type SendMessageRequest struct {
	Role    string `json:"role" example:"user"`
	Message string `json:"message" example:"Hello, how are you?"`
}

// SendMessageResponse represents the request body for sending a message.
type SendMessageResponse struct {
	Role    string `json:"role" example:"assistant"`
	Message string `json:"message" example:"I'm fine, thank you!"`
}
