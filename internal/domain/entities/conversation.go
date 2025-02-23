package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Conversation struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	AgentID   primitive.ObjectID `json:"agent_id" bson:"agent_id"`
	Messages  []Message          `json:"messages" bson:"messages"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
	Active    bool               `json:"active" bson:"active"`
}

func NewConversation(agentID primitive.ObjectID) *Conversation {
	return &Conversation{
		ID:        primitive.NewObjectID(),
		AgentID:   agentID,
		Messages:  make([]Message, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}
}
