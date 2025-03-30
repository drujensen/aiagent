package apicontrollers

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/services"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	e.GET("/chats/:id", c.GetChat)
	e.POST("/chats", c.CreateChat)
	e.PUT("/chats/:id", c.UpdateChat)
	e.DELETE("/chats/:id", c.DeleteChat)
	e.POST("/chats/:id/messages", c.SendMessage)
}

// ListChats handles GET requests to list all chats
func (c *ChatController) ListChats(ctx echo.Context) error {
	chats, err := c.chatService.ListChats(ctx.Request().Context())
	if err != nil {
		return c.handleError(ctx, err, http.StatusInternalServerError)
	}
	return ctx.JSON(http.StatusOK, chats)
}

// GetChat handles GET requests to retrieve a specific chat
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

// CreateChat handles POST requests to create a new chat
func (c *ChatController) CreateChat(ctx echo.Context) error {
	var input struct {
		AgentID primitive.ObjectID `json:"agent_id"`
		Name    string             `json:"name"`
	}
	if err := ctx.Bind(&input); err != nil {
		return c.handleError(ctx, "Invalid request body", http.StatusBadRequest)
	}

	chat, err := c.chatService.CreateChat(ctx.Request().Context(), input.AgentID.Hex(), input.Name)
	if err != nil {
		return c.handleError(ctx, err, http.StatusInternalServerError)
	}

	return ctx.JSON(http.StatusCreated, chat)
}

// UpdateChat handles PUT requests to update an existing chat
func (c *ChatController) UpdateChat(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return c.handleError(ctx, "Missing chat ID", http.StatusBadRequest)
	}

	var input struct {
		Name string `json:"name"`
	}
	if err := ctx.Bind(&input); err != nil {
		return c.handleError(ctx, "Invalid request body", http.StatusBadRequest)
	}

	chat, err := c.chatService.UpdateChat(ctx.Request().Context(), id, input.Name)
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

// DeleteChat handles DELETE requests to delete a specific chat
func (c *ChatController) DeleteChat(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return c.handleError(ctx, "Missing chat ID", http.StatusBadRequest)
	}

	if err := c.chatService.DeleteChat(ctx.Request().Context(), id); err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return c.handleError(ctx, "Chat not found", http.StatusNotFound)
		default:
			return c.handleError(ctx, err, http.StatusInternalServerError)
		}
	}

	return ctx.NoContent(http.StatusNoContent)
}

// SendMessage handles POST requests to send a new message to a chat
func (c *ChatController) SendMessage(ctx echo.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return c.handleError(ctx, "Missing chat ID", http.StatusBadRequest)
	}

	var message entities.Message
	if err := ctx.Bind(&message); err != nil {
		return c.handleError(ctx, "Invalid request body", http.StatusBadRequest)
	}

	newMessage, err := c.chatService.SendMessage(ctx.Request().Context(), id, message)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return c.handleError(ctx, "Chat not found", http.StatusNotFound)
		default:
			return c.handleError(ctx, err, http.StatusInternalServerError)
		}
	}

	return ctx.JSON(http.StatusOK, newMessage)
}

// handleError handles errors and returns them in a consistent format
func (c *ChatController) handleError(ctx echo.Context, err interface{}, statusCode int) error {
	c.logger.Error("Error occurred", zap.Any("error", err))
	return ctx.JSON(statusCode, map[string]interface{}{
		"error": err,
	})
}
