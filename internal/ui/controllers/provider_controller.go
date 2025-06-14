package uicontrollers

import (
	"html/template"
	"net/http"

	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/services"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ProviderController struct {
	logger          *zap.Logger
	tmpl            *template.Template
	providerService services.ProviderService
}

func NewProviderController(logger *zap.Logger, tmpl *template.Template, providerService services.ProviderService) *ProviderController {
	return &ProviderController{
		logger:          logger,
		tmpl:            tmpl,
		providerService: providerService,
	}
}

func (c *ProviderController) RegisterRoutes(e *echo.Echo) {
	e.GET("/providers", c.ListProvidersHandler)
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
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Provider ID is required"})
	}

	provider, err := c.providerService.GetProvider(eCtx.Request().Context(), id)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Provider not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load provider")
		}
	}

	return eCtx.JSON(http.StatusOK, provider)
}
