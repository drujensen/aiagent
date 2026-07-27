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
	mu             sync.RWMutex
	toolInstances  map[string]entities.Tool
	toolFactory    *tools.ToolFactory
	configResolver interfaces.ConfigResolver
	// toolSources supplies dynamically-discovered tools (currently: MCP
	// servers) that have no ToolData row and are merged in at read time -
	// see JsonToolRepository for the full rationale (identical for both
	// backends). Set once at construction; not guarded by mu.
	toolSources []tools.ToolSource
	logger      *zap.Logger
}

func NewToolRepository(collection *mongo.Collection, toolFactory *tools.ToolFactory, configResolver interfaces.ConfigResolver, toolSources []tools.ToolSource, logger *zap.Logger) (*ToolRepository, error) {
	toolRepository := &ToolRepository{
		collection:     collection,
		toolInstances:  make(map[string]entities.Tool),
		toolFactory:    toolFactory,
		configResolver: configResolver,
		toolSources:    toolSources,
		logger:         logger,
	}
	// Load initial tool instances
	if err := toolRepository.reloadToolInstances(); err != nil {
		return nil, err
	}
	return toolRepository, nil
}

// findInSources looks up a tool by name across all configured tool sources -
// see JsonToolRepository.findInSources for the full rationale (identical for
// both backends).
func (t *ToolRepository) findInSources(ctx context.Context, name string) (entities.Tool, error) {
	for _, source := range t.toolSources {
		sourceTools, err := source.ListTools(ctx)
		if err != nil {
			t.logger.Warn("Failed to list tools from source, skipping", zap.String("source", source.Name()), zap.Error(err))
			continue
		}
		for _, tool := range sourceTools {
			if tool.Name() == name {
				return tool, nil
			}
		}
	}
	return nil, nil
}

func (t *ToolRepository) ListTools() ([]entities.Tool, error) {
	t.mu.RLock()
	var toolList []entities.Tool
	for _, tool := range t.toolInstances {
		toolList = append(toolList, tool)
	}
	t.mu.RUnlock()

	for _, source := range t.toolSources {
		sourceTools, err := source.ListTools(context.Background())
		if err != nil {
			t.logger.Warn("Failed to list tools from source, skipping", zap.String("source", source.Name()), zap.Error(err))
			continue
		}
		toolList = append(toolList, sourceTools...)
	}
	return toolList, nil
}

func (t *ToolRepository) GetToolByName(name string) (entities.Tool, error) {
	t.mu.RLock()
	tool, exists := t.toolInstances[name]
	t.mu.RUnlock()
	if exists {
		return tool, nil
	}

	return t.findInSources(context.Background(), name)
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
		// Resolve #{VAR}# placeholders once here, at singleton-construction
		// time - see JsonToolRepository.reloadToolInstances for the full
		// rationale (identical for both backends).
		resolvedConfig, err := t.resolveConfig(toolData.Configuration)
		if err != nil {
			t.logger.Warn("Skipping tool due to unresolvable configuration", zap.String("tool_name", toolData.Name), zap.Error(err))
			continue
		}
		tool := toolFactoryEntry.Factory(toolData.Name, toolData.Description, resolvedConfig, t.logger)
		newInstances[toolData.Name] = tool
	}

	t.mu.Lock()
	t.toolInstances = newInstances
	t.mu.Unlock()
	return nil
}

// resolveConfig merges override on top of base (override wins on key
// collision) and resolves #{VAR}# placeholders in the result. Both maps are
// left untouched; the returned map is always a fresh allocation.
func (t *ToolRepository) resolveConfig(base map[string]string, override ...map[string]string) (map[string]string, error) {
	merged := make(map[string]string, len(base))
	for k, v := range base {
		merged[k] = v
	}
	for _, o := range override {
		for k, v := range o {
			merged[k] = v
		}
	}
	return t.configResolver.ResolveConfiguration(merged)
}

// GetToolForChat mints a tool instance scoped to one chat turn, with its
// configuration already resolved - see the interface doc comment for the
// Stateful-tool exception.
func (t *ToolRepository) GetToolForChat(ctx context.Context, name string, config map[string]string) (entities.Tool, error) {
	var toolData entities.ToolData
	err := t.collection.FindOne(ctx, bson.M{"name": name}).Decode(&toolData)
	if err == mongo.ErrNoDocuments {
		// Not a native, factory-built tool - check configured tool sources
		// (MCP servers) - see JsonToolRepository.GetToolForChat for the full
		// rationale (identical for both backends).
		sourceTool, sourceErr := t.findInSources(ctx, name)
		if sourceErr != nil {
			return nil, sourceErr
		}
		if sourceTool != nil {
			return sourceTool, nil
		}
		return nil, errors.NotFoundErrorf("tool not found: %s", name)
	}
	if err != nil {
		return nil, errors.InternalErrorf("failed to get toolData for %s: %v", name, err)
	}

	factoryEntry, err := t.toolFactory.GetFactoryByName(toolData.ToolType)
	if err != nil {
		return nil, err
	}

	if factoryEntry.Stateful {
		// Never mint a fresh instance for a stateful tool - see
		// JsonToolRepository.GetToolForChat for the full rationale
		// (identical for both backends).
		t.mu.RLock()
		defer t.mu.RUnlock()
		instance, exists := t.toolInstances[name]
		if !exists {
			return nil, errors.NotFoundErrorf("tool instance not found: %s", name)
		}
		return instance, nil
	}

	resolvedConfig, err := t.resolveConfig(toolData.Configuration, config)
	if err != nil {
		return nil, errors.InternalErrorf("failed to resolve configuration for tool %s: %v", name, err)
	}

	return factoryEntry.Factory(toolData.Name, toolData.Description, resolvedConfig, t.logger), nil
}

var _ interfaces.ToolRepository = (*ToolRepository)(nil)
