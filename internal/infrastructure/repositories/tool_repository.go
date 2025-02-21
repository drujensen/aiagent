package repositories

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// TimeoutDuration is the duration for context timeouts
	TimeoutDuration = 10 * time.Second
)

// Tool represents the structure of a tool document in MongoDB
type Tool struct {
	ID        string    `bson:"_id,omitempty"`
	Name      string    `bson:"name"`
	Category  string    `bson:"category"`
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// ToolRepository manages CRUD operations for Tools in MongoDB
type ToolRepository struct {
	client *mongo.Client
	db     *mongo.Database
}

// NewToolRepository creates a new ToolRepository instance
func NewToolRepository(connectionString string, databaseName string) (*ToolRepository, error) {
	clientOptions := options.Client().ApplyURI(connectionString)
	ctx, cancel := context.WithTimeout(context.Background(), TimeoutDuration)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	log.Println("Connected to MongoDB!")

	db := client.Database(databaseName)
	return &ToolRepository{
		client: client,
		db:     db,
	}, nil
}

// CreateTool adds a new tool to the database
func (r *ToolRepository) CreateTool(tool Tool) (string, error) {
	collection := r.db.Collection("tools")

	tool.CreatedAt = time.Now()
	tool.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, tool)
	if err != nil {
		return "", err
	}

	return result.InsertedID.(string), nil
}

// GetTool retrieves a tool by its ID
func (r *ToolRepository) GetTool(id string) (*Tool, error) {
	collection := r.db.Collection("tools")

	var tool Tool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&tool)
	if err != nil {
		return nil, err
	}

	return &tool, nil
}

// UpdateTool modifies an existing tool in the database
func (r *ToolRepository) UpdateTool(id string, updatedTool Tool) error {
	collection := r.db.Collection("tools")

	updatedTool.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "name", Value: updatedTool.Name},
				{Key: "category", Value: updatedTool.Category},
				{Key: "updated_at", Value: updatedTool.UpdatedAt},
			}},
		},
	)

	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// DeleteTool removes a tool from the database
func (r *ToolRepository) DeleteTool(id string) error {
	collection := r.db.Collection("tools")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}
