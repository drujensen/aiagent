package integrations

import (
	"testing"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"go.uber.org/zap/zaptest"
)

func TestNewOpenAIIntegration_EndpointRouting(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name         string
		modelFamily  string
		modelName    string
		expectedPath string
	}{
		{
			name:         "o1 model uses responses endpoint",
			modelFamily:  "o",
			modelName:    "o1-preview",
			expectedPath: "/v1/responses",
		},
		{
			name:         "codex model uses responses endpoint",
			modelFamily:  "codex",
			modelName:    "codex-python",
			expectedPath: "/v1/responses",
		},
		{
			name:         "gpt model uses chat completions endpoint",
			modelFamily:  "gpt-4",
			modelName:    "gpt-4",
			expectedPath: "/v1/chat/completions",
		},
		{
			name:         "o4 model uses responses endpoint",
			modelFamily:  "o",
			modelName:    "o4-mini",
			expectedPath: "/v1/responses",
		},
		{
			name:         "empty family defaults to chat completions",
			modelFamily:  "",
			modelName:    "some-model",
			expectedPath: "/v1/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &entities.Model{
				Family:    tt.modelFamily,
				ModelName: tt.modelName,
			}

			integration, err := NewOpenAIIntegration("https://api.openai.com", "test-key", model, nil, logger)
			if err != nil {
				t.Fatalf("Failed to create integration: %v", err)
			}

			// Check that the correct endpoint was set
			if !containsString(integration.baseURL, tt.expectedPath) {
				t.Errorf("Expected endpoint to contain %s, got %s", tt.expectedPath, integration.baseURL)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

func TestOpenAIIntegration_generateResponseV2(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name           string
		modelFamily    string
		mockResponse   string
		expectedOutput string
		expectToolCall bool
	}{
		{
			name:        "basic message response",
			modelFamily: "o",
			mockResponse: `{
				"id": "resp_test",
				"object": "response",
				"created_at": 1234567890,
				"status": "completed",
				"model": "o1-preview",
				"output": [
					{
						"type": "message",
						"content": [
							{
								"type": "output_text",
								"text": "Hello, this is a test response."
							}
						]
					}
				],
				"usage": {
					"input_tokens": 10,
					"output_tokens": 5,
					"total_tokens": 15
				}
			}`,
			expectedOutput: "Hello, this is a test response.",
			expectToolCall: false,
		},
		{
			name:        "reasoning response",
			modelFamily: "o",
			mockResponse: `{
				"id": "resp_test",
				"object": "response",
				"created_at": 1234567890,
				"status": "completed",
				"model": "o1-preview",
				"output": [
					{
						"type": "reasoning",
						"summary": "I need to analyze this request carefully."
					},
					{
						"type": "message",
						"content": [
							{
								"type": "output_text",
								"text": "After careful analysis, here's my response."
							}
						]
					}
				],
				"usage": {
					"input_tokens": 10,
					"output_tokens": 8,
					"total_tokens": 18
				}
			}`,
			expectedOutput: "I need to analyze this request carefully.\nAfter careful analysis, here's my response.",
			expectToolCall: false,
		},
		{
			name:        "function call response",
			modelFamily: "o",
			mockResponse: `{
				"id": "resp_test",
				"object": "response",
				"created_at": 1234567890,
				"status": "completed",
				"model": "o1-preview",
				"output": [
					{
						"type": "function_call",
						"call_id": "call_123",
						"name": "test_tool",
						"arguments": "{\"param\": \"value\"}"
					}
				],
				"usage": {
					"input_tokens": 10,
					"output_tokens": 3,
					"total_tokens": 13
				}
			}`,
			expectedOutput: "",
			expectToolCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &entities.Model{
				Family:    tt.modelFamily,
				ModelName: "o1-preview",
			}

			integration, err := NewOpenAIIntegration("https://api.openai.com", "test-key", model, nil, logger)
			if err != nil {
				t.Fatalf("Failed to create integration: %v", err)
			}

			// Note: Full integration testing would require mocking HTTP responses
			// This test verifies the endpoint routing works correctly
			if tt.modelFamily == "o" && !containsString(integration.baseURL, "/v1/responses") {
				t.Errorf("Expected responses endpoint for o1 model, got %s", integration.baseURL)
			}
		})
	}
}
