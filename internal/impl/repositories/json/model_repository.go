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

	"github.com/google/uuid"
)

type JsonModelRepository struct {
	mu       sync.RWMutex
	filePath string
	data     []*entities.Model
}

func NewJSONModelRepository(storageDir string) (interfaces.ModelRepository, error) {
	filePath := filepath.Join(storageDir, "models.json")
	repo := &JsonModelRepository{
		filePath: filePath,
		data:     []*entities.Model{},
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func copyModel(m *entities.Model) *entities.Model {
	return &entities.Model{
		ID:               m.ID,
		Name:             m.Name,
		ProviderID:       m.ProviderID,
		ProviderType:     m.ProviderType,
		ModelName:        m.ModelName,
		APIKey:           m.APIKey,
		Temperature:      copyFloat64(m.Temperature),
		MaxTokens:        copyIntPtr(m.MaxTokens),
		ContextWindow:    copyIntPtr(m.ContextWindow),
		ReasoningEffort:  m.ReasoningEffort,
		Family:           m.Family,
		Reasoning:        m.Reasoning,
		ToolCall:         m.ToolCall,
		TemperatureCap:   m.TemperatureCap,
		Attachment:       m.Attachment,
		StructuredOutput: m.StructuredOutput,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

// load and save are only ever called by exported methods that already hold
// r.mu, so neither acquires the lock itself.
func (r *JsonModelRepository) load() error {
	data, err := os.ReadFile(r.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return errors.InternalErrorf("failed to read models.json: %v", err)
	}

	var models []*entities.Model
	if err := json.Unmarshal(data, &models); err != nil {
		return errors.InternalErrorf("failed to unmarshal models.json: %v", err)
	}

	for _, model := range models {
		if model.ID == "" {
			return errors.InternalErrorf("model is missing an ID")
		}
		if _, err := uuid.Parse(model.ID); err != nil {
			return errors.InternalErrorf("model has an invalid UUID: %v", err)
		}
	}

	r.data = models
	return nil
}

func (r *JsonModelRepository) save() error {
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return errors.InternalErrorf("failed to marshal models: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return errors.InternalErrorf("failed to create directory: %v", err)
	}

	return atomicWriteFile(r.filePath, data)
}

func (r *JsonModelRepository) ListModels(ctx context.Context) ([]*entities.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modelsCopy := make([]*entities.Model, len(r.data))
	for i, m := range r.data {
		modelsCopy[i] = copyModel(m)
	}
	return modelsCopy, nil
}

func (r *JsonModelRepository) GetModel(ctx context.Context, id string) (*entities.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, model := range r.data {
		if model.ID == id {
			return copyModel(model), nil
		}
	}
	return nil, errors.NotFoundErrorf("model not found: %s", id)
}

func (r *JsonModelRepository) CreateModel(ctx context.Context, model *entities.Model) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if model.ID == "" {
		model.ID = uuid.New().String()
	}
	model.CreatedAt = time.Now()
	model.UpdatedAt = model.CreatedAt

	r.data = append(r.data, copyModel(model))
	return r.save()
}

func (r *JsonModelRepository) UpdateModel(ctx context.Context, model *entities.Model) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, m := range r.data {
		if m.ID == model.ID {
			model.UpdatedAt = time.Now()
			r.data[i] = copyModel(model)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("model not found: %s", model.ID)
}

func (r *JsonModelRepository) DeleteModel(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, m := range r.data {
		if m.ID == id {
			r.data = slices.Delete(r.data, i, i+1)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("model not found: %s", id)
}

func (r *JsonModelRepository) GetModelsByProvider(ctx context.Context, providerID string) ([]*entities.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []*entities.Model
	for _, model := range r.data {
		if model.ProviderID == providerID {
			models = append(models, copyModel(model))
		}
	}
	return models, nil
}

func copyFloat64(f *float64) *float64 {
	if f == nil {
		return nil
	}
	copied := *f
	return &copied
}

func copyIntPtr(i *int) *int {
	if i == nil {
		return nil
	}
	copied := *i
	return &copied
}

var _ interfaces.ModelRepository = (*JsonModelRepository)(nil)
