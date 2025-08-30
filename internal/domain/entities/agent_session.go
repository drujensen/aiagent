package entities

import (
	"fmt"
	"time"
)

type AgentSession struct {
	ID          string                 `json:"id" bson:"_id"`
	ParentAgent string                 `json:"parent_agent" bson:"parent_agent"`
	Subagent    string                 `json:"subagent" bson:"subagent"`
	TaskID      string                 `json:"task_id" bson:"task_id"`
	Status      string                 `json:"status" bson:"status"` // pending, active, completed, failed
	CreatedAt   time.Time              `json:"created_at" bson:"created_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	Result      interface{}            `json:"result,omitempty" bson:"result,omitempty"`
	Error       string                 `json:"error,omitempty" bson:"error,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty" bson:"context,omitempty"`
}

func NewAgentSession(parentAgentID, subagentID, taskID string) *AgentSession {
	return &AgentSession{
		ID:          generateSessionID(),
		ParentAgent: parentAgentID,
		Subagent:    subagentID,
		TaskID:      taskID,
		Status:      "pending",
		CreatedAt:   time.Now(),
		Context:     make(map[string]interface{}),
	}
}

func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
