package entities

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        string    `json:"id" bson:"id"`
	Role      string    `json:"role" bson:"role"`
	Content   string    `json:"content" bson:"content"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

func NewMessage(role, content string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
}
