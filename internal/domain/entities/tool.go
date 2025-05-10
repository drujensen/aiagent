package entities

import (
	"time"

	"github.com/google/uuid"
)

type ToolData struct {
	ID            string            `json:"id" bson:"_id"`
	ToolType      string            `json:"tool_type" bson:"tool_type"`
	Name          string            `json:"name" bson:"name"`
	Description   string            `json:"description" bson:"description"`
	Configuration map[string]string `json:"configurations" bson:"configurations"`
	CreatedAt     time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at" bson:"updated_at"`
}

func NewToolData(toolType, name, description string, config map[string]string) *ToolData {
	return &ToolData{
		ID:            uuid.New().String(),
		ToolType:      toolType,
		Name:          name,
		Description:   description,
		Configuration: config,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

type Item struct {
	Type string
}

type Parameter struct {
	Name        string
	Type        string
	Enum        []string
	Items       []Item
	Description string
	Required    bool
}

type Tool interface {
	Name() string
	Description() string
	FullDescription() string
	Configuration() map[string]string
	UpdateConfiguration(config map[string]string)
	Parameters() []Parameter
	Execute(arguments string) (string, error)
}
