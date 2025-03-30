package services

import (
	"context"
	"fmt"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/impl/config"
	"aiagent/internal/impl/integrations"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	agentObjID, err := primitive.ObjectIDFromHex(agentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent ID: %v", err)
	}

	chat := entities.NewChat(agentObjID, name)
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
	agent, err := s.agentRepo.GetAgent(ctx, chat.AgentID.Hex())
	if err != nil {
		return nil, err
	}

	// Get the provider
	provider, err := s.providerRepo.GetProvider(ctx, agent.ProviderID.Hex())
	if err != nil {
		// Fallback to get provider by type if ID lookup fails
		s.logger.Warn("Failed to get provider by ID, attempting to find by type",
			zap.String("agent_id", agent.ID.Hex()),
			zap.String("provider_id", agent.ProviderID.Hex()),
			zap.String("provider_type", string(agent.ProviderType)),
			zap.Error(err))

		providerByType, typeErr := s.providerRepo.GetProviderByType(ctx, agent.ProviderType)
		if typeErr != nil {
			return nil, errors.InternalErrorf("failed to get provider %s and fallback by type %s also failed: %v", agent.ProviderID.Hex(), agent.ProviderType, err)
		}

		// Update the agent with the new provider ID for future calls
		agent.ProviderID = providerByType.ID
		if updateErr := s.agentRepo.UpdateAgent(ctx, agent); updateErr != nil {
			s.logger.Error("Failed to update agent with new provider ID", zap.String("agent_id", agent.ID.Hex()), zap.Error(updateErr))
		}

		provider = providerByType
	}

	resolvedAPIKey, err := s.config.ResolveAPIKey(agent.APIKey)
	if err != nil {
		return nil, errors.InternalErrorf("failed to resolve API key for agent %s: %v", agent.ID.Hex(), err)
	}

	// Create AI model integration based on provider type
	aiModelFactory := integrations.NewAIModelFactory(s.toolRepo, s.logger)
	aiModel, err := aiModelFactory.CreateModelIntegration(agent, provider, resolvedAPIKey)
	if err != nil {
		return nil, errors.InternalErrorf("failed to initialize AI model: %v", err)
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
	currentTokens := systemTokens

	// Check if we need to compress messages
	totalMessageTokens := systemTokens
	for i := 0; i < len(chat.Messages); i++ {
		totalMessageTokens += estimateTokens(&chat.Messages[i])
	}

	if float64(totalMessageTokens) > compressionThreshold && len(chat.Messages) > 0 {
		// Compress messages
		compressedMessages, originalMessagesReplaced, err := s.compressMessages(ctx, chat, agent, provider, resolvedAPIKey, tokenLimit)
		if err != nil {
			s.logger.Warn("Failed to compress messages", zap.Error(err))
			// Fall back to normal message selection if compression fails
			var tempMessages []*entities.Message
			for i := len(chat.Messages) - 1; i >= 0; i-- {
				msg := chat.Messages[i]
				msgTokens := estimateTokens(&msg)
				if currentTokens+msgTokens > tokenLimit {
					break
				}
				tempMessages = append([]*entities.Message{&msg}, tempMessages...)
				currentTokens += msgTokens
			}
			messagesToSend = append(messagesToSend, tempMessages...)
		} else {
			// If we successfully compressed messages, update the chat's messages array
			if originalMessagesReplaced {
				// Save the updated chat with compressed messages to the database
				if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
					s.logger.Warn("Failed to update chat with compressed messages", zap.Error(err))
					// Continue with the compressed messages in memory even if save failed
				}
			}
			messagesToSend = append(messagesToSend, compressedMessages...)
		}
	} else {
		// Normal message selection within token limit
		var tempMessages []*entities.Message
		for i := len(chat.Messages) - 1; i >= 0; i-- {
			msg := chat.Messages[i]
			msgTokens := estimateTokens(&msg)
			if currentTokens+msgTokens > tokenLimit {
				break
			}
			tempMessages = append([]*entities.Message{&msg}, tempMessages...)
			currentTokens += msgTokens
		}
		messagesToSend = append(messagesToSend, tempMessages...)
	}

	tools := []*interfaces.ToolIntegration{}
	for _, toolName := range agent.Tools {
		tool, err := s.toolRepo.GetToolByName(toolName)
		if err != nil {
			return nil, errors.InternalErrorf("failed to get tool %s: %v", toolName, err)
		}
		tools = append(tools, tool)
	}

	// Check for cancellation
	if ctx.Err() == context.Canceled {
		return nil, errors.CanceledErrorf("message processing was canceled")
	}

	options := map[string]interface{}{
		"temperature": 0.0,
		"max_tokens":  maxTokens,
	}
	if agent.Temperature != nil {
		options["temperature"] = *agent.Temperature
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
	// Approximate additional tokens for role, etc.
	return contentTokens + 4
}

// compressMessages summarizes older messages to reduce token count while preserving context
// Returns the compressed messages, a flag indicating if the chat messages were replaced, and any error
func (s *chatService) compressMessages(
	ctx context.Context,
	chat *entities.Chat,
	agent *entities.Agent,
	provider *entities.Provider,
	apiKey string,
	tokenLimit int,
) ([]*entities.Message, bool, error) {
	// Calculate how many messages to summarize (approx 40% of older messages)
	numMessagesToKeep := int(float64(len(chat.Messages)) * 0.6)
	if numMessagesToKeep < 1 {
		numMessagesToKeep = 1 // Always keep at least the most recent message
	}

	// Split messages: older ones to summarize and recent ones to keep
	summarizeEndIdx := len(chat.Messages) - numMessagesToKeep
	if summarizeEndIdx < 1 {
		summarizeEndIdx = 1 // Ensure we have at least one message to summarize
	}

	messagesToSummarize := chat.Messages[:summarizeEndIdx]
	recentMessagesToKeep := chat.Messages[summarizeEndIdx:]

	// Create AI model for summarization
	aiModelFactory := integrations.NewAIModelFactory(s.toolRepo, s.logger)
	aiModel, err := aiModelFactory.CreateModelIntegration(agent, provider, apiKey)
	if err != nil {
		return nil, false, errors.InternalErrorf("failed to initialize AI model for summarization: %v", err)
	}

	// Create summary prompt
	summaryPrompt := &entities.Message{
		Role:    "system",
		Content: "You are an expert at summarizing conversation history. Create a concise summary of the following conversation that captures all important context, decisions, and information. The summary will be used as context for future messages in this conversation. Focus on key facts, goals, decisions, and relevant details. Your summary should be complete enough that the conversation can continue without losing context.",
	}

	// Messages for the summarization request
	var historyMsgs []*entities.Message
	historyMsgs = append(historyMsgs, summaryPrompt)

	// Add messages to summarize
	for i := 0; i < len(messagesToSummarize); i++ {
		msg := messagesToSummarize[i]
		historyMsgs = append(historyMsgs, &msg)
	}

	// Generate summary
	options := map[string]interface{}{
		"temperature": 0.0,
		"max_tokens":  1000, // Allow sufficient tokens for a detailed summary
	}

	// Check for cancellation
	if ctx.Err() == context.Canceled {
		return nil, false, errors.CanceledErrorf("message summarization was canceled")
	}

	summaryResponse, err := aiModel.GenerateResponse(ctx, historyMsgs, nil, options)
	if err != nil {
		return nil, false, errors.InternalErrorf("failed to generate summary: %v", err)
	}

	if len(summaryResponse) == 0 {
		return nil, false, fmt.Errorf("no summary generated")
	}

	// Create a single message that contains the summary
	summaryMsg := &entities.Message{
		ID:        uuid.New().String(),
		Role:      "assistant",
		Content:   "Summary of previous conversation: " + summaryResponse[0].Content,
		Timestamp: time.Now(),
	}

	// Create new array with summary message + recent messages to keep
	var newMessages []entities.Message
	newMessages = append(newMessages, *summaryMsg)
	newMessages = append(newMessages, recentMessagesToKeep...)

	// Replace the chat's messages with the compressed version
	chat.Messages = newMessages

	// Prepare the messages to return for the current message processing
	var compressedMessages []*entities.Message
	compressedMessages = append(compressedMessages, summaryMsg)

	// Add recent messages to keep
	for i := 0; i < len(recentMessagesToKeep); i++ {
		msg := recentMessagesToKeep[i]
		compressedMessages = append(compressedMessages, &msg)
	}

	// Verify we're not exceeding token limit
	currentTokens := estimateTokens(summaryMsg)
	var finalMessages []*entities.Message
	finalMessages = append(finalMessages, summaryMsg)

	// Add as many of the recent messages as possible within token limit
	for i := 0; i < len(recentMessagesToKeep); i++ {
		msgTokens := estimateTokens(&recentMessagesToKeep[i])
		if currentTokens+msgTokens > tokenLimit {
			break
		}
		finalMessages = append(finalMessages, &recentMessagesToKeep[i])
		currentTokens += msgTokens
	}

	return finalMessages, true, nil
}

// verify that chatService implements ChatService
var _ ChatService = &chatService{}
