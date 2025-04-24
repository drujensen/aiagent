package tools

import (
	"aiagent/internal/domain/entities"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type ImageGenerationTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
	client        *http.Client
}

func NewImageGenerationTool(name, description string, configuration map[string]string, logger *zap.Logger) *ImageGenerationTool {
	return &ImageGenerationTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
		client:        &http.Client{},
	}
}

func (t *ImageGenerationTool) Name() string {
	return t.name
}

func (t *ImageGenerationTool) Description() string {
	return t.description
}

func (t *ImageGenerationTool) Configuration() map[string]string {
	return t.configuration
}

func (t *ImageGenerationTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config // Update with new config, e.g., provider, API key, etc.
}

func (t *ImageGenerationTool) FullDescription() string {
	var b strings.Builder
	b.WriteString(t.Description() + "\n\n")
	b.WriteString("Configuration for this tool:\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *ImageGenerationTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "prompt",
			Type:        "string",
			Description: "The text prompt for image generation",
			Required:    true,
		},
		{
			Name:        "n",
			Type:        "integer",
			Description: "Number of images to generate (1-10)",
			Required:    false,
		},
		{
			Name:        "model",
			Type:        "string",
			Description: "Model to use (e.g., 'dall-e-3' for OpenAI)",
			Required:    false,
		},
	}
}

func (t *ImageGenerationTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing image generation", zap.String("arguments", arguments))
	var args struct {
		Prompt string `json:"prompt"`
		N      int    `json:"n"`
		Model  string `json:"model"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if args.N < 1 || args.N > 10 {
		args.N = 1
	}

	provider, ok := t.configuration["provider"]
	if !ok {
		return "", fmt.Errorf("provider not configured")
	}
	apiKey, ok := t.configuration["api_key"]
	if !ok || apiKey == "" {
		return "", fmt.Errorf("API key not configured")
	}
	baseURL, ok := t.configuration["base_url"]
	if !ok || baseURL == "" {
		return "", fmt.Errorf("base URL not configured")
	}

	t.logger.Info("Attempting API call", zap.String("provider", provider), zap.String("baseURL", baseURL))
	url := baseURL
	body := map[string]interface{}{
		"prompt": args.Prompt,
		"n":      args.N,
	}
	if strings.ToLower(provider) == "openai" && args.Model != "" {
		body["model"] = args.Model
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.logger.Error("API request failed", zap.Int("status", resp.StatusCode), zap.String("url", url))
		return fmt.Sprintf("API error: %s", resp.Status), nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		return fmt.Sprintf("Images generated: %v", data), nil
	}
	return "Image generation completed, but no data returned", nil
}

var _ entities.Tool = (*ImageGenerationTool)(nil)
