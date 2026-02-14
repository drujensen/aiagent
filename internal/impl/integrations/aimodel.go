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

// AIModelIntegration implements the base OpenAI-compatible API
type AIModelIntegration struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	model      string
	toolRepo   interfaces.ToolRepository
	logger     *zap.Logger
	lastUsage  *entities.Usage
}

// NewAIModelIntegration creates a new base integration for OpenAI-compatible APIs
func NewAIModelIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*AIModelIntegration, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}
	if model == "" {
		return nil, fmt.Errorf("model cannot be empty")
	}
	return &AIModelIntegration{
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
func (m *AIModelIntegration) ModelName() string {
	m.logger.Info("Using OpenAI-compatible model", zap.String("model", m.model))
	return m.model
}

// ProviderType returns the type of provider
func (m *AIModelIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderOpenAI
}

// convertToOpenAIMessages converts message entities to OpenAI API format
func convertToOpenAIMessages(messages []*entities.Message) []map[string]any {
	apiMessages := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		apiMsg := map[string]any{
			"role": msg.Role,
		}

		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			apiMsg["tool_calls"] = msg.ToolCalls
		} else if msg.Role == "tool" {
			apiMsg["tool_call_id"] = msg.ToolCallID
			apiMsg["content"] = msg.Content
		} else {
			apiMsg["content"] = msg.Content
		}

		apiMessages = append(apiMessages, apiMsg)
	}
	return apiMessages
}

// GenerateResponse generates a response from the OpenAI-compatible API with incremental saving
func (m *AIModelIntegration) GenerateResponse(ctx context.Context, messages []*entities.Message, toolList []*entities.Tool, options map[string]any, callback interfaces.MessageCallback) ([]*entities.Message, error) {
	// Prepare tool definitions for OpenAI
	tools := make([]map[string]any, len(toolList))
	for i, tool := range toolList {
		// Check for cancellation
		if ctx.Err() != nil {
			return nil, fmt.Errorf("operation canceled by user")
		}

		tools[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        (*tool).Name(),
				"description": (*tool).Description(),
				"parameters":  (*tool).Schema(),
			},
		}
	}

	// Format request body
	reqBody := map[string]any{
		"model":      m.model,
		"messages":   convertToOpenAIMessages(messages),
		"max_tokens": options["max_tokens"],
	}
	if temp, ok := options["temperature"]; ok {
		reqBody["temperature"] = temp
	}
	if len(tools) > 0 {
		reqBody["tools"] = tools
	}

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
		m.logger.Info("Sending request to OpenAI-compatible API", zap.String("body", string(jsonBody)))

		req, err := http.NewRequestWithContext(ctx, "POST", m.baseURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+m.apiKey)

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
				m.logger.Error("OpenAI-compatible API error",
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
		m.logger.Info("OpenAI-compatible response", zap.String("body", string(respBody)))

		// Parse response
		var responseBody struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int    `json:"created"`
			Model   string `json:"model"`
			Choices []struct {
				Index   int `json:"index"`
				Message struct {
					Role      string              `json:"role"`
					Content   string              `json:"content"`
					ToolCalls []entities.ToolCall `json:"tool_calls"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(respBody, &responseBody); err != nil {
			return nil, fmt.Errorf("error decoding response: %v", err)
		}

		if len(responseBody.Choices) == 0 {
			return nil, fmt.Errorf("no choices in response")
		}

		choice := responseBody.Choices[0]
		message := choice.Message

		// Track usage
		m.lastUsage.PromptTokens = responseBody.Usage.PromptTokens
		m.lastUsage.CompletionTokens = responseBody.Usage.CompletionTokens
		m.lastUsage.TotalTokens = responseBody.Usage.TotalTokens

		// Log tool calls
		if len(message.ToolCalls) > 0 {
			m.logger.Info("Tool calls generated", zap.Any("toolCalls", message.ToolCalls))
		} else {
			m.logger.Info("No tool calls generated")
		}

		// Only continue if finish_reason indicates tool calls
		if choice.FinishReason == "tool_calls" {
			toolCallMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   message.Content,
				ToolCalls: message.ToolCalls,
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, toolCallMessage)

			// Save incrementally if callback is provided
			if callback != nil {
				if err := callback([]*entities.Message{toolCallMessage}); err != nil {
					m.logger.Error("Failed to save tool call message incrementally", zap.Error(err))
				}
			}

			for _, toolCall := range message.ToolCalls {
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

				// Append tool result to messages for next iteration
				reqBody["messages"] = append(reqBody["messages"].([]map[string]any), map[string]any{
					"role":       "assistant",
					"content":    message.Content,
					"tool_calls": []entities.ToolCall{toolCall},
				})
				reqBody["messages"] = append(reqBody["messages"].([]map[string]any), map[string]any{
					"role":         "tool",
					"content":      toolResult,
					"tool_call_id": toolCall.ID,
				})
			}
		} else {
			// Any other finish_reason is treated as final
			finalMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   message.Content,
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
	newMessages = ensureToolCallResponsesOpenAI(newMessages, m.logger)

	m.logger.Info("Generated messages", zap.Any("messages", newMessages))
	return newMessages, nil
}

// ensureToolCallResponsesOpenAI validates that every tool call has a corresponding response
// and creates error responses for any orphaned tool calls
func ensureToolCallResponsesOpenAI(messages []*entities.Message, logger *zap.Logger) []*entities.Message {
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
			logger.Warn("Found orphaned tool call without response in OpenAI integration", zap.String("tool_call_id", toolCallID))
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
func (m *AIModelIntegration) extractDiffFromResult(result string) string {
	var resultData struct {
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal([]byte(result), &resultData); err == nil && resultData.Diff != "" {
		return resultData.Diff
	}
	return ""
}

// GetUsage returns usage information
func (m *AIModelIntegration) GetUsage() (*entities.Usage, error) {
	return m.lastUsage, nil
}

// GetLastUsage returns the usage from the last API call
func (m *AIModelIntegration) GetLastUsage() (*entities.Usage, error) {
	return m.lastUsage, nil
}

// Ensure AIModelIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*AIModelIntegration)(nil)
