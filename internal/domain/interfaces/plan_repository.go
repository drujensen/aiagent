package interfaces

import (
	"context"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

type PlanRepository interface {
	CreatePlan(ctx context.Context, plan *entities.Plan) error
	UpdatePlan(ctx context.Context, plan *entities.Plan) error
	DeletePlan(ctx context.Context, id string) error
	GetPlan(ctx context.Context, id string) (*entities.Plan, error)
	ListPlans(ctx context.Context) ([]*entities.Plan, error)
}
