package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/infrastructure/integrations"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MessageListener func(conversationID string, message entities.Message)

type ChatService interface {
	SendMessage(ctx context.Context, conversationID string, message entities.Message) error
	CreateConversation(ctx context.Context, agentID string) (*entities.Conversation, error)
	ListActiveConversations(ctx context.Context) ([]*entities.Conversation, error)
	GetConversation(ctx context.Context, id string) (*entities.Conversation, error)
	AddMessageListener(listener MessageListener)
}

type chatService struct {
	conversationRepo interfaces.ConversationRepository
	agentRepo        interfaces.AgentRepository
	messageListeners []MessageListener
}

func NewChatService(conversationRepo interfaces.ConversationRepository, agentRepo interfaces.AgentRepository) *chatService {
	return &chatService{
		conversationRepo: conversationRepo,
		agentRepo:        agentRepo,
		messageListeners: []MessageListener{},
	}
}

func (s *chatService) AddMessageListener(listener MessageListener) {
	s.messageListeners = append(s.messageListeners, listener)
}

func (s *chatService) SendMessage(ctx context.Context, conversationID string, message entities.Message) error {
	if conversationID == "" {
		return fmt.Errorf("conversation ID is required")
	}
	if message.Role == "" || message.Content == "" {
		return fmt.Errorf("message role and content are required")
	}

	conversation, err := s.conversationRepo.GetConversation(ctx, conversationID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("conversation not found: %s", conversationID)
		}
		return fmt.Errorf("failed to retrieve conversation: %w", err)
	}

	message.ID = uuid.New().String()
	message.Timestamp = time.Now()
	conversation.Messages = append(conversation.Messages, message)
	conversation.UpdatedAt = time.Now()

	if err := s.conversationRepo.UpdateConversation(ctx, conversation); err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	go func(conv *entities.Conversation) {
		bgCtx := context.Background()
		agent, err := s.agentRepo.GetAgent(bgCtx, conv.AgentID.Hex())
		if err != nil {
			log.Printf("Failed to get agent %s: %v", conv.AgentID.Hex(), err)
			return
		}

		aiModel, err := integrations.NewGenericAIModel(agent.Endpoint, agent.APIKey, agent.Model)
		if err != nil {
			log.Printf("Failed to initialize AI model: %v", err)
			return
		}

		messages := make([]map[string]string, len(conv.Messages))
		for i, msg := range conv.Messages {
			messages[i] = map[string]string{
				"role":    msg.Role,
				"content": msg.Content,
			}
			if msg.Role == "tool" {
				messages[i]["tool_call_id"] = msg.ID // Assuming ID is used as tool_call_id
			}
		}

		// Add available tools to options if any
		options := map[string]interface{}{}
		if len(agent.Tools) > 0 {
			toolList := make([]map[string]string, 0, len(agent.Tools))
			for _, toolID := range agent.Tools {
				tool := integrations.GetToolByID(toolID.Hex())
				if tool != nil {
					toolList = append(toolList, map[string]string{
						"name": tool.Name(),
					})
				}
			}
			options["tools"] = toolList
		}

		if agent.Temperature != nil {
			options["temperature"] = *agent.Temperature
		} else {
			options["temperature"] = 0.7 // Default
		}
		if agent.MaxTokens != nil {
			options["max_tokens"] = *agent.MaxTokens
		} else {
			options["max_tokens"] = 1024 // Default
		}

		response, err := aiModel.GenerateResponse(messages, options)
		if err != nil {
			log.Printf("Failed to generate AI response: %v", err)
			return
		}

		aiMessage := entities.Message{
			ID:        uuid.New().String(),
			Role:      "assistant",
			Content:   response,
			Timestamp: time.Now(),
		}

		conv.Messages = append(conv.Messages, aiMessage)
		conv.UpdatedAt = time.Now()

		if err := s.conversationRepo.UpdateConversation(bgCtx, conv); err != nil {
			log.Printf("Failed to update conversation with AI response: %v", err)
			return
		}

		for _, listener := range s.messageListeners {
			listener(conversationID, aiMessage)
		}
	}(conversation)

	for _, listener := range s.messageListeners {
		listener(conversationID, message)
	}

	return nil
}

func (s *chatService) CreateConversation(ctx context.Context, agentID string) (*entities.Conversation, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	agentObjID, err := primitive.ObjectIDFromHex(agentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent ID: %v", err)
	}

	conversation := entities.NewConversation(agentObjID)
	if err := s.conversationRepo.CreateConversation(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %v", err)
	}

	return conversation, nil
}

func (s *chatService) ListActiveConversations(ctx context.Context) ([]*entities.Conversation, error) {
	conversations, err := s.conversationRepo.ListConversations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %v", err)
	}

	var activeConversations []*entities.Conversation
	for _, conv := range conversations {
		if conv.Active {
			activeConversations = append(activeConversations, conv)
		}
	}
	return activeConversations, nil
}

func (s *chatService) GetConversation(ctx context.Context, id string) (*entities.Conversation, error) {
	if id == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}

	conversation, err := s.conversationRepo.GetConversation(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("conversation not found: %s", id)
		}
		return nil, fmt.Errorf("failed to retrieve conversation: %v", err)
	}
	return conversation, nil
}
