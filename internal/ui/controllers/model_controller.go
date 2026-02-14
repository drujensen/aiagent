package uicontrollers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"
	domainErrs "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/services"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"html/template"
)

type ModelController struct {
	logger          *zap.Logger
	tmpl            *template.Template
	modelService    services.ModelService
	providerService services.ProviderService
}

func NewModelController(logger *zap.Logger, tmpl *template.Template, modelService services.ModelService, providerService services.ProviderService) *ModelController {
	return &ModelController{
		logger:          logger,
		tmpl:            tmpl,
		modelService:    modelService,
		providerService: providerService,
	}
}

func (c *ModelController) RegisterRoutes(e *echo.Echo) {
	e.GET("/models/new", c.ModelFormHandler)
	e.POST("/models", c.CreateModelHandler)
	e.GET("/models/:id/edit", c.ModelFormHandler)
	e.PUT("/models/:id", c.UpdateModelHandler)
	e.DELETE("/models/:id", c.DeleteModelHandler)
}

func (c *ModelController) ModelFormHandler(eCtx echo.Context) error {
	var model *entities.Model
	var isEdit bool

	path := eCtx.Request().URL.Path
	if strings.HasSuffix(path, "/edit") {
		isEdit = true
		id := eCtx.Param("id")
		if id == "" {
			return eCtx.String(http.StatusBadRequest, "Model ID is required for editing")
		}
		var err error
		model, err = c.modelService.GetModel(eCtx.Request().Context(), id)
		if err != nil {
			switch err.(type) {
			case *domainErrs.NotFoundError:
				return eCtx.Redirect(http.StatusFound, "/models")
			default:
				return eCtx.String(http.StatusInternalServerError, "Failed to load model")
			}
		}
	}

	providers, err := c.providerService.ListProviders(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list providers", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load providers")
	}

	modelData := struct {
		ID              string
		Name            string
		ProviderID      string
		ProviderType    entities.ProviderType
		ModelName       string
		APIKey          string
		Temperature     *float64
		MaxTokens       *int
		ContextWindow   *int
		ReasoningEffort string
	}{
		APIKey: "#{API_KEY}#", // Default placeholder
	}

	if model != nil {
		modelData.ID = model.ID
		modelData.Name = model.Name
		modelData.ProviderID = model.ProviderID
		modelData.ProviderType = model.ProviderType
		modelData.ModelName = model.ModelName
		modelData.APIKey = model.APIKey
		modelData.Temperature = model.Temperature
		modelData.MaxTokens = model.MaxTokens
		modelData.ContextWindow = model.ContextWindow
		modelData.ReasoningEffort = model.ReasoningEffort
	} else {
		modelData.ID = uuid.New().String()
	}

	data := map[string]any{
		"Title":           "Model Form",
		"ContentTemplate": "model_form_content",
		"Model":           modelData,
		"IsEdit":          isEdit,
		"Providers":       providers,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ModelController) CreateModelHandler(eCtx echo.Context) error {
	name := eCtx.FormValue("name")
	providerTypeStr := eCtx.FormValue("provider_type")
	modelName := eCtx.FormValue("model_name")
	apiKey := eCtx.FormValue("api_key")

	if name == "" || providerTypeStr == "" || modelName == "" {
		return eCtx.String(http.StatusBadRequest, "Name, provider, and model name are required")
	}

	providerType := entities.ProviderType(providerTypeStr)
	providerID := providerTypeStr

	// Parse optional fields
	var temperature *float64
	if tempStr := eCtx.FormValue("temperature"); tempStr != "" {
		if temp, err := parseFloat(tempStr); err == nil {
			temperature = &temp
		}
	}

	var maxTokens *int
	if maxTokensStr := eCtx.FormValue("max_tokens"); maxTokensStr != "" {
		if mt, err := parseInt(maxTokensStr); err == nil {
			maxTokens = &mt
		}
	}

	var contextWindow *int
	if cwStr := eCtx.FormValue("context_window"); cwStr != "" {
		if cw, err := parseInt(cwStr); err == nil {
			contextWindow = &cw
		}
	}

	reasoningEffort := eCtx.FormValue("reasoning_effort")

	model := entities.NewModel(name, providerID, providerType, modelName, apiKey, temperature, maxTokens, contextWindow, reasoningEffort,
		"",    // Family (unknown for manually created models)
		false, // Reasoning
		true,  // ToolCall (assume true for manually created models)
		true,  // Temperature (assume true for manually created models)
		false, // Attachment
		false, // StructuredOutput
	)

	if err := c.modelService.CreateModel(eCtx.Request().Context(), model); err != nil {
		c.logger.Error("Failed to create model", zap.Error(err))
		return eCtx.String(http.StatusBadRequest, "Failed to create model: "+err.Error())
	}

	eCtx.Response().Header().Set("HX-Trigger", `{"refreshModels": true}`)
	return eCtx.String(http.StatusOK, "Model created successfully")
}

func (c *ModelController) UpdateModelHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Model ID is required")
	}

	// Get existing model
	existing, err := c.modelService.GetModel(eCtx.Request().Context(), id)
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to get model")
	}

	name := eCtx.FormValue("name")
	providerTypeStr := eCtx.FormValue("provider_type")
	modelName := eCtx.FormValue("model_name")
	apiKey := eCtx.FormValue("api_key")

	if name == "" || providerTypeStr == "" || modelName == "" {
		return eCtx.String(http.StatusBadRequest, "Name, provider, and model name are required")
	}

	providerType := entities.ProviderType(providerTypeStr)
	providerID := providerTypeStr

	// Parse optional fields
	var temperature *float64
	if tempStr := eCtx.FormValue("temperature"); tempStr != "" {
		if temp, err := parseFloat(tempStr); err == nil {
			temperature = &temp
		}
	}

	var maxTokens *int
	if maxTokensStr := eCtx.FormValue("max_tokens"); maxTokensStr != "" {
		if mt, err := parseInt(maxTokensStr); err == nil {
			maxTokens = &mt
		}
	}

	var contextWindow *int
	if cwStr := eCtx.FormValue("context_window"); cwStr != "" {
		if cw, err := parseInt(cwStr); err == nil {
			contextWindow = &cw
		}
	}

	reasoningEffort := eCtx.FormValue("reasoning_effort")

	model := &entities.Model{
		ID:               id,
		Name:             name,
		ProviderID:       providerID,
		ProviderType:     providerType,
		ModelName:        modelName,
		APIKey:           apiKey,
		Temperature:      temperature,
		MaxTokens:        maxTokens,
		ContextWindow:    contextWindow,
		ReasoningEffort:  reasoningEffort,
		Family:           existing.Family,
		Reasoning:        existing.Reasoning,
		ToolCall:         existing.ToolCall,
		TemperatureCap:   existing.TemperatureCap,
		Attachment:       existing.Attachment,
		StructuredOutput: existing.StructuredOutput,
		CreatedAt:        existing.CreatedAt,
		UpdatedAt:        existing.UpdatedAt,
	}

	if err := c.modelService.UpdateModel(eCtx.Request().Context(), model); err != nil {
		c.logger.Error("Failed to update model", zap.Error(err))
		return eCtx.String(http.StatusBadRequest, "Failed to update model: "+err.Error())
	}

	eCtx.Response().Header().Set("HX-Trigger", `{"refreshModels": true}`)
	return eCtx.String(http.StatusOK, "Model updated successfully")
}

func (c *ModelController) DeleteModelHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Model ID is required")
	}

	if err := c.modelService.DeleteModel(eCtx.Request().Context(), id); err != nil {
		c.logger.Error("Failed to delete model", zap.Error(err))
		return eCtx.String(http.StatusBadRequest, "Failed to delete model: "+err.Error())
	}

	eCtx.Response().Header().Set("HX-Trigger", `{"refreshModels": true}`)
	return eCtx.String(http.StatusOK, "Model deleted successfully")
}

// Helper functions for parsing
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
