package interfaces

import (
	"context"

	"aiagent/internal/domain/entities"
)

type ConversationRepository interface {
	CreateConversation(ctx context.Context, conversation *entities.Conversation) error
	UpdateConversation(ctx context.Context, conversation *entities.Conversation) error
	DeleteConversation(ctx context.Context, id string) error
	GetConversation(ctx context.Context, id string) (*entities.Conversation, error)
	ListConversations(ctx context.Context) ([]*entities.Conversation, error)
}
