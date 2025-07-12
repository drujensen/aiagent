package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"github.com/go-openapi/spec"
	"go.uber.org/zap"
)

type SwaggerTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
	client        *http.Client
}

func NewSwaggerTool(name, description string, configuration map[string]string, logger *zap.Logger) *SwaggerTool {
	return &SwaggerTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
		client:        &http.Client{},
	}
}

func (t *SwaggerTool) Name() string {
	return t.name
}

func (t *SwaggerTool) Description() string {
	return t.description
}

func (t *SwaggerTool) Configuration() map[string]string {
	return t.configuration
}

func (t *SwaggerTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *SwaggerTool) FullDescription() string {
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

func (t *SwaggerTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "fetch",
			Type:        "boolean",
			Description: "Whether to fetch and return the Swagger API specification (default: true)",
			Required:    false,
		},
	}
}

type SwaggerEndpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

type SwaggerResponse struct {
	Endpoints []SwaggerEndpoint `json:"endpoints"`
	RawSpec   string            `json:"raw_spec,omitempty"`
}

func (t *SwaggerTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing Swagger tool", zap.String("arguments", arguments))

	var args struct {
		Fetch bool `json:"fetch"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	// Default to true if not specified
	if arguments == "" {
		args.Fetch = true
	}

	if !args.Fetch {
		return "", fmt.Errorf("no action requested")
	}

	swaggerURL := t.configuration["swagger_url"]
	if swaggerURL == "" {
		t.logger.Error("Swagger URL not configured")
		return "", fmt.Errorf("swagger_url configuration is required")
	}

	// Fetch the Swagger JSON
	req, err := http.NewRequest("GET", swaggerURL, nil)
	if err != nil {
		t.logger.Error("Failed to create request", zap.Error(err))
		return "", err
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := t.client.Do(req)
	if err != nil {
		t.logger.Error("Failed to fetch Swagger spec", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.logger.Error("Failed to read response body", zap.Error(err))
		return "", err
	}

	// Parse the Swagger JSON
	var swaggerSpec spec.Swagger
	if err := json.Unmarshal(body, &swaggerSpec); err != nil {
		t.logger.Error("Failed to parse Swagger JSON", zap.Error(err))
		return "", err
	}

	// Extract endpoints
	var endpoints []SwaggerEndpoint
	for path, pathItem := range swaggerSpec.Paths.Paths {
		if pathItem.Get != nil {
			endpoints = append(endpoints, SwaggerEndpoint{
				Method:      "GET",
				Path:        path,
				Description: pathItem.Get.Summary,
			})
		}
		if pathItem.Post != nil {
			endpoints = append(endpoints, SwaggerEndpoint{
				Method:      "POST",
				Path:        path,
				Description: pathItem.Post.Summary,
			})
		}
		if pathItem.Put != nil {
			endpoints = append(endpoints, SwaggerEndpoint{
				Method:      "PUT",
				Path:        path,
				Description: pathItem.Put.Summary,
			})
		}
		if pathItem.Patch != nil {
			endpoints = append(endpoints, SwaggerEndpoint{
				Method:      "PATCH",
				Path:        path,
				Description: pathItem.Patch.Summary,
			})
		}
		if pathItem.Delete != nil {
			endpoints = append(endpoints, SwaggerEndpoint{
				Method:      "DELETE",
				Path:        path,
				Description: pathItem.Delete.Summary,
			})
		}
	}

	// Prepare response
	response := SwaggerResponse{
		Endpoints: endpoints,
		RawSpec:   string(body), // Optionally include the raw spec
	}
	resultJSON, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal response", zap.Error(err))
		return "", err
	}

	t.logger.Info("Swagger spec fetched and parsed", zap.String("url", swaggerURL))
	return string(resultJSON), nil
}

var _ entities.Tool = (*SwaggerTool)(nil)
