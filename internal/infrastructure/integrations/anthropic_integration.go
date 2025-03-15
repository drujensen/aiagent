package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AnthropicIntegration implements the Anthropic Claude API
type AnthropicIntegration struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	model      string
	toolRepo   interfaces.ToolRepository
	logger     *zap.Logger
	lastUsage  *entities.Usage
}

// NewAnthropicIntegration creates a new Anthropic integration
func NewAnthropicIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*AnthropicIntegration, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}
	if model == "" {
		return nil, fmt.Errorf("model cannot be empty")
	}
	return &AnthropicIntegration{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		model:      model,
		toolRepo:   toolRepo,
		logger:     logger,
		lastUsage:  &entities.Usage{},
	}, nil
}

// ModelName returns the name of the model being used
func (m *AnthropicIntegration) ModelName() string {
	m.logger.Info("Using Anthropic model", zap.String("model", m.model))
	return m.model
}

// ProviderType returns the type of provider
func (m *AnthropicIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderAnthropic
}

// convertToAnthropicMessages converts message entities to Anthropic API format
func convertToAnthropicMessages(messages []*entities.Message) []map[string]interface{} {
	apiMessages := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == "system" {
			// Handle system message specially for Anthropic
			continue
		}
		
		apiMsg := map[string]interface{}{}
		
		// Map roles: user -> user, assistant -> assistant, tool -> tool
		switch msg.Role {
		case "user":
			apiMsg["role"] = "user"
			apiMsg["content"] = msg.Content
		case "assistant":
			apiMsg["role"] = "assistant"
			if len(msg.ToolCalls) > 0 {
				// Format tool calls for Anthropic's API
				content := make([]map[string]interface{}, 0)
				content = append(content, map[string]interface{}{
					"type": "text",
					"text": "",
				})
				
				for _, tc := range msg.ToolCalls {
					content = append(content, map[string]interface{}{
						"type": "tool_use",
						"id":   tc.ID,
						"name": tc.Function.Name,
						"input": json.RawMessage(tc.Function.Arguments),
					})
				}
				apiMsg["content"] = content
			} else {
				apiMsg["content"] = msg.Content
			}
		case "tool":
			apiMsg["role"] = "tool"
			apiMsg["content"] = msg.Content
			apiMsg["tool_call_id"] = msg.ToolCallID
		}
		
		apiMessages = append(apiMessages, apiMsg)
	}
	return apiMessages
}

// GenerateResponse generates a response from the Anthropic API
func (m *AnthropicIntegration) GenerateResponse(messages []*entities.Message, toolList []*interfaces.ToolIntegration, options map[string]interface{}) ([]*entities.Message, error) {
	// Prepare tool definitions for Anthropic
	tools := make([]map[string]interface{}, len(toolList))
	for i, tool := range toolList {
		// Parse the parameters for this tool
		requiredFields := make([]string, 0)
		for _, param := range (*tool).Parameters() {
			if param.Required {
				requiredFields = append(requiredFields, param.Name)
			}
		}

		properties := make(map[string]interface{})
		for _, param := range (*tool).Parameters() {
			property := map[string]interface{}{
				"type":        param.Type,
				"description": param.Description,
			}
			if len(param.Enum) > 0 {
				property["enum"] = param.Enum
			}
			properties[param.Name] = property
		}

		// Anthropic tool format
		tools[i] = map[string]interface{}{
			"name":        (*tool).Name(),
			"description": (*tool).Description(),
			"input_schema": map[string]interface{}{
				"type":       "object",
				"properties": properties,
				"required":   requiredFields,
			},
		}
	}

	// Extract system message if present
	var systemPrompt string
	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			break
		}
	}

	// Format request body according to Anthropic's API
	reqBody := map[string]interface{}{
		"model":      m.model,
		"max_tokens": options["max_tokens"],
	}

	// Add temperature if specified
	if temp, ok := options["temperature"]; ok {
		reqBody["temperature"] = temp
	}

	// Add system prompt if present
	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
	}

	// Add tools if present
	if len(tools) > 0 {
		reqBody["tools"] = tools
	}

	// Convert messages to Anthropic format
	apiMessages := convertToAnthropicMessages(messages)
	reqBody["messages"] = apiMessages

	var newMessages []*entities.Message

	// Handle tool calls in a loop similar to OpenAI
	for {
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request: %v", err)
		}

		// Make the request to Anthropic API
		req, err := http.NewRequest("POST", m.baseURL+"/v1/messages", bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", m.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		
		// Log the request for debugging
		m.logger.Info("Sending request to Anthropic", 
			zap.String("model", m.model), 
			zap.String("url", m.baseURL+"/v1/messages"))

		var resp *http.Response
		for attempt := 0; attempt < 3; attempt++ {
			resp, err = m.httpClient.Do(req)
			if err != nil {
				if attempt < 2 {
					m.logger.Warn("Error making request, retrying", zap.Error(err))
					time.Sleep(time.Duration(attempt+1) * time.Second)
					continue
				}
				return nil, fmt.Errorf("error making request: %v", err)
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				if attempt < 2 {
					time.Sleep(time.Duration(attempt+1) * time.Second)
					continue
				}
				return nil, fmt.Errorf("rate limit exceeded")
			}
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				bodyStr := string(body)
				m.logger.Error("Anthropic API error",
					zap.Int("status_code", resp.StatusCode),
					zap.String("body", bodyStr),
					zap.String("model", m.model))
				return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, bodyStr)
			}
			break
		}
		defer resp.Body.Close()

		// Parse Anthropic API response
		var responseBody struct {
			Type       string `json:"type"`
			Id         string `json:"id"`
			Model      string `json:"model"`
			StopReason string `json:"stop_reason"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Content []struct {
				Type     string `json:"type"`
				Text     string `json:"text,omitempty"`
				ToolUse  *struct {
					Id    string          `json:"id"`
					Name  string          `json:"name"`
					Input json.RawMessage `json:"input"`
				} `json:"tool_use,omitempty"`
			} `json:"content"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
			return nil, fmt.Errorf("error decoding response: %v", err)
		}

		// Track usage
		m.lastUsage.PromptTokens = responseBody.Usage.InputTokens
		m.lastUsage.CompletionTokens = responseBody.Usage.OutputTokens
		m.lastUsage.TotalTokens = responseBody.Usage.InputTokens + responseBody.Usage.OutputTokens

		// Process response content
		var toolCalls []entities.ToolCall
		var textContent string

		for _, content := range responseBody.Content {
			if content.Type == "text" {
				textContent += content.Text
			} else if content.Type == "tool_use" && content.ToolUse != nil {
				toolCall := entities.ToolCall{
					ID:   content.ToolUse.Id,
					Type: "function",
				}
				toolCall.Function.Name = content.ToolUse.Name
				toolCall.Function.Arguments = string(content.ToolUse.Input)
				toolCalls = append(toolCalls, toolCall)
			}
		}

		if len(toolCalls) == 0 {
			// No tool calls, add the final assistant message
			finalMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   textContent,
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, finalMessage)
			break
		} else {
			// There are tool calls, create a message for the assistant's tool call
			toolCallMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   textContent,
				ToolCalls: toolCalls,
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, toolCallMessage)

			// Handle tool calls - execute them and add responses
			for _, toolCall := range toolCalls {
				toolName := toolCall.Function.Name
				tool, err := m.toolRepo.GetToolByName(toolName)
				if err != nil {
					return nil, fmt.Errorf("failed to get tool %s: %v", toolName, err)
				}

				var toolResult string
				if tool != nil {
					result, err := (*tool).Execute(toolCall.Function.Arguments)
					if err != nil {
						toolResult = fmt.Sprintf("Tool %s execution failed: %v", toolName, err)
					} else {
						toolResult = result
					}
				} else {
					toolResult = fmt.Sprintf("Tool %s not found", toolName)
				}

				// Create tool response message
				toolResponseMessage := &entities.Message{
					ID:         uuid.New().String(),
					Role:       "tool",
					Content:    toolResult,
					ToolCallID: toolCall.ID,
					Timestamp:  time.Now(),
				}
				newMessages = append(newMessages, toolResponseMessage)

				// Add tool response to messages for next API call
				apiMessages = append(apiMessages, map[string]interface{}{
					"role":         "tool",
					"content":      toolResult,
					"tool_call_id": toolCall.ID,
				})
			}

			// Add assistant message to messages for next API call
			apiMessages = append(apiMessages, map[string]interface{}{
				"role":    "assistant",
				"content": toolCalls,
			})

			// Update request for next iteration
			reqBody["messages"] = apiMessages
		}
	}

	return newMessages, nil
}

// GetUsage returns usage information
func (m *AnthropicIntegration) GetUsage() (*entities.Usage, error) {
	return m.lastUsage, nil
}

// Ensure AnthropicIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*AnthropicIntegration)(nil)