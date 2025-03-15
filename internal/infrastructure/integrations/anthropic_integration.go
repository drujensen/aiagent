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
		httpClient: &http.Client{Timeout: 600 * time.Second},
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
		if msg.Role == "system" || msg.Role == "tool" { // Skip system and tool roles for initial request
			continue
		}

		apiMsg := map[string]interface{}{}

		switch msg.Role {
		case "user":
			apiMsg["role"] = "user"
			apiMsg["content"] = msg.Content
		case "assistant":
			apiMsg["role"] = "assistant"
			if len(msg.ToolCalls) > 0 {
				content := make([]map[string]interface{}, 0)
				content = append(content, map[string]interface{}{
					"type": "text",
					"text": "",
				})
				for _, tc := range msg.ToolCalls {
					content = append(content, map[string]interface{}{
						"type":  "tool_use",
						"id":    tc.ID,
						"name":  tc.Function.Name,
						"input": json.RawMessage(tc.Function.Arguments),
					})
				}
				apiMsg["content"] = content
			} else {
				apiMsg["content"] = msg.Content
			}
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

	// Format request body
	reqBody := map[string]interface{}{
		"model":      m.model,
		"max_tokens": options["max_tokens"],
	}
	if temp, ok := options["temperature"]; ok {
		reqBody["temperature"] = temp
	}
	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
	}
	if len(tools) > 0 {
		reqBody["tools"] = tools
	}

	// Convert messages to Anthropic format (initially excluding tool roles)
	apiMessages := convertToAnthropicMessages(messages)
	reqBody["messages"] = apiMessages

	var newMessages []*entities.Message

	// Tool call handling loop
	for {
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request: %v", err)
		}
		m.logger.Info("Sending request to Anthropic", zap.String("body", string(jsonBody)))

		req, err := http.NewRequest("POST", m.baseURL+"/v1/messages", bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", m.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

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
				m.logger.Error("Anthropic API error",
					zap.Int("status_code", resp.StatusCode),
					zap.String("body", string(body)))
				return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
			}
			break
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response: %v", err)
		}
		m.logger.Info("Anthropic response", zap.String("body", string(respBody)))

		// Parse response
		var responseBody struct {
			Type       string `json:"type"`
			Id         string `json:"id"`
			Model      string `json:"model"`
			StopReason string `json:"stop_reason"`
			Usage      struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Content []struct {
				Type    string `json:"type"`
				Text    string `json:"text,omitempty"`
				ToolUse struct {
					ID    string          `json:"id"`
					Name  string          `json:"name"`
					Input json.RawMessage `json:"input"`
				} `json:"tool_use,omitempty"`
			} `json:"content"`
		}
		if err := json.Unmarshal(respBody, &responseBody); err != nil {
			return nil, fmt.Errorf("error decoding response: %v", err)
		}

		// Log each content block
		for i, content := range responseBody.Content {
			if content.Type == "tool_use" {
				m.logger.Info("Parsed tool_use block",
					zap.Int("index", i),
					zap.String("id", content.ToolUse.ID),
					zap.String("name", content.ToolUse.Name),
					zap.String("input", string(content.ToolUse.Input)))
			} else if content.Type == "text" {
				m.logger.Info("Parsed text block",
					zap.Int("index", i),
					zap.String("type", content.Type),
					zap.String("text", content.Text))
			}
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
			} else if content.Type == "tool_use" {
				toolCall := entities.ToolCall{
					ID:   content.ToolUse.ID,
					Type: "function",
				}
				toolCall.Function.Name = content.ToolUse.Name
				toolCall.Function.Arguments = string(content.ToolUse.Input)
				toolCalls = append(toolCalls, toolCall)
				m.logger.Info("Tool use processed",
					zap.String("id", content.ToolUse.ID),
					zap.String("name", content.ToolUse.Name),
					zap.String("input", string(content.ToolUse.Input)))
			}
		}

		if len(toolCalls) > 0 {
			m.logger.Info("Tool calls generated", zap.Any("toolCalls", toolCalls))
		} else {
			m.logger.Info("No tool calls generated")
		}

		if len(toolCalls) == 0 {
			finalMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   textContent,
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, finalMessage)
			break
		} else {
			toolCallMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   textContent,
				ToolCalls: toolCalls,
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, toolCallMessage)

			for _, toolCall := range toolCalls {
				toolName := toolCall.Function.Name
				tool, err := m.toolRepo.GetToolByName(toolName)
				if err != nil {
					m.logger.Warn("Failed to get tool", zap.String("toolName", toolName), zap.Error(err))
					continue // Skip to next tool call if tool not found
				}

				var toolResult string
				if tool != nil {
					result, err := (*tool).Execute(toolCall.Function.Arguments)
					if err != nil {
						toolResult = fmt.Sprintf("Tool %s execution failed: %v", toolName, err)
						m.logger.Warn("Tool execution failed", zap.String("toolName", toolName), zap.Error(err))
					} else {
						toolResult = result
					}
				} else {
					toolResult = fmt.Sprintf("Tool %s not found", toolName)
					m.logger.Warn("Tool not found", zap.String("toolName", toolName))
				}

				toolResponseMessage := &entities.Message{
					ID:         uuid.New().String(),
					Role:       "tool",
					Content:    toolResult,
					ToolCallID: toolCall.ID,
					Timestamp:  time.Now(),
				}
				newMessages = append(newMessages, toolResponseMessage)

				// Append tool result to apiMessages for next iteration
				if toolCall.ID != "" { // Only append if we have a valid tool_call_id
					apiMessages = append(apiMessages, map[string]interface{}{
						"role": "user", // Use "user" role to report tool result as per Anthropic's convention
						"content": []map[string]interface{}{
							{
								"type":         "tool_result",
								"tool_call_id": toolCall.ID,
								"content":      toolResult,
							},
						},
					})
				}
			}

			reqBody["messages"] = apiMessages
		}
	}

	m.logger.Info("Generated messages", zap.Any("messages", newMessages))
	return newMessages, nil
}

// GetUsage returns usage information
func (m *AnthropicIntegration) GetUsage() (*entities.Usage, error) {
	return m.lastUsage, nil
}

// Ensure AnthropicIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*AnthropicIntegration)(nil)
