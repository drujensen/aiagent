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

// AIModelIntegration implements the Base API
type AIModelIntegration struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	model      string
	toolRepo   interfaces.ToolRepository
	logger     *zap.Logger
	lastUsage  *entities.Usage
}

// NewAIModelIntegration creates a new Base integration
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

// For a generic Base-compatible API
func NewGenericAIModelIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*AIModelIntegration, error) {
	return NewAIModelIntegration(baseURL, apiKey, model, toolRepo, logger)
}

// ModelName returns the name of the model being used
func (m *AIModelIntegration) ModelName() string {
	return m.model
}

// ProviderType returns the type of provider
func (m *AIModelIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderGeneric
}

// convertToBaseMessages converts message entities to the Base API format
func convertToBaseMessages(messages []*entities.Message) []map[string]any {
	apiMessages := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		apiMsg := map[string]any{
			"role": msg.Role,
		}
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			apiMsg["tool_calls"] = msg.ToolCalls

			if msg.Content == "" {
				apiMsg["content"] = "Executing tool call."
			} else {
				apiMsg["content"] = msg.Content
			}
		} else {
			apiMsg["content"] = msg.Content
			if msg.Role == "tool" {
				apiMsg["tool_call_id"] = msg.ToolCallID
			}
		}
		apiMessages = append(apiMessages, apiMsg)
	}
	return apiMessages
}

func (m *AIModelIntegration) GenerateResponse(ctx context.Context, messages []*entities.Message, toolList []*entities.Tool, options map[string]any, callback interfaces.MessageCallback) ([]*entities.Message, error) {
	if ctx.Err() != nil {
		// Return empty results for early cancellation (no work done yet)
		return []*entities.Message{}, nil
	}

	tools := make([]map[string]any, len(toolList))
	for i, tool := range toolList {
		if ctx.Err() != nil {
			// Return empty results for early cancellation (no work done yet)
			return []*entities.Message{}, nil
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
			if param.Type == "array" && len(param.Items) > 0 {
				property["items"] = map[string]any{
					"type": param.Items[0].Type,
				}
			}
			properties[param.Name] = property
		}

		tools[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        (*tool).Name(),
				"description": (*tool).FullDescription(),
				"parameters": map[string]any{
					"type":       "object",
					"properties": properties,
					"required":   requiredFields,
				},
				"strict": true,
			},
		}
	}

	reqBody := map[string]any{
		"model": m.model,
	}
	if len(tools) > 0 {
		reqBody["tools"] = tools
	}

	for key, value := range options {
		reqBody[key] = value
	}

	apiMessages := convertToBaseMessages(messages)
	reqBody["messages"] = apiMessages

	var newMessages []*entities.Message
	iterationCount := 0
	const maxIterations = 25 // Reasonable limit for most tasks
	var lastAssistantContents []string
	consecutiveFailures := 0
	const maxConsecutiveFailures = 3 // Stop after 3 consecutive tool failures

	for {
		iterationCount++
		m.logger.Info("Starting AI processing iteration", zap.Int("iteration", iterationCount))

		// Prevent infinite loops by limiting iterations
		if iterationCount > maxIterations {
			m.logger.Warn("Maximum iterations reached, ending processing", zap.Int("maxIterations", maxIterations))
			finalMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   fmt.Sprintf("Processing stopped after %d iterations to prevent infinite loops. This may indicate the task is too complex or the AI is stuck. Try breaking down the task into smaller steps.", maxIterations),
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, finalMessage)
			if callback != nil {
				if err := callback([]*entities.Message{finalMessage}); err != nil {
					m.logger.Error("Failed to save safeguard message incrementally", zap.Error(err))
				}
			}
			break
		}

		if ctx.Err() == context.Canceled {
			m.logger.Info("Processing canceled by user", zap.Int("iteration", iterationCount))
			return nil, fmt.Errorf("operation canceled by user")
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request: %v", err)
		}

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
				if ctx.Err() == context.Canceled {
					return nil, fmt.Errorf("operation canceled by user")
				}
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
				m.logger.Error("Unexpected status code", zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
				return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
			}
			break
		}
		defer resp.Body.Close()

		var responseBody struct {
			Choices []struct {
				FinishReason string `json:"finish_reason"`
				Message      struct {
					Content   string              `json:"content"`
					ToolCalls []entities.ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response: %v", err)
		}

		var content string
		var toolCalls []entities.ToolCall

		// Try standard decoding first
		if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&responseBody); err != nil {
			return nil, fmt.Errorf("error decoding response: %v", err)
		}
		if len(responseBody.Choices) == 0 {
			return nil, fmt.Errorf("no choices in response")
		}

		content = responseBody.Choices[0].Message.Content
		toolCalls = responseBody.Choices[0].Message.ToolCalls

		// Store usage data
		m.lastUsage.PromptTokens = responseBody.Usage.PromptTokens
		m.lastUsage.CompletionTokens = responseBody.Usage.CompletionTokens
		m.lastUsage.TotalTokens = responseBody.Usage.TotalTokens

		finishReason := responseBody.Choices[0].FinishReason

		// Validate content for malformed tool call syntax
		if strings.Contains(content, "<xai:function_call>") || strings.Contains(content, `{"content">`) ||
			strings.Contains(content, "<parameter") || strings.Contains(content, "</parameter") ||
			strings.Contains(content, "\u003c/") || strings.Contains(content, "\u003e") {
			m.logger.Warn("Detected malformed tool call syntax in AI response, sanitizing",
				zap.String("content", content))
			// Remove specific malformed patterns to prevent model confusion
			content = sanitizeMalformedContent(content)
			// Clear tool calls since content was malformed
			toolCalls = nil
			finishReason = "stop"
		}

		// Check for repetitive responses to prevent infinite loops
		// Only check non-empty content, as empty content with tool calls is legitimate exploration
		if strings.TrimSpace(content) != "" {
			lastAssistantContents = append(lastAssistantContents, content)
			if len(lastAssistantContents) > 5 {
				lastAssistantContents = lastAssistantContents[1:] // keep last 5
			}
			if len(lastAssistantContents) == 5 && lastAssistantContents[0] == lastAssistantContents[1] && lastAssistantContents[1] == lastAssistantContents[2] && lastAssistantContents[2] == lastAssistantContents[3] && lastAssistantContents[3] == lastAssistantContents[4] {
				m.logger.Warn("Detected repetitive AI responses, ending processing to prevent infinite loop")
				finalMessage := &entities.Message{
					ID:        uuid.New().String(),
					Role:      "assistant",
					Content:   "Processing stopped due to repetitive AI responses. The AI appears to be stuck in a loop. Try rephrasing your request or breaking it into smaller steps.",
					Timestamp: time.Now(),
				}
				newMessages = append(newMessages, finalMessage)
				if callback != nil {
					if err := callback([]*entities.Message{finalMessage}); err != nil {
						m.logger.Error("Failed to save safeguard message incrementally", zap.Error(err))
					}
				}
				break
			}
		}

		m.logger.Info("AI response analysis",
			zap.String("finishReason", finishReason),
			zap.Int("toolCallsCount", len(toolCalls)),
			zap.Bool("hasContent", content != ""),
			zap.Int("iteration", iterationCount))

		// Handle different finish_reason values
		shouldBreak := false
		var finalContent string

		switch finishReason {
		case "tool_calls":
			m.logger.Info("AI requested tool calls - continuing processing",
				zap.Int("toolCallsCount", len(toolCalls)))
			shouldBreak = false
		case "stop":
			m.logger.Info("AI finished with stop - ending processing")
			finalContent = content
			shouldBreak = true
		case "length":
			m.logger.Warn("AI finished due to length limit - ending processing")
			finalContent = content + "\n\n[Response truncated due to length limit]"
			shouldBreak = true
		case "content_filter":
			m.logger.Warn("AI finished due to content filter - ending processing")
			finalContent = content + "\n\n[Response filtered by content policy]"
			shouldBreak = true
		default:
			m.logger.Warn("Unknown finish_reason - treating as completion",
				zap.String("finishReason", finishReason))
			finalContent = content
			shouldBreak = true
		}

		if shouldBreak {
			finalMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   finalContent,
				Timestamp: time.Now(),
			}

			// Add usage information to the final assistant message
			if m.lastUsage != nil && (m.lastUsage.PromptTokens > 0 || m.lastUsage.CompletionTokens > 0) {
				finalMessage.AddUsage(m.lastUsage.PromptTokens, m.lastUsage.CompletionTokens, 0, 0) // Cost will be calculated later by chat service
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

		// If we didn't break above, there are tool calls to process
		if len(toolCalls) == 0 {
			// This shouldn't happen, but break to be safe
			break
		}

		contentMsg := "Executing tool call."
		if len(toolCalls) > 0 {
			contentMsg = fmt.Sprintf("Executing %s tool with parameters: %s", toolCalls[0].Function.Name, toolCalls[0].Function.Arguments)
		}
		toolCallMessage := &entities.Message{
			ID:        uuid.New().String(),
			Role:      "assistant",
			Content:   contentMsg,
			ToolCalls: toolCalls,
			Timestamp: time.Now(),
		}

		// Add usage information to the tool call message
		if m.lastUsage != nil && (m.lastUsage.PromptTokens > 0 || m.lastUsage.CompletionTokens > 0) {
			toolCallMessage.AddUsage(m.lastUsage.PromptTokens, m.lastUsage.CompletionTokens, 0, 0) // Cost will be calculated later by chat service
		}

		newMessages = append(newMessages, toolCallMessage)

		// Save incrementally if callback is provided
		if callback != nil {
			if err := callback([]*entities.Message{toolCallMessage}); err != nil {
				m.logger.Error("Failed to save tool call message incrementally", zap.Error(err))
			}
		}

		assistantMsg := map[string]any{
			"role":       "assistant",
			"content":    contentMsg,
			"tool_calls": toolCalls,
		}
		apiMessages = append(apiMessages, assistantMsg)

		for _, toolCall := range toolCalls {
			if ctx.Err() == context.Canceled {
				// Return partial results - validate completed tool calls and return what we have
				newMessages = ensureToolCallResponses(newMessages, m.logger)
				return newMessages, nil
			}

			if toolCall.Type != "function" {
				continue
			}
			toolName := toolCall.Function.Name

			var toolResult string
			var toolError string
			var diff string

			tool, err := m.toolRepo.GetToolByName(toolName)
			if err != nil {
				m.logger.Warn("Tool not found, treating as execution error", zap.String("toolName", toolName), zap.Error(err))
				toolResult = fmt.Sprintf("Tool %s not found: %v", toolName, err)
				toolError = err.Error()
			} else if tool != nil {
				// Validate tool arguments length to prevent excessive token usage
				const maxArgLength = 10000 // 10KB limit
				if len(toolCall.Function.Arguments) > maxArgLength {
					m.logger.Warn("Tool arguments too long, truncating",
						zap.String("toolName", toolName),
						zap.Int("originalLength", len(toolCall.Function.Arguments)),
						zap.Int("maxLength", maxArgLength))
					toolCall.Function.Arguments = toolCall.Function.Arguments[:maxArgLength] + "...[truncated]"
				}

				m.logger.Info("Executing tool", zap.String("toolName", toolName))
				result, err := (*tool).Execute(toolCall.Function.Arguments)
				if err != nil {
					m.logger.Error("Tool execution failed",
						zap.String("toolName", toolName),
						zap.Error(err))
					// Return JSON error response for consistency
					errorResponse := map[string]interface{}{
						"summary": fmt.Sprintf("❌ Tool %s failed: %s", toolName, err.Error()),
						"success": false,
						"error":   err.Error(),
					}
					errorJson, jsonErr := json.Marshal(errorResponse)
					if jsonErr != nil {
						// Fallback to plain text if JSON marshaling fails
						toolResult = fmt.Sprintf("Tool %s execution failed: %v", toolName, err)
					} else {
						toolResult = string(errorJson)
					}
					toolError = err.Error()
					consecutiveFailures++
				} else {
					m.logger.Info("Tool executed successfully",
						zap.String("toolName", toolName),
						zap.Int("resultLength", len(result)))
					toolResult = result
					// Extract diff if it's a file write operation
					if toolName == "FileWrite" {
						diff = m.extractDiffFromResult(result)
					}
					consecutiveFailures = 0 // Reset on success
				}
			}
			// Extract full data for AI and summary for TUI
			fullContent, summaryContent := m.extractToolContent(toolName, toolResult, toolError)

			// Create tool call event with summary for TUI
			toolEvent := entities.NewToolCallEvent(toolCall.ID, toolName, toolCall.Function.Arguments, summaryContent, toolError, diff, nil)

			// Publish real-time event for TUI updates (Web UI uses message history refresh)
			events.PublishToolCallEvent(toolEvent)

			var content string
			if toolError != "" {
				content = fmt.Sprintf("Tool %s failed with error: %s", toolName, toolError)
			} else {
				content = fullContent // Full data for AI reasoning
			}

			var newMessage = &entities.Message{
				ID:             uuid.New().String(),
				Role:           "tool",
				Content:        content,
				ToolCallID:     toolCall.ID,
				ToolCallEvents: []entities.ToolCallEvent{*toolEvent},
				Timestamp:      time.Now(),
			}
			newMessages = append(newMessages, newMessage)

			// Save incrementally if callback is provided
			if callback != nil {
				if err := callback([]*entities.Message{newMessage}); err != nil {
					m.logger.Error("Failed to save tool response message incrementally", zap.Error(err))
				}
			}

			toolResponseMsg := map[string]any{
				"role":         "tool",
				"content":      content,
				"tool_call_id": toolCall.ID,
			}
			apiMessages = append(apiMessages, toolResponseMsg)
		}

		// Check for too many consecutive tool failures
		if consecutiveFailures >= maxConsecutiveFailures {
			m.logger.Warn("Too many consecutive tool failures, ending processing",
				zap.Int("consecutiveFailures", consecutiveFailures),
				zap.Int("maxConsecutiveFailures", maxConsecutiveFailures))
			finalMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   fmt.Sprintf("Processing stopped due to %d consecutive tool failures. The tools may be malfunctioning or the task may be impossible to complete. Try breaking down the task into smaller steps.", consecutiveFailures),
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, finalMessage)
			if callback != nil {
				if err := callback([]*entities.Message{finalMessage}); err != nil {
					m.logger.Error("Failed to save consecutive failure message incrementally", zap.Error(err))
				}
			}
			break
		}

		reqBody["messages"] = apiMessages

		m.logger.Info("Completed iteration, preparing for next AI call",
			zap.Int("iteration", iterationCount),
			zap.Int("totalMessages", len(apiMessages)))
	}

	// Validate that all tool calls have responses before returning
	newMessages = ensureToolCallResponses(newMessages, m.logger)

	return newMessages, nil
}

// ensureToolCallResponses validates that every tool call has a corresponding response
// and creates error responses for any orphaned tool calls
func ensureToolCallResponses(messages []*entities.Message, logger *zap.Logger) []*entities.Message {
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
			logger.Warn("Found orphaned tool call without response in AI integration", zap.String("tool_call_id", toolCallID))
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
	// First, try to extract from top-level diff field
	var resultData struct {
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal([]byte(result), &resultData); err == nil && resultData.Diff != "" {
		return resultData.Diff
	}

	// If that fails, try to parse as a generic map and look for diff field
	var genericData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &genericData); err == nil {
		if diff, ok := genericData["diff"]; ok {
			if diffStr, ok := diff.(string); ok && diffStr != "" {
				return diffStr
			}
		}
	}

	// If still no diff found, check if the result contains diff-like content
	// (unified diff format starts with --- or +++ or @@)
	if strings.Contains(result, "---") || strings.Contains(result, "+++") || strings.Contains(result, "@@") {
		return result
	}

	return ""
}

// GetUsage returns the token usage statistics
func (m *AIModelIntegration) GetUsage() (*entities.Usage, error) {
	return m.lastUsage, nil
}

// sanitizeMalformedContent removes specific malformed tool call patterns that confuse the AI
func sanitizeMalformedContent(content string) string {
	// Remove specific malformed patterns seen in AI responses
	content = strings.ReplaceAll(content, "<xai:function_call>", "")
	content = strings.ReplaceAll(content, "</xai:function_call", "")
	content = strings.ReplaceAll(content, "<parameter", "")
	content = strings.ReplaceAll(content, "</parameter", "")
	content = strings.ReplaceAll(content, `{"content">`, "")
	// Remove common malformed tag patterns
	content = strings.ReplaceAll(content, "\u003c/", "") // </
	content = strings.ReplaceAll(content, "\u003e", "")  // >
	return content
}

// extractToolContent extracts full content for AI and summary for TUI from tool results
func (m *AIModelIntegration) extractToolContent(toolName, result, toolError string) (fullContent, summaryContent string) {
	// Try to parse as JSON (both success and error responses are now JSON)
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &jsonData); err != nil {
		// Fallback: not JSON, use result as-is (shouldn't happen with updated tools)
		if toolError != "" {
			// Sanitize error message to prevent model confusion from malformed patterns
			sanitizedError := sanitizeMalformedContent(toolError)
			fullContent = fmt.Sprintf("ERROR: Tool %s failed with error: %s", toolName, sanitizedError)
			summaryContent = fullContent
		} else {
			fullContent = fmt.Sprintf("SUCCESS: Tool %s completed: %s", toolName, result)
			summaryContent = result
		}
		return
	}

	// Extract summary for TUI
	if summary, ok := jsonData["summary"].(string); ok {
		summaryContent = summary
	} else {
		summaryContent = result
	}

	// For AI, return the JSON directly to allow structured parsing
	fullContent = result

	return
}

// Ensure AIModelIntegration implements the AIModelIntegration interface
var _ interfaces.AIModelIntegration = (*AIModelIntegration)(nil)
