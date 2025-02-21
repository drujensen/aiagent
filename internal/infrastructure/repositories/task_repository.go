package repositories

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/repositories"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/**
 * @description
 * This file implements the MongoDB-backed TaskRepository for the AI Workflow Automation Platform.
 * It provides concrete implementations of the TaskRepository interface defined in the domain layer,
 * managing CRUD operations for Task entities stored in MongoDB's 'tasks' collection.
 *
 * Key features:
 * - MongoDB Integration: Persists and retrieves Task data with status, hierarchy, and message history.
 * - Domain Alignment: Operates on entities.Task and matches the TaskRepository interface.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides the Task entity definition.
 * - aiagent/internal/domain/repositories: Provides the TaskRepository interface and ErrNotFound.
 * - go.mongodb.org/mongo-driver/mongo: MongoDB driver for database operations.
 *
 * @notes
 * - Status is stored as a string enum (e.g., "pending") per the TaskStatus type.
 * - Messages field stores task context (tool outputs, AI responses), updated explicitly in UpdateTask.
 * - ParentTaskID links to another Task, assumed valid by the service layer.
 */

// MongoTaskRepository is the MongoDB implementation of the TaskRepository interface.
// It handles CRUD operations for Task entities using a MongoDB collection.
type MongoTaskRepository struct {
	collection *mongo.Collection
}

// NewMongoTaskRepository creates a new MongoTaskRepository instance with the given collection.
//
// Parameters:
// - collection: The MongoDB collection handle for the 'tasks' collection.
//
// Returns:
// - *MongoTaskRepository: A new instance ready to manage Task entities.
func NewMongoTaskRepository(collection *mongo.Collection) *MongoTaskRepository {
	return &MongoTaskRepository{
		collection: collection,
	}
}

// CreateTask inserts a new task into the MongoDB collection and sets the task’s ID.
// It initializes CreatedAt and UpdatedAt to the current time.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - task: Pointer to the Task entity to insert; its ID is updated upon success.
//
// Returns:
// - error: Nil on success, or an error if insertion fails (e.g., database error).
func (r *MongoTaskRepository) CreateTask(ctx context.Context, task *entities.Task) error {
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.ID = ""

	result, err := r.collection.InsertOne(ctx, task)
	if err != nil {
		return err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		task.ID = oid.Hex()
	} else {
		return mongo.ErrInvalidIndexValue
	}

	return nil
}

// GetTask retrieves a task by its ID from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the task to retrieve.
//
// Returns:
// - *entities.Task: The retrieved task, or nil if not found or an error occurs.
// - error: Nil on success, ErrNotFound if the task doesn’t exist, or another error otherwise.
func (r *MongoTaskRepository) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	var task entities.Task
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, repositories.ErrNotFound
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return nil, repositories.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &task, nil
}

// UpdateTask updates an existing task in the MongoDB collection.
// It sets UpdatedAt to the current time and updates all fields, including Messages.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - task: Pointer to the Task entity with updated fields; must have a valid ID.
//
// Returns:
// - error: Nil on success, ErrNotFound if the task doesn’t exist, or another error otherwise.
func (r *MongoTaskRepository) UpdateTask(ctx context.Context, task *entities.Task) error {
	task.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(task.ID)
	if err != nil {
		return repositories.ErrNotFound
	}

	update := bson.M{
		"$set": bson.M{
			"description":                task.Description,
			"assigned_to":                task.AssignedTo,
			"parent_task_id":             task.ParentTaskID,
			"status":                     task.Status,
			"result":                     task.Result,
			"requires_human_interaction": task.RequiresHumanInteraction,
			"messages":                   task.Messages,
			"updated_at":                 task.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return repositories.ErrNotFound
	}

	return nil
}

// DeleteTask deletes a task by its ID from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the task to delete.
//
// Returns:
// - error: Nil on success, ErrNotFound if the task doesn’t exist, or another error otherwise.
func (r *MongoTaskRepository) DeleteTask(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return repositories.ErrNotFound
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return repositories.ErrNotFound
	}

	return nil
}

// ListTasks retrieves all tasks from the MongoDB collection.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
//
// Returns:
// - []*entities.Task: Slice of all tasks, empty if none exist.
// - error: Nil on success, or an error if retrieval fails (e.g., database error).
func (r *MongoTaskRepository) ListTasks(ctx context.Context) ([]*entities.Task, error) {
	var tasks []*entities.Task
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var task entities.Task
		if err := cursor.Decode(&task); err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

// Notes:
// - Edge case: Invalid AssignedTo or ParentTaskID references are not checked here; handled in TaskService.
// - Assumption: The 'tasks' collection exists; created implicitly by MongoDB on first insert.
// - Limitation: No filtering by status or hierarchy; extendable if needed for workflow monitoring.
