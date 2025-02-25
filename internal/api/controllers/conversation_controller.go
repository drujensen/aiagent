package apicontrollers

import (
	"net/http"

	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"

	"github.com/labstack/echo/v4"
)

type ConversationController struct {
	ChatService services.ChatService
	Config      *config.Config
}

func NewConversationController(chatService services.ChatService, cfg *config.Config) *ConversationController {
	return &ConversationController{
		ChatService: chatService,
		Config:      cfg,
	}
}

func (c *ConversationController) CreateConversation(eCtx echo.Context) error {
	if eCtx.Request().Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		return eCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}
	var req struct {
		AgentID string `json:"agent_id"`
		Name    string `json:"name"` // Include name field as updated
	}
	if err := eCtx.Bind(&req); err != nil {
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	conversation, err := c.ChatService.CreateConversation(eCtx.Request().Context(), req.AgentID, req.Name)
	if err != nil {
		return eCtx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return eCtx.JSON(http.StatusCreated, map[string]string{"id": conversation.ID.Hex()})
}
