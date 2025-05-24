package repositories_mongo

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/impl/tools"

	"go.mongodb.org/mongo-driver/bson"
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
		collection:    collection,
		toolInstances: make(map[string]*entities.Tool),
		toolFactory:   toolFactory,
		logger:        logger,
	}
	// Load initial tool instances
	if err := toolRepository.reloadToolInstances(); err != nil {
		return nil, err
	}
	return toolRepository, nil
}

func (t *ToolRepository) ListTools() ([]*entities.Tool, error) {
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
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&toolData)
	if err == mongo.ErrNoDocuments {
		return nil, errors.NotFoundErrorf("toolData not found")
	}
	if err != nil {
		return nil, errors.InternalErrorf("failed to get toolData: %v", err)
	}

	return &toolData, nil
}

func (t *ToolRepository) CreateToolData(ctx context.Context, toolData *entities.ToolData) error {
	_, err := t.collection.InsertOne(ctx, toolData)
	if err != nil {
		return errors.InternalErrorf("failed to create toolData: %v", err)
	}

	return t.reloadToolInstances()
}

func (t *ToolRepository) UpdateToolData(ctx context.Context, toolData *entities.ToolData) error {
	toolData.UpdatedAt = time.Now()

	result, err := t.collection.UpdateOne(ctx, bson.M{"_id": toolData.ID}, bson.M{"$set": toolData})
	if err != nil {
		return errors.InternalErrorf("failed to update toolData: %v", err)
	}
	if result.MatchedCount == 0 {
		return errors.NotFoundErrorf("toolData not found: %s", toolData.ID)
	}

	return t.reloadToolInstances()
}

func (t *ToolRepository) DeleteToolData(ctx context.Context, id string) error {
	result, err := t.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return errors.InternalErrorf("failed to delete toolData: %v", err)
	}
	if result.DeletedCount == 0 {
		return errors.NotFoundErrorf("toolData not found: %s", id)
	}

	return t.reloadToolInstances()
}

func (t *ToolRepository) reloadToolInstances() error {
	t.toolInstances = make(map[string]*entities.Tool)
	toolDataList, err := t.ListToolData(context.Background())
	if err != nil {
		t.logger.Error("Failed to load tool instances", zap.Error(err))
		return errors.InternalErrorf("failed to load tool instances: %v", err)
	}

	for _, toolData := range toolDataList {
		toolFactoryEntry, err := t.toolFactory.GetFactoryByName(toolData.ToolType)
		if err != nil {
			t.logger.Warn("Skipping tool due to unknown type", zap.String("tool_type", toolData.ToolType), zap.Error(err))
			continue
		}
		tool := toolFactoryEntry.Factory(toolData.Name, toolData.Description, toolData.Configuration, t.logger)
		t.toolInstances[toolData.Name] = &tool
	}
	return nil
}

var _ interfaces.ToolRepository = (*ToolRepository)(nil)
