package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
)

type JsonChatRepository struct {
	filePath string
	data     []*entities.Chat
}

func NewJSONChatRepository(dataDir string) (interfaces.ChatRepository, error) {
	filePath := filepath.Join(dataDir, ".aiagent", "chats.json")
	repo := &JsonChatRepository{
		filePath: filePath,
		data:     []*entities.Chat{},
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *JsonChatRepository) load() error {
	data, err := os.ReadFile(r.filePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet, start with empty data
	}
	if err != nil {
		return errors.InternalErrorf("failed to read chats.json: %v", err)
	}

	var chats []*entities.Chat
	if err := json.Unmarshal(data, &chats); err != nil {
		return errors.InternalErrorf("failed to unmarshal chats.json: %v", err)
	}

	// Validate UUIDs
	for _, chat := range chats {
		if chat.ID == "" {
			return errors.InternalErrorf("chat is missing an ID")
		}
		if _, err := uuid.Parse(chat.ID); err != nil {
			return errors.InternalErrorf("chat has an invalid UUID: %v", err)
		}
	}

	r.data = chats
	return nil
}

func (r *JsonChatRepository) save() error {
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return errors.InternalErrorf("failed to marshal chats: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(r.filePath), 0755); err != nil {
		return errors.InternalErrorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return errors.InternalErrorf("failed to write chats.json: %v", err)
	}

	return nil
}

func (r *JsonChatRepository) ListChats(ctx context.Context) ([]*entities.Chat, error) {
	chatsCopy := make([]*entities.Chat, len(r.data))
	for i, c := range r.data {
		messagesCopy := make([]entities.Message, len(c.Messages))
		copy(messagesCopy, c.Messages)
		chatsCopy[i] = &entities.Chat{
			ID:        c.ID,
			AgentID:   c.AgentID,
			ModelID:   c.ModelID,
			Name:      c.Name,
			Messages:  messagesCopy,
			Usage:     c.Usage,
			Active:    c.Active,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		}
	}

	// Sort chatsCopy by UpdatedAt in descending order
	sort.Slice(chatsCopy, func(i, j int) bool {
		return chatsCopy[i].UpdatedAt.Before(chatsCopy[j].UpdatedAt)
	})

	return chatsCopy, nil
}

func (r *JsonChatRepository) GetChat(ctx context.Context, id string) (*entities.Chat, error) {
	for _, chat := range r.data {
		if chat.ID == id {
			messagesCopy := make([]entities.Message, len(chat.Messages))
			copy(messagesCopy, chat.Messages)
			return &entities.Chat{
				ID:        chat.ID,
				AgentID:   chat.AgentID,
				ModelID:   chat.ModelID,
				Name:      chat.Name,
				Messages:  messagesCopy,
				Usage:     chat.Usage,
				Active:    chat.Active,
				CreatedAt: chat.CreatedAt,
				UpdatedAt: chat.UpdatedAt,
			}, nil
		}
	}
	return nil, errors.NotFoundErrorf("chat not found: %s", id)
}

func (r *JsonChatRepository) CreateChat(ctx context.Context, chat *entities.Chat) error {
	if chat.ID == "" {
		chat.ID = uuid.New().String()
	}
	chat.CreatedAt = time.Now()
	chat.UpdatedAt = chat.CreatedAt

	r.data = append(r.data, chat)
	return r.save()
}

func (r *JsonChatRepository) UpdateChat(ctx context.Context, chat *entities.Chat) error {
	for i, c := range r.data {
		if c.ID == chat.ID {
			chat.UpdatedAt = time.Now()
			r.data[i] = chat
			return r.save()
		}
	}
	return errors.NotFoundErrorf("chat not found: %s", chat.ID)
}

func (r *JsonChatRepository) DeleteChat(ctx context.Context, id string) error {
	for i, c := range r.data {
		if c.ID == id {
			r.data = slices.Delete(r.data, i, i+1)
			return r.save()
		}
	}
	return errors.NotFoundErrorf("chat not found: %s", id)
}

var _ interfaces.ChatRepository = (*JsonChatRepository)(nil)
