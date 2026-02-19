package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

const (
	defaultUserAgent = "AIAgents/1.0 (Autonomous; +https://github.com/drujensen/aiagents)"
)

type FetchTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
	client        *http.Client
}

func NewFetchTool(name, description string, configuration map[string]string, logger *zap.Logger) *FetchTool {
	return &FetchTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
		client:        &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *FetchTool) Name() string {
	return f.name
}

func (f *FetchTool) Description() string {
	return f.description
}

func (t *FetchTool) Configuration() map[string]string {
	return t.configuration
}

func (t *FetchTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *FetchTool) FullDescription() string {
	var b strings.Builder

	// Add description
	b.WriteString(t.Description())
	b.WriteString("\n\n")

	// Add configuration header
	b.WriteString("Configuration for this tool:\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")

	// Loop through configuration and add key-value pairs to the table
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}

	return b.String()
}

func (t *FetchTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch content from",
			},
			"format": map[string]any{
				"type":        "string",
				"description": "The format to return the content in (text, markdown, or html). Defaults to markdown.",
				"enum":        []string{"text", "markdown", "html"},
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Optional timeout in seconds (max 120)",
			},
		},
		"required":             []string{"url", "format"},
		"additionalProperties": false,
	}
}

func (t *FetchTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing fetch operation", zap.String("arguments", arguments))

	var rawArgs map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &rawArgs); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	url := ""
	if u, ok := rawArgs["url"].(string); ok {
		url = u
	}
	format := "markdown"
	if f, ok := rawArgs["format"].(string); ok {
		format = f
	}
	timeoutVal, _ := rawArgs["timeout"].(float64)
	timeout := int(timeoutVal)

	if url == "" {
		t.logger.Error("url is required")
		return "", fmt.Errorf("url is required")
	}

	if timeout > 0 && timeout <= 120 {
		t.client.Timeout = time.Duration(timeout) * time.Second
	}

	userAgent := t.configuration["user_agent"]
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	// For webfetch, default to GET
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	content := string(body)

	// Convert based on format
	switch format {
	case "text":
		// Already text
	case "html":
		// Assume it's already HTML
	case "markdown":
		// Simple conversion: wrap in code blocks or something, but for now, return as is
		content = fmt.Sprintf("```\n%s\n```", content)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	return fmt.Sprintf(`{"content": %q, "status_code": %d}`, content, resp.StatusCode), nil
}

func (t *FetchTool) get(url string, headers []string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	return t.doRequest(req, headers)
}

func (t *FetchTool) post(url string, headers []string, body string) (string, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return "", err
	}
	return t.doRequest(req, headers)
}

func (t *FetchTool) patch(url string, headers []string, body string) (string, error) {
	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return "", err
	}
	return t.doRequest(req, headers)
}

func (t *FetchTool) put(url string, headers []string, body string) (string, error) {
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return "", err
	}
	return t.doRequest(req, headers)
}

func (t *FetchTool) deleteRequest(url string, headers []string) (string, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return "", err
	}
	return t.doRequest(req, headers)
}

func (t *FetchTool) head(url string, headers []string) (string, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", err
	}
	return t.doRequest(req, headers)
}

func (t *FetchTool) options(url string, headers []string) (string, error) {
	req, err := http.NewRequest("OPTIONS", url, nil)
	if err != nil {
		return "", err
	}
	return t.doRequest(req, headers)
}

func (t *FetchTool) doRequest(req *http.Request, headers []string) (string, error) {
	// Set default User-Agent
	userAgent := t.configuration["user_agent"]
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	req.Header.Set("User-Agent", userAgent)

	// Add custom headers
	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			req.Header.Set(key, value)
		}
	}

	resp, err := t.client.Do(req)
	if err != nil {
		t.logger.Error("Request failed", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.logger.Error("Failed to read response body", zap.Error(err))
		return "", err
	}

	// Create TUI-friendly summary
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("🌐 HTTP %s: %s\n", req.Method, req.URL.String()))
	summary.WriteString(fmt.Sprintf("📊 Status: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode)))

	// Show response body preview (first 200 characters)
	bodyStr := string(body)
	if len(bodyStr) > 200 {
		summary.WriteString(fmt.Sprintf("📄 Response: %s...\n", bodyStr[:200]))
	} else {
		summary.WriteString(fmt.Sprintf("📄 Response: %s\n", bodyStr))
	}

	// Create JSON response with summary for TUI and full data for AI
	response := struct {
		Summary string `json:"summary"`
		Method  string `json:"method"`
		URL     string `json:"url"`
		Status  int    `json:"status"`
		Body    string `json:"body"`
	}{
		Summary: summary.String(),
		Method:  req.Method,
		URL:     req.URL.String(),
		Status:  resp.StatusCode,
		Body:    bodyStr,
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal fetch response", zap.Error(err))
		return summary.String(), nil // Fallback to summary only
	}

	t.logger.Debug("Request completed",
		zap.Int("status", resp.StatusCode),
		zap.String("url", req.URL.String()))
	return string(jsonResult), nil
}

var _ entities.Tool = (*FetchTool)(nil)
