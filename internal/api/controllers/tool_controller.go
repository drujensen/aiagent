package controllers

import (
	"net/http"

	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"

	"github.com/labstack/echo/v4"
)

type ToolController struct {
	ToolService services.ToolService
	Config      *config.Config
}

func (c *ToolController) ListTools(eCtx echo.Context) error {
	if eCtx.Request().Method != http.MethodGet {
		return eCtx.String(http.StatusMethodNotAllowed, "Method not allowed")
	}

	if eCtx.Request().Header.Get("X-API-Key") != c.Config.LocalAPIKey {
		return eCtx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	tools, err := c.ToolService.ListTools(eCtx.Request().Context())
	if err != nil {
		return eCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list tools"})
	}

	return eCtx.JSON(http.StatusOK, tools)
}
