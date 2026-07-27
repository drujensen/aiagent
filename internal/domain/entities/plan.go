package entities

import (
	"time"

	"github.com/google/uuid"
)

// Feature groups related Tasks under a Plan.
type Feature struct {
	Name        string `json:"name" bson:"name"`
	Description string `json:"description" bson:"description"`
}

// Plan is the durable output of the research/design/plan pipeline Skills: a
// goal, its constraints, and the Features it decomposes into. The actual
// units of fan-out work are Task records (see task.go) referencing this
// Plan's ID via Task.PlanID.
type Plan struct {
	ID          string    `json:"id" bson:"_id"`
	Goal        string    `json:"goal" bson:"goal"`
	Constraints []string  `json:"constraints,omitempty" bson:"constraints,omitempty"`
	Features    []Feature `json:"features,omitempty" bson:"features,omitempty"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

func NewPlan(goal string, constraints []string, features []Feature) *Plan {
	return &Plan{
		ID:          uuid.New().String(),
		Goal:        goal,
		Constraints: constraints,
		Features:    features,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// Implement the list.Item interface (matches the pattern used by Task, Agent, etc.)
func (p *Plan) FilterValue() string {
	return p.Goal
}

func (p *Plan) Title() string {
	return p.Goal
}

func (p *Plan) Description() string {
	return p.CreatedAt.Format("2006-01-02 15:04")
}
