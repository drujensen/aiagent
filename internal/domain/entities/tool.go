package entities

import "time"

// Tool represents the structure of a tool document in MongoDB
type Tool struct {
	ID        string    `bson:"_id,omitempty"`
	Name      string    `bson:"name"`
	Category  string    `bson:"category"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}
