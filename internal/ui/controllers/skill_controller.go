package uicontrollers

import (
	"html/template"
	"net/http"

	"github.com/drujensen/aiagent/internal/domain/services"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type SkillController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	skillService services.SkillService
}

func NewSkillController(logger *zap.Logger, tmpl *template.Template, skillService services.SkillService) *SkillController {
	return &SkillController{
		logger:       logger,
		tmpl:         tmpl,
		skillService: skillService,
	}
}

func (c *SkillController) RegisterRoutes(e *echo.Echo) {
	e.GET("/skills", c.ListSkillsHandler)
}

func (c *SkillController) ListSkillsHandler(eCtx echo.Context) error {
	skills, err := c.skillService.ListSkills(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list skills", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load skills")
	}

	data := map[string]any{
		"Title":           "AI Agents - Skills",
		"ContentTemplate": "skills_list_content",
		"Skills":          skills,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}
