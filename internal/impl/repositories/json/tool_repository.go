package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/impl/tools"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type jsonToolRepository struct {
	filePath      string
	data          []*entities.ToolData
	toolInstances map[string]*entities.Tool
	toolFactory   *tools.ToolFactory
	logger        *zap.Logger
	mu            sync.RWMutex
}

func NewJSONToolRepository(dataDir string, toolFactory *tools.ToolFactory, logger *zap.Logger) (interfaces.ToolRepository, error) {
	filePath := filepath.Join(dataDir, ".aiagent", "tools.json")
	repo := &jsonToolRepository{
		filePath:      filePath,
		data:          []*entities.ToolData{},
		toolInstances: make(map[string]*entities.Tool),
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

func (r *jsonToolRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet, start with empty data
	}
	if err != nil {
		return errors.InternalErrorf("failed to read tools.json: %v", err)
	}

	var tools []*entities.ToolData
	if err := json.Unmarshal(data, &tools); err != nil {
		return errors.InternalErrorf("failed to unmarshal tools.json: %v", err)
	}

	// Validate UUIDs
	for _, tool := range tools {
		if tool.ID == "" {
			return errors.InternalErrorf("tool is missing an ID")
		}
		if _, err := uuid.Parse(tool.ID); err != nil {
			return errors.InternalErrorf("tool has an invalid UUID: %v", err)
		}
	}

	r.data = tools
	return nil
}

func (r *jsonToolRepository) save() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return errors.InternalErrorf("failed to marshal tools: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return errors.InternalErrorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return errors.InternalErrorf("failed to write tools.json: %v", err)
	}

	return nil
}

func (r *jsonToolRepository) reloadToolInstances() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.toolInstances = make(map[string]*entities.Tool)
	for _, toolData := range r.data {
		toolFactoryEntry, err := r.toolFactory.GetFactoryByName(toolData.ToolType)
		if err != nil {
			r.logger.Warn("Skipping tool due to unknown type", zap.String("tool_type", toolData.ToolType), zap.Error(err))
			continue
		}
		tool := toolFactoryEntry.Factory(toolData.Name, toolData.Description, toolData.Configuration, r.logger)
		r.toolInstances[toolData.Name] = &tool
	}
	return nil
}

func (r *jsonToolRepository) ListTools() ([]*entities.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []*entities.Tool
	for _, tool := range r.toolInstances {
		tools = append(tools, tool)
	}
	return tools, nil
}

func (r *jsonToolRepository) GetToolByName(name string) (*entities.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.toolInstances[name]
	if !exists {
		return nil, nil
	}
	return tool, nil
}

func (r *jsonToolRepository) RegisterTool(name string, tool *entities.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.toolInstances[name]; exists {
		return errors.DuplicateErrorf("tool with the same name already exists")
	}
	r.toolInstances[name] = tool
	return nil
}

func (r *jsonToolRepository) ListToolData(ctx context.Context) ([]*entities.ToolData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	toolDataCopy := make([]*entities.ToolData, len(r.data))
	for i, t := range r.data {
		toolDataCopy[i] = &entities.ToolData{
			ID:            t.ID,
			Name:          t.Name,
			Description:   t.Description,
			ToolType:      t.ToolType,
			Configuration: t.Configuration,
			CreatedAt:     t.CreatedAt,
			UpdatedAt:     t.UpdatedAt,
		}
	}
	return toolDataCopy, nil
}

func (r *jsonToolRepository) GetToolData(ctx context.Context, id string) (*entities.ToolData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, toolData := range r.data {
		if toolData.ID == id {
			return &entities.ToolData{
				ID:            toolData.ID,
				Name:          toolData.Name,
				Description:   toolData.Description,
				ToolType:      toolData.ToolType,
				Configuration: toolData.Configuration,
				CreatedAt:     toolData.CreatedAt,
				UpdatedAt:     toolData.UpdatedAt,
			}, nil
		}
	}
	return nil, errors.NotFoundErrorf("toolData not found: %s", id)
}

func (r *jsonToolRepository) CreateToolData(ctx context.Context, toolData *entities.ToolData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if toolData.ID == "" {
		toolData.ID = uuid.New().String()
	}
	toolData.CreatedAt = time.Now()
	toolData.UpdatedAt = toolData.CreatedAt

	r.data = append(r.data, toolData)
	if err := r.save(); err != nil {
		return err
	}
	return r.reloadToolInstances()
}

func (r *jsonToolRepository) UpdateToolData(ctx context.Context, toolData *entities.ToolData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, t := range r.data {
		if t.ID == toolData.ID {
			toolData.UpdatedAt = time.Now()
			r.data[i] = toolData
			if err := r.save(); err != nil {
				return err
			}
			return r.reloadToolInstances()
		}
	}
	return errors.NotFoundErrorf("toolData not found: %s", toolData.ID)
}

func (r *jsonToolRepository) DeleteToolData(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, t := range r.data {
		if t.ID == id {
			r.data = append(r.data[:i], r.data[i+1:]...)
			if err := r.save(); err != nil {
				return err
			}
			return r.reloadToolInstances()
		}
	}
	return errors.NotFoundErrorf("toolData not found: %s", id)
}

var _ interfaces.ToolRepository = (*jsonToolRepository)(nil)
