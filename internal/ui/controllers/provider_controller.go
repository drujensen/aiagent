package uicontrollers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/services"

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
	e.GET("/providers/new", c.ProviderFormHandler)
	e.POST("/providers", c.CreateProviderHandler)
	e.GET("/providers/:id/edit", c.ProviderFormHandler)
	e.PUT("/providers/:id", c.UpdateProviderHandler)
	e.DELETE("/providers/:id", c.DeleteProviderHandler)

	e.GET("/api/providers/:id", c.GetProviderHandler)
	e.GET("/api/debug/providers", c.DebugProvidersHandler)
	e.POST("/api/debug/providers/reset", c.ResetProvidersHandler)
}

func (c *ProviderController) ProviderFormHandler(eCtx echo.Context) error {
	var provider *entities.Provider
	var err error

	id := eCtx.Param("id")
	if id != "" {
		provider, err = c.providerService.GetProvider(eCtx.Request().Context(), id)
		if err != nil {
			switch err.(type) {
			case *errors.NotFoundError:
				return eCtx.String(http.StatusNotFound, "Provider not found")
			default:
				return eCtx.String(http.StatusInternalServerError, "Failed to load provider")
			}
		}
	} else {
		// Create an empty provider for new form
		provider = &entities.Provider{}
	}

	data := map[string]interface{}{
		"Title":           "Provider Configuration",
		"ContentTemplate": "provider_form_content",
		"Provider":        provider,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ProviderController) ListProvidersHandler(eCtx echo.Context) error {
	providers, err := c.providerService.ListProviders(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list providers", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load providers")
	}

	data := map[string]interface{}{
		"Title":           "Providers",
		"ContentTemplate": "providers_list_content",
		"Providers":       providers,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ProviderController) GetProviderModelsHandler(eCtx echo.Context) error {
	providerID := eCtx.QueryParam("provider_id")
	if providerID == "" {
		return eCtx.String(http.StatusBadRequest, "Provider ID is required")
	}

	provider, err := c.providerService.GetProvider(eCtx.Request().Context(), providerID)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Provider not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load provider")
		}
	}

	data := map[string]interface{}{
		"Models": provider.Models,
	}

	var buf bytes.Buffer
	if err := c.tmpl.ExecuteTemplate(&buf, "provider_models_partial", data); err != nil {
		c.logger.Error("Failed to render provider models", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to render provider models")
	}

	return eCtx.HTML(http.StatusOK, buf.String())
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

func (c *ProviderController) CreateProviderHandler(eCtx echo.Context) error {
	name := eCtx.FormValue("name")
	providerType := entities.ProviderType(eCtx.FormValue("type"))
	baseURL := eCtx.FormValue("base_url")
	apiKeyName := eCtx.FormValue("api_key_name")

	if name == "" || string(providerType) == "" || baseURL == "" || apiKeyName == "" {
		return eCtx.String(http.StatusBadRequest, "All fields are required")
	}

	// Process models information from form
	modelNames := eCtx.Request().Form["model_names[]"]
	inputPrices := eCtx.Request().Form["input_prices[]"]
	outputPrices := eCtx.Request().Form["output_prices[]"]
	contextWindows := eCtx.Request().Form["context_windows[]"]

	if len(modelNames) != len(inputPrices) || len(modelNames) != len(outputPrices) || len(modelNames) != len(contextWindows) {
		return eCtx.String(http.StatusBadRequest, "Inconsistent model data")
	}

	// Create models
	var models []entities.ModelPricing
	for i, name := range modelNames {
		if name == "" {
			continue
		}

		inputPrice, err := strconv.ParseFloat(inputPrices[i], 64)
		if err != nil {
			inputPrice = 0
		}

		outputPrice, err := strconv.ParseFloat(outputPrices[i], 64)
		if err != nil {
			outputPrice = 0
		}

		contextWindow, err := strconv.Atoi(contextWindows[i])
		if err != nil {
			contextWindow = 4096 // Default
		}

		models = append(models, entities.ModelPricing{
			Name:                name,
			InputPricePerMille:  inputPrice,
			OutputPricePerMille: outputPrice,
			ContextWindow:       contextWindow,
		})
	}

	_, err := c.providerService.CreateProvider(eCtx.Request().Context(), name, providerType, baseURL, apiKeyName, models)
	if err != nil {
		c.logger.Error("Failed to create provider", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to create provider: "+err.Error())
	}

	eCtx.Response().Header().Set("HX-Redirect", "/providers")
	return eCtx.String(http.StatusOK, "Provider created successfully")
}

func (c *ProviderController) UpdateProviderHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	name := eCtx.FormValue("name")
	providerType := entities.ProviderType(eCtx.FormValue("type"))
	baseURL := eCtx.FormValue("base_url")
	apiKeyName := eCtx.FormValue("api_key_name")

	if id == "" || name == "" || string(providerType) == "" || baseURL == "" || apiKeyName == "" {
		return eCtx.String(http.StatusBadRequest, "All fields are required")
	}

	// Process models information from form
	modelNames := eCtx.Request().Form["model_names[]"]
	inputPrices := eCtx.Request().Form["input_prices[]"]
	outputPrices := eCtx.Request().Form["output_prices[]"]
	contextWindows := eCtx.Request().Form["context_windows[]"]

	if len(modelNames) != len(inputPrices) || len(modelNames) != len(outputPrices) || len(modelNames) != len(contextWindows) {
		return eCtx.String(http.StatusBadRequest, "Inconsistent model data")
	}

	// Create models
	var models []entities.ModelPricing
	for i, name := range modelNames {
		if name == "" {
			continue
		}

		inputPrice, err := strconv.ParseFloat(inputPrices[i], 64)
		if err != nil {
			inputPrice = 0
		}

		outputPrice, err := strconv.ParseFloat(outputPrices[i], 64)
		if err != nil {
			outputPrice = 0
		}

		contextWindow, err := strconv.Atoi(contextWindows[i])
		if err != nil {
			contextWindow = 4096 // Default
		}

		models = append(models, entities.ModelPricing{
			Name:                name,
			InputPricePerMille:  inputPrice,
			OutputPricePerMille: outputPrice,
			ContextWindow:       contextWindow,
		})
	}

	_, err := c.providerService.UpdateProvider(eCtx.Request().Context(), id, name, providerType, baseURL, apiKeyName, models)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Provider not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load provider")
		}
	}

	eCtx.Response().Header().Set("HX-Redirect", "/providers")
	return eCtx.String(http.StatusOK, "Provider updated successfully")
}

func (c *ProviderController) DeleteProviderHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Provider ID is required")
	}

	err := c.providerService.DeleteProvider(eCtx.Request().Context(), id)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Provider not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load provider")
		}
	}

	// Return success
	return eCtx.String(http.StatusOK, "Provider deleted successfully")
}

// ResetProvidersHandler resets providers to default configuration
func (c *ProviderController) ResetProvidersHandler(eCtx echo.Context) error {
	c.logger.Info("Resetting providers to default configuration")

	err := c.providerService.ResetDefaultProviders(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to reset providers", zap.Error(err))
		return eCtx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to reset providers: " + err.Error(),
		})
	}

	providers, err := c.providerService.ListProviders(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list providers after reset", zap.Error(err))
	} else {
		c.logger.Info("Providers reset successfully", zap.Int("provider_count", len(providers)))
		for i, p := range providers {
			c.logger.Info("Provider details",
				zap.Int("index", i),
				zap.String("id", p.ID.Hex()),
				zap.String("name", p.Name),
				zap.Int("models_count", len(p.Models)))
		}
	}

	return eCtx.JSON(http.StatusOK, map[string]string{
		"message": "Providers reset successfully",
		"count":   fmt.Sprintf("%d", len(providers)),
	})
}

// DebugProvidersHandler returns JSON information about all providers for debugging
func (c *ProviderController) DebugProvidersHandler(eCtx echo.Context) error {
	providers, err := c.providerService.ListProviders(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list providers", zap.Error(err))
		return eCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list providers"})
	}

	// Create a simpler format for debugging
	debugInfo := []map[string]interface{}{}
	for _, p := range providers {
		providerInfo := map[string]interface{}{
			"id":           p.ID.Hex(),
			"name":         p.Name,
			"type":         string(p.Type),
			"baseURL":      p.BaseURL,
			"apiKeyName":   p.APIKeyName,
			"models_count": len(p.Models),
			"models":       []map[string]interface{}{},
		}

		for _, m := range p.Models {
			modelInfo := map[string]interface{}{
				"name":          m.Name,
				"inputPrice":    m.InputPricePerMille,
				"outputPrice":   m.OutputPricePerMille,
				"contextWindow": m.ContextWindow,
			}
			providerInfo["models"] = append(providerInfo["models"].([]map[string]interface{}), modelInfo)
		}

		debugInfo = append(debugInfo, providerInfo)
	}

	return eCtx.JSON(http.StatusOK, debugInfo)
}
