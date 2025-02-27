package apicontrollers

import (
	"net/http"

	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"

	"github.com/labstack/echo/v4"
)

type ChatController struct {
	ChatService services.ChatService
	Config      *config.Config
}

func NewChatController(chatService services.ChatService, cfg *config.Config) *ChatController {
	return &ChatController{
		ChatService: chatService,
		Config:      cfg,
	}
}

func (c *ChatController) CreateChat(eCtx echo.Context) error {
	if eCtx.Request().Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		return eCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}
	var req struct {
		AgentID string `json:"agent_id"`
		Name    string `json:"name"`
	}
	if err := eCtx.Bind(&req); err != nil {
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	chat, err := c.ChatService.CreateChat(eCtx.Request().Context(), req.AgentID, req.Name)
	if err != nil {
		return eCtx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return eCtx.JSON(http.StatusCreated, map[string]string{"id": chat.ID.Hex()})
}
