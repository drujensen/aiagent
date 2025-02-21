package repositories

import (
	"context"

	"aiagent/internal/domain/entities"
)

// ConversationRepository defines the interface for managing Conversation entities in the domain layer.
// It provides methods to create, update, delete, and retrieve conversations, abstracting the data
// storage implementation (e.g., MongoDB). This interface supports chat-based human interaction
// without coupling the domain logic to specific storage details.
//
// Key features:
// - CRUD Operations: Handles the full lifecycle of conversations.
// - Context Usage: Includes context.Context for timeout and cancellation support.
// - Error Handling: Uses ErrNotFound (from package scope) for non-existent conversations in GetConversation and DeleteConversation.
//
// Dependencies:
// - context: For handling request timeouts and cancellations.
// - aiagent/internal/domain/entities: Provides the Conversation entity definition.
//
// Notes:
// - CreateConversation sets the conversation's ID in place, assuming storage generates it.
// - UpdateConversation requires a valid ID and typically appends messages.
// - ListConversations returns pointers to optimize memory usage for message-heavy conversations.
type ConversationRepository interface {
	// CreateConversation inserts a new conversation into the repository and sets the conversation's ID.
	// The conversation's ID field is updated with the generated ID upon successful insertion.
	// Returns an error if the insertion fails (e.g., invalid TaskID, database error).
	CreateConversation(ctx context.Context, conversation *entities.Conversation) error

	// UpdateConversation updates an existing conversation in the repository.
	// The conversation must have a valid ID; returns an error if it does not exist
	// or if the update fails (e.g., message validation error, database issue).
	UpdateConversation(ctx context.Context, conversation *entities.Conversation) error

	// DeleteConversation deletes a conversation by its ID.
	// Returns ErrNotFound if no conversation with the given ID exists; otherwise, returns
	// an error if the deletion fails (e.g., database connectivity issue).
	DeleteConversation(ctx context.Context, id string) error

	// GetConversation retrieves a conversation by its ID.
	// Returns a pointer to the Conversation and nil error on success, or nil and ErrNotFound
	// if the conversation does not exist, or nil and another error for other failures (e.g., database error).
	GetConversation(ctx context.Context, id string) (*entities.Conversation, error)

	// ListConversations retrieves all conversations in the repository.
	// Returns a slice of pointers to Conversation entities; returns an empty slice if no conversations exist,
	// or an error if the retrieval fails (e.g., database connection lost).
	ListConversations(ctx context.Context) ([]*entities.Conversation, error)
}

// Notes:
// - Error handling expects detailed errors from implementations (e.g., "conversation not found: id=xyz").
// - Edge case: Empty Messages slice is valid at creation, populated via updates.
// - Assumption: Conversation IDs are strings, consistent with MongoDB ObjectID usage.
// - Limitation: No filtering for active conversations; could be added for chat UI optimization.
