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
	"go.uber.org/zap"
)

type ChatService interface {
	SendMessage(ctx context.Context, chatID string, message entities.Message) (*entities.Message, error)
	CreateChat(ctx context.Context, agentID, name string) (*entities.Chat, error)
	ListActiveChats(ctx context.Context) ([]*entities.Chat, error)
	GetChat(ctx context.Context, id string) (*entities.Chat, error)
	UpdateChat(ctx context.Context, id, name string) (*entities.Chat, error)
	DeleteChat(ctx context.Context, id string) error
}

type chatService struct {
	chatRepo     interfaces.ChatRepository
	agentRepo    interfaces.AgentRepository
	providerRepo interfaces.ProviderRepository
	toolRepo     interfaces.ToolRepository
	config       *config.Config
	logger       *zap.Logger
}

func NewChatService(
	chatRepo interfaces.ChatRepository, 
	agentRepo interfaces.AgentRepository, 
	providerRepo interfaces.ProviderRepository,
	toolRepo interfaces.ToolRepository, 
	cfg *config.Config, 
	logger *zap.Logger,
) *chatService {
	return &chatService{
		chatRepo:     chatRepo,
		agentRepo:    agentRepo,
		providerRepo: providerRepo,
		toolRepo:     toolRepo,
		config:       cfg,
		logger:       logger,
	}
}

// estimateTokens approximates the token count for a message.
func estimateTokens(msg *entities.Message) int {
	contentTokens := (len(msg.Content) + 3) / 4
	// Approximate additional tokens for role, etc.
	return contentTokens + 4
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

	s.logger.Info("Before update", zap.Any("chat", chat))

	if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
		return nil, fmt.Errorf("failed to update chat: %w", err)
	}

	// Generate AI response synchronously
	agent, err := s.agentRepo.GetAgent(ctx, chat.AgentID.Hex())
	if err != nil {
		return nil, fmt.Errorf("failed to get agent %s: %v", chat.AgentID.Hex(), err)
	}

	// Get the provider
	provider, err := s.providerRepo.GetProvider(ctx, agent.ProviderID.Hex())
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s: %v", agent.ProviderID.Hex(), err)
	}

	resolvedAPIKey, err := s.config.ResolveAPIKey(agent.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve API key for agent %s: %v", agent.ID.Hex(), err)
	}

	// Create AI model integration based on provider type
	aiModelFactory := integrations.NewAIModelFactory(s.toolRepo, s.logger)
	aiModel, err := aiModelFactory.CreateModelIntegration(agent, provider, resolvedAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI model: %v", err)
	}

	contextLength := 128000
	if agent.ContextWindow != nil {
		contextLength = *agent.ContextWindow
	}

	maxTokens := 4096
	if agent.MaxTokens != nil {
		maxTokens = *agent.MaxTokens
	}

	tokenLimit := contextLength - maxTokens

	// Create system message
	systemMessage := &entities.Message{
		Role:    "system",
		Content: agent.SystemPrompt,
	}
	systemTokens := estimateTokens(systemMessage)
	if systemTokens > tokenLimit {
		return nil, fmt.Errorf("system prompt too large for the context window")
	}

	// Select messages within token limit
	var tempMessages []*entities.Message
	currentTokens := systemTokens
	for i := len(chat.Messages) - 1; i >= 0; i-- {
		msg := chat.Messages[i]
		msgTokens := estimateTokens(&msg)
		if currentTokens+msgTokens > tokenLimit {
			break
		}
		tempMessages = append([]*entities.Message{&msg}, tempMessages...)
		currentTokens += msgTokens
	}

	messagesToSend := append([]*entities.Message{systemMessage}, tempMessages...)

	tools := []*interfaces.ToolIntegration{}
	for _, toolName := range agent.Tools {
		tool, err := s.toolRepo.GetToolByName(toolName)
		if err != nil {
			return nil, fmt.Errorf("failed to get tool %s: %v", toolName, err)
		}
		tools = append(tools, tool)
	}

	options := map[string]interface{}{
		"temperature": 0.0,
		"max_tokens":  maxTokens,
	}
	if agent.Temperature != nil {
		options["temperature"] = *agent.Temperature
	}

	newMessages, err := aiModel.GenerateResponse(messagesToSend, tools, options)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AI response: %v", err)
	}

	// Get usage information for billing
	usage, err := aiModel.GetUsage()
	if err != nil {
		s.logger.Warn("Failed to get usage info", zap.Error(err))
	}

	// Get pricing for this model
	var inputPricePerMille, outputPricePerMille float64
	modelPricing := provider.GetModelPricing(agent.Model)
	if modelPricing != nil {
		inputPricePerMille = modelPricing.InputPricePerMille
		outputPricePerMille = modelPricing.OutputPricePerMille
	}

	// Add usage information to the last message (typically the final assistant response)
	if len(newMessages) > 0 && usage != nil {
		lastMsg := newMessages[len(newMessages)-1]
		lastMsg.AddUsage(usage.PromptTokens, usage.CompletionTokens, inputPricePerMille, outputPricePerMille)
	}

	// Append all new messages to the chat's message history
	for _, msg := range newMessages {
		chat.Messages = append(chat.Messages, *msg)
	}
	
	// Update chat usage totals
	chat.UpdateUsage()
	chat.UpdatedAt = time.Now()

	if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
		return nil, fmt.Errorf("failed to update chat with AI response: %v", err)
	}

	// Return the last message (final assistant response)
	if len(newMessages) > 0 {
		return newMessages[len(newMessages)-1], nil
	}
	return nil, fmt.Errorf("no AI response generated")
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

func (s *chatService) DeleteChat(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("chat ID is required")
	}
	err := s.chatRepo.DeleteChat(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("chat not found: %s", id)
		}
		return fmt.Errorf("failed to delete chat: %v", err)
	}
	return nil
}
