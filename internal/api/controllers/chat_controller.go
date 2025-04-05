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

// ListChats godoc
// @Summary List all chats
// @Description Retrieves a list of all chats.
// @Tags chats
// @Produce json
// @Success 200 {array} entities.Chat "Successfully retrieved list of chats"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/chats [get]
func (c *ChatController) ListChats(ctx echo.Context) error {
	chats, err := c.chatService.ListChats(ctx.Request().Context())
	if err != nil {
		return c.handleError(ctx, err, http.StatusInternalServerError)
	}
	return ctx.JSON(http.StatusOK, chats)
}

// GetChat godoc
// @Summary Get a chat by ID
// @Description Retrieves a chat's information by its ID.
// @Tags chats
// @Produce json
// @Param id path string true "Chat ID"
// @Success 200 {object} entities.Chat "Successfully retrieved chat"
// @Failure 400 {object} map[string]interface{} "Invalid chat ID"
// @Failure 404 {object} map[string]interface{} "Chat not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/chats/{id} [get]
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
// @Summary Create a new chat
// @Description Creates a new chat with the provided information.
// @Tags chats
// @Accept json
// @Produce json
// @Param request body CreateChatRequest true "Chat information to create"
// @Success 201 {object} entities.Chat "Successfully created chat"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/chats [post]
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

// UpdateChat godoc
// @Summary Update an existing chat
// @Description Updates an existing chat with the provided information.
// @Tags chats
// @Accept json
// @Produce json
// @Param id path string true "Chat ID"
// @Param request body UpdateChatRequest true "Chat information to update"
// @Success 200 {object} entities.Chat "Successfully updated chat"
// @Failure 400 {object} map[string]interface{} "Invalid request body or chat ID"
// @Failure 404 {object} map[string]interface{} "Chat not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/chats/{id} [put]
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

// DeleteChat godoc
// @Summary Delete a chat
// @Description Deletes a chat by its ID.
// @Tags chats
// @Produce json
// @Param id path string true "Chat ID"
// @Success 204 "Successfully deleted chat"
// @Failure 400 {object} map[string]interface{} "Invalid chat ID"
// @Failure 404 {object} map[string]interface{} "Chat not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/chats/{id} [delete]
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

// SendMessage godoc
// @Summary Send a message to a chat
// @Description Sends a new message to a chat.
// @Tags chats
// @Accept json
// @Produce json
// @Param id path string true "Chat ID"
// @Param message body entities.Message true "Message to send"
// @Success 200 {object} entities.Message "Successfully sent message"
// @Failure 400 {object} map[string]interface{} "Invalid request body or chat ID"
// @Failure 404 {object} map[string]interface{} "Chat not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/chats/{id}/messages [post]
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

// CreateChatRequest represents the request body for creating a new chat.
type CreateChatRequest struct {
	AgentID string `json:"agent_id" example:"60d0ddb0f0a4a729c0a8e9b1"`
	Name    string `json:"name" example:"My Chat"`
}

// UpdateChatRequest represents the request body for updating a chat.
type UpdateChatRequest struct {
	AgentID string `json:"agent_id" example:"60d0ddb0f0a4a729c0a8e9b1"`
	Name    string `json:"name" example:"Updated Chat Name"`
}
