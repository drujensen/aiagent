package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
)

type JsonModelRepository struct {
	filePath string
	data     []*entities.Model
}

func NewJSONModelRepository(dataDir string) (interfaces.ModelRepository, error) {
	filePath := filepath.Join(dataDir, ".aiagent", "models.json")
	repo := &JsonModelRepository{
		filePath: filePath,
		data:     []*entities.Model{},
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *JsonModelRepository) load() error {
	data, err := os.ReadFile(r.filePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet, start with empty data
	}
	if err != nil {
		return errors.InternalErrorf("failed to read models.json: %v", err)
	}

	var models []*entities.Model
	if err := json.Unmarshal(data, &models); err != nil {
		return errors.InternalErrorf("failed to unmarshal models.json: %v", err)
	}

	// Validate UUIDs
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

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return errors.InternalErrorf("failed to write models.json: %v", err)
	}

	return nil
}

func (r *JsonModelRepository) ListModels(ctx context.Context) ([]*entities.Model, error) {
	modelsCopy := make([]*entities.Model, len(r.data))
	for i, m := range r.data {
		modelsCopy[i] = &entities.Model{
			ID:            m.ID,
			Name:          m.Name,
			ProviderID:    m.ProviderID,
			ProviderType:  m.ProviderType,
			ContextWindow: m.ContextWindow,
			Capabilities: entities.ModelCapabilities{
				SupportsTools:  m.Capabilities.SupportsTools,
				SupportsImages: m.Capabilities.SupportsImages,
				SupportsVision: m.Capabilities.SupportsVision,
			},
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		}
	}
	return modelsCopy, nil
}

func (r *JsonModelRepository) GetModel(ctx context.Context, id string) (*entities.Model, error) {
	for _, model := range r.data {
		if model.ID == id {
			return &entities.Model{
				ID:            model.ID,
				Name:          model.Name,
				ProviderID:    model.ProviderID,
				ProviderType:  model.ProviderType,
				ContextWindow: model.ContextWindow,
				Capabilities: entities.ModelCapabilities{
					SupportsTools:  model.Capabilities.SupportsTools,
					SupportsImages: model.Capabilities.SupportsImages,
					SupportsVision: model.Capabilities.SupportsVision,
				},
				CreatedAt: model.CreatedAt,
				UpdatedAt: model.UpdatedAt,
			}, nil
		}
	}
	return nil, errors.NotFoundErrorf("model not found: %s", id)
}

func (r *JsonModelRepository) CreateModel(ctx context.Context, model *entities.Model) error {
	if model.ID == "" {
		model.ID = uuid.New().String()
	}
	model.CreatedAt = time.Now()
	model.UpdatedAt = model.CreatedAt

	r.data = append(r.data, model)
	return r.save()
}

func (r *JsonModelRepository) UpdateModel(ctx context.Context, model *entities.Model) error {
	for i, m := range r.data {
		if m.ID == model.ID {
			model.UpdatedAt = time.Now()
			r.data[i] = model
			return r.save()
		}
	}
	return errors.NotFoundErrorf("model not found: %s", model.ID)
}

func (r *JsonModelRepository) DeleteModel(ctx context.Context, id string) error {
	for i, m := range r.data {
		if m.ID == id {
			r.data = slices.Delete(r.data, i, i+1)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("model not found: %s", id)
}

var _ interfaces.ModelRepository = (*JsonModelRepository)(nil)
