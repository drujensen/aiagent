package repositories

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ToolRepository manages CRUD operations for Tools in MongoDB
type ToolRepository struct {
	client *mongo.Client
	db     *mongo.Database
}

// NewToolRepository creates a new ToolRepository instance
func NewToolRepository(connectionString string, databaseName string) (*ToolRepository, error) {
	clientOptions := options.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return nil, err
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
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

	result, err := collection.InsertOne(context.TODO(), tool)
	if err != nil {
		return "", err
	}

	return result.InsertedID.(string), nil
}

// GetTool retrieves a tool by its ID
func (r *ToolRepository) GetTool(id string) (*Tool, error) {
	collection := r.db.Collection("tools")

	var tool Tool
	err := collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&tool)
	if err != nil {
		return nil, err
	}

	return &tool, nil
}

// UpdateTool modifies an existing tool in the database
func (r *ToolRepository) UpdateTool(id string, updatedTool Tool) error {
	collection := r.db.Collection("tools")

	updatedTool.UpdatedAt = time.Now()

	result, err := collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": id},
		bson.D{
			{"$set", bson.D{
				{"name", updatedTool.Name},
				{"category", updatedTool.Category},
				{"updated_at", updatedTool.UpdatedAt},
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

	result, err := collection.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}
