package tools

import (
	"aiagent/internal/domain/entities"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type VisionTool struct {
	NameField            string
	DescriptionField     string
	FullDescriptionField string
	ConfigurationField   map[string]string
}

func (v *VisionTool) Name() string {
	return v.NameField
}

func (v *VisionTool) Description() string {
	return v.DescriptionField
}

func (v *VisionTool) FullDescription() string {
	return v.FullDescriptionField
}

func (v *VisionTool) Configuration() map[string]string {
	return v.ConfigurationField
}

func (v *VisionTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "image_path",
			Type:        "string",
			Description: "Path to the image file for base64 encoding",
			Required:    false,
		},
		{
			Name:        "image_url",
			Type:        "string",
			Description: "Direct URL to the image",
			Required:    false,
		},
		{
			Name:        "prompt",
			Type:        "string",
			Description: "Text prompt for the image",
			Required:    true,
		},
	}
}

func (v *VisionTool) Execute(arguments string) (string, error) {
	fmt.Println("\rExecuting VisionTool:", arguments)
	var args map[string]string
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	prompt, ok := args["prompt"]
	if !ok {
		return "", fmt.Errorf("prompt is required")
	}

	var imageURL string
	if args["image_path"] != "" {
		base64Image, err := EncodeImageToBase64(args["image_path"])
		if err != nil {
			return "", err
		}
		imageURL = base64Image
	} else if args["image_url"] != "" {
		imageURL = args["image_url"]
	}

	messages := []Message{
		{
			Role: "user",
			Content: []MessageContent{
				{
					Type: "image_url",
					ImageURL: struct {
						URL    string `json:"url"`
						Detail string `json:"detail"`
					}{URL: imageURL, Detail: "high"},
				},
				{
					Type: "text",
					Text: prompt,
				},
			},
		},
	}

	return v.VisionAPIRequest(messages)
}

func (v *VisionTool) UpdateConfiguration(config map[string]string) {
	v.ConfigurationField = config
}

type MessageContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL struct {
		URL    string `json:"url"`
		Detail string `json:"detail"`
	} `json:"image_url,omitempty"`
}

type Message struct {
	Role    string           `json:"role"`
	Content []MessageContent `json:"content"`
}

type RequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

func (v *VisionTool) VisionAPIRequest(messages []Message) (string, error) {
	provider := v.ConfigurationField["provider"]
	apiKey := v.ConfigurationField["api_key"]
	baseURL := v.ConfigurationField["base_url"]
	model := v.ConfigurationField["model"]

	if provider == "xai" {
		if baseURL == "" {
			baseURL = "https://api.x.ai/v1" // Default for xai
		}
		if model == "" {
			model = "grok-2-vision-latest" // Default model for xai
		}
	} else if provider == "openai" {
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1" // Default for openai (adjust if needed)
		}
		if model == "" {
			model = "gpt-4-vision-preview" // Default model for openai vision
		}
	} else {
		return "", fmt.Errorf("unsupported provider")
	}

	// Rest of the function remains the same, using the resolved baseURL and model
	url := baseURL + "/chat/completions" // Use the resolved baseURL
	body := RequestBody{
		Model:    model, // e.g., "grok-2-vision-latest"
		Messages: messages,
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(respBody), nil // Response contains the AI's output
}

// Example function to encode an image to base64
func EncodeImageToBase64(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data), nil
}
