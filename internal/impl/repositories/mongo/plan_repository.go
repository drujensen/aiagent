package repositories_mongo

import (
	"context"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoPlanRepository struct {
	collection *mongo.Collection
}

func NewMongoPlanRepository(collection *mongo.Collection) *MongoPlanRepository {
	return &MongoPlanRepository{
		collection: collection,
	}
}

func (r *MongoPlanRepository) ListPlans(ctx context.Context) ([]*entities.Plan, error) {
	var plans []*entities.Plan
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.InternalErrorf("failed to list plans: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var plan entities.Plan
		if err := cursor.Decode(&plan); err != nil {
			return nil, errors.InternalErrorf("failed to decode plan: %v", err)
		}
		plans = append(plans, &plan)
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.InternalErrorf("failed to list plans: %v", err)
	}

	return plans, nil
}

func (r *MongoPlanRepository) GetPlan(ctx context.Context, id string) (*entities.Plan, error) {
	var plan entities.Plan
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&plan)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("plan not found")
	}
	if err != nil {
		return nil, errors.InternalErrorf("failed to get plan: %v", err)
	}

	return &plan, nil
}

func (r *MongoPlanRepository) CreatePlan(ctx context.Context, plan *entities.Plan) error {
	if plan.ID == "" {
		plan.ID = uuid.New().String()
	}
	_, err := r.collection.InsertOne(ctx, plan)
	if err != nil {
		return errors.InternalErrorf("failed to create plan: %v", err)
	}

	return nil
}

func (r *MongoPlanRepository) UpdatePlan(ctx context.Context, plan *entities.Plan) error {
	plan.UpdatedAt = time.Now()

	update, err := bson.Marshal(bson.M{
		"$set": plan,
	})
	if err != nil {
		return errors.InternalErrorf("failed to marshal plan: %v", err)
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": plan.ID}, update)
	if err != nil {
		return errors.InternalErrorf("failed to update plan: %v", err)
	}
	if result.MatchedCount == 0 {
		return errors.NotFoundErrorf("plan not found: %s", plan.ID)
	}

	return nil
}

func (r *MongoPlanRepository) DeletePlan(ctx context.Context, id string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return errors.InternalErrorf("failed to delete plan: %v", err)
	}
	if result.DeletedCount == 0 {
		return errors.NotFoundErrorf("plan not found: %s", id)
	}

	return nil
}

var _ interfaces.PlanRepository = (*MongoPlanRepository)(nil)
