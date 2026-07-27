package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"github.com/google/uuid"
)

type JsonChatRepository struct {
	mu       sync.RWMutex
	filePath string
	data     map[string]*entities.Chat
}

func NewJSONChatRepository(storageDir string) (interfaces.ChatRepository, error) {
	filePath := filepath.Join(storageDir, "chats.json")
	repo := &JsonChatRepository{
		filePath: filePath,
		data:     make(map[string]*entities.Chat),
	}

	if err := repo.load(); err != nil {
		return nil, err
	}

	return repo, nil
}

// deepCopyChat returns a Chat whose Usage pointer, and whose Messages'
// Usage pointers/ToolCalls/ToolCallEvents slices, are all newly allocated -
// never aliasing the argument's backing memory. Required on every read and
// write path so a caller mutating its own copy (e.g. Chat.UpdateUsage) can
// never race with a concurrent reader/writer touching the repository's copy.
func deepCopyChat(c *entities.Chat) *entities.Chat {
	messagesCopy := make([]entities.Message, len(c.Messages))
	for i, m := range c.Messages {
		messagesCopy[i] = deepCopyMessage(m)
	}

	var usageCopy *entities.ChatUsage
	if c.Usage != nil {
		u := *c.Usage
		usageCopy = &u
	}

	return &entities.Chat{
		ID:           c.ID,
		AgentID:      c.AgentID,
		ModelID:      c.ModelID,
		Name:         c.Name,
		Messages:     messagesCopy,
		Usage:        usageCopy,
		Active:       c.Active,
		ParentChatID: c.ParentChatID,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

func deepCopyMessage(m entities.Message) entities.Message {
	mc := m

	if m.Usage != nil {
		u := *m.Usage
		mc.Usage = &u
	}

	if m.ToolCalls != nil {
		mc.ToolCalls = make([]entities.ToolCall, len(m.ToolCalls))
		copy(mc.ToolCalls, m.ToolCalls)
	}

	if m.ToolCallEvents != nil {
		mc.ToolCallEvents = make([]entities.ToolCallEvent, len(m.ToolCallEvents))
		for i, e := range m.ToolCallEvents {
			mc.ToolCallEvents[i] = deepCopyToolCallEvent(e)
		}
	}

	return mc
}

func deepCopyToolCallEvent(e entities.ToolCallEvent) entities.ToolCallEvent {
	ec := e
	if e.Metadata != nil {
		ec.Metadata = make(map[string]string, len(e.Metadata))
		for k, v := range e.Metadata {
			ec.Metadata[k] = v
		}
	}
	return ec
}

// load and save are only ever called by exported methods that already hold
// r.mu, so neither acquires the lock itself.
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

	return atomicWriteFile(r.filePath, data)
}

func (r *JsonChatRepository) ListChats(ctx context.Context) ([]*entities.Chat, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chatsCopy := make([]*entities.Chat, 0, len(r.data))
	for _, c := range r.data {
		chatsCopy = append(chatsCopy, deepCopyChat(c))
	}
	sort.Slice(chatsCopy, func(i, j int) bool {
		return chatsCopy[i].UpdatedAt.After(chatsCopy[j].UpdatedAt)
	})
	return chatsCopy, nil
}

func (r *JsonChatRepository) GetChat(ctx context.Context, id string) (*entities.Chat, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chat, exists := r.data[id]
	if !exists {
		return nil, errors.NotFoundErrorf("chat not found: %s", id)
	}

	return deepCopyChat(chat), nil
}

func (r *JsonChatRepository) CreateChat(ctx context.Context, chat *entities.Chat) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if chat.ID == "" {
		chat.ID = uuid.New().String()
	}
	chat.CreatedAt = time.Now()
	chat.UpdatedAt = chat.CreatedAt

	// Store a deep copy, not the caller's pointer - the caller may keep
	// mutating its own object (e.g. via UpdateUsage) after this returns,
	// which must never be visible to a concurrent reader of the repository.
	r.data[chat.ID] = deepCopyChat(chat)
	return r.save()
}

func (r *JsonChatRepository) UpdateChat(ctx context.Context, chat *entities.Chat) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[chat.ID]; !exists {
		return errors.NotFoundErrorf("chat not found: %s", chat.ID)
	}
	chat.UpdatedAt = time.Now()
	r.data[chat.ID] = deepCopyChat(chat)
	return r.save()
}

func (r *JsonChatRepository) DeleteChat(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[id]; !exists {
		return errors.NotFoundErrorf("chat not found: %s", id)
	}
	delete(r.data, id)
	return r.save()
}

var _ interfaces.ChatRepository = (*JsonChatRepository)(nil)
