package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/events"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

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
func convertToAnthropicMessages(messages []*entities.Message) []map[string]any {
	apiMessages := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == "system" { // Skip system for initial request
			continue
		}

		apiMsg := map[string]any{}

		switch msg.Role {
		case "user":
			apiMsg["role"] = "user"
			apiMsg["content"] = msg.Content
		case "assistant":
			apiMsg["role"] = "assistant"
			if len(msg.ToolCalls) > 0 {
				content := make([]map[string]any, 0)
				for _, tc := range msg.ToolCalls {
					content = append(content, map[string]any{
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
		case "tool":
			apiMsg["role"] = "user" // Use "user" role to report tool result as per Anthropic's convention
			apiMsg["content"] = []map[string]any{
				{
					"type":        "tool_result",
					"tool_use_id": msg.ToolCallID,
					"content":     msg.Content,
				},
			}
		}

		apiMessages = append(apiMessages, apiMsg)
	}
	return apiMessages
}

// GenerateResponse generates a response from the Anthropic API
func (m *AnthropicIntegration) GenerateResponse(ctx context.Context, messages []*entities.Message, toolList []*entities.Tool, options map[string]any) ([]*entities.Message, error) {
	// Prepare tool definitions for Anthropic
	tools := make([]map[string]any, len(toolList))
	for i, tool := range toolList {
		// Check for cancellation
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("operation canceled by user")
		}

		requiredFields := make([]string, 0)
		for _, param := range (*tool).Parameters() {
			if param.Required {
				requiredFields = append(requiredFields, param.Name)
			}
		}

		properties := make(map[string]any)
		for _, param := range (*tool).Parameters() {
			property := map[string]any{
				"type":        param.Type,
				"description": param.Description,
			}
			if len(param.Enum) > 0 {
				property["enum"] = param.Enum
			}
			properties[param.Name] = property
		}

		tools[i] = map[string]any{
			"name":        (*tool).Name(),
			"description": (*tool).Description(),
			"input_schema": map[string]any{
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
	reqBody := map[string]any{
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
		// Check for cancellation before sending request
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("operation canceled by user")
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request: %v", err)
		}
		m.logger.Info("Sending request to Anthropic", zap.String("body", string(jsonBody)))

		req, err := http.NewRequestWithContext(ctx, "POST", m.baseURL+"/v1/messages", bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", m.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		var resp *http.Response
		for attempt := 0; attempt < 3; attempt++ {
			// Check for cancellation before making request
			if ctx.Err() == context.Canceled {
				return nil, fmt.Errorf("operation canceled by user")
			}

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
				Type  string          `json:"type"`
				Text  string          `json:"text,omitempty"`
				Id    string          `json:"id"`
				Name  string          `json:"name"`
				Input json.RawMessage `json:"input"`
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
					zap.String("id", content.Id),
					zap.String("name", content.Name),
					zap.String("input", string(content.Input)))
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
					ID:   content.Id,
					Type: "function",
				}
				toolCall.Function.Name = content.Name
				toolCall.Function.Arguments = string(content.Input)
				toolCalls = append(toolCalls, toolCall)
				m.logger.Info("Tool use processed",
					zap.String("id", content.Id),
					zap.String("name", content.Name),
					zap.String("input", string(content.Input)))
			}
		}

		if len(toolCalls) > 0 {
			m.logger.Info("Tool calls generated", zap.Any("toolCalls", toolCalls))
		} else {
			m.logger.Info("No tool calls generated")
		}

		// Break if no tool calls OR if stop_reason indicates completion
		if len(toolCalls) == 0 || responseBody.StopReason == "end_turn" || responseBody.StopReason == "max_tokens" || responseBody.StopReason == "stop_sequence" {
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
				// Check for cancellation before executing tool
				if ctx.Err() == context.Canceled {
					return nil, fmt.Errorf("operation canceled by user")
				}

				toolName := toolCall.Function.Name
				tool, err := m.toolRepo.GetToolByName(toolName)

				var toolResult string
				var toolError string
				var diff string
				if err != nil {
					toolResult = fmt.Sprintf("Tool %s could not be retrieved: %v", toolName, err)
					toolError = err.Error()
					m.logger.Warn("Failed to get tool", zap.String("toolName", toolName), zap.Error(err))
				} else if tool != nil {
					result, err := (*tool).Execute(toolCall.Function.Arguments)
					if err != nil {
						toolResult = fmt.Sprintf("Tool %s execution failed: %v", toolName, err)
						toolError = err.Error()
						m.logger.Warn("Tool execution failed", zap.String("toolName", toolName), zap.Error(err))
					} else {
						toolResult = result
						// Extract diff if it's a file write operation
						if toolName == "FileWrite" {
							diff = m.extractDiffFromResult(result)
						}
					}
				} else {
					toolResult = fmt.Sprintf("Tool %s not found", toolName)
					toolError = "Tool not found"
					m.logger.Warn("Tool not found", zap.String("toolName", toolName))
				}
				// Create tool call event
				toolEvent := entities.NewToolCallEvent(toolName, toolCall.Function.Arguments, toolResult, toolError, diff, nil)

				// Publish real-time event for TUI updates
				events.PublishToolCallEvent(toolEvent)

				var content string
				if toolError != "" {
					content = fmt.Sprintf("Tool %s failed with error: %s", toolName, toolError)
				} else {
					content = fmt.Sprintf("Tool %s succeeded: %s", toolName, toolResult)
				}

				toolResponseMessage := &entities.Message{
					ID:             uuid.New().String(),
					Role:           "tool",
					Content:        content,
					ToolCallID:     toolCall.ID,
					ToolCallEvents: []entities.ToolCallEvent{*toolEvent},
					Timestamp:      time.Now(),
				}
				newMessages = append(newMessages, toolResponseMessage)

				// Append tool result to apiMessages for next iteration
				if toolCall.ID != "" {
					apiMessages = append(apiMessages, map[string]any{
						"role": "assistant",
						"content": []map[string]any{
							{
								"type":  "tool_use",
								"id":    toolCall.ID,
								"name":  toolName,
								"input": json.RawMessage(toolCall.Function.Arguments),
							},
						},
					})
					apiMessages = append(apiMessages, map[string]any{
						"role": "user", // Use "user" role to report tool result as per Anthropic's convention
						"content": []map[string]any{
							{
								"type":        "tool_result",
								"tool_use_id": toolCall.ID,
								"content":     toolResult,
							},
						},
					})
				}
			}

			reqBody["messages"] = apiMessages
		}
	}

	// Validate that all tool calls have responses before returning
	newMessages = ensureToolCallResponsesAnthropic(newMessages, m.logger)

	m.logger.Info("Generated messages", zap.Any("messages", newMessages))
	return newMessages, nil
}

// ensureToolCallResponsesAnthropic validates that every tool call has a corresponding response
// and creates error responses for any orphaned tool calls
func ensureToolCallResponsesAnthropic(messages []*entities.Message, logger *zap.Logger) []*entities.Message {
	// Collect all tool call IDs from assistant messages
	toolCallIDs := make(map[string]bool)
	for _, msg := range messages {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				toolCallIDs[toolCall.ID] = false // false = no response found yet
			}
		}
	}

	// Mark tool calls that have responses
	for _, msg := range messages {
		if msg.Role == "tool" && msg.ToolCallID != "" {
			if _, exists := toolCallIDs[msg.ToolCallID]; exists {
				toolCallIDs[msg.ToolCallID] = true // response found
			}
		}
	}

	// Create error responses for orphaned tool calls
	for toolCallID, hasResponse := range toolCallIDs {
		if !hasResponse {
			logger.Warn("Found orphaned tool call without response in Anthropic integration", zap.String("tool_call_id", toolCallID))
			errorMessage := &entities.Message{
				ID:         uuid.New().String(),
				Role:       "tool",
				Content:    "Tool execution failed: No response generated",
				ToolCallID: toolCallID,
				Timestamp:  time.Now(),
			}
			messages = append(messages, errorMessage)
		}
	}

	return messages
}

// extractDiffFromResult extracts diff from FileWrite tool result
func (m *AnthropicIntegration) extractDiffFromResult(result string) string {
	var resultData struct {
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal([]byte(result), &resultData); err == nil && resultData.Diff != "" {
		return resultData.Diff
	}
	return ""
}

// GetUsage returns usage information
func (m *AnthropicIntegration) GetUsage() (*entities.Usage, error) {
	return m.lastUsage, nil
}

// Ensure AnthropicIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*AnthropicIntegration)(nil)
