package uicontrollers

import (
	"html/template"
	"net/http"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/services"
	"aiagent/internal/impl/tools"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ToolController struct {
	logger      *zap.Logger
	tmpl        *template.Template
	toolService services.ToolService
	toolFactory *tools.ToolFactory
}

func NewToolController(logger *zap.Logger, tmpl *template.Template, toolService services.ToolService, toolFactory *tools.ToolFactory) *ToolController {
	return &ToolController{
		logger:      logger,
		tmpl:        tmpl,
		toolService: toolService,
		toolFactory: toolFactory,
	}
}

func (c *ToolController) RegisterRoutes(e *echo.Echo) {
	e.GET("/tools/new", c.ToolFormHandler)
	e.GET("/tools/:id/edit", c.ToolFormHandler)
	e.POST("/tools", c.CreateToolHandler)
	e.PUT("/tools/:id", c.UpdateToolHandler)
	e.GET("/tools/defaults", c.GetToolTypeDefaultsHandler)
	e.DELETE("/tools/:id", c.DeleteToolHandler)
}

func (c *ToolController) ToolFormHandler(eCtx echo.Context) error {
	factories, err := c.toolFactory.ListFactories()
	if err != nil {
		c.logger.Error("Failed to list tool factories", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	var tool *entities.ToolData
	isEdit := strings.HasSuffix(eCtx.Request().URL.Path, "/edit")
	if isEdit {
		id := eCtx.Param("id")
		if id == "" {
			return eCtx.String(http.StatusBadRequest, "Tool ID is required for editing")
		}
		tool, err = c.toolService.GetToolData(eCtx.Request().Context(), id)
		if err != nil {
			switch err.(type) {
			case *errors.NotFoundError:
				return eCtx.Redirect(http.StatusFound, "/")
			default:
				c.logger.Error("Failed to load tool", zap.Error(err))
				return eCtx.String(http.StatusInternalServerError, "Failed to load tool")
			}
		}
	}

	if tool == nil {
		tool = entities.NewToolData("", "", "", nil)
	}

	data := map[string]interface{}{
		"Title":           "AI Agents - Tool Form",
		"ContentTemplate": "tool_form_content",
		"Tool":            tool,
		"ToolTypes":       factories,
		"IsEdit":          isEdit,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ToolController) GetToolTypeDefaultsHandler(eCtx echo.Context) error {
	toolType := eCtx.QueryParam("tool_type")
	if toolType == "" {
		return eCtx.HTML(http.StatusOK, `
						<div class="form-group">
								<label for="name">Name:</label>
								<input type="text" id="name" name="name" class="form-control" value="" required>
						</div>
						<div class="form-group">
								<label for="description">Description:</label>
								<textarea id="description" name="description" class="form-control" required></textarea>
						</div>
						<div class="form-group" id="configuration-fields">
								<label>Configuration:</label>
								<p>Select a tool type to configure.</p>
								<small class="form-text">Use #{value}# for environment variables</small>
						</div>
				`)
	}

	factory, err := c.toolFactory.GetFactoryByName(toolType)
	if err != nil {
		c.logger.Error("Failed to get tool factory", zap.String("tool_type", toolType), zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Invalid tool type")
	}

	configFields := ""
	for _, key := range factory.ConfigKeys {
		configFields += `
						<div class="config-pair">
								<input type="text" name="config_key[]" class="form-control" value="` + key + `" placeholder="Key" readonly>
								<input type="text" name="config_value[]" class="form-control" value="" placeholder="Value for ` + key + `">
						</div>
				`
	}

	html := `
				<div class="form-group">
						<label for="name">Name:</label>
						<input type="text" id="name" name="name" class="form-control" value="` + factory.Name + `" required>
				</div>
				<div class="form-group">
						<label for="description">Description:</label>
						<textarea id="description" name="description" class="form-control" required>` + factory.Description + `</textarea>
				</div>
				<div class="form-group" id="configuration-fields">
						<label>Configuration:</label>
						` + configFields + `
						<small class="form-text">Use #{value}# for environment variables</small>
				</div>
		`
	return eCtx.HTML(http.StatusOK, html)
}

func (c *ToolController) CreateToolHandler(eCtx echo.Context) error {
	tool := &entities.ToolData{
		ID:            eCtx.FormValue("id"),
		ToolType:      eCtx.FormValue("tool_type"),
		Name:          eCtx.FormValue("name"),
		Description:   eCtx.FormValue("description"),
		Configuration: make(map[string]string),
	}

	c.logger.Debug("Creating tool", zap.String("name", tool.Name), zap.String("tool_type", tool.ToolType))

	// Parse dynamic configuration fields
	configKeys := eCtx.Request().Form["config_key[]"]
	configValues := eCtx.Request().Form["config_value[]"]
	if len(configKeys) != len(configValues) {
		return eCtx.String(http.StatusBadRequest, "Mismatch between configuration keys and values")
	}
	for i, key := range configKeys {
		tool.Configuration[key] = configValues[i]
	}

	if err := c.toolService.CreateToolData(eCtx.Request().Context(), tool); err != nil {
		c.logger.Error("Failed to create tool", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to create tool: "+err.Error())
	}

	c.logger.Info("Tool created successfully", zap.String("name", tool.Name))
	eCtx.Response().Header().Set("HX-Trigger", `{"refreshTools": true}`)
	return eCtx.HTML(http.StatusOK, "<div>Tool created successfully!</div>")
}

func (c *ToolController) UpdateToolHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		c.logger.Error("Tool ID is missing in update request")
		return eCtx.String(http.StatusBadRequest, "Tool ID is required")
	}

	tool := &entities.ToolData{
		ID:            id,
		ToolType:      eCtx.FormValue("tool_type"),
		Name:          eCtx.FormValue("name"),
		Description:   eCtx.FormValue("description"),
		Configuration: make(map[string]string),
	}

	// Parse dynamic configuration fields
	configKeys := eCtx.Request().Form["config_key[]"]
	configValues := eCtx.Request().Form["config_value[]"]
	if len(configKeys) != len(configValues) {
		return eCtx.String(http.StatusBadRequest, "Mismatch between configuration keys and values")
	}
	for i, key := range configKeys {
		tool.Configuration[key] = configValues[i]
	}

	if err := c.toolService.UpdateToolData(eCtx.Request().Context(), tool); err != nil {
		c.logger.Error("Failed to update tool", zap.String("id", id), zap.Error(err))
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Tool not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to update tool: "+err.Error())
		}
	}

	c.logger.Info("Tool updated successfully", zap.String("id", id))
	eCtx.Response().Header().Set("HX-Trigger", `{"refreshTools": true}`)
	return eCtx.HTML(http.StatusOK, "<div>Tool updated successfully!</div>")
}

func (c *ToolController) DeleteToolHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		c.logger.Error("Tool ID is missing in delete request")
		return eCtx.String(http.StatusBadRequest, "Tool ID is required")
	}

	if err := c.toolService.DeleteToolData(eCtx.Request().Context(), id); err != nil {
		c.logger.Error("Failed to delete tool", zap.String("id", id), zap.Error(err))
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Tool not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to delete tool: "+err.Error())
		}
	}

	c.logger.Info("Tool deleted successfully", zap.String("id", id))
	eCtx.Response().Header().Set("HX-Trigger", `{"refreshTools": true}`)
	return eCtx.HTML(http.StatusOK, "<div>Tool deleted successfully!</div>")
}
