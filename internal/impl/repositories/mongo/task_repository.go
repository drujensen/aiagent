package repositories_mongo

import (
	"context"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoTaskRepository struct {
	collection *mongo.Collection
}

func NewMongoTaskRepository(collection *mongo.Collection) *MongoTaskRepository {
	return &MongoTaskRepository{
		collection: collection,
	}
}

func (r *MongoTaskRepository) ListTasks(ctx context.Context) ([]*entities.Task, error) {
	var tasks []*entities.Task
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.InternalErrorf("failed to list tasks: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var task entities.Task
		if err := cursor.Decode(&task); err != nil {
			return nil, errors.InternalErrorf("failed to decode task: %v", err)
		}
		tasks = append(tasks, &task)
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.InternalErrorf("failed to list tasks: %v", err)
	}

	return tasks, nil
}

func (r *MongoTaskRepository) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	var task entities.Task
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("task not found")
	}
	if err != nil {
		return nil, errors.InternalErrorf("failed to get task: %v", err)
	}

	return &task, nil
}

func (r *MongoTaskRepository) CreateTask(ctx context.Context, task *entities.Task) error {
	_, err := r.collection.InsertOne(ctx, task)
	if err != nil {
		return errors.InternalErrorf("failed to create task: %v", err)
	}

	return nil
}

func (r *MongoTaskRepository) UpdateTask(ctx context.Context, task *entities.Task) error {
	task.UpdatedAt = time.Now()

	update, err := bson.Marshal(bson.M{
		"$set": task,
	})
	if err != nil {
		return errors.InternalErrorf("failed to marshal task: %v", err)
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": task.ID}, update)
	if err != nil {
		return errors.InternalErrorf("failed to update task: %v", err)
	}
	if result.MatchedCount == 0 {
		return errors.NotFoundErrorf("task not found: %s", task.ID)
	}

	return nil
}

func (r *MongoTaskRepository) DeleteTask(ctx context.Context, id string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return errors.InternalErrorf("failed to delete task: %v", err)
	}
	if result.DeletedCount == 0 {
		return errors.NotFoundErrorf("task not found: %s", id)
	}

	return nil
}

var _ interfaces.TaskRepository = (*MongoTaskRepository)(nil)
