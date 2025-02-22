package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// GenericAIModel implements the AIModel interface for interacting with any
// OpenAI-compatible API provider (e.g., OpenAI, Grok, Together.ai, Llama.cpp).
// It handles generating responses and tracking token usage using a configurable
// base URL and API key.
//
// Key features:
// - Generates AI responses using the /v1/chat/completions endpoint
// - Tracks token usage for each API call
// - Handles rate limits with exponential backoff retries
// - Supports multiple providers with the same OpenAI-style API
//
// Dependencies:
// - net/http: For making HTTP requests to the AI provider API
// - encoding/json: For JSON encoding/decoding of requests and responses
// - io/ioutil: For reading response bodies
//
// Notes:
// - Assumes all providers support the OpenAI chat completions API structure
// - Prompt parameter in GenerateResponse is treated as the user message
// - Options map can include "system_prompt", "model", "temperature", "max_tokens"
// - Implements retry logic for rate limit errors (HTTP 429)
// - Stores last token usage for retrieval via GetTokenUsage
type GenericAIModel struct {
	baseURL    string       // Base URL for the AI provider API (e.g., https://api.openai.com)
	apiKey     string       // API key for authentication with the provider
	httpClient *http.Client // HTTP client for making API requests
	lastUsage  int          // Token usage from the last API call
}

// NewGenericAIModel creates a new GenericAIModel instance with the provided base URL and API key.
// It initializes the HTTP client with a default timeout of 30 seconds and validates inputs.
//
// Parameters:
// - baseURL: The base URL of the AI provider (e.g., "https://api.openai.com")
// - apiKey: The API key for authenticating with the provider
//
// Returns:
// - A pointer to the initialized GenericAIModel
// - An error if baseURL or apiKey is empty
func NewGenericAIModel(baseURL, apiKey string) (*GenericAIModel, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}
	return &GenericAIModel{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		lastUsage:  0,
	}, nil
}

// GenerateResponse generates a response using the provider's chat completions API.
// It constructs the request with the provided prompt and options, handles API
// calls, and stores token usage.
//
// Parameters:
// - prompt: The user prompt to send to the AI model
// - options: Map of optional parameters, including:
//   - "system_prompt" (string): System prompt for the agent's configuration
//   - "model" (string): Model name (default: "default-model")
//   - "temperature" (float64): Sampling temperature (default: 0.7)
//   - "max_tokens" (int): Maximum tokens for the response
//
// Returns:
// - The generated response text
// - An error if the API call fails or response parsing fails
func (m *GenericAIModel) GenerateResponse(prompt string, options map[string]interface{}) (string, error) {
	// Construct messages array for the API request
	// Start with the user prompt
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}
	// Check if a system prompt is provided in options and prepend it
	if systemPrompt, ok := options["system_prompt"].(string); ok {
		messages = append([]map[string]string{{"role": "system", "content": systemPrompt}}, messages...)
	}

	// Extract model and other parameters from options with defaults
	model, _ := options["model"].(string)
	if model == "" {
		model = "default-model" // Default model if not specified
	}
	temperature, _ := options["temperature"].(float64)
	if temperature == 0 {
		temperature = 0.7 // Default temperature if not specified or zero
	}
	maxTokens, _ := options["max_tokens"].(int)

	// Construct request body for the API
	reqBody := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": temperature,
	}
	// Include max_tokens only if specified
	if maxTokens > 0 {
		reqBody["max_tokens"] = maxTokens
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP POST request to the provider's chat completions endpoint
	req, err := http.NewRequest("POST", m.baseURL+"/v1/chat/completions", bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	// Implement retry logic for handling rate limits (HTTP 429)
	attempts := 0
	maxAttempts := 5
	for attempts < maxAttempts {
		resp, err := m.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		// Handle rate limit errors (429 Too Many Requests)
		if resp.StatusCode == http.StatusTooManyRequests {
			attempts++
			if attempts >= maxAttempts {
				return "", fmt.Errorf("max retry attempts reached for rate limit")
			}
			// Exponential backoff: wait longer with each attempt (1s, 2s, 4s, etc.)
			time.Sleep(time.Duration(attempts) * time.Second)
			continue
		} else if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			return "", fmt.Errorf("API error: %s (status: %d)", string(body), resp.StatusCode)
		}

		// Read and parse the API response
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response body: %w", err)
		}
		var result struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				TotalTokens int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("failed to unmarshal response: %w", err)
		}
		if len(result.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}
		generatedText := result.Choices[0].Message.Content
		m.lastUsage = result.Usage.TotalTokens
		return generatedText, nil
	}
	return "", fmt.Errorf("unexpected error in retry loop")
}

// GetTokenUsage returns the token usage from the last API call.
// This allows tracking of token consumption for cost management.
//
// Returns:
// - The total tokens used in the last API call
// - An error (currently always nil, included for interface compliance)
func (m *GenericAIModel) GetTokenUsage() (int, error) {
	return m.lastUsage, nil
}
