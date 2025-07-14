package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
		client:        &http.Client{},
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

func (t *FetchTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"GET", "POST", "PATCH", "PUT", "DELETE", "HEAD", "OPTIONS"},
			Description: "The HTTP operation to perform",
			Required:    true,
		},
		{
			Name:        "url",
			Type:        "string",
			Description: "The URL to fetch. Must include the protocol (e.g., http:// or https://)",
			Required:    true,
		},
		{
			Name:        "headers",
			Type:        "array",
			Items:       []entities.Item{{Type: "string"}},
			Description: "Array of headers in the format 'key:value' to include in the request",
			Required:    false,
		},
		{
			Name:        "body",
			Type:        "string",
			Description: "The BODY of the request",
			Required:    false,
		},
	}
}

func (t *FetchTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing fetch operation", zap.String("arguments", arguments))

	var args struct {
		Operation string   `json:"operation"`
		Url       string   `json:"url"`
		Headers   []string `json:"headers"`
		Body      string   `json:"body"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	if args.Operation == "" || args.Url == "" {
		t.logger.Error("Operation and url are required")
		return "", fmt.Errorf("operation and url are required")
	}

	userAgent := t.configuration["user_agent"]
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	switch args.Operation {
	case "GET":
		return t.get(args.Url, args.Headers)
	case "POST":
		return t.post(args.Url, args.Headers, args.Body)
	case "PATCH":
		return t.patch(args.Url, args.Headers, args.Body)
	case "PUT":
		return t.put(args.Url, args.Headers, args.Body)
	case "DELETE":
		return t.deleteRequest(args.Url, args.Headers)
	case "HEAD":
		return t.head(args.Url, args.Headers)
	case "OPTIONS":
		return t.options(args.Url, args.Headers)
	default:
		t.logger.Error("Unsupported operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("unsupported operation: %s", args.Operation)
	}
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

	result := struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	}{
		Status: resp.StatusCode,
		Body:   string(body),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.logger.Error("Failed to marshal response", zap.Error(err))
		return "", err
	}

	t.logger.Debug("Request completed",
		zap.Int("status", resp.StatusCode),
		zap.String("url", req.URL.String()))
	return string(resultJSON), nil
}

var _ entities.Tool = (*FetchTool)(nil)
