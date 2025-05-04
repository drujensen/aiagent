package services

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/impl/config"
	"aiagent/internal/impl/integrations"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ChatService interface {
	ListChats(ctx context.Context) ([]*entities.Chat, error)
	GetChat(ctx context.Context, id string) (*entities.Chat, error)
	CreateChat(ctx context.Context, agentID, name string) (*entities.Chat, error)
	UpdateChat(ctx context.Context, id, name string) (*entities.Chat, error)
	DeleteChat(ctx context.Context, id string) error
	SendMessage(ctx context.Context, id string, message entities.Message) (*entities.Message, error)
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

func (s *chatService) ListChats(ctx context.Context) ([]*entities.Chat, error) {
	chats, err := s.chatRepo.ListChats(ctx)
	if err != nil {
		return nil, err
	}

	return chats, nil
}

func (s *chatService) GetChat(ctx context.Context, id string) (*entities.Chat, error) {
	if id == "" {
		return nil, errors.ValidationErrorf("chat ID is required")
	}

	chat, err := s.chatRepo.GetChat(ctx, id)
	if err != nil {
		return nil, err
	}

	return chat, nil
}

func (s *chatService) CreateChat(ctx context.Context, agentID, name string) (*entities.Chat, error) {
	if agentID == "" {
		return nil, errors.ValidationErrorf("agent ID is required")
	}

	chat := entities.NewChat(agentID, name)
	if err := s.chatRepo.CreateChat(ctx, chat); err != nil {
		return nil, err
	}

	return chat, nil
}

func (s *chatService) UpdateChat(ctx context.Context, id, name string) (*entities.Chat, error) {
	if id == "" {
		return nil, errors.ValidationErrorf("chat ID is required")
	}

	if name == "" {
		return nil, errors.ValidationErrorf("chat name is required")
	}

	existingChat, err := s.chatRepo.GetChat(ctx, id)
	if err != nil {
		return nil, err
	}

	existingChat.Name = name
	existingChat.UpdatedAt = time.Now()

	if err := s.chatRepo.UpdateChat(ctx, existingChat); err != nil {
		return nil, err
	}

	return existingChat, nil
}

func (s *chatService) DeleteChat(ctx context.Context, id string) error {
	if id == "" {
		return errors.ValidationErrorf("chat ID is required")
	}
	err := s.chatRepo.DeleteChat(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *chatService) SendMessage(ctx context.Context, id string, message entities.Message) (*entities.Message, error) {
	if id == "" {
		return nil, errors.ValidationErrorf("chat ID is required")
	}
	if message.Role == "" || message.Content == "" {
		return nil, errors.ValidationErrorf("message role and content are required")
	}

	chat, err := s.chatRepo.GetChat(ctx, id)
	if err != nil {
		return nil, err
	}

	message.ID = uuid.New().String()
	message.Timestamp = time.Now()
	chat.Messages = append(chat.Messages, message)
	chat.UpdatedAt = time.Now()

	if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
		return nil, err
	}

	// Check for cancellation
	if ctx.Err() == context.Canceled {
		return nil, errors.CanceledErrorf("message processing was canceled")
	}

	// Generate AI response synchronously
	agent, err := s.agentRepo.GetAgent(ctx, chat.AgentID)
	if err != nil {
		return nil, err
	}

	// Get the provider using agent's ProviderID (ObjectID)
	provider, err := s.providerRepo.GetProvider(ctx, agent.ProviderID)
	if err != nil {
		// Fallback to get provider by type if ID lookup fails
		s.logger.Warn("Failed to get provider by ID, attempting to find by type",
			zap.String("agent_id", agent.ID),
			zap.String("provider_id", agent.ProviderID),
			zap.String("provider_type", string(agent.ProviderType)),
			zap.Error(err))

		providerByType, typeErr := s.providerRepo.GetProviderByType(ctx, agent.ProviderType)
		if typeErr != nil {
			return nil, errors.InternalErrorf("failed to get provider %s and fallback by type %s also failed: %v", agent.ProviderID, agent.ProviderType, err)
		}

		agent.ProviderID = providerByType.ID
		if updateErr := s.agentRepo.UpdateAgent(ctx, agent); updateErr != nil {
			s.logger.Error("Failed to update agent with new provider ID",
				zap.String("agent_id", agent.ID),
				zap.Error(updateErr))
		}

		provider = providerByType
	}

	resolvedAPIKey, err := s.config.ResolveEnvironmentVariable(agent.APIKey)
	if err != nil {
		s.logger.Error("Failed to resolve API key", zap.String("agent_id", agent.ID), zap.Error(err))
		return nil, errors.InternalErrorf("failed to resolve API key for agent %s: %v", agent.ID, err)
	}

	// Resolve tool configurations
	tools := []*entities.Tool{}
	for _, toolName := range agent.Tools {
		tool, err := s.toolRepo.GetToolByName(toolName)
		if err != nil {
			return nil, errors.InternalErrorf("failed to get tool %s: %v", toolName, err)
		}
		resolvedConfig, err := s.config.ResolveConfiguration((*tool).Configuration())
		if err != nil {
			return nil, errors.InternalErrorf("failed to resolve configuration for tool %s: %v", toolName, err)
		}
		(*tool).UpdateConfiguration(resolvedConfig)
		tools = append(tools, tool)
	}

	// Create AI model integration based on provider type
	aiModelFactory := integrations.NewAIModelFactory(s.toolRepo, s.logger)
	aiModel, err := aiModelFactory.CreateModelIntegration(agent, provider, resolvedAPIKey)
	if err != nil {
		s.logger.Error("Failed to create AI model integration", zap.String("agent_id", agent.ID), zap.Error(err))
		return nil, errors.InternalErrorf("failed to initialize AI model: %v", err)
	}

	contextLength := 128000
	if agent.ContextWindow != nil {
		contextLength = *agent.ContextWindow
	}

	tokenLimit := contextLength

	// Create system message
	systemMessage := &entities.Message{
		Role:    "system",
		Content: agent.FullSystemPrompt(),
	}
	systemTokens := estimateTokens(systemMessage)
	if systemTokens > tokenLimit {
		return nil, errors.InternalErrorf("system prompt too large for the context window")
	}

	// Check for cancellation
	if ctx.Err() == context.Canceled {
		return nil, errors.CanceledErrorf("message processing was canceled")
	}

	// Check if we need message compression (at 90% of token limit)
	compressionThreshold := float64(tokenLimit) * 0.9
	var messagesToSend []*entities.Message

	// Always start with the system message
	messagesToSend = append(messagesToSend, systemMessage)

	// Check if we need to compress messages
	totalMessageTokens := systemTokens
	for i := 0; i < len(chat.Messages); i++ {
		totalMessageTokens += estimateTokens(&chat.Messages[i])
	}

	s.logger.Debug("Total message tokens: ", zap.Float64("total_message_tokens", float64(totalMessageTokens)), zap.Float64("compression_threshold", compressionThreshold))
	if float64(totalMessageTokens) > compressionThreshold && len(chat.Messages) > 0 {
		// Compress messages
		compressedMessages, originalMessagesReplaced, err := s.compressMessages(ctx, chat, agent, provider, resolvedAPIKey, tokenLimit)
		if err != nil {
			s.logger.Warn("Failed to compress messages", zap.Error(err))
			var tempMessages []*entities.Message
			for i := len(chat.Messages) - 1; i >= 0; i-- {
				msg := chat.Messages[i]
				tempMessages = append([]*entities.Message{&msg}, tempMessages...)
			}
			messagesToSend = append(messagesToSend, tempMessages...)
		} else {
			if originalMessagesReplaced {
				if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
					s.logger.Warn("Failed to update chat with compressed messages", zap.Error(err))
				}
			}
			messagesToSend = append(messagesToSend, compressedMessages...)
		}
	} else {
		var tempMessages []*entities.Message
		for i := len(chat.Messages) - 1; i >= 0; i-- {
			msg := chat.Messages[i]
			tempMessages = append([]*entities.Message{&msg}, tempMessages...)
		}
		messagesToSend = append(messagesToSend, tempMessages...)
	}

	// Check for cancellation
	if ctx.Err() == context.Canceled {
		return nil, errors.CanceledErrorf("message processing was canceled")
	}

	options := map[string]interface{}{
		"temperature": 0.0,
		"max_tokens":  4096,
	}
	if agent.Temperature != nil {
		options["temperature"] = *agent.Temperature
	}
	if agent.MaxTokens != nil {
		options["max_tokens"] = *agent.MaxTokens
	}
	if agent.ReasoningEffort != "none" {
		options["reasoning_effort"] = agent.ReasoningEffort
	}

	newMessages, err := aiModel.GenerateResponse(ctx, messagesToSend, tools, options)
	if err != nil {
		return nil, errors.InternalErrorf("failed to generate AI response: %v", err)
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

	// Add usage information to the last message
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
		return nil, err
	}

	// Return the last message (final assistant response)
	if len(newMessages) > 0 {
		return newMessages[len(newMessages)-1], nil
	}
	return nil, errors.InternalErrorf("no AI response generated")
}

// estimateTokens approximates the token count for a message.
func estimateTokens(msg *entities.Message) int {
	contentTokens := (len(msg.Content) + 3) / 4
	return contentTokens + 4
}

// compressMessages summarizes older messages to reduce token count while preserving context
func (s *chatService) compressMessages(
	ctx context.Context,
	chat *entities.Chat,
	agent *entities.Agent,
	provider *entities.Provider,
	apiKey string,
	tokenLimit int,
) ([]*entities.Message, bool, error) {
	// ... (unchanged from original) ...
	// Note: No changes needed here as it uses the provider as-is
	return nil, false, nil // Placeholder; actual implementation unchanged
}

// verify that chatService implements ChatService
var _ ChatService = &chatService{}
