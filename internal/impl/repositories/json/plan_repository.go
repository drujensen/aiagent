package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
)

type JsonPlanRepository struct {
	mu       sync.RWMutex
	filePath string
	data     []*entities.Plan
}

func NewJSONPlanRepository(storageDir string) (interfaces.PlanRepository, error) {
	filePath := filepath.Join(storageDir, "plans.json")
	repo := &JsonPlanRepository{
		filePath: filePath,
		data:     []*entities.Plan{},
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

// copyPlan deep-copies Constraints and Features (both slices) so a returned
// Plan never shares backing storage with r.data - matching the deep-copy
// contract established in Phase 1 (see chat_repository.go's deepCopyChat).
func copyPlan(p *entities.Plan) *entities.Plan {
	return &entities.Plan{
		ID:          p.ID,
		Goal:        p.Goal,
		Constraints: slices.Clone(p.Constraints),
		Features:    slices.Clone(p.Features),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// load and save are only ever called by exported methods that already hold
// r.mu, so neither acquires the lock itself.
func (r *JsonPlanRepository) load() error {
	data, err := os.ReadFile(r.filePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet, start with empty data
	}
	if err != nil {
		return errors.InternalErrorf("failed to read plans.json: %v", err)
	}

	var plans []*entities.Plan
	if err := json.Unmarshal(data, &plans); err != nil {
		return errors.InternalErrorf("failed to unmarshal plans.json: %v", err)
	}

	// Validate UUIDs
	for _, plan := range plans {
		if plan.ID == "" {
			return errors.InternalErrorf("plan is missing an ID")
		}
		if _, err := uuid.Parse(plan.ID); err != nil {
			return errors.InternalErrorf("plan has an invalid UUID: %v", err)
		}
	}

	r.data = plans
	return nil
}

func (r *JsonPlanRepository) save() error {
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return errors.InternalErrorf("failed to marshal plans: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return errors.InternalErrorf("failed to create directory: %v", err)
	}

	return atomicWriteFile(r.filePath, data)
}

func (r *JsonPlanRepository) ListPlans(ctx context.Context) ([]*entities.Plan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plansCopy := make([]*entities.Plan, len(r.data))
	for i, p := range r.data {
		plansCopy[i] = copyPlan(p)
	}
	return plansCopy, nil
}

func (r *JsonPlanRepository) GetPlan(ctx context.Context, id string) (*entities.Plan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, plan := range r.data {
		if plan.ID == id {
			return copyPlan(plan), nil
		}
	}
	return nil, errors.NotFoundErrorf("plan not found: %s", id)
}

func (r *JsonPlanRepository) CreatePlan(ctx context.Context, plan *entities.Plan) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if plan.ID == "" {
		plan.ID = uuid.New().String()
	}
	plan.CreatedAt = time.Now()
	plan.UpdatedAt = plan.CreatedAt

	r.data = append(r.data, copyPlan(plan))
	return r.save()
}

func (r *JsonPlanRepository) UpdatePlan(ctx context.Context, plan *entities.Plan) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, p := range r.data {
		if p.ID == plan.ID {
			plan.UpdatedAt = time.Now()
			r.data[i] = copyPlan(plan)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("plan not found: %s", plan.ID)
}

func (r *JsonPlanRepository) DeletePlan(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, p := range r.data {
		if p.ID == id {
			r.data = slices.Delete(r.data, i, i+1)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("plan not found: %s", id)
}

var _ interfaces.PlanRepository = (*JsonPlanRepository)(nil)
