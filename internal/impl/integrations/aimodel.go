package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
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
			var toolCalls []map[string]any
			for _, tc := range msg.ToolCalls {
				tcMap := map[string]any{
					"id":   tc.ID,
					"type": tc.Type,
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				}
				toolCalls = append(toolCalls, tcMap)
			}
			apiMsg["tool_calls"] = toolCalls
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

// toolExecResult holds the outcome of a single tool call execution.
type toolExecResult struct {
	ToolCall    entities.ToolCall
	ToolName    string
	ToolResult  string // raw result sent back to the LLM
	Content     string // display content sent to the TUI / events
	ToolError   string
	Diff        string
	ToolEvent   *entities.ToolCallEvent
	ToolMessage *entities.Message
}

// injectToolArgs injects framework-managed fields (session_id, parent_chat_id) into
// a tool's JSON argument string. Fields that are already present are not overwritten.
func injectToolArgs(args, toolName, chatID string) string {
	if chatID == "" {
		return args
	}
	var m map[string]any
	if json.Unmarshal([]byte(args), &m) != nil {
		return args
	}
	changed := false
	if toolName == "TodoWrite" {
		if _, exists := m["session_id"]; !exists {
			m["session_id"] = chatID
			changed = true
		}
	}
	if _, exists := m["parent_chat_id"]; !exists {
		m["parent_chat_id"] = chatID
		changed = true
	}
	if !changed {
		return args
	}
	if b, err := json.Marshal(m); err == nil {
		return string(b)
	}
	return args
}

// extractDiffStatic extracts a diff string from a FileWrite tool result JSON.
func extractDiffStatic(result string) string {
	var d struct {
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal([]byte(result), &d); err == nil {
		return d.Diff
	}
	return ""
}

// executeToolsParallel runs all toolCalls concurrently, publishes ToolCallEvents
// in real-time as each tool completes, and returns results in the original order.
func executeToolsParallel(
	ctx context.Context,
	toolCalls []entities.ToolCall,
	toolRepo interfaces.ToolRepository,
	options map[string]any,
	logger *zap.Logger,
) []toolExecResult {
	chatID, _ := options["session_id"].(string)
	results := make([]toolExecResult, len(toolCalls))
	var wg sync.WaitGroup

	for i, toolCall := range toolCalls {
		wg.Add(1)
		go func(i int, toolCall entities.ToolCall) {
			defer wg.Done()

			toolName := toolCall.Function.Name
			args := injectToolArgs(toolCall.Function.Arguments, toolName, chatID)

			var toolResult, toolError, diff string
			tool, err := toolRepo.GetToolByName(toolName)
			if err != nil {
				toolResult = fmt.Sprintf("Tool %s could not be retrieved: %v", toolName, err)
				toolError = err.Error()
				logger.Warn("Failed to get tool", zap.String("toolName", toolName), zap.Error(err))
			} else if tool != nil {
				result, execErr := tool.Execute(ctx, args)
				if execErr != nil {
					toolResult = fmt.Sprintf("Tool %s execution failed: %v", toolName, execErr)
					toolError = execErr.Error()
					logger.Warn("Tool execution failed", zap.String("toolName", toolName), zap.Error(execErr))
				} else {
					toolResult = result
					if toolName == "Write" || toolName == "Edit" {
						diff = extractDiffStatic(result)
					}
				}
			} else {
				toolResult = fmt.Sprintf("Tool %s not found", toolName)
				toolError = "Tool not found"
				logger.Warn("Tool not found", zap.String("toolName", toolName))
			}

			content := toolResult
			if toolError != "" {
				content = fmt.Sprintf("Tool %s failed with error: %s", toolName, toolError)
			}

			toolEvent := entities.NewToolCallEvent(toolCall.ID, toolName, toolCall.Function.Arguments, content, toolError, diff, chatID, nil)
			events.PublishToolCallEvent(toolEvent)

			toolMessage := &entities.Message{
				ID:             uuid.New().String(),
				Role:           "tool",
				Content:        content,
				ToolCallID:     toolCall.ID,
				ToolCallEvents: []entities.ToolCallEvent{*toolEvent},
				Timestamp:      time.Now(),
			}

			results[i] = toolExecResult{
				ToolCall:    toolCall,
				ToolName:    toolName,
				ToolResult:  toolResult,
				Content:     content,
				ToolError:   toolError,
				Diff:        diff,
				ToolEvent:   toolEvent,
				ToolMessage: toolMessage,
			}
		}(i, toolCall)
	}

	wg.Wait()
	return results
}

// GenerateResponse generates a response from the OpenAI-compatible API with incremental saving
func (m *AIModelIntegration) GenerateResponse(ctx context.Context, messages []*entities.Message, toolList []entities.Tool, options map[string]any, callback interfaces.MessageCallback) ([]*entities.Message, error) {
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
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  tool.Schema(),
			},
		}
	}

	// Format request body
	reqBody := map[string]any{
		"model":    m.model,
		"messages": convertToOpenAIMessages(messages),
	}

	// Handle max tokens parameter (may be max_tokens or max_completion_tokens)
	if maxCompletionTokens, ok := options["max_completion_tokens"]; ok {
		reqBody["max_completion_tokens"] = maxCompletionTokens
	} else if maxTokens, ok := options["max_tokens"]; ok {
		reqBody["max_tokens"] = maxTokens
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
		req.Header.Set(m.authHeaderName(), m.authHeaderValue())

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

				// Check for context window errors
				if resp.StatusCode == http.StatusBadRequest {
					if contextErr := m.parseOpenAIContextError(body); contextErr != nil {
						return nil, contextErr
					}
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
					Role      string           `json:"role"`
					Content   string           `json:"content"`
					ToolCalls []map[string]any `json:"tool_calls"`
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

		// Parse tool calls
		var toolCalls []entities.ToolCall
		for _, tcMap := range message.ToolCalls {
			var tc entities.ToolCall
			if id, ok := tcMap["id"].(string); ok {
				tc.ID = m.customizeToolCallID(id)
			}
			if typ, ok := tcMap["type"].(string); ok {
				tc.Type = typ
			}
			if tc.Type == "" {
				tc.Type = "function"
			}
			if fn, ok := tcMap["function"].(map[string]any); ok {
				if name, ok := fn["name"].(string); ok {
					tc.Function.Name = name
				}
				if args, ok := fn["arguments"].(string); ok {
					tc.Function.Arguments = args
				}
			}

			toolCalls = append(toolCalls, tc)
		}
		m.lastUsage.PromptTokens = responseBody.Usage.PromptTokens
		m.lastUsage.CompletionTokens = responseBody.Usage.CompletionTokens
		m.lastUsage.TotalTokens = responseBody.Usage.TotalTokens

		// Log tool calls
		if len(toolCalls) > 0 {
			m.logger.Info("Tool calls generated", zap.Any("toolCalls", toolCalls))
		} else {
			m.logger.Info("No tool calls generated")
		}

		// Only continue if finish_reason indicates tool calls
		if choice.FinishReason == "tool_calls" {
			toolCallMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   message.Content,
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

			// Append the assistant message with tool calls to the request messages for the next API call
			assistantMessageAPI := convertToOpenAIMessages([]*entities.Message{toolCallMessage})[0]
			reqBody["messages"] = append(reqBody["messages"].([]map[string]any), assistantMessageAPI)

			// Execute all tool calls in parallel, then process results in order.
			toolResults := executeToolsParallel(ctx, toolCalls, m.toolRepo, options, m.logger)
			for _, r := range toolResults {
				newMessages = append(newMessages, r.ToolMessage)

				if callback != nil {
					if err := callback([]*entities.Message{r.ToolMessage}); err != nil {
						m.logger.Error("Failed to save tool response message incrementally", zap.Error(err))
					}
				}

				apiMsg := map[string]any{
					"role":         "tool",
					"content":      r.ToolResult,
					"tool_call_id": r.ToolCall.ID,
				}
				reqBody["messages"] = append(reqBody["messages"].([]map[string]any), apiMsg)
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

// customizeToolCallID allows providers to customize tool call IDs (e.g., for format requirements)
func (m *AIModelIntegration) customizeToolCallID(originalID string) string {
	return originalID
}

// authHeaderName returns the header name for authentication
func (m *AIModelIntegration) authHeaderName() string {
	return "Authorization"
}

// authHeaderValue returns the header value for authentication
func (m *AIModelIntegration) authHeaderValue() string {
	return "Bearer " + m.apiKey
}

// parseOpenAIContextError checks if the error response is related to context window limits
func (m *AIModelIntegration) parseOpenAIContextError(body []byte) error {
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		return nil // Not a valid JSON error response, return nil to let caller handle
	}

	if errorResp.Error.Type == "invalid_request_error" || errorResp.Error.Type == "" {
		errMsg := strings.ToLower(errorResp.Error.Message)
		if strings.Contains(errMsg, "maximum context length") ||
			strings.Contains(errMsg, "too many tokens") ||
			strings.Contains(errMsg, "token limit") ||
			strings.Contains(errMsg, "context") ||
			strings.Contains(errMsg, "reduce.*length") {
			return errors.ContextWindowErrorf("OpenAI-compatible context window exceeded: %s", errorResp.Error.Message)
		}
	}

	return nil
}

// Ensure AIModelIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*AIModelIntegration)(nil)
