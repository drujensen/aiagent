package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/repositories"
)

/**
 * @description
 * This file implements the ChatService for the AI Workflow Automation Platform.
 * It manages conversations for human interaction with AI agents, including message sending,
 * conversation creation, listing active conversations, and retrieving specific conversations.
 * The service interacts with the ConversationRepository and TaskRepository to persist and retrieve data,
 * and supports real-time message notifications via listeners for WebSocket integration.
 *
 * Key features:
 * - Conversation Management: Creates, updates, and retrieves conversations tied to tasks.
 * - Message Handling: Validates and appends messages to conversations with unique IDs.
 * - Active Conversation Listing: Filters conversations based on task status.
 * - Real-time Notifications: Supports message listeners for WebSocket broadcasting.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides Conversation, Message, and Task entities.
 * - aiagent/internal/domain/repositories: Provides ConversationRepository and TaskRepository interfaces.
 * - github.com/google/uuid: Used for generating unique message IDs.
 * - context: For timeout and cancellation support.
 * - fmt: For error message formatting.
 * - time: For timestamp handling.
 *
 * @notes
 * - Messages are validated based on their type (chat or tool) to ensure required fields.
 * - Conversations are only created for tasks requiring human interaction, verified via TaskRepository.
 * - Active conversations are filtered by task status, fetching task details for accuracy.
 * - Message listeners enable WebSocket integration, notifying the hub of new messages.
 * - Error handling wraps repository errors with context for better debugging.
 * - Assumption: TaskRepository and ConversationRepository are properly implemented and injected.
 * - Limitation: ListActiveConversations may be inefficient for large datasets; pagination could be added later.
 */

// MessageListener is a callback function type for notifying when a new message is added to a conversation.
// It is used to broadcast messages to connected WebSocket clients in real-time.
//
// Parameters:
// - conversationID: The ID of the conversation the message belongs to.
// - message: The Message entity that was added.
type MessageListener func(conversationID string, message entities.Message)

// ChatService defines the interface for managing conversations and messages in the domain layer.
// It supports human oversight by enabling message exchange and conversation tracking.
type ChatService interface {
	// SendMessage appends a message to a conversation and updates the repository.
	// Validates the message based on its type and generates a unique ID.
	// Returns an error if the conversation doesn't exist or persistence fails.
	SendMessage(ctx context.Context, conversationID string, message entities.Message) error

	// ReceiveMessage retrieves the latest message from a conversation.
	// Useful for UI initial load or reconnection; returns an error if no messages exist.
	ReceiveMessage(ctx context.Context, conversationID string) (entities.Message, error)

	// CreateConversation initializes a new conversation for a task requiring human interaction.
	// Returns the created conversation or an error if the task is invalid.
	CreateConversation(ctx context.Context, taskID string) (*entities.Conversation, error)

	// ListActiveConversations retrieves conversations awaiting human input.
	// Filters based on task status (RequiresHumanInteraction=true and Status!=TaskCompleted).
	ListActiveConversations(ctx context.Context) ([]*entities.Conversation, error)

	// GetConversation retrieves a conversation by its ID.
	// Returns the conversation or an error if it doesn’t exist or retrieval fails.
	GetConversation(ctx context.Context, id string) (*entities.Conversation, error)

	// AddMessageListener registers a listener to be notified of new messages.
	// Used for real-time updates via WebSocket.
	AddMessageListener(listener MessageListener)
}

// chatService implements the ChatService interface.
// It uses repositories to manage conversations and tasks, ensuring domain consistency.
type chatService struct {
	conversationRepo repositories.ConversationRepository
	taskRepo         repositories.TaskRepository
	messageListeners []MessageListener
}

// NewChatService creates a new instance of chatService with the given repositories.
//
// Parameters:
// - conversationRepo: Repository for managing Conversation entities.
// - taskRepo: Repository for managing Task entities to verify conversation eligibility.
//
// Returns:
// - *chatService: A new instance implementing ChatService.
func NewChatService(conversationRepo repositories.ConversationRepository, taskRepo repositories.TaskRepository) *chatService {
	return &chatService{
		conversationRepo: conversationRepo,
		taskRepo:         taskRepo,
		messageListeners: []MessageListener{},
	}
}

// AddMessageListener registers a listener to be notified when new messages are added.
// It appends the listener to the service's list, ensuring thread safety in the caller.
//
// Parameters:
// - listener: The callback function to invoke for new messages.
func (s *chatService) AddMessageListener(listener MessageListener) {
	s.messageListeners = append(s.messageListeners, listener)
}

// SendMessage appends a validated message to a conversation and updates the repository.
// Generates a unique ID, sets the timestamp, and notifies listeners for real-time updates.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - conversationID: ID of the conversation to append the message to.
// - message: Message entity to append, validated based on Type.
//
// Returns:
// - error: Nil on success, or an error if validation fails or persistence errors occur.
func (s *chatService) SendMessage(ctx context.Context, conversationID string, message entities.Message) error {
	// Validate conversationID is provided
	if conversationID == "" {
		return fmt.Errorf("conversation ID is required")
	}

	// Validate message fields based on type
	switch message.Type {
	case entities.ChatMessageType:
		if message.Content == "" || message.Sender == "" {
			return fmt.Errorf("chat message must have content and sender")
		}
		// Clear tool-specific fields if present
		message.ToolName = ""
		message.Request = ""
		message.Result = ""
	case entities.ToolMessageType:
		if message.ToolName == "" || message.Request == "" || message.Result == "" {
			return fmt.Errorf("tool message must have tool_name, request, and result")
		}
		// Clear chat-specific fields if present
		message.Content = ""
		message.Sender = ""
	default:
		return fmt.Errorf("invalid message type: %s", message.Type)
	}

	// Generate unique ID and set timestamp
	message.ID = uuid.New().String()
	message.Timestamp = time.Now()

	// Retrieve existing conversation
	conversation, err := s.conversationRepo.GetConversation(ctx, conversationID)
	if err != nil {
		if err == repositories.ErrNotFound {
			return fmt.Errorf("conversation not found: %s", conversationID)
		}
		return fmt.Errorf("failed to retrieve conversation: %w", err)
	}

	// Append message and update timestamp
	conversation.Messages = append(conversation.Messages, message)
	conversation.UpdatedAt = time.Now()

	// Persist updated conversation
	if err := s.conversationRepo.UpdateConversation(ctx, conversation); err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	// Notify listeners for real-time updates
	for _, listener := range s.messageListeners {
		listener(conversationID, message)
	}

	return nil
}

// ReceiveMessage retrieves the most recent message from a conversation.
// Returns an error if the conversation doesn’t exist or has no messages.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - conversationID: ID of the conversation to retrieve the latest message from.
//
// Returns:
// - entities.Message: The latest message in the conversation.
// - error: Nil on success, or an error if retrieval fails or no messages exist.
func (s *chatService) ReceiveMessage(ctx context.Context, conversationID string) (entities.Message, error) {
	if conversationID == "" {
		return entities.Message{}, fmt.Errorf("conversation ID is required")
	}

	// Retrieve conversation
	conversation, err := s.conversationRepo.GetConversation(ctx, conversationID)
	if err != nil {
		if err == repositories.ErrNotFound {
			return entities.Message{}, fmt.Errorf("conversation not found: %s", conversationID)
		}
		return entities.Message{}, fmt.Errorf("failed to retrieve conversation: %w", err)
	}

	// Check if there are messages
	if len(conversation.Messages) == 0 {
		return entities.Message{}, fmt.Errorf("no messages in conversation: %s", conversationID)
	}

	// Return the latest message (last in slice)
	return conversation.Messages[len(conversation.Messages)-1], nil
}

// CreateConversation initializes a new conversation for a task requiring human interaction.
// Verifies the task exists and requires human input before creation.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - taskID: ID of the task to associate with the conversation.
//
// Returns:
// - *entities.Conversation: The newly created conversation.
// - error: Nil on success, or an error if task validation or creation fails.
func (s *chatService) CreateConversation(ctx context.Context, taskID string) (*entities.Conversation, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID is required")
	}

	// Verify task exists and requires human interaction
	task, err := s.taskRepo.GetTask(ctx, taskID)
	if err != nil {
		if err == repositories.ErrNotFound {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to retrieve task: %w", err)
	}
	if !task.RequiresHumanInteraction {
		return nil, fmt.Errorf("task does not require human interaction: %s", taskID)
	}

	// Initialize new conversation
	now := time.Now()
	conversation := &entities.Conversation{
		TaskID:    taskID,
		Messages:  []entities.Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Persist conversation
	if err := s.conversationRepo.CreateConversation(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conversation, nil
}

// ListActiveConversations retrieves conversations where human input is still needed.
// Filters based on associated task’s RequiresHumanInteraction and Status fields.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
//
// Returns:
// - []*entities.Conversation: Slice of active conversations, empty if none.
// - error: Nil on success, or an error if retrieval fails.
func (s *chatService) ListActiveConversations(ctx context.Context) ([]*entities.Conversation, error) {
	// Retrieve all conversations
	conversations, err := s.conversationRepo.ListConversations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	// Filter active conversations
	var activeConversations []*entities.Conversation
	for _, conv := range conversations {
		task, err := s.taskRepo.GetTask(ctx, conv.TaskID)
		if err != nil {
			if err == repositories.ErrNotFound {
				// Skip if task is deleted; conversation is no longer active
				continue
			}
			return nil, fmt.Errorf("failed to retrieve task for conversation %s: %w", conv.ID, err)
		}

		// Include if task requires human interaction and is not completed
		if task.RequiresHumanInteraction && task.Status != entities.TaskCompleted {
			activeConversations = append(activeConversations, conv)
		}
	}

	return activeConversations, nil
}

// GetConversation retrieves a specific conversation by its ID.
// Used by the UI to fetch message history for display.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The ID of the conversation to retrieve.
//
// Returns:
// - *entities.Conversation: The retrieved conversation, or nil if not found.
// - error: Nil on success, ErrNotFound if conversation doesn’t exist, or another error otherwise.
func (s *chatService) GetConversation(ctx context.Context, id string) (*entities.Conversation, error) {
	if id == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}
	conversation, err := s.conversationRepo.GetConversation(ctx, id)
	if err != nil {
		if err == repositories.ErrNotFound {
			return nil, fmt.Errorf("conversation not found: %s", id)
		}
		return nil, fmt.Errorf("failed to retrieve conversation: %w", err)
	}
	return conversation, nil
}
