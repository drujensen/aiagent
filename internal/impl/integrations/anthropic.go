package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
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
		httpClient: &http.Client{Timeout: 30 * time.Minute},
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

// GenerateResponse generates a response from the Anthropic API with incremental saving
func (m *AnthropicIntegration) GenerateResponse(ctx context.Context, messages []*entities.Message, toolList []entities.Tool, options map[string]any, callback interfaces.MessageCallback) ([]*entities.Message, error) {
	// Prepare tool definitions for Anthropic
	tools := make([]map[string]any, len(toolList))
	for i, tool := range toolList {
		// Check for cancellation
		if ctx.Err() != nil {
			return nil, fmt.Errorf("operation canceled by user")
		}

		tools[i] = map[string]any{
			"name":         tool.Name(),
			"description":  tool.Description(),
			"input_schema": tool.Schema(),
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

				// Check for context window errors on any error status
				if contextErr := m.parseAnthropicContextError(body); contextErr != nil {
					return nil, contextErr
				}

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

		// Only continue if stop_reason indicates tool use
		if responseBody.StopReason == "tool_use" {
			toolCallMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   textContent,
				ToolCalls: toolCalls,
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, toolCallMessage)

			// Save incrementally if callback is provided
			if callback != nil {
				if err := callback([]*entities.Message{toolCallMessage}); err != nil {
					m.logger.Error("Failed to save tool call message incrementally", zap.Error(err))
				}
			}

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
					result, err := tool.Execute(toolCall.Function.Arguments)
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
				// Use raw tool result for both AI and UI display
				var content string
				if toolError != "" {
					content = fmt.Sprintf("Tool %s failed with error: %s", toolName, toolError)
				} else {
					content = toolResult
				}

				// Create tool call event with raw result for UI formatting
				toolEvent := entities.NewToolCallEvent(toolCall.ID, toolName, toolCall.Function.Arguments, content, toolError, diff, nil)

				// Publish real-time event for TUI updates
				events.PublishToolCallEvent(toolEvent)

				toolResponseMessage := &entities.Message{
					ID:             uuid.New().String(),
					Role:           "tool",
					Content:        content,
					ToolCallID:     toolCall.ID,
					ToolCallEvents: []entities.ToolCallEvent{*toolEvent},
					Timestamp:      time.Now(),
				}
				newMessages = append(newMessages, toolResponseMessage)

				// Save incrementally if callback is provided
				if callback != nil {
					if err := callback([]*entities.Message{toolResponseMessage}); err != nil {
						m.logger.Error("Failed to save tool response message incrementally", zap.Error(err))
					}
				}

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
		} else {
			// Any other stop_reason is treated as final
			finalMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   textContent,
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, finalMessage)

			// Save incrementally if callback is provided
			if callback != nil {
				if err := callback([]*entities.Message{finalMessage}); err != nil {
					m.logger.Error("Failed to save final message incrementally", zap.Error(err))
				}
			}

			break
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

// GetLastUsage returns the usage from the last API call
func (m *AnthropicIntegration) GetLastUsage() (*entities.Usage, error) {
	return m.lastUsage, nil
}

// parseAnthropicContextError checks if the error response is related to context window limits
func (m *AnthropicIntegration) parseAnthropicContextError(body []byte) error {
	// Try to parse as structured error first
	var errorResp struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil {
		m.logger.Debug("Parsed structured Anthropic error", zap.String("type", errorResp.Type), zap.String("error_type", errorResp.Error.Type), zap.String("message", errorResp.Error.Message))

		// Check for context-related errors regardless of error type
		errMsg := strings.ToLower(errorResp.Error.Message)
		if strings.Contains(errMsg, "too long") ||
			strings.Contains(errMsg, "token limit") ||
			strings.Contains(errMsg, "context") ||
			strings.Contains(errMsg, "maximum length") ||
			strings.Contains(errMsg, "context_length_exceeded") ||
			strings.Contains(errMsg, "prompt is too long") ||
			strings.Contains(errMsg, "input too long") {
			return errors.ContextWindowErrorf("Anthropic context window exceeded: %s", errorResp.Error.Message)
		}

		// Also check for system errors that might indicate context issues
		if errorResp.Type == "error" && (errorResp.Error.Type == "system_error" || errorResp.Error.Type == "internal_error") {
			if strings.Contains(errMsg, "context") || strings.Contains(errMsg, "token") || strings.Contains(errMsg, "length") {
				return errors.ContextWindowErrorf("Anthropic system error (likely context): %s", errorResp.Error.Message)
			}
		}
	} else {
		// If not structured JSON, check if it's a raw error message that contains context-related text
		bodyStr := strings.ToLower(string(body))
		m.logger.Debug("Checking raw Anthropic error for context issues", zap.String("body", bodyStr))

		if strings.Contains(bodyStr, "too long") ||
			strings.Contains(bodyStr, "token limit") ||
			strings.Contains(bodyStr, "context") ||
			strings.Contains(bodyStr, "maximum length") ||
			strings.Contains(bodyStr, "context_length_exceeded") ||
			strings.Contains(bodyStr, "prompt is too long") ||
			strings.Contains(bodyStr, "input too long") ||
			strings.Contains(bodyStr, "context window") {
			return errors.ContextWindowErrorf("Anthropic context window exceeded (raw error): %s", string(body))
		}
	}

	return nil
}

// Ensure AnthropicIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*AnthropicIntegration)(nil)
