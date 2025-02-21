/**
 * @description
 * This file defines the HomeController for the AI Workflow Automation Platform within the internal/ui package.
 * It handles rendering the main home page, serving as the entry point for the web UI. The controller uses Mustache
 * templating via the raymond library to render server-side HTML, supporting both full page loads and HTMX-driven
 * partial rendering for dynamic updates.
 *
 * Key features:
 * - Home Page Rendering: Displays a welcoming message and navigation instructions.
 * - HTMX Support: Returns full HTML (layout + content) for initial loads or just the content partial for HTMX requests.
 * - Template Integration: Leverages layout.html with header.html, sidebar.html, and home.html partials.
 *
 * @dependencies
 * - github.com/aymerick/raymond: Mustache templating library for server-side rendering.
 * - net/http: Standard Go package for HTTP handling.
 * - os: For manual file reading of partials.
 * - go.uber.org/zap: Structured logging for errors.
 *
 * @notes
 * - Template paths are relative to the project root with ./ prefix (e.g., ./internal/ui/templates/).
 * - Assumes layout.html exists with {{> header}}, {{> sidebar}}, and {{> content}} partials.
 * - Edge case: If templates fail to load or render, an HTTP 500 error is returned with a logged message.
 * - Assumption: Static assets (e.g., htmx.min.js, styles.css) are served via main.go.
 */

package ui

import (
	"net/http"
	"os"

	"github.com/aymerick/raymond"
	"go.uber.org/zap"
)

// HomeController manages UI-related requests for the home page.
type HomeController struct {
	logger *zap.Logger // Logger for error reporting
}

// NewHomeController creates a new HomeController instance with the provided logger.
//
// Parameters:
// - logger: A zap.Logger instance for logging errors and events.
//
// Returns:
// - *HomeController: A new instance of HomeController.
func NewHomeController(logger *zap.Logger) *HomeController {
	return &HomeController{
		logger: logger,
	}
}

// HomeHandler handles requests to the root path ("/"), rendering the home page.
// It supports both full page rendering (initial load) and partial rendering (HTMX requests).
//
// Parameters:
// - w: HTTP response writer for sending the rendered HTML.
// - r: HTTP request containing headers (e.g., HX-Request) and context.
//
// Behavior:
// - If HX-Request header is present, renders only the home.html partial.
// - Otherwise, renders the full page using layout.html with registered partials (header, sidebar, content).
// - Logs and returns errors if template loading or rendering fails.
func (c *HomeController) HomeHandler(w http.ResponseWriter, r *http.Request) {
	// Data to pass to the template
	data := map[string]interface{}{
		"title": "AI Workflow Automation Platform", // Page title
	}

	// Check if this is an HTMX request
	isHtmxRequest := r.Header.Get("HX-Request") == "true"

	if isHtmxRequest {
		// Render only the content partial for HTMX
		tmpl, err := raymond.ParseFile("./internal/ui/templates/home.html")
		if err != nil {
			c.logger.Error("Failed to parse home.html template", zap.Error(err))
			http.Error(w, "Internal server error: template parsing failed", http.StatusInternalServerError)
			return
		}
		rendered, err := tmpl.Exec(data)
		if err != nil {
			c.logger.Error("Failed to render home.html template", zap.Error(err))
			http.Error(w, "Internal server error: template rendering failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(rendered))
		return
	}

	// Render the full page with layout
	tmpl, err := raymond.ParseFile("./internal/ui/templates/layout.html")
	if err != nil {
		c.logger.Error("Failed to parse layout.html template", zap.Error(err))
		http.Error(w, "Internal server error: template parsing failed", http.StatusInternalServerError)
		return
	}

	// Manually register partials
	headerContent, err := os.ReadFile("./internal/ui/templates/header.html")
	if err != nil {
		c.logger.Error("Failed to read header.html", zap.Error(err))
		http.Error(w, "Internal server error: failed to read header partial", http.StatusInternalServerError)
		return
	}
	tmpl.RegisterPartial("header", string(headerContent))

	sidebarContent, err := os.ReadFile("./internal/ui/templates/sidebar.html")
	if err != nil {
		c.logger.Error("Failed to read sidebar.html", zap.Error(err))
		http.Error(w, "Internal server error: failed to read sidebar partial", http.StatusInternalServerError)
		return
	}
	tmpl.RegisterPartial("sidebar", string(sidebarContent))

	contentContent, err := os.ReadFile("./internal/ui/templates/home.html")
	if err != nil {
		c.logger.Error("Failed to read home.html for content partial", zap.Error(err))
		http.Error(w, "Internal server error: failed to read content partial", http.StatusInternalServerError)
		return
	}
	tmpl.RegisterPartial("content", string(contentContent))

	// Render the template with registered partials
	rendered, err := tmpl.Exec(data)
	if err != nil {
		c.logger.Error("Failed to render layout.html template", zap.Error(err))
		http.Error(w, "Internal server error: template rendering failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(rendered))
}
