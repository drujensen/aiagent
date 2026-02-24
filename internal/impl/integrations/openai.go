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
	"github.com/drujensen/aiagent/internal/domain/events"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// OpenAIIntegration implements the OpenAI API
// For now, we'll use Base implementation as OpenAI uses an OpenAI-compatible API,
// but in the future this could have OpenAI-specific customizations
type OpenAIIntegration struct {
	*AIModelIntegration
}

// NewOpenAIIntegration creates a new OpenAI integration
func NewOpenAIIntegration(baseURL, apiKey string, model *entities.Model, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*OpenAIIntegration, error) {
	// Determine endpoint based on model family and name
	// All o-series models and codex models use the responses API
	endpoint := baseURL + "/v1/chat/completions"

	modelNameLower := strings.ToLower(model.ModelName)
	familyLower := strings.ToLower(model.Family)
	if strings.Contains(modelNameLower, "codex") ||
		strings.Contains(modelNameLower, "o1") ||
		strings.Contains(modelNameLower, "o3") ||
		strings.Contains(modelNameLower, "o4") ||
		familyLower == "o" ||
		strings.HasPrefix(familyLower, "o") {
		// All o-series models (o1, o3, o4, etc.) and codex models use responses API
		endpoint = baseURL + "/v1/responses"
	}
	// GPT models use chat completions API

	openAIIntegration, err := NewAIModelIntegration(endpoint, apiKey, model.ModelName, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &OpenAIIntegration{
		AIModelIntegration: openAIIntegration,
	}, nil
}

// GenerateResponse generates a response using the appropriate OpenAI API based on the endpoint
func (m *OpenAIIntegration) GenerateResponse(ctx context.Context, messages []*entities.Message, toolList []entities.Tool, options map[string]any, callback interfaces.MessageCallback) ([]*entities.Message, error) {
	// Check if this is a responses API endpoint (for all o-series and codex models)
	if strings.Contains(m.baseURL, "/v1/responses") {
		return m.generateResponseV2(ctx, messages, toolList, options, callback)
	}

	// Use the standard chat completions API for GPT models
	return m.AIModelIntegration.GenerateResponse(ctx, messages, toolList, options, callback)
}

// generateResponseV2 handles the /v1/responses API for o1 and codex models with proper tool call support
func (m *OpenAIIntegration) generateResponseV2(ctx context.Context, messages []*entities.Message, toolList []entities.Tool, options map[string]any, callback interfaces.MessageCallback) ([]*entities.Message, error) {
	// Prepare tool definitions for OpenAI responses API (flattened format)
	tools := make([]map[string]any, len(toolList))
	for i, tool := range toolList {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("operation canceled by user")
		}

		tools[i] = map[string]any{
			"type":        "function",
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  tool.Schema(),
		}
	}

	// Convert initial messages to input format and extract instructions
	inputItems, instructions := m.convertMessagesToInputItems(messages)

	var allMessages []*entities.Message
	var previousResponseID string
	var lastUsage struct {
		InputTokens     int `json:"input_tokens"`
		OutputTokens    int `json:"output_tokens"`
		TotalTokens     int `json:"total_tokens"`
		ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	}

	// Tool call execution loop
	for {
		// Check for cancellation
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("operation canceled by user")
		}

		// Format request body for /v1/responses API
		reqBody := map[string]any{
			"model": m.model,
			"input": inputItems,
		}
		if instructions != "" {
			reqBody["instructions"] = instructions
		}
		if maxTokens, ok := options["max_tokens"]; ok {
			reqBody["max_output_tokens"] = maxTokens
		}
		// Note: temperature is not supported for o-series models using /v1/responses API
		if len(tools) > 0 {
			reqBody["tools"] = tools
			reqBody["tool_choice"] = "auto"
		}
		if instructions != "" {
			reqBody["instructions"] = instructions
		}
		if maxTokens, ok := options["max_tokens"]; ok {
			reqBody["max_output_tokens"] = maxTokens
		}
		if len(tools) > 0 {
			reqBody["tools"] = tools
			reqBody["tool_choice"] = "auto"
		}
		if previousResponseID != "" {
			reqBody["previous_response_id"] = previousResponseID
		}

		// Make the API request
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request: %v", err)
		}
		m.logger.Info("Sending request to OpenAI /v1/responses API", zap.String("body", string(jsonBody)))

		req, err := http.NewRequestWithContext(ctx, "POST", m.baseURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+m.apiKey)

		var resp *http.Response
		for attempt := 0; attempt < 3; attempt++ {
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
				m.logger.Error("OpenAI /v1/responses API error",
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
		m.logger.Info("OpenAI /v1/responses response", zap.String("body", string(respBody)))

		// Parse response from /v1/responses API
		var responseBody struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created_at"`
			Status  string `json:"status"`
			Model   string `json:"model"`
			Output  []struct {
				ID      string      `json:"id,omitempty"`
				Type    string      `json:"type"`
				Status  string      `json:"status,omitempty"`
				Text    string      `json:"text,omitempty"`
				Summary interface{} `json:"summary,omitempty"`
				Content []struct {
					Type        string      `json:"type,omitempty"`
					Text        string      `json:"text,omitempty"`
					Annotations interface{} `json:"annotations,omitempty"`
					Logprobs    interface{} `json:"logprobs,omitempty"`
				} `json:"content,omitempty"`
				// Function call specific fields
				CallID    string `json:"call_id,omitempty"`
				Name      string `json:"name,omitempty"`
				Arguments string `json:"arguments,omitempty"`
				Role      string `json:"role,omitempty"`
			} `json:"output"`
			Usage struct {
				InputTokens     int `json:"input_tokens"`
				OutputTokens    int `json:"output_tokens"`
				TotalTokens     int `json:"total_tokens"`
				ReasoningTokens int `json:"reasoning_tokens,omitempty"`
			} `json:"usage"`
		}

		if err := json.Unmarshal(respBody, &responseBody); err != nil {
			return nil, fmt.Errorf("error decoding response: %v", err)
		}

		if len(responseBody.Output) == 0 {
			m.logger.Error("Received response with empty output array",
				zap.String("response_id", responseBody.ID),
				zap.String("model", responseBody.Model))
			return nil, fmt.Errorf("response contains no output items")
		}

		// Store response ID for potential follow-up requests
		previousResponseID = responseBody.ID

		// Update usage tracking
		lastUsage = responseBody.Usage

		// Extract content and tool calls from output items
		var content strings.Builder
		var toolCalls []entities.ToolCall

		for i, item := range responseBody.Output {
			m.logger.Debug("Processing output item",
				zap.Int("index", i),
				zap.String("type", item.Type),
				zap.String("id", item.ID))

			switch item.Type {
			case "reasoning":
				// Extract reasoning content
				reasoningText := m.extractReasoningContent(item)
				if reasoningText != "" {
					if content.Len() > 0 {
						content.WriteString("\n")
					}
					content.WriteString(reasoningText)
				}
			case "message":
				// Extract text from message content
				messageText := m.extractMessageContent(item)
				if messageText != "" {
					if content.Len() > 0 {
						content.WriteString("\n")
					}
					content.WriteString(messageText)
				}
			case "function_call":
				// Parse function call
				if item.Name != "" && item.Arguments != "" {
					toolCall := entities.ToolCall{
						ID:   item.CallID,
						Type: "function",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      item.Name,
							Arguments: item.Arguments,
						},
					}
					toolCalls = append(toolCalls, toolCall)
					m.logger.Info("Parsed tool call", zap.String("name", item.Name), zap.String("call_id", item.CallID))
				} else {
					m.logger.Warn("Incomplete function_call item missing name or arguments",
						zap.String("item_id", item.ID),
						zap.String("call_id", item.CallID),
						zap.String("name", item.Name),
						zap.Bool("has_arguments", item.Arguments != ""))
				}
			default:
				// Try to extract text from any item that has it
				if item.Text != "" {
					if content.Len() > 0 {
						content.WriteString("\n")
					}
					content.WriteString(item.Text)
				}
			}
		}

		// Create assistant message if we have content or tool calls
		if content.Len() > 0 || len(toolCalls) > 0 {
			assistantMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   content.String(),
				ToolCalls: toolCalls,
				Timestamp: time.Now(),
			}
			allMessages = append(allMessages, assistantMessage)

			m.logger.Info("Created assistant message",
				zap.String("message_id", assistantMessage.ID),
				zap.Bool("has_content", content.Len() > 0),
				zap.Int("content_length", content.Len()),
				zap.Int("tool_call_count", len(toolCalls)))

			// Save incrementally if callback provided
			if callback != nil {
				if err := callback([]*entities.Message{assistantMessage}); err != nil {
					m.logger.Error("Failed to save assistant message incrementally", zap.Error(err))
				}
			}
		} else {
			m.logger.Warn("No content or tool calls found in response output",
				zap.Int("output_items", len(responseBody.Output)),
				zap.String("response_id", responseBody.ID))
		}

		// If there are tool calls, execute them and continue the loop
		if len(toolCalls) > 0 {
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
					// Inject session_id into todowrite tool arguments
					args := toolCall.Function.Arguments
					if toolName == "todowrite" {
						if sessionID, ok := options["session_id"].(string); ok && sessionID != "" {
							var argsMap map[string]any
							if err := json.Unmarshal([]byte(args), &argsMap); err == nil {
								if _, exists := argsMap["session_id"]; !exists {
									argsMap["session_id"] = sessionID
									if newArgs, err := json.Marshal(argsMap); err == nil {
										args = string(newArgs)
									}
								}
							}
						}
					}

					result, err := tool.Execute(args)
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
				var displayContent string
				if toolError != "" {
					displayContent = fmt.Sprintf("Tool %s failed with error: %s", toolName, toolError)
				} else {
					displayContent = toolResult
				}

				// Create tool call event
				toolEvent := entities.NewToolCallEvent(toolCall.ID, toolName, toolCall.Function.Arguments, displayContent, toolError, diff, nil)

				// Publish real-time event for TUI updates
				events.PublishToolCallEvent(toolEvent)

				// Create tool response message
				toolResponseMessage := &entities.Message{
					ID:             uuid.New().String(),
					Role:           "tool",
					Content:        displayContent,
					ToolCallID:     toolCall.ID,
					ToolCallEvents: []entities.ToolCallEvent{*toolEvent},
					Timestamp:      time.Now(),
				}
				allMessages = append(allMessages, toolResponseMessage)

				// Save incrementally if callback provided
				if callback != nil {
					if err := callback([]*entities.Message{toolResponseMessage}); err != nil {
						m.logger.Error("Failed to save tool response message incrementally", zap.Error(err))
					}
				}

				// Append tool result to input for next request
				toolInputItem := map[string]any{
					"type":    "function_call_output",
					"call_id": toolCall.ID,
					"output":  toolResult,
				}
				inputItems = append(inputItems, toolInputItem)
			}
			// Continue the loop to make another API call with tool results
		} else {
			// No more tool calls, exit the loop
			break
		}
	}

	// Update usage tracking with the last response
	if len(allMessages) > 0 {
		// Note: Usage tracking would need to be accumulated across multiple requests
		// For now, we'll track the last response's usage
		m.lastUsage.PromptTokens = lastUsage.InputTokens
		m.lastUsage.CompletionTokens = lastUsage.OutputTokens
		m.lastUsage.TotalTokens = lastUsage.TotalTokens
	}

	// Ensure tool call responses are validated
	allMessages = ensureToolCallResponsesOpenAIResponses(allMessages, m.logger)

	m.logger.Info("Generated messages from /v1/responses API", zap.Any("messages", allMessages))
	return allMessages, nil
}

// convertMessagesToInputItems converts message entities to /v1/responses input format
func (m *OpenAIIntegration) convertMessagesToInputItems(messages []*entities.Message) ([]map[string]any, string) {
	var inputItems []map[string]any
	var instructions string

	for _, msg := range messages {
		if msg.Role == "system" {
			// Extract system messages to instructions parameter
			if instructions != "" {
				instructions += "\n"
			}
			instructions += msg.Content
		} else if msg.Role == "tool" {
			// Skip tool messages for Responses API - tool results are handled via conversation state
			// The Responses API maintains tool call context through previous_response_id
			m.logger.Debug("Skipping tool message in Responses API input conversion",
				zap.String("tool_call_id", msg.ToolCallID))
		} else {
			// Convert to input item format for user/assistant messages
			item := map[string]any{
				"role":    msg.Role,
				"content": msg.Content,
			}
			inputItems = append(inputItems, item)
		}
	}

	return inputItems, instructions
}

// extractReasoningContent extracts text content from a reasoning output item
func (m *OpenAIIntegration) extractReasoningContent(item struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"`
	Status  string      `json:"status,omitempty"`
	Text    string      `json:"text,omitempty"`
	Summary interface{} `json:"summary,omitempty"`
	Content []struct {
		Type        string      `json:"type,omitempty"`
		Text        string      `json:"text,omitempty"`
		Annotations interface{} `json:"annotations,omitempty"`
		Logprobs    interface{} `json:"logprobs,omitempty"`
	} `json:"content,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Role      string `json:"role,omitempty"`
}) string {
	// Try summary first
	if item.Summary != nil {
		switch s := item.Summary.(type) {
		case string:
			if s != "" {
				return s
			}
		case []interface{}:
			if len(s) > 0 {
				var summaryParts []string
				for _, elem := range s {
					if str := fmt.Sprintf("%v", elem); str != "" {
						summaryParts = append(summaryParts, str)
					}
				}
				if len(summaryParts) > 0 {
					return strings.Join(summaryParts, " ")
				}
			}
		default:
			if str := fmt.Sprintf("%v", s); str != "" && str != "[]" {
				return str
			}
		}
	}

	// Try content field
	for _, contentItem := range item.Content {
		if contentItem.Type == "text" && contentItem.Text != "" {
			return contentItem.Text
		}
	}

	// Fallback to direct text field
	return item.Text
}

// extractMessageContent extracts text content from a message output item
func (m *OpenAIIntegration) extractMessageContent(item struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"`
	Status  string      `json:"status,omitempty"`
	Text    string      `json:"text,omitempty"`
	Summary interface{} `json:"summary,omitempty"`
	Content []struct {
		Type        string      `json:"type,omitempty"`
		Text        string      `json:"text,omitempty"`
		Annotations interface{} `json:"annotations,omitempty"`
		Logprobs    interface{} `json:"logprobs,omitempty"`
	} `json:"content,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Role      string `json:"role,omitempty"`
}) string {
	for _, contentItem := range item.Content {
		if contentItem.Type == "output_text" && contentItem.Text != "" {
			return contentItem.Text
		}
	}
	// Fallback to direct text field
	return item.Text
}

// extractDiffFromResult extracts diff from FileWrite tool result
func (m *OpenAIIntegration) extractDiffFromResult(result string) string {
	var resultData struct {
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal([]byte(result), &resultData); err == nil && resultData.Diff != "" {
		return resultData.Diff
	}
	return ""
}

// ensureToolCallResponsesOpenAIResponses validates that every tool call has a corresponding response
// and creates error responses for any orphaned tool calls (specific to Responses API)
func ensureToolCallResponsesOpenAIResponses(messages []*entities.Message, logger *zap.Logger) []*entities.Message {
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

// ProviderType returns the type of provider
func (m *OpenAIIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderOpenAI
}

// Ensure OpenAIIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*OpenAIIntegration)(nil)
