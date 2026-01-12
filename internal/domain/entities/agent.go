package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ID           string    `json:"id" bson:"_id"`
	Name         string    `json:"name" bson:"name"`
	SystemPrompt string    `json:"system_prompt" bson:"system_prompt"`
	Tools        []string  `json:"tools,omitempty" bson:"tools,omitempty"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
}

func NewAgent(name, systemPrompt string, tools []string) *Agent {
	return &Agent{
		ID:           uuid.New().String(),
		Name:         name,
		SystemPrompt: systemPrompt,
		Tools:        tools,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// Implement the list.Item interface
func (a *Agent) FilterValue() string {
	return a.Name
}

func (a *Agent) Title() string {
	return a.Name
}

func (a *Agent) Description() string {
	toolCount := len(a.Tools)
	if toolCount > 0 {
		return fmt.Sprintf("Tools: %d", toolCount)
	}
	return "No tools configured"
}

func (a *Agent) FullSystemPrompt() string {
	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-01-02 15:04:05")
	return "Your name is " + a.Name + "\nCurrent date and time is " + formattedTime + "\n" + a.SystemPrompt
}
