package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
)

type JsonChatRepository struct {
	filePath string
	data     map[string]*entities.Chat
}

func NewJSONChatRepository(dataDir string) (interfaces.ChatRepository, error) {
	filePath := filepath.Join(dataDir, ".aiagent", "chats.json")
	repo := &JsonChatRepository{
		filePath: filePath,
		data:     make(map[string]*entities.Chat),
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

	// Convert slice to map
	r.data = make(map[string]*entities.Chat)
	for _, chat := range chats {
		r.data[chat.ID] = chat
	}
	return nil
}

func (r *JsonChatRepository) save() error {
	// Convert map to slice for JSON serialization
	chats := make([]*entities.Chat, 0, len(r.data))
	for _, chat := range r.data {
		chats = append(chats, chat)
	}

	data, err := json.MarshalIndent(chats, "", "  ")
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
	chatsCopy := make([]*entities.Chat, 0, len(r.data))
	for _, c := range r.data {
		messagesCopy := make([]entities.Message, len(c.Messages))
		copy(messagesCopy, c.Messages)
		chatsCopy = append(chatsCopy, &entities.Chat{
			ID:        c.ID,
			AgentID:   c.AgentID,
			ModelID:   c.ModelID,
			Name:      c.Name,
			Messages:  messagesCopy,
			Usage:     c.Usage,
			Active:    c.Active,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}
	sort.Slice(chatsCopy, func(i, j int) bool {
		return chatsCopy[i].UpdatedAt.After(chatsCopy[j].UpdatedAt)
	})
	return chatsCopy, nil
}

func (r *JsonChatRepository) GetChat(ctx context.Context, id string) (*entities.Chat, error) {
	chat, exists := r.data[id]
	if !exists {
		return nil, errors.NotFoundErrorf("chat not found: %s", id)
	}

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

func (r *JsonChatRepository) CreateChat(ctx context.Context, chat *entities.Chat) error {
	if chat.ID == "" {
		chat.ID = uuid.New().String()
	}
	chat.CreatedAt = time.Now()
	chat.UpdatedAt = chat.CreatedAt

	r.data[chat.ID] = chat
	return r.save()
}

func (r *JsonChatRepository) UpdateChat(ctx context.Context, chat *entities.Chat) error {
	if _, exists := r.data[chat.ID]; !exists {
		return errors.NotFoundErrorf("chat not found: %s", chat.ID)
	}
	chat.UpdatedAt = time.Now()
	r.data[chat.ID] = chat
	return r.save()
}

func (r *JsonChatRepository) DeleteChat(ctx context.Context, id string) error {
	if _, exists := r.data[id]; !exists {
		return errors.NotFoundErrorf("chat not found: %s", id)
	}
	delete(r.data, id)
	return r.save()
}

var _ interfaces.ChatRepository = (*JsonChatRepository)(nil)
