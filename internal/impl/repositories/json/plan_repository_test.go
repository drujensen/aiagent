package repositories_json

import (
	"context"
	"testing"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"github.com/stretchr/testify/assert"
)

func TestJsonPlanRepository_CRUDRoundTrip(t *testing.T) {
	storageDir := t.TempDir()
	repo, err := NewJSONPlanRepository(storageDir)
	assert.NoError(t, err)

	ctx := context.Background()
	plan := entities.NewPlan("Ship feature X", []string{"no downtime"}, []entities.Feature{
		{Name: "Auth", Description: "Add login"},
	})

	assert.NoError(t, repo.CreatePlan(ctx, plan))
	assert.NotEmpty(t, plan.ID)

	fetched, err := repo.GetPlan(ctx, plan.ID)
	assert.NoError(t, err)
	assert.Equal(t, plan.Goal, fetched.Goal)
	assert.Equal(t, plan.Constraints, fetched.Constraints)
	assert.Equal(t, plan.Features, fetched.Features)

	fetched.Goal = "Ship feature X v2"
	assert.NoError(t, repo.UpdatePlan(ctx, fetched))

	updated, err := repo.GetPlan(ctx, plan.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Ship feature X v2", updated.Goal)

	all, err := repo.ListPlans(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 1)

	assert.NoError(t, repo.DeletePlan(ctx, plan.ID))
	_, err = repo.GetPlan(ctx, plan.ID)
	assert.Error(t, err)
}

// TestJsonTaskRepository_PlanFieldsRoundTrip confirms the additive PlanID
// and AssignedAgentID fields (and the existing DueDate) survive a JSON
// repository round-trip, and that copyTask does not alias the caller's copy
// with repo-internal storage.
func TestJsonTaskRepository_PlanFieldsRoundTrip(t *testing.T) {
	storageDir := t.TempDir()
	repo, err := NewJSONTaskRepository(storageDir)
	assert.NoError(t, err)

	ctx := context.Background()
	task := entities.NewTask("Implement login", "Add JWT auth", entities.TaskPriorityHigh)
	task.PlanID = "plan-123"
	task.AssignedAgentID = "agent-456"
	dueDate := task.CreatedAt
	task.DueDate = &dueDate

	assert.NoError(t, repo.CreateTask(ctx, task))

	fetched, err := repo.GetTask(ctx, task.ID)
	assert.NoError(t, err)
	assert.Equal(t, "plan-123", fetched.PlanID)
	assert.Equal(t, "agent-456", fetched.AssignedAgentID)
	assert.NotNil(t, fetched.DueDate)
	assert.True(t, dueDate.Equal(*fetched.DueDate))

	// Mutating the fetched copy must not alias repo-internal state.
	fetched.PlanID = "mutated"
	again, err := repo.GetTask(ctx, task.ID)
	assert.NoError(t, err)
	assert.Equal(t, "plan-123", again.PlanID)
}
