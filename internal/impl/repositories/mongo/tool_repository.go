package repositories_mongo

import (
	"context"
	"sync"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"github.com/drujensen/aiagent/internal/impl/tools"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type ToolRepository struct {
	collection *mongo.Collection
	// mu guards toolInstances only. reloadToolInstances builds its
	// replacement map without holding mu (so it never blocks concurrent
	// reads on the old map, and never re-enters ListToolData while
	// holding a lock ListToolData itself does not need), then swaps it in
	// under a brief write lock.
	mu            sync.RWMutex
	toolInstances map[string]entities.Tool
	toolFactory   *tools.ToolFactory
	logger        *zap.Logger
}

func NewToolRepository(collection *mongo.Collection, toolFactory *tools.ToolFactory, logger *zap.Logger) (*ToolRepository, error) {
	toolRepository := &ToolRepository{
		collection:    collection,
		toolInstances: make(map[string]entities.Tool),
		toolFactory:   toolFactory,
		logger:        logger,
	}
	// Load initial tool instances
	if err := toolRepository.reloadToolInstances(); err != nil {
		return nil, err
	}
	return toolRepository, nil
}

func (t *ToolRepository) ListTools() ([]entities.Tool, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var toolList []entities.Tool
	for _, tool := range t.toolInstances {
		toolList = append(toolList, tool)
	}
	return toolList, nil
}

func (t *ToolRepository) GetToolByName(name string) (entities.Tool, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	tool, exists := t.toolInstances[name]
	if !exists {
		return nil, nil
	}
	return tool, nil
}

func (t *ToolRepository) RegisterTool(name string, tool entities.Tool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.toolInstances[name]; exists {
		return errors.DuplicateErrorf("tool with the same name already exists")
	}
	t.toolInstances[name] = tool
	return nil
}

func copyToolData(td *entities.ToolData) *entities.ToolData {
	var configCopy map[string]string
	if td.Configuration != nil {
		configCopy = make(map[string]string, len(td.Configuration))
		for k, v := range td.Configuration {
			configCopy[k] = v
		}
	}
	return &entities.ToolData{
		ID:            td.ID,
		Name:          td.Name,
		Description:   td.Description,
		ToolType:      td.ToolType,
		Configuration: configCopy,
		CreatedAt:     td.CreatedAt,
		UpdatedAt:     td.UpdatedAt,
	}
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
	toolDataList, err := t.ListToolData(context.Background())
	if err != nil {
		t.logger.Error("Failed to load tool instances", zap.Error(err))
		return errors.InternalErrorf("failed to load tool instances: %v", err)
	}

	newInstances := make(map[string]entities.Tool)
	for _, toolData := range toolDataList {
		toolFactoryEntry, err := t.toolFactory.GetFactoryByName(toolData.ToolType)
		if err != nil {
			t.logger.Warn("Skipping tool due to unknown type", zap.String("tool_type", toolData.ToolType), zap.Error(err))
			continue
		}
		configCopy := copyToolData(toolData).Configuration
		tool := toolFactoryEntry.Factory(toolData.Name, toolData.Description, configCopy, t.logger)
		newInstances[toolData.Name] = tool
	}

	t.mu.Lock()
	t.toolInstances = newInstances
	t.mu.Unlock()
	return nil
}

var _ interfaces.ToolRepository = (*ToolRepository)(nil)
