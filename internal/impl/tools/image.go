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

type ImageTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
	client        *http.Client
}

func NewImageTool(name, description string, configuration map[string]string, logger *zap.Logger) *ImageTool {
	return &ImageTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
		client:        &http.Client{},
	}
}

func (t *ImageTool) Name() string {
	return t.name
}

func (t *ImageTool) Description() string {
	return t.description
}

func (t *ImageTool) Configuration() map[string]string {
	return t.configuration
}

func (t *ImageTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *ImageTool) FullDescription() string {
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

func (t *ImageTool) Parameters() []entities.Parameter {
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
	}
}

func (t *ImageTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing image generation", zap.String("arguments", arguments))
	fmt.Println("Executing image generation with arguments:", arguments)
	var args struct {
		Prompt string `json:"prompt"`
		N      int    `json:"n"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err), zap.String("arguments", arguments))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Prompt == "" {
		t.logger.Error("Prompt is required but not provided")
		return "", fmt.Errorf("prompt is required")
	}

	if args.N < 1 || args.N > 10 {
		args.N = 1
	}

	provider, ok := t.configuration["provider"]
	if !ok {
		t.logger.Error("Provider not configured")
		return "", fmt.Errorf("provider not configured")
	}
	apiKey, ok := t.configuration["api_key"]
	if !ok || apiKey == "" {
		t.logger.Error("API key not configured")
		return "", fmt.Errorf("API key not configured")
	}
	baseURL, ok := t.configuration["base_url"]
	if !ok || baseURL == "" {
		if strings.ToLower(provider) == "openai" {
			baseURL = "https://api.openai.com/v1/images/generations"
		} else if strings.ToLower(provider) == "xai" {
			baseURL = "https://api.x.ai/v1/images/generations"

		} else {
			t.logger.Error("Base URL not configured")
			return "", fmt.Errorf("base URL not configured")
		}
	}
	model, ok := t.configuration["model"]
	if !ok || model == "" {
		if strings.ToLower(provider) == "openai" {
			model = "dall-e-3"
		} else if strings.ToLower(provider) == "xai" {
			model = "grok-2-image"
		} else {
			t.logger.Error("Model not configured")
			return "", fmt.Errorf("Model not configured")
		}
	}

	t.logger.Info("Attempting API call", zap.String("provider", provider), zap.String("baseURL", baseURL))
	url := baseURL
	body := map[string]any{
		"prompt": args.Prompt,
		"n":      args.N,
		"model":  model,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.logger.Error("Failed to marshal request body", zap.Error(err), zap.String("url", url))
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		t.logger.Error("Failed to create request", zap.Error(err), zap.String("url", url))
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		t.logger.Error("API request error", zap.Error(err), zap.String("url", url))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.logger.Error("API request failed", zap.Int("status", resp.StatusCode), zap.String("url", url))
		return fmt.Sprintf("API error: %s", resp.Status), nil
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	t.logger.Debug("API response", zap.Any("result", result))
	if data, ok := result["data"].([]any); ok && len(data) > 0 {
		var markdownLinks strings.Builder
		for i, item := range data {
			if image, ok := item.(map[string]any); ok {
				if url, ok := image["url"].(string); ok {
					markdownLinks.WriteString(fmt.Sprintf("![Image %d](%s)\n", i+1, url))
					revisedPrompt := image["revised_prompt"].(string)
					markdownLinks.WriteString(fmt.Sprintf("%s\n", revisedPrompt))
				} else {
					t.logger.Warn("Image URL not found", zap.Any("item", item))
				}
			} else {
				t.logger.Warn("Unexpected image format", zap.Any("item", item))
			}
		}
		t.logger.Info("Image generation successful", zap.String("response", markdownLinks.String()))
		return markdownLinks.String(), nil
	}
	t.logger.Warn("No image data returned from API")
	return "Image generation completed, but no data returned", nil
}

var _ entities.Tool = (*ImageTool)(nil)
