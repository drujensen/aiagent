package uicontrollers

import (
	"html/template"
	"net/http"
	"strings"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/services"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ToolController struct {
	logger      *zap.Logger
	tmpl        *template.Template
	toolService services.ToolService
}

func NewToolController(logger *zap.Logger, tmpl *template.Template, toolService services.ToolService) *ToolController {
	return &ToolController{
		logger:      logger,
		tmpl:        tmpl,
		toolService: toolService,
	}
}

func (c *ToolController) RegisterRoutes(e *echo.Echo) {
	e.GET("/tools/new", c.ToolFormHandler)
	e.GET("/tools/:id/edit", c.ToolFormHandler)
}

func (c *ToolController) ToolFormHandler(eCtx echo.Context) error {
	tools, err := c.toolService.ListTools()
	if err != nil {
		c.logger.Error("Failed to list tools", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Internal server error")
	}

	toolNames := []string{}
	for _, tool := range tools {
		toolNames = append(toolNames, (*tool).Name())
	}

	var tool *entities.ToolData
	path := eCtx.Request().URL.Path
	isEdit := strings.HasSuffix(path, "/edit")
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
				return eCtx.String(http.StatusInternalServerError, "Failed to load agent")
			}
		}
	}

	toolData := struct {
		ID            string
		Name          string
		Description   string
		Configuration map[string]string
	}{}

	if tool != nil {
		toolData.ID = tool.ID.Hex()
		toolData.Name = tool.Name
		toolData.Description = tool.Description
		toolData.Configuration = tool.Configuration
	}

	data := map[string]interface{}{
		"Title":           "Tool Form",
		"ContentTemplate": "tool_form_content",
		"Tool":            toolData,
		"Tools":           tools,
	}

	eCtx.Response().Header().Set("Content-Type", "text/html")
	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}
