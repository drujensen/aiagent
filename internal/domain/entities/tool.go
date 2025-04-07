package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ToolData struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ToolType      string             `json:"tool_type" bson:"tool_type"`
	Name          string             `json:"name" bson:"name"`
	Description   string             `json:"description" bson:"description"`
	Configuration map[string]string  `json:"configurations" bson:"configurations"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
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
	Configuration() map[string]string
	Parameters() []Parameter
	Execute(arguments string) (string, error)
}
