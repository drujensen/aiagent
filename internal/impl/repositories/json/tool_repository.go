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
	mu            sync.RWMutex
	filePath      string
	data          []*entities.ToolData
	toolInstances map[string]entities.Tool
	toolFactory   *tools.ToolFactory
	logger        *zap.Logger
}

func NewJSONToolRepository(storageDir string, toolFactory *tools.ToolFactory, logger *zap.Logger) (interfaces.ToolRepository, error) {
	filePath := filepath.Join(storageDir, "tools.json")
	repo := &JsonToolRepository{
		filePath:      filePath,
		data:          []*entities.ToolData{},
		toolInstances: make(map[string]entities.Tool),
		toolFactory:   toolFactory,
		logger:        logger,
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	if err := repo.reloadToolInstances(); err != nil {
		return nil, err
	}

	return repo, nil
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
		// Pass a copy of the configuration map into the factory so the
		// resulting tool instance never shares backing storage with r.data.
		configCopy := copyToolData(toolData).Configuration
		tool := toolFactoryEntry.Factory(toolData.Name, toolData.Description, configCopy, r.logger)
		r.toolInstances[toolData.Name] = tool
	}
	return nil
}

func (r *JsonToolRepository) ListTools() ([]entities.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var toolList []entities.Tool
	for _, tool := range r.toolInstances {
		toolList = append(toolList, tool)
	}
	return toolList, nil
}

func (r *JsonToolRepository) GetToolByName(name string) (entities.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.toolInstances[name]
	if !exists {
		return nil, nil
	}
	return tool, nil
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
