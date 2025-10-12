package entities

import (
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ID              string    `json:"id" bson:"_id"`
	Name            string    `json:"name" bson:"name"`
	DefaultModelID  string    `json:"default_model_id" bson:"default_model_id"`
	SystemPrompt    string    `json:"system_prompt" bson:"system_prompt"`
	Temperature     *float64  `json:"temperature,omitempty" bson:"temperature,omitempty"`
	ReasoningEffort string    `json:"reasoning_effort" bson:"reasoning_effort"` // low, medium, high, or none
	Tools           []string  `json:"tools,omitempty" bson:"tools,omitempty"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" bson:"updated_at"`
}

func NewAgent(name, defaultModelID, systemPrompt string, tools []string) *Agent {
	return &Agent{
		ID:             uuid.New().String(),
		Name:           name,
		DefaultModelID: defaultModelID,
		SystemPrompt:   systemPrompt,
		Tools:          tools,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
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
	return "Agent"
}

func (a *Agent) FullSystemPrompt() string {
	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-01-02 15:04:05")
	return "Your name is " + a.Name + "\nCurrent date and time is " + formattedTime + "\n" + a.SystemPrompt
}
