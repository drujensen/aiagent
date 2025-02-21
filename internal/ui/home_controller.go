/**
 * @description
 * This file defines the HomeController for the AI Workflow Automation Platform within the internal/ui package.
 * It handles rendering the main home page, serving as the entry point for the web UI. The controller uses Go's
 * built-in html/template package for server-side HTML rendering, supporting both full page loads and HTMX-driven
 * partial rendering for dynamic updates.
 *
 * Key features:
 * - Home Page Rendering: Displays a welcoming message and navigation instructions.
 * - HTMX Support: Returns full HTML (layout + partials) for initial loads or just the home partial for HTMX requests.
 * - Template Integration: Leverages layout.html with header, sidebar, and home partials.
 *
 * @dependencies
 * - html/template: Standard Go package for templating.
 * - net/http: Standard Go package for HTTP handling.
 * - go.uber.org/zap: Structured logging for errors.
 *
 * @notes
 * - Template paths are relative to the project root (e.g., ./internal/ui/templates/).
 * - Assumes layout.html exists with {{template "header" .}}, {{template "sidebar" .}}, and {{template "home" .}}.
 * - Edge case: If templates fail to load or render, an HTTP 500 error is returned with a logged message.
 * - Assumption: Static assets (e.g., htmx.min.js, styles.css) are served via main.go.
 */

package ui

import (
	"html/template"
	"net/http"

	"go.uber.org/zap"
)

// HomeController manages UI-related requests for the home page.
type HomeController struct {
	logger *zap.Logger        // Logger for error reporting
	tmpl   *template.Template // Pre-parsed templates for rendering
}

// NewHomeController creates a new HomeController instance with pre-parsed templates.
//
// Parameters:
// - logger: A zap.Logger instance for logging errors.
//
// Returns:
// - *HomeController: A new instance of HomeController with loaded templates.
func NewHomeController(logger *zap.Logger) *HomeController {
	tmpl, err := template.ParseFiles(
		"./internal/ui/templates/layout.html",
		"./internal/ui/templates/header.html",
		"./internal/ui/templates/sidebar.html",
		"./internal/ui/templates/home.html",
		"./internal/ui/templates/agent_list.html",
		"./internal/ui/templates/agent_form.html",
	)
	if err != nil {
		logger.Fatal("Failed to parse templates", zap.Error(err))
	}
	return &HomeController{
		logger: logger,
		tmpl:   tmpl,
	}
}

// HomeHandler handles requests to the root path ("/"), rendering the home page.
//
// Parameters:
// - w: HTTP response writer for sending the rendered HTML.
// - r: HTTP request containing headers (e.g., HX-Request) and context.
//
// Behavior:
// - If HX-Request header is present, renders only the "home" template.
// - Otherwise, renders the full "layout" template with embedded partials.
// - Logs and returns errors if rendering fails.
func (c *HomeController) HomeHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "AI Workflow Automation Platform",
	}
	isHtmxRequest := r.Header.Get("HX-Request") == "true"
	tmplName := "layout"
	if isHtmxRequest {
		tmplName = "home"
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, tmplName, data); err != nil {
		c.logger.Error("Failed to render template", zap.String("template", tmplName), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (c *HomeController) AgentListHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Agents",
		"Agents": []struct{ Name, Prompt string }{
			{"Agent1", "Do task 1"},
			{"Agent2", "Do task 2"},
		},
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "agent_list", data); err != nil {
		c.logger.Error("Failed to render agent_list", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (c *HomeController) AgentFormHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Agent Form",
		"Agent": struct {
			ID            string
			Name          string
			Prompt        string
			Configuration struct {
				APIKey, LocalURL string
				Temperature      float64
				ThinkingTime     int
			}
			HumanInteractionEnabled bool
		}{},
		"Tools": []struct {
			ID, Name string
			Selected bool
		}{{ID: "1", Name: "Tool1"}},
		"Providers": []struct {
			Value, Label string
			Selected     bool
		}{{Value: "openai", Label: "OpenAI"}},
	}
	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "agent_form", data); err != nil {
		c.logger.Error("Failed to render agent_form", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
