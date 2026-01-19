package uicontrollers

import (
	"html/template"
	"net/http"

	"github.com/drujensen/aiagent/internal/domain/services"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ProviderController struct {
	logger              *zap.Logger
	tmpl                *template.Template
	providerService     services.ProviderService
	modelRefreshService services.ModelRefreshService
}

func NewProviderController(logger *zap.Logger, tmpl *template.Template, providerService services.ProviderService, modelRefreshService services.ModelRefreshService) *ProviderController {
	return &ProviderController{
		logger:              logger,
		tmpl:                tmpl,
		providerService:     providerService,
		modelRefreshService: modelRefreshService,
	}
}

func (c *ProviderController) RegisterRoutes(e *echo.Echo) {
	e.GET("/providers", c.ListProvidersHandler)
	e.POST("/providers/refresh", c.RefreshProvidersHandler)
	e.GET("/api/providers/:id", c.GetProviderHandler)
}

func (c *ProviderController) ListProvidersHandler(eCtx echo.Context) error {
	providers, err := c.providerService.ListProviders(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list providers", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load providers")
	}

	data := map[string]any{
		"Title":           "AI Agents - Providers",
		"ContentTemplate": "providers_list_content",
		"Providers":       providers,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ProviderController) GetProviderHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Provider ID or type is required"})
	}

	// First try to get by ID
	provider, err := c.providerService.GetProvider(eCtx.Request().Context(), id)
	if err == nil {
		return eCtx.JSON(http.StatusOK, provider)
	}

	// If not found by ID, try to find by type (for backward compatibility)
	providers, err := c.providerService.ListProviders(eCtx.Request().Context())
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to load providers")
	}

	for _, p := range providers {
		if string(p.Type) == id {
			return eCtx.JSON(http.StatusOK, p)
		}
	}

	return eCtx.String(http.StatusNotFound, "Provider not found")
}

func (c *ProviderController) RefreshProvidersHandler(eCtx echo.Context) error {
	c.logger.Info("Starting provider refresh from models.dev")

	if err := c.modelRefreshService.RefreshAllProviders(eCtx.Request().Context()); err != nil {
		c.logger.Error("Failed to refresh providers", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to refresh providers: "+err.Error())
	}

	c.logger.Info("Provider refresh completed successfully")
	eCtx.Response().Header().Set("HX-Trigger", `{"refreshProviders": true}`)
	return eCtx.String(http.StatusOK, "Providers refreshed successfully from models.dev!")
}
