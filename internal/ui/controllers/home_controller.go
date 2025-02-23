package uicontrollers

import (
	"html/template"
	"net/http"

	"go.uber.org/zap"
)

type HomeController struct {
	logger *zap.Logger
	tmpl   *template.Template
}

func NewHomeController(logger *zap.Logger, tmpl *template.Template) *HomeController {
	return &HomeController{
		logger: logger,
		tmpl:   tmpl,
	}
}

func (c *HomeController) HomeHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":           "AI Chat Application",
		"ContentTemplate": "home_content",
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render template", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
