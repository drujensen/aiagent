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
}

func NewAIModelIntegration(baseURL, apiKey, modelName string) (*AIModelIntegration, error) {
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

func (m *AIModelIntegration) GenerateResponse(messages []map[string]string, options map[string]interface{}) (string, error) {
	// Prepare tool definitions
	var tools []map[string]interface{}
	if toolList, ok := options["tools"].([]map[string]string); ok && len(toolList) > 0 {
		tools = make([]map[string]interface{}, len(toolList))
		for i, tool := range toolList {
			tools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool["name"],
					"description": fmt.Sprintf("Execute the %s tool", tool["name"]),
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"input": map[string]interface{}{
								"type":        "string",
								"description": "Input for the tool",
							},
						},
						"required": []string{"input"},
					},
				},
			}
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
			var args struct {
				Input string `json:"input"`
			}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return "", fmt.Errorf("error parsing tool arguments: %v", err)
			}

			tool := GetToolByName(toolName)
			var toolResult string
			if tool != nil {
				result, err := tool.Execute(args.Input)
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
