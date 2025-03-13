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

type AIModelIntegration struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	lastUsage  *entities.Usage
	modelName  string
	toolRepo   interfaces.ToolRepository
	logger     *zap.Logger
}

func NewAIModelIntegration(baseURL, apiKey, modelName string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*AIModelIntegration, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}
	if modelName == "" {
		return nil, fmt.Errorf("modelName cannot be empty")
	}
	return &AIModelIntegration{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 600 * time.Second},
		lastUsage:  &entities.Usage{},
		modelName:  modelName,
		toolRepo:   toolRepo,
		logger:     logger,
	}, nil
}

func convertToAPIMessages(messages []*entities.Message) []map[string]interface{} {
	apiMessages := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		apiMsg := map[string]interface{}{
			"role": msg.Role,
		}
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			apiMsg["tool_calls"] = msg.ToolCalls
			apiMsg["content"] = nil
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

func (m *AIModelIntegration) GenerateResponse(messages []*entities.Message, toolList []*interfaces.ToolIntegration, options map[string]interface{}) ([]*entities.Message, error) {
	// Prepare tool definitions
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
			"type": "function",
			"function": map[string]interface{}{
				"name":        (*tool).Name(),
				"description": (*tool).Description(),
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": properties,
					"required":   requiredFields,
				},
			},
		}
	}

	reqBody := map[string]interface{}{
		"model": m.modelName,
	}
	// Add tools to request body if any
	if len(tools) > 0 {
		reqBody["tools"] = tools
	}

	// Add options to request body
	for key, value := range options {
		reqBody[key] = value
	}

	// Convert initial messages to API format
	apiMessages := convertToAPIMessages(messages)
	reqBody["messages"] = apiMessages

	m.logger.Info("Messages before tool calls", zap.Any("messages", apiMessages))

	var newMessages []*entities.Message

	// Loop to handle tool calls
	for {
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request: %v", err)
		}

		req, err := http.NewRequest("POST", m.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+m.apiKey)

		var resp *http.Response
		for attempt := 0; attempt < 3; attempt++ {
			resp, err = m.httpClient.Do(req)
			if err != nil {
				if attempt < 2 {
					m.logger.Warn("Error making request, retrying", zap.Error(err))
					m.logger.Warn("jsonBody", zap.String("jsonBody", string(jsonBody)))
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

		if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
			return nil, fmt.Errorf("error decoding response: %v", err)
		}
		if len(responseBody.Choices) == 0 {
			return nil, fmt.Errorf("no choices in response")
		}

		// Store usage data
		m.lastUsage.PromptTokens = responseBody.Usage.PromptTokens
		m.lastUsage.CompletionTokens = responseBody.Usage.CompletionTokens
		m.lastUsage.TotalTokens = responseBody.Usage.TotalTokens

		choice := responseBody.Choices[0].Message

		if len(choice.ToolCalls) == 0 {
			// No tool calls, add the final assistant message
			finalMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   choice.Content,
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, finalMessage)
			break
		} else {
			// There are tool calls, create a message for the assistant's tool call
			toolCallMessage := &entities.Message{
				ID:        uuid.New().String(),
				Role:      "assistant",
				Content:   "",
				ToolCalls: choice.ToolCalls,
				Timestamp: time.Now(),
			}
			newMessages = append(newMessages, toolCallMessage)

			// Append assistant message with tool calls to apiMessages
			assistantMsg := map[string]interface{}{
				"role":       "assistant",
				"content":    nil,
				"tool_calls": choice.ToolCalls,
			}
			apiMessages = append(apiMessages, assistantMsg)

			// Handle tool calls
			for _, toolCall := range choice.ToolCalls {
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

				// Create tool response message
				toolResponseMessage := &entities.Message{
					ID:         uuid.New().String(),
					Role:       "tool",
					Content:    toolResult,
					ToolCallID: toolCall.ID,
					Timestamp:  time.Now(),
				}
				newMessages = append(newMessages, toolResponseMessage)

				// Append tool response to apiMessages
				toolResponseMsg := map[string]interface{}{
					"role":         "tool",
					"content":      toolResult,
					"tool_call_id": toolCall.ID,
				}
				apiMessages = append(apiMessages, toolResponseMsg)
			}

			m.logger.Info("Messages after tool calls", zap.Any("messages", newMessages))

			// Prepare for next iteration
			reqBody["messages"] = apiMessages
		}
	}

	return newMessages, nil
}

func (m *AIModelIntegration) GetUsage() (*entities.Usage, error) {
	return m.lastUsage, nil
}

func (m *AIModelIntegration) ModelName() string {
	return m.modelName
}

func (m *AIModelIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderGeneric
}

var _ interfaces.AIModelIntegration = (*AIModelIntegration)(nil)
