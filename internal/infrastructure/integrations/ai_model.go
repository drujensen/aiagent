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

type GenericAIModel struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	lastUsage  int
	modelName  string
}

func NewGenericAIModel(baseURL, apiKey, modelName string) (*GenericAIModel, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}
	if modelName == "" {
		return nil, fmt.Errorf("modelName cannot be empty")
	}
	return &GenericAIModel{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		lastUsage:  0,
		modelName:  modelName,
	}, nil
}

func (m *GenericAIModel) GenerateResponse(messages []map[string]string, options map[string]interface{}) (string, error) {
	reqBody := map[string]interface{}{
		"model":    m.modelName,
		"messages": messages,
	}
	for key, value := range options {
		reqBody[key] = value
	}

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
				Content string `json:"content"`
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
	return responseBody.Choices[0].Message.Content, nil
}

func (m *GenericAIModel) GetTokenUsage() (int, error) {
	return m.lastUsage, nil
}

var _ interfaces.AIModelIntegration = (*GenericAIModel)(nil)
