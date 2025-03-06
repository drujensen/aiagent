package services

import (
	"context"
	"fmt"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/infrastructure/config"
	"aiagent/internal/infrastructure/integrations"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChatService interface {
	SendMessage(ctx context.Context, chatID string, message entities.Message) (*entities.Message, error)
	CreateChat(ctx context.Context, agentID, name string) (*entities.Chat, error)
	ListActiveChats(ctx context.Context) ([]*entities.Chat, error)
	GetChat(ctx context.Context, id string) (*entities.Chat, error)
	UpdateChat(ctx context.Context, id, name string) (*entities.Chat, error)
}

type chatService struct {
	chatRepo  interfaces.ChatRepository
	agentRepo interfaces.AgentRepository
	config    *config.Config
}

func NewChatService(chatRepo interfaces.ChatRepository, agentRepo interfaces.AgentRepository, cfg *config.Config) *chatService {
	return &chatService{
		chatRepo:  chatRepo,
		agentRepo: agentRepo,
		config:    cfg,
	}
}

func (s *chatService) SendMessage(ctx context.Context, chatID string, message entities.Message) (*entities.Message, error) {
	if chatID == "" {
		return nil, fmt.Errorf("chat ID is required")
	}
	if message.Role == "" || message.Content == "" {
		return nil, fmt.Errorf("message role and content are required")
	}

	chat, err := s.chatRepo.GetChat(ctx, chatID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("chat not found: %s", chatID)
		}
		return nil, fmt.Errorf("failed to retrieve chat: %w", err)
	}

	message.ID = uuid.New().String()
	message.Timestamp = time.Now()
	chat.Messages = append(chat.Messages, message)
	chat.UpdatedAt = time.Now()

	if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
		return nil, fmt.Errorf("failed to update chat: %w", err)
	}

	// Generate AI response synchronously
	agent, err := s.agentRepo.GetAgent(ctx, chat.AgentID.Hex())
	if err != nil {
		return nil, fmt.Errorf("failed to get agent %s: %v", chat.AgentID.Hex(), err)
	}

	resolvedAPIKey, err := s.config.ResolveAPIKey(agent.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve API key for agent %s: %v", agent.ID.Hex(), err)
	}

	aiModel, err := integrations.NewAIModelIntegration(agent.Endpoint, resolvedAPIKey, agent.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI model: %v", err)
	}

	messages := make([]map[string]string, len(chat.Messages))
	for i, msg := range chat.Messages {
		messages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.Role == "tool" {
			messages[i]["tool_call_id"] = msg.ID
		}
	}

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
		options["temperature"] = 0.0
	}
	if agent.MaxTokens != nil {
		options["max_tokens"] = *agent.MaxTokens
	} else {
		options["max_tokens"] = 128000
	}

	response, err := aiModel.GenerateResponse(messages, options)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AI response: %v", err)
	}

	aiMessage := entities.NewMessage("assistant", response)
	chat.Messages = append(chat.Messages, *aiMessage)
	chat.UpdatedAt = time.Now()

	if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
		return nil, fmt.Errorf("failed to update chat with AI response: %v", err)
	}

	return aiMessage, nil
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
