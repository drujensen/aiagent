package entities

import "encoding/json"

type ChatResponse struct {
	ID      string `json:"id" bson:"id"`
	Object  string `json:"object" bson:"object"`
	Created int64  `json:"created" bson:"created"`
	Model   string `json:"model" bson:"model"`
	Choices []struct {
		Index        int     `json:"index" bson:"index"`
		Message      Message `json:"message" bson:"message"`
		FinishReason string  `json:"finish_reason" bson:"finish_reason"`
	} `json:"choices" bson:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens" bson:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens" bson:"completion_tokens"`
		TotalTokens      int `json:"total_tokens" bson:"total_tokens"`
	} `json:"usage" bson:"usage"`
}

func NewChatResponse(rawJSON []byte) (*ChatResponse, error) {
	var resp ChatResponse
	err := json.Unmarshal(rawJSON, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (cr *ChatResponse) GetAssistantMessages() []Message {
	messages := make([]Message, 0)
	for _, choice := range cr.Choices {
		if choice.Message.Role == "assistant" {
			messages = append(messages, choice.Message)
		}
	}
	return messages
}
