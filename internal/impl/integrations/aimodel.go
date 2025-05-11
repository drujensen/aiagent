package integrations

import (
	"bytes"
	"context"
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
		httpClient: &http.Client{Timeout: 600 * time.Second},
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

func (m *AIModelIntegration) GenerateResponse(ctx context.Context, messages []*entities.Message, toolList []*entities.Tool, options map[string]any) ([]*entities.Message, error) {
	if ctx.Err() == context.Canceled {
		return nil, fmt.Errorf("operation canceled by user")
	}

	tools := make([]map[string]any, len(toolList))
	for i, tool := range toolList {
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

	for {
		if ctx.Err() == context.Canceled {
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
				Message struct {
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

		if len(toolCalls) == 0 {
			finalMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   content,
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, finalMessage)
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
		newMessages = append(newMessages, toolCallMessage)

		assistantMsg := map[string]any{
			"role":       "assistant",
			"content":    contentMsg,
			"tool_calls": toolCalls,
		}
		apiMessages = append(apiMessages, assistantMsg)

		for _, toolCall := range toolCalls {
			if ctx.Err() == context.Canceled {
				return nil, fmt.Errorf("operation canceled by user")
			}

			if toolCall.Type != "function" {
				continue
			}
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

			var newMessage *entities.Message

			newMessage = &entities.Message{
				ID:         uuid.New().String(),
				Role:       "tool",
				Content:    fmt.Sprintf("Tool %s responded: %s", toolName, toolResult),
				ToolCallID: toolCall.ID,
				Timestamp:  time.Now(),
			}
			newMessages = append(newMessages, newMessage)

			toolResponseMsg := map[string]any{
				"role":         "tool",
				"content":      newMessage.Content,
				"tool_call_id": toolCall.ID,
			}
			apiMessages = append(apiMessages, toolResponseMsg)
		}

		reqBody["messages"] = apiMessages
	}

	return newMessages, nil
}

// GetUsage returns the token usage statistics
func (m *AIModelIntegration) GetUsage() (*entities.Usage, error) {
	return m.lastUsage, nil
}

// Ensure AIModelIntegration implements the AIModelIntegration interface
var _ interfaces.AIModelIntegration = (*AIModelIntegration)(nil)
