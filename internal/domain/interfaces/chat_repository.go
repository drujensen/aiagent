package interfaces

import (
	"context"

	"aiagent/internal/domain/entities"
)

type ChatRepository interface {
	CreateChat(ctx context.Context, chat *entities.Chat) error
	UpdateChat(ctx context.Context, chat *entities.Chat) error
	DeleteChat(ctx context.Context, id string) error
	GetChat(ctx context.Context, id string) (*entities.Chat, error)
	ListChats(ctx context.Context) ([]*entities.Chat, error)
}
