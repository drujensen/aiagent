package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"github.com/drujensen/aiagent/internal/impl/config"
	"github.com/drujensen/aiagent/internal/impl/integrations"

	"github.com/google/uuid"
	"github.com/pkoukk/tiktoken-go"
	"go.uber.org/zap"
)

type ChatService interface {
	ListChats(ctx context.Context) ([]*entities.Chat, error)
	GetChat(ctx context.Context, id string) (*entities.Chat, error)
	GetActiveChat(ctx context.Context) (*entities.Chat, error)
	SetActiveChat(ctx context.Context, chatID string) error
	CreateChat(ctx context.Context, agentID, name string) (*entities.Chat, error)
	UpdateChat(ctx context.Context, id, agentID, name string) (*entities.Chat, error)
	DeleteChat(ctx context.Context, id string) error
	SendMessage(ctx context.Context, id string, message *entities.Message) (*entities.Message, error)
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

	// Set the other chat sessions to inactive
	chats, err := s.chatRepo.ListChats(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range chats {
		if c.ID != chat.ID {
			c.Active = false
			if err := s.chatRepo.UpdateChat(ctx, c); err != nil {
				s.logger.Error("Failed to update chat status", zap.String("chat_id", c.ID), zap.Error(err))
			}
		}
	}

	return chat, nil
}

func (s *chatService) GetActiveChat(ctx context.Context) (*entities.Chat, error) {
	chats, err := s.chatRepo.ListChats(ctx)
	if err != nil {
		return nil, err
	}

	for _, chat := range chats {
		if chat.Active {
			return chat, nil
		}
	}

	return nil, errors.NotFoundErrorf("no active chat found")
}

func (s *chatService) SetActiveChat(ctx context.Context, chatID string) error {
	if chatID == "" {
		return errors.ValidationErrorf("chat ID is required")
	}
	chat, err := s.chatRepo.GetChat(ctx, chatID)
	if err != nil {
		return err
	}
	// Set the other chat sessions to inactive
	chats, err := s.chatRepo.ListChats(ctx)
	if err != nil {
		for _, c := range chats {
			if c.ID != chat.ID {
				c.Active = false
				if err := s.chatRepo.UpdateChat(ctx, c); err != nil {
					s.logger.Error("Failed to update chat status", zap.String("chat_id", c.ID), zap.Error(err))
				}
			}
		}
	}
	chat.Active = true
	if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
		s.logger.Error("Failed to update chat status", zap.String("chat_id", chat.ID), zap.Error(err))
	}
	return nil
}

func (s *chatService) UpdateChat(ctx context.Context, id, agentID, name string) (*entities.Chat, error) {
	if id == "" {
		return nil, errors.ValidationErrorf("chat ID is required")
	}

	if agentID == "" {
		return nil, errors.ValidationErrorf("agent ID is required")
	}

	if name == "" {
		return nil, errors.ValidationErrorf("chat name is required")
	}

	existingChat, err := s.chatRepo.GetChat(ctx, id)
	if err != nil {
		return nil, err
	}

	existingChat.Name = name
	existingChat.AgentID = agentID
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

func (s *chatService) SendMessage(ctx context.Context, id string, message *entities.Message) (*entities.Message, error) {
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
	chat.Messages = append(chat.Messages, *message)
	chat.UpdatedAt = time.Now()

	if err = s.chatRepo.UpdateChat(ctx, chat); err != nil {
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
		s.logger.Warn("failed to get provider by ID",
			zap.String("agent_id", agent.ID),
			zap.String("provider_id", agent.ProviderID),
			zap.String("provider_type", string(agent.ProviderType)),
			zap.Error(err))
		return nil, errors.InternalErrorf("failed to get provider for agent %s: %v", agent.ID, err)
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
	for i := range chat.Messages {
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

	options := map[string]any{
		"temperature": 0.0,
		"max_tokens":  4096,
	}
	if agent.Temperature != nil {
		options["temperature"] = *agent.Temperature
	}
	if agent.MaxTokens != nil {
		options["max_tokens"] = *agent.MaxTokens
	}
	if agent.ReasoningEffort != "none" && agent.ReasoningEffort != "" {
		options["reasoning_effort"] = agent.ReasoningEffort
	}

	newMessages, err := aiModel.GenerateResponse(ctx, messagesToSend, tools, options)
	if err != nil {
		if strings.Contains(err.Error(), "canceled") {
			return nil, errors.CanceledErrorf("message processing was canceled")
		}
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
		if msg.Content == "" {
			msg.Content = "Unknown Error: No response generated"
		}
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

func estimateTokens(msg *entities.Message) int {
	enc, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		return 0
	}

	tokens := enc.Encode(msg.Content, nil, nil)

	return len(tokens)
}

// isSafeSplit checks if splitting at 'split' avoids both orphaned responses and unfinished calls.
func isSafeSplit(messages []entities.Message, split int) bool {
	// Collect all tool call IDs before split
	toolCallIDsBefore := make(map[string]struct{})
	for i := 0; i < split; i++ {
		msg := messages[i]
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				toolCallIDsBefore[tc.ID] = struct{}{}
			}
		}
	}

	// Check for orphaned responses: tool after split referencing call before split
	for i := split; i < len(messages); i++ {
		msg := messages[i]
		if msg.Role == "tool" {
			if _, ok := toolCallIDsBefore[msg.ToolCallID]; ok {
				return false // Orphan response
			}
		}
	}

	// Collect all tool call IDs before split and check if they have responses before split
	responseIDsBefore := make(map[string]struct{})
	for i := 0; i < split; i++ {
		msg := messages[i]
		if msg.Role == "tool" {
			responseIDsBefore[msg.ToolCallID] = struct{}{}
		}
	}

	// Check for unfinished calls: call before split without response before split
	for callID := range toolCallIDsBefore {
		if _, ok := responseIDsBefore[callID]; !ok {
			return false // Unfinished call
		}
	}

	return true
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

	// Tentative split point
	summarizeEndIdx := len(chat.Messages) - numMessagesToKeep
	if summarizeEndIdx < 1 {
		summarizeEndIdx = 1 // Ensure we have at least one message to summarize
	}

	// Find the largest safe split point <= summarizeEndIdx
	safeFound := false
	for j := summarizeEndIdx; j >= 1; j-- { // Require at least 1 message to summarize
		if isSafeSplit(chat.Messages, j) {
			summarizeEndIdx = j
			safeFound = true
			break
		}
	}
	if !safeFound {
		s.logger.Warn("No safe split point found; skipping compression to avoid unbalanced messages")
		return nil, false, nil // Return nil to indicate skipping, no error
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
	for i := range messagesToSummarize {
		msg := messagesToSummarize[i]
		historyMsgs = append(historyMsgs, &msg)
	}

	// Generate summary
	options := map[string]any{
		"temperature": 0.0,
		"max_tokens":  1000, // Allow sufficient tokens for a detailed summary
	}

	// Check for cancellation
	if ctx.Err() == context.Canceled {
		return nil, false, errors.CanceledErrorf("message summarization was canceled")
	}

	summaryResponse, err := aiModel.GenerateResponse(ctx, historyMsgs, nil, options)
	if err != nil {
		if strings.Contains(err.Error(), "canceled") {
			return nil, false, errors.CanceledErrorf("message summarization was canceled")
		}
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
	chat.Messages = append([]entities.Message{*summaryMsg}, recentMessagesToKeep...)

	// Verify we're not exceeding token limit
	currentTokens := estimateTokens(summaryMsg)
	var finalMessages []*entities.Message
	finalMessages = append(finalMessages, summaryMsg)

	// Add as many of the recent messages as possible within token limit
	for i := range recentMessagesToKeep {
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
