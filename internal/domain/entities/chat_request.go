package entities

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
}

func NewChatRequest(agent *Agent, conversation *Conversation) *ChatRequest {
	messages := make([]Message, len(conversation.Messages))
	for i, msg := range conversation.Messages {
		messages[i] = msg
	}

	if agent.SystemPrompt != "" {
		systemMsg := Message{
			Role:    "system",
			Content: agent.SystemPrompt,
		}
		messages = append([]Message{systemMsg}, messages...)
	}

	return &ChatRequest{
		Model:       agent.Model,
		Messages:    messages,
		Temperature: agent.Temperature,
		MaxTokens:   agent.MaxTokens,
	}
}
