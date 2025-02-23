package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Tool struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}

func NewTool(name, description string) *Tool {
	return &Tool{
		ID:          primitive.NewObjectID(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
