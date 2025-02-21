package integrations

import (
	"fmt"
)

// Open API Integration. This will be an http client that accesses Open AI API

type OpenAI struct {
	APIKey string
}

// NewOpenAI creates a new OpenAI client with the given API key
func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{
		APIKey: apiKey,
	}
}

// GenerateResponse generates a response from the OpenAI API
func (openai *OpenAI) GenerateResponse(prompt string) (string, error) {
	// Call OpenAI API here
	return fmt.Sprintf("Response from OpenAI API for prompt: %s", prompt), nil
}
