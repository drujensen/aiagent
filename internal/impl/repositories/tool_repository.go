package repositories

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/impl/tools"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type ToolRepository struct {
	collection    *mongo.Collection
	toolInstances map[string]*entities.Tool
	toolFactory   *tools.ToolFactory
	logger        *zap.Logger
}

func NewToolRepository(collection *mongo.Collection, toolFactory *tools.ToolFactory, logger *zap.Logger) (*ToolRepository, error) {
	toolRepository := &ToolRepository{
		collection:  collection,
		toolFactory: toolFactory,
		logger:      logger,
	}

	return toolRepository, nil
}

func (t *ToolRepository) ListTools() ([]*entities.Tool, error) {
	if t.toolInstances == nil {
		t.reloadToolInstances()
	}

	var tools []*entities.Tool
	for _, tool := range t.toolInstances {
		tools = append(tools, tool)
	}
	return tools, nil
}

func (t *ToolRepository) GetToolByName(name string) (*entities.Tool, error) {
	tool, exists := t.toolInstances[name]
	if !exists {
		return nil, nil
	}
	return tool, nil
}

func (t *ToolRepository) RegisterTool(name string, tool *entities.Tool) error {
	if _, exists := t.toolInstances[name]; exists {
		return errors.DuplicateErrorf("tool with the same name already exists")
	}
	t.toolInstances[name] = tool
	return nil
}

func (r *ToolRepository) ListToolData(ctx context.Context) ([]*entities.ToolData, error) {
	var toolDatas []*entities.ToolData
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, errors.InternalErrorf("failed to list toolDatas: %v", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var toolData entities.ToolData
		if err := cursor.Decode(&toolData); err != nil {
			return nil, errors.InternalErrorf("failed to decode toolData: %v", err)
		}
		toolDatas = append(toolDatas, &toolData)
	}

	if err := cursor.Err(); err != nil {
		return nil, errors.InternalErrorf("failed to list toolDatas: %v", err)
	}

	return toolDatas, nil
}

func (r *ToolRepository) GetToolData(ctx context.Context, id string) (*entities.ToolData, error) {
	var toolData entities.ToolData
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.NotFoundErrorf("toolData not found")
	}

	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&toolData)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("toolData not found")
	}
	if err != nil {
		return nil, errors.InternalErrorf("failed to get toolData: %v", err)
	}

	return &toolData, nil
}

func (t *ToolRepository) CreateToolData(ctx context.Context, toolData *entities.ToolData) error {
	toolData.CreatedAt = time.Now()
	toolData.UpdatedAt = time.Now()
	toolData.ID = primitive.NewObjectID()

	result, err := t.collection.InsertOne(ctx, toolData)
	if err != nil {
		return errors.InternalErrorf("failed to create toolData: %v", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		toolData.ID = oid
	} else {
		return errors.ValidationErrorf("failed to convert InsertedID to ObjectID")
	}

	t.reloadToolInstances()

	return nil
}

func (t *ToolRepository) UpdateToolData(ctx context.Context, toolData *entities.ToolData) error {
	toolData.UpdatedAt = time.Now()

	oid, err := primitive.ObjectIDFromHex(toolData.ID.Hex())
	if err != nil {
		return errors.NotFoundErrorf("toolData not found: %s", toolData.ID.Hex())
	}

	// Convert the toolData struct to BSON
	update, err := bson.Marshal(bson.M{
		"$set": toolData,
	})
	if err != nil {
		return errors.InternalErrorf("failed to marshal toolData: %v", err)
	}

	result, err := t.collection.UpdateOne(ctx, bson.M{"_id": oid}, update)
	if err != nil {
		return errors.InternalErrorf("failed to update toolData: %v", err)
	}
	if result.MatchedCount == 0 {
		return errors.NotFoundErrorf("toolData not found: %s", toolData.ID.Hex())
	}

	t.reloadToolInstances()

	return nil
}

func (t *ToolRepository) DeleteToolData(ctx context.Context, id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.NotFoundErrorf("toolData not found")
	}

	result, err := t.collection.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		return errors.InternalErrorf("failed to delete toolData: %v", err)
	}
	if result.DeletedCount == 0 {
		return errors.NotFoundErrorf("toolData not found: %s", id)
	}

	t.reloadToolInstances()

	return nil
}

func (t *ToolRepository) reloadToolInstances() error {
	t.toolInstances = make(map[string]*entities.Tool)

	toolDataList, err := t.ListToolData(context.Background())
	if err != nil {
		return errors.InternalErrorf("failed to load tool instances: %v", err)
	}

	for _, toolData := range toolDataList {
		toolFactoryEntry, err := t.toolFactory.GetFactoryByName(toolData.ToolType)
		if err != nil {
			return errors.InternalErrorf("failed to get tool factory: %v", err)
		}
		tool := toolFactoryEntry.Factory(toolData.Name, toolData.Description, toolData.Configuration, t.logger)
		t.toolInstances[toolData.Name] = &tool
	}
	return nil
}

var _ interfaces.ToolRepository = (*ToolRepository)(nil)
