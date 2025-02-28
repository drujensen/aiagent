package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/infrastructure/config"
	"aiagent/internal/infrastructure/integrations"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MessageListener func(chatID string, message entities.Message)

type ChatService interface {
	SendMessage(ctx context.Context, chatID string, message entities.Message) error
	CreateChat(ctx context.Context, agentID, name string) (*entities.Chat, error)
	ListActiveChats(ctx context.Context) ([]*entities.Chat, error)
	GetChat(ctx context.Context, id string) (*entities.Chat, error)
	UpdateChat(ctx context.Context, id, name string) (*entities.Chat, error)
	AddMessageListener(listener MessageListener)
}

type chatService struct {
	chatRepo         interfaces.ChatRepository
	agentRepo        interfaces.AgentRepository
	config           *config.Config
	messageListeners []MessageListener
}

func NewChatService(chatRepo interfaces.ChatRepository, agentRepo interfaces.AgentRepository, cfg *config.Config) *chatService {
	return &chatService{
		chatRepo:         chatRepo,
		agentRepo:        agentRepo,
		config:           cfg, // Store the config
		messageListeners: []MessageListener{},
	}
}

func (s *chatService) AddMessageListener(listener MessageListener) {
	s.messageListeners = append(s.messageListeners, listener)
}

func (s *chatService) SendMessage(ctx context.Context, chatID string, message entities.Message) error {
	if chatID == "" {
		return fmt.Errorf("chat ID is required")
	}
	if message.Role == "" || message.Content == "" {
		return fmt.Errorf("message role and content are required")
	}

	chat, err := s.chatRepo.GetChat(ctx, chatID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("chat not found: %s", chatID)
		}
		return fmt.Errorf("failed to retrieve chat: %w", err)
	}

	message.ID = uuid.New().String()
	message.Timestamp = time.Now()
	chat.Messages = append(chat.Messages, message)
	chat.UpdatedAt = time.Now()

	if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
		return fmt.Errorf("failed to update chat: %w", err)
	}

	go func(conv *entities.Chat) {
		bgCtx := context.Background()
		agent, err := s.agentRepo.GetAgent(bgCtx, conv.AgentID.Hex())
		if err != nil {
			log.Printf("Failed to get agent %s: %v", conv.AgentID.Hex(), err)
			return
		}

		// Resolve the API key before initializing the AI model
		resolvedAPIKey, err := s.config.ResolveAPIKey(agent.APIKey)
		if err != nil {
			log.Printf("Failed to resolve API key for agent %s: %v", agent.ID.Hex(), err)
			return
		}

		aiModel, err := integrations.NewGenericAIModel(agent.Endpoint, resolvedAPIKey, agent.Model)
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

		if err := s.chatRepo.UpdateChat(bgCtx, conv); err != nil {
			log.Printf("Failed to update chat with AI response: %v", err)
			return
		}

		for _, listener := range s.messageListeners {
			listener(chatID, aiMessage)
		}
	}(chat)

	return nil
}

func (s *chatService) CreateChat(ctx context.Context, agentID, name string) (*entities.Chat, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	agentObjID, err := primitive.ObjectIDFromHex(agentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent ID: %v", err)
	}

	chat := entities.NewChat(agentObjID, name)
	if err := s.chatRepo.CreateChat(ctx, chat); err != nil {
		return nil, fmt.Errorf("failed to create chat: %v", err)
	}

	return chat, nil
}

func (s *chatService) UpdateChat(ctx context.Context, id, name string) (*entities.Chat, error) {
	if id == "" {
		return nil, fmt.Errorf("chat ID is required")
	}

	if name == "" {
		return nil, fmt.Errorf("chat name is required")
	}

	existingChat, err := s.chatRepo.GetChat(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("chat not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get existing chat: %v", err)
	}

	existingChat.Name = name
	existingChat.UpdatedAt = time.Now()

	if err := s.chatRepo.UpdateChat(ctx, existingChat); err != nil {
		return nil, fmt.Errorf("failed to update chat: %v", err)
	}

	return existingChat, nil
}

func (s *chatService) ListActiveChats(ctx context.Context) ([]*entities.Chat, error) {
	chats, err := s.chatRepo.ListChats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list chats: %v", err)
	}

	var activeChats []*entities.Chat
	for _, conv := range chats {
		if conv.Active {
			activeChats = append(activeChats, conv)
		}
	}
	return activeChats, nil
}

func (s *chatService) GetChat(ctx context.Context, id string) (*entities.Chat, error) {
	if id == "" {
		return nil, fmt.Errorf("chat ID is required")
	}

	chat, err := s.chatRepo.GetChat(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("chat not found: %s", id)
		}
		return nil, fmt.Errorf("failed to retrieve chat: %v", err)
	}

	return chat, nil
}
