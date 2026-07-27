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
	"github.com/drujensen/aiagent/internal/impl/tools"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type JsonToolRepository struct {
	// mu guards data and toolInstances. Unexported helpers (load, save,
	// reloadToolInstances) assume the caller already holds mu - only
	// exported methods acquire it, since reloadToolInstances is called
	// from inside Create/Update/DeleteToolData and would otherwise
	// self-deadlock on a non-reentrant sync.RWMutex.
	mu             sync.RWMutex
	filePath       string
	data           []*entities.ToolData
	toolInstances  map[string]entities.Tool
	toolFactory    *tools.ToolFactory
	configResolver interfaces.ConfigResolver
	// toolSources supplies dynamically-discovered tools (currently: MCP
	// servers) that have no ToolData row and are merged in at read time by
	// ListTools/GetToolByName/GetToolForChat, alongside the native,
	// factory-built tools in toolInstances. Set once at construction; not
	// guarded by mu since it's never mutated afterward.
	toolSources []tools.ToolSource
	logger      *zap.Logger
}

func NewJSONToolRepository(storageDir string, toolFactory *tools.ToolFactory, configResolver interfaces.ConfigResolver, toolSources []tools.ToolSource, logger *zap.Logger) (interfaces.ToolRepository, error) {
	filePath := filepath.Join(storageDir, "tools.json")
	repo := &JsonToolRepository{
		filePath:       filePath,
		data:           []*entities.ToolData{},
		toolInstances:  make(map[string]entities.Tool),
		toolFactory:    toolFactory,
		configResolver: configResolver,
		toolSources:    toolSources,
		logger:         logger,
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	if err := repo.reloadToolInstances(); err != nil {
		return nil, err
	}

	return repo, nil
}

// findInSources looks up a tool by name across all configured tool sources
// (currently: MCP servers), returning the first match. Each call re-fetches
// the source's tool list, since MCP tool sets can change between calls
// (e.g. a server restart) and this repository does not cache them.
func (r *JsonToolRepository) findInSources(ctx context.Context, name string) (entities.Tool, error) {
	for _, source := range r.toolSources {
		sourceTools, err := source.ListTools(ctx)
		if err != nil {
			r.logger.Warn("Failed to list tools from source, skipping", zap.String("source", source.Name()), zap.Error(err))
			continue
		}
		for _, t := range sourceTools {
			if t.Name() == name {
				return t, nil
			}
		}
	}
	return nil, nil
}

func copyToolData(t *entities.ToolData) *entities.ToolData {
	var configCopy map[string]string
	if t.Configuration != nil {
		configCopy = make(map[string]string, len(t.Configuration))
		for k, v := range t.Configuration {
			configCopy[k] = v
		}
	}
	return &entities.ToolData{
		ID:            t.ID,
		Name:          t.Name,
		Description:   t.Description,
		ToolType:      t.ToolType,
		Configuration: configCopy,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
}

// load, save, and reloadToolInstances assume r.mu is already held by the
// caller (see the field comment on mu above).
func (r *JsonToolRepository) load() error {
	data, err := os.ReadFile(r.filePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet, start with empty data
	}
	if err != nil {
		return errors.InternalErrorf("failed to read tools.json: %v", err)
	}

	var toolData []*entities.ToolData
	if err := json.Unmarshal(data, &toolData); err != nil {
		return errors.InternalErrorf("failed to unmarshal tools.json: %v", err)
	}

	// Validate UUIDs
	for _, tool := range toolData {
		if tool.ID == "" {
			return errors.InternalErrorf("tool is missing an ID")
		}
		if _, err := uuid.Parse(tool.ID); err != nil {
			return errors.InternalErrorf("tool has an invalid UUID: %v", err)
		}
	}

	r.data = toolData
	return nil
}

func (r *JsonToolRepository) save() error {
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return errors.InternalErrorf("failed to marshal tools: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return errors.InternalErrorf("failed to create directory: %v", err)
	}

	return atomicWriteFile(r.filePath, data)
}

func (r *JsonToolRepository) reloadToolInstances() error {
	r.toolInstances = make(map[string]entities.Tool)
	for _, toolData := range r.data {
		toolFactoryEntry, err := r.toolFactory.GetFactoryByName(toolData.ToolType)
		if err != nil {
			r.logger.Warn("Skipping tool due to unknown type", zap.String("tool_type", toolData.ToolType), zap.Error(err))
			continue
		}
		// Resolve #{VAR}# placeholders once here, at singleton-construction
		// time. This matters most for Stateful tools (see
		// ToolFactoryEntry.Stateful) - they never go through
		// GetToolForChat's per-turn resolution, so this is the only place
		// their configuration is ever resolved. It's also harmless for
		// non-stateful tools: GetToolForChat mints its own freshly-resolved
		// instance for those, so this copy is only ever used as the
		// fallback path (GetToolByName).
		resolvedConfig, err := r.resolveConfig(toolData.Configuration)
		if err != nil {
			r.logger.Warn("Skipping tool due to unresolvable configuration", zap.String("tool_name", toolData.Name), zap.Error(err))
			continue
		}
		tool := toolFactoryEntry.Factory(toolData.Name, toolData.Description, resolvedConfig, r.logger)
		r.toolInstances[toolData.Name] = tool
	}
	return nil
}

// resolveConfig merges override on top of base (override wins on key
// collision) and resolves #{VAR}# placeholders in the result. Both maps are
// left untouched; the returned map is always a fresh allocation.
func (r *JsonToolRepository) resolveConfig(base map[string]string, override ...map[string]string) (map[string]string, error) {
	merged := make(map[string]string, len(base))
	for k, v := range base {
		merged[k] = v
	}
	for _, o := range override {
		for k, v := range o {
			merged[k] = v
		}
	}
	return r.configResolver.ResolveConfiguration(merged)
}

// GetToolForChat mints a tool instance scoped to one chat turn, with its
// configuration already resolved - see the interface doc comment for the
// Stateful-tool exception.
func (r *JsonToolRepository) GetToolForChat(ctx context.Context, name string, config map[string]string) (entities.Tool, error) {
	r.mu.RLock()
	var toolData *entities.ToolData
	for _, t := range r.data {
		if t.Name == name {
			toolData = t
			break
		}
	}
	r.mu.RUnlock()

	if toolData == nil {
		// Not a native, factory-built tool - check configured tool sources
		// (MCP servers). Source-provided tools have no ToolData row and no
		// per-turn config resolution of their own; the instance returned by
		// the source is used directly.
		sourceTool, err := r.findInSources(ctx, name)
		if err != nil {
			return nil, err
		}
		if sourceTool != nil {
			return sourceTool, nil
		}
		return nil, errors.NotFoundErrorf("tool not found: %s", name)
	}

	factoryEntry, err := r.toolFactory.GetFactoryByName(toolData.ToolType)
	if err != nil {
		return nil, err
	}

	if factoryEntry.Stateful {
		// Never mint a fresh instance for a stateful tool - that would
		// break state that must persist across turns and, if it wrote
		// resolved config back into the shared singleton, would
		// reintroduce the exact per-turn shared-write race this method
		// exists to avoid. Its configuration was already resolved once, in
		// reloadToolInstances, at construction time.
		r.mu.RLock()
		defer r.mu.RUnlock()
		instance, exists := r.toolInstances[name]
		if !exists {
			return nil, errors.NotFoundErrorf("tool instance not found: %s", name)
		}
		return instance, nil
	}

	resolvedConfig, err := r.resolveConfig(toolData.Configuration, config)
	if err != nil {
		return nil, errors.InternalErrorf("failed to resolve configuration for tool %s: %v", name, err)
	}

	return factoryEntry.Factory(toolData.Name, toolData.Description, resolvedConfig, r.logger), nil
}

func (r *JsonToolRepository) ListTools() ([]entities.Tool, error) {
	r.mu.RLock()
	var toolList []entities.Tool
	for _, tool := range r.toolInstances {
		toolList = append(toolList, tool)
	}
	r.mu.RUnlock()

	for _, source := range r.toolSources {
		sourceTools, err := source.ListTools(context.Background())
		if err != nil {
			r.logger.Warn("Failed to list tools from source, skipping", zap.String("source", source.Name()), zap.Error(err))
			continue
		}
		toolList = append(toolList, sourceTools...)
	}
	return toolList, nil
}

func (r *JsonToolRepository) GetToolByName(name string) (entities.Tool, error) {
	r.mu.RLock()
	tool, exists := r.toolInstances[name]
	r.mu.RUnlock()
	if exists {
		return tool, nil
	}

	return r.findInSources(context.Background(), name)
}

func (r *JsonToolRepository) RegisterTool(name string, tool entities.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.toolInstances[name]; exists {
		return errors.DuplicateErrorf("tool with the same name already exists")
	}
	r.toolInstances[name] = tool
	return nil
}

func (r *JsonToolRepository) ListToolData(ctx context.Context) ([]*entities.ToolData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	toolDataCopy := make([]*entities.ToolData, len(r.data))
	for i, t := range r.data {
		toolDataCopy[i] = copyToolData(t)
	}
	return toolDataCopy, nil
}

func (r *JsonToolRepository) GetToolData(ctx context.Context, id string) (*entities.ToolData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, toolData := range r.data {
		if toolData.ID == id {
			return copyToolData(toolData), nil
		}
	}
	return nil, errors.NotFoundErrorf("toolData not found: %s", id)
}

func (r *JsonToolRepository) CreateToolData(ctx context.Context, toolData *entities.ToolData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if toolData.ID == "" {
		toolData.ID = uuid.New().String()
	}
	toolData.CreatedAt = time.Now()
	toolData.UpdatedAt = toolData.CreatedAt

	r.data = append(r.data, copyToolData(toolData))
	if err := r.save(); err != nil {
		return err
	}
	return r.reloadToolInstances()
}

func (r *JsonToolRepository) UpdateToolData(ctx context.Context, toolData *entities.ToolData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, t := range r.data {
		if t.ID == toolData.ID {
			toolData.UpdatedAt = time.Now()
			r.data[i] = copyToolData(toolData)
			if err := r.save(); err != nil {
				return err
			}
			return r.reloadToolInstances()
		}
	}
	return errors.NotFoundErrorf("toolData not found: %s", toolData.ID)
}

func (r *JsonToolRepository) DeleteToolData(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, tool := range r.data {
		if tool.ID == id {
			r.data = slices.Delete(r.data, index, index+1)
			if err := r.save(); err != nil {
				return err
			}
			return r.reloadToolInstances()
		}
	}
	return errors.NotFoundErrorf("toolData not found: %s", id)
}

var _ interfaces.ToolRepository = (*JsonToolRepository)(nil)
