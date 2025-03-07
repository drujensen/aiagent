package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"aiagent/internal/domain/interfaces"
)

type AIModelIntegration struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	lastUsage  int
	modelName  string
	toolRepo   interfaces.ToolRepository
}

func NewAIModelIntegration(baseURL, apiKey, modelName string, toolRepo interfaces.ToolRepository) (*AIModelIntegration, error) {
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
		httpClient: &http.Client{Timeout: 30 * time.Second},
		lastUsage:  0,
		modelName:  modelName,
		toolRepo:   toolRepo,
	}, nil
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func (m *AIModelIntegration) GenerateResponse(messages []map[string]string, toolList []*interfaces.ToolIntegration, options map[string]interface{}) (string, error) {

	// Prepare tool definitions
	tools := make([]map[string]interface{}, len(toolList))
	for i, tool := range toolList {
		requiredFields := make([]string, 0)
		for _, param := range (*tool).Parameters() {
			if param.Required {
				requiredFields = append(requiredFields, param.Name)
			}
		}

		properties := make([]map[string]interface{}, len(toolList))
		for i, param := range (*tool).Parameters() {
			property := map[string]interface{}{
				param.Name: map[string]interface{}{
					"type":        param.Type,
					"description": param.Description,
				},
			}

			// Only add enum if it has values
			if len(param.Enum) > 0 {
				property[param.Name].(map[string]interface{})["enum"] = param.Enum
			}

			properties[i] = property
		}

		tools[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        (*tool).Name(),
				"description": (*tool).Description(),
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": properties,
				},
				"required": requiredFields,
			},
		}
	}

	reqBody := map[string]interface{}{
		"model":    m.modelName,
		"messages": messages,
	}
	if len(tools) > 0 {
		reqBody["tools"] = tools
	}
	for key, value := range options {
		if key != "tools" {
			reqBody[key] = value
		}
	}

	var finalResponse string
	for {
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("error marshaling request: %v", err)
		}

		req, err := http.NewRequest("POST", m.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonBody))
		if err != nil {
			return "", fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+m.apiKey)

		var resp *http.Response
		for attempt := 0; attempt < 3; attempt++ {
			resp, err = m.httpClient.Do(req)
			if err != nil {
				if attempt < 2 {
					time.Sleep(time.Duration(attempt+1) * time.Second)
					continue
				}
				return "", fmt.Errorf("error making request: %v", err)
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				if attempt < 2 {
					time.Sleep(time.Duration(attempt+1) * time.Second)
					continue
				}
				return "", fmt.Errorf("rate limit exceeded")
			}
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
			}
			break
		}
		defer resp.Body.Close()

		var responseBody struct {
			Choices []struct {
				Message struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				TotalTokens int `json:"total_tokens"`
			} `json:"usage"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
			return "", fmt.Errorf("error decoding response: %v", err)
		}
		if len(responseBody.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}

		m.lastUsage = responseBody.Usage.TotalTokens
		choice := responseBody.Choices[0].Message

		if len(choice.ToolCalls) == 0 {
			// No tool calls, return the content
			finalResponse = choice.Content
			break
		}

		// Handle tool calls
		for _, toolCall := range choice.ToolCalls {
			if toolCall.Type != "function" {
				continue
			}
			toolName := toolCall.Function.Name

			tool, err := m.toolRepo.GetToolByName(toolName)
			if err != nil {
				return "", fmt.Errorf("failed to get tool %s: %v", toolName, err)
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

			// Append tool response message
			messages = append(messages, map[string]string{
				"role":         "tool",
				"content":      toolResult,
				"tool_call_id": toolCall.ID,
			})
		}

		// Prepare for next iteration
		reqBody["messages"] = messages
	}

	return finalResponse, nil
}

func (m *AIModelIntegration) GetTokenUsage() (int, error) {
	return m.lastUsage, nil
}

var _ interfaces.AIModelIntegration = (*AIModelIntegration)(nil)
