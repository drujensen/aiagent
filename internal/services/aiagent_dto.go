package services

import (
    "encoding/json"
    "aiagent/internal/domain/models"
)

// AIAgentDTO is a data transfer object for serializing AIAgent

type AIAgentDTO struct {
    Prompt string       `json:"prompt"`
    Tools  []ToolDTO    `json:"tools"`
}

// ToolDTO is a data transfer object for serializing Tool

type ToolDTO struct {
    ID           string                 `json:"id"`
    Name         string                 `json:"name"`
    Type         string                 `json:"type"`
    Configuration map[string]interface{} `json:"configuration"`
}

// NewAIAgentDTO creates a new AIAgentDTO from an AIAgent
func NewAIAgentDTO(agent *models.AIAgent) *AIAgentDTO {
    tools := make([]ToolDTO, len(agent.Tools))
    for i, tool := range agent.Tools {
        tools[i] = ToolDTO{
            ID:           tool.ID,
            Name:         tool.Name,
            Type:         tool.Type,
            Configuration: tool.Configuration,
        }
    }
    return &AIAgentDTO{
        Prompt: agent.Prompt,
        Tools:  tools,
    }
}

// ToJSON serializes the AIAgentDTO to a JSON string
func (dto *AIAgentDTO) ToJSON() (string, error) {
    bytes, err := json.Marshal(dto)
    if err != nil {
        return "", err
    }
    return string(bytes), nil
}
