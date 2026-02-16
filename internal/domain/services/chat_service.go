package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/events"
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
	CreateChat(ctx context.Context, agentID, modelID, name string) (*entities.Chat, error)
	UpdateChat(ctx context.Context, id, agentID, modelID, name string) (*entities.Chat, error)
	DeleteChat(ctx context.Context, id string) error
	SendMessage(ctx context.Context, id string, message *entities.Message) (*entities.Message, error)
	SaveMessagesIncrementally(ctx context.Context, chatID string, messages []*entities.Message) error
	CalculateTotalChatCost(ctx context.Context, chatID string) (float64, error)
}

type chatService struct {
	chatRepo     interfaces.ChatRepository
	agentRepo    interfaces.AgentRepository
	modelRepo    interfaces.ModelRepository
	providerRepo interfaces.ProviderRepository
	toolRepo     interfaces.ToolRepository
	config       *config.Config
	logger       *zap.Logger
}

func NewChatService(
	chatRepo interfaces.ChatRepository,
	agentRepo interfaces.AgentRepository,
	modelRepo interfaces.ModelRepository,
	providerRepo interfaces.ProviderRepository,
	toolRepo interfaces.ToolRepository,
	cfg *config.Config,
	logger *zap.Logger,
) *chatService {
	return &chatService{
		chatRepo:     chatRepo,
		agentRepo:    agentRepo,
		modelRepo:    modelRepo,
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

func (s *chatService) CreateChat(ctx context.Context, agentID, modelID, name string) (*entities.Chat, error) {
	if agentID == "" {
		return nil, errors.ValidationErrorf("agent ID is required")
	}
	if modelID == "" {
		return nil, errors.ValidationErrorf("model ID is required")
	}

	chat := entities.NewChat(agentID, modelID, name)
	if err := s.chatRepo.CreateChat(ctx, chat); err != nil {
		return nil, err
	}

	setOtherChatsInactive(ctx, s.chatRepo, chat.ID)

	return chat, nil
}

func setOtherChatsInactive(ctx context.Context, chatRepo interfaces.ChatRepository, activeChatID string) error {
	chats, err := chatRepo.ListChats(ctx)
	if err != nil {
		return err
	}
	for _, c := range chats {
		if c.ID != activeChatID {
			c.Active = false
			if err := chatRepo.UpdateChat(ctx, c); err != nil {
				return err
			}
		}
	}
	return nil
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
	if err := setOtherChatsInactive(ctx, s.chatRepo, chat.ID); err != nil {
		return err
	}
	chat.Active = true
	if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
		s.logger.Error("Failed to update chat status", zap.String("chat_id", chat.ID), zap.Error(err))
	}
	return nil
}

func (s *chatService) UpdateChat(ctx context.Context, id, agentID, modelID, name string) (*entities.Chat, error) {
	if id == "" {
		return nil, errors.ValidationErrorf("chat ID is required")
	}

	if agentID == "" {
		return nil, errors.ValidationErrorf("agent ID is required")
	}

	if modelID == "" {
		return nil, errors.ValidationErrorf("model ID is required")
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
	existingChat.ModelID = modelID
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

func (s *chatService) SaveMessagesIncrementally(ctx context.Context, chatID string, messages []*entities.Message) error {
	if chatID == "" {
		return errors.ValidationErrorf("chat ID is required")
	}
	if len(messages) == 0 {
		return nil
	}

	chat, err := s.chatRepo.GetChat(ctx, chatID)
	if err != nil {
		return err
	}

	// Append new messages to chat
	for _, msg := range messages {
		if msg.Content == "" {
			msg.Content = "Unknown Error: No response generated"
		}
		chat.Messages = append(chat.Messages, *msg)
	}

	// Update chat usage totals
	chat.UpdateUsage()
	chat.UpdatedAt = time.Now()

	if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
		return err
	}

	// Publish message history change event for live updates
	events.PublishMessageHistoryEvent(chatID, messages)

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

	// Get Model instead of Agent for inference settings
	model, err := s.modelRepo.GetModel(ctx, chat.ModelID)
	if err != nil {
		return nil, err
	}

	// Get Agent for behavior settings (system prompt, tools)
	agent, err := s.agentRepo.GetAgent(ctx, chat.AgentID)
	if err != nil {
		return nil, err
	}

	// Get provider using model's ProviderID
	provider, err := s.providerRepo.GetProvider(ctx, model.ProviderID)
	if err != nil {
		s.logger.Warn("failed to get provider by ID",
			zap.String("model_id", chat.ModelID),
			zap.String("provider_id", model.ProviderID),
			zap.Error(err))
		return nil, errors.InternalErrorf("failed to get provider for model %s: %v", chat.ModelID, err)
	}

	// Resolve API key from provider
	apiKeyReference := "#{" + provider.APIKeyName + "}#"
	resolvedAPIKey, err := s.config.ResolveEnvironmentVariable(apiKeyReference)
	if err != nil {
		s.logger.Error("Failed to resolve API key", zap.String("provider_id", provider.ID), zap.Error(err))
		return nil, errors.InternalErrorf("failed to resolve API key for provider %s: %v", provider.ID, err)
	}

	contextLength := 128000 // default
	if model.ContextWindow != nil {
		contextLength = *model.ContextWindow
	}

	tokenLimit := contextLength

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
		compressedMessages, originalMessagesReplaced, err := s.compressMessages(ctx, chat, model, provider, resolvedAPIKey, tokenLimit)
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
		"max_tokens":  16384, // Increased for complex multi-tool tasks
		"session_id":  chat.ID,
	}
	if model.Temperature != nil {
		options["temperature"] = *model.Temperature
	}
	if model.MaxTokens != nil {
		options["max_tokens"] = *model.MaxTokens
	}
	if model.ReasoningEffort != "none" && model.ReasoningEffort != "" {
		options["reasoning_effort"] = model.ReasoningEffort
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
	aiModel, err := aiModelFactory.CreateModelIntegration(model, provider, resolvedAPIKey)
	if err != nil {
		s.logger.Error("Failed to create AI model integration", zap.String("model_id", model.ID), zap.Error(err))
		return nil, errors.InternalErrorf("failed to initialize AI model: %v", err)
	}

	// Create a callback function for incremental message saving
	messageCallback := func(messages []*entities.Message) error {
		return s.SaveMessagesIncrementally(ctx, chat.ID, messages)
	}

	// Use the callback method for incremental saving
	newMessages, err := aiModel.GenerateResponse(ctx, messagesToSend, tools, options, messageCallback)
	if err != nil {
		if strings.Contains(err.Error(), "canceled") {
			return nil, errors.CanceledErrorf("message processing was canceled")
		}
		return nil, errors.InternalErrorf("failed to generate AI response: %v", err)
	}

	// Get usage information for billing
	totalUsage, err := aiModel.GetUsage()
	if err != nil {
		s.logger.Warn("Failed to get total usage info", zap.Error(err))
	}
	lastUsage, err := aiModel.GetLastUsage()
	if err != nil {
		s.logger.Warn("Failed to get last usage info", zap.Error(err))
	}

	// Get pricing for this model
	var inputPricePerMille, outputPricePerMille float64
	modelPricing := provider.GetModelPricing(model.ModelName)
	if modelPricing != nil {
		inputPricePerMille = modelPricing.InputPricePerMille
		outputPricePerMille = modelPricing.OutputPricePerMille
	} else {
		s.logger.Warn("No pricing found for model", zap.String("model", model.ModelName), zap.String("provider", provider.Name))
	}

	// Calculate total cost
	var totalCost float64
	if totalUsage != nil {
		totalCost = (float64(totalUsage.PromptTokens)*inputPricePerMille + float64(totalUsage.CompletionTokens)*outputPricePerMille) / 1000000.0
	}

	// Add usage information to the last message
	if len(newMessages) > 0 && lastUsage != nil {
		lastMsg := newMessages[len(newMessages)-1]
		lastMsg.AddUsage(lastUsage.PromptTokens, lastUsage.CompletionTokens, inputPricePerMille, outputPricePerMille)
		lastMsg.Usage.Cost = totalCost

		// Save the updated message with usage information
		if err := s.SaveMessagesIncrementally(ctx, chat.ID, []*entities.Message{lastMsg}); err != nil {
			s.logger.Warn("Failed to save message with usage information", zap.Error(err))
		}
	}

	// Check if this is a partial response due to cancellation
	isPartialResponse := ctx.Err() == context.Canceled && len(newMessages) > 0
	if isPartialResponse {
		s.logger.Info("Processing partial response due to cancellation", zap.Int("messageCount", len(newMessages)))
		// For partial responses, add a note about cancellation
		if len(newMessages) > 0 {
			lastMsg := newMessages[len(newMessages)-1]
			if lastMsg.Role == "assistant" && !strings.Contains(lastMsg.Content, "cancelled") {
				lastMsg.Content += "\n\n[Processing cancelled by user - partial results shown]"
			}
		}
	}

	// Validate that all tool calls have responses
	newMessages = s.ensureToolCallResponses(newMessages)

	// Check for compression instructions in tool results
	if err := s.processCompressionInstructions(ctx, chat, newMessages); err != nil {
		s.logger.Warn("Failed to process compression instructions", zap.Error(err))
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

// processCompressionInstructions checks for compression instructions in tool messages
// and executes compression if found
func (s *chatService) processCompressionInstructions(ctx context.Context, chat *entities.Chat, messages []*entities.Message) error {
	for _, msg := range messages {
		if msg.Role == "tool" && strings.Contains(msg.Content, "compression_instruction") {
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Content), &result); err != nil {
				continue // Not a valid JSON result, skip
			}

			if instruction, ok := result["compression_instruction"].(map[string]interface{}); ok {
				action, _ := instruction["action"].(string)
				if action == "compress_range" {
					// EMIT TOOL CALL EVENT FOR COMPRESSION START
					event := entities.NewToolCallEvent(
						uuid.New().String(),            // toolCallID
						"compression",                  // toolName
						fmt.Sprintf("%v", instruction), // arguments
						"",                             // result (will be updated later)
						"",                             // error
						"",                             // diff
						map[string]string{
							"chat_id": chat.ID,
							"action":  "start_compression",
						},
					)
					events.PublishToolCallEvent(event)

					return s.executeCompressionInstruction(ctx, chat, instruction)
				}
			}
		}
	}
	return nil
}

// executeCompressionInstruction performs the actual compression based on the instruction
func (s *chatService) executeCompressionInstruction(ctx context.Context, chat *entities.Chat, instruction map[string]interface{}) error {
	startIdx, _ := instruction["start_message_index"].(float64)
	endIdx, _ := instruction["end_message_index"].(float64)
	summaryType, _ := instruction["summary_type"].(string)
	description, _ := instruction["description"].(string)

	s.logger.Info("Executing compression instruction",
		zap.Int("start_index", int(startIdx)),
		zap.Int("end_index", int(endIdx)),
		zap.String("summary_type", summaryType))

	// Get current messages
	messages := chat.Messages

	// Validate range
	start := int(startIdx)
	end := int(endIdx)
	if start < 0 || end >= len(messages) || start > end {
		return fmt.Errorf("invalid compression range: start=%d, end=%d, total_messages=%d", start, end, len(messages))
	}

	// Extract messages to compress
	messagesToCompress := messages[start : end+1]

	// Analyze content to enhance summarization
	contentAnalysis := s.analyzeMessages(messagesToCompress)

	// Create summarization prompt based on type and content analysis
	var prompt string
	switch summaryType {
	case "task_cleanup":
		prompt = s.buildTaskCleanupPrompt(contentAnalysis)
	case "plan_update":
		prompt = s.buildPlanUpdatePrompt(contentAnalysis)
	case "context_preservation":
		prompt = s.buildContextPreservationPrompt(contentAnalysis)
	case "full_reset":
		prompt = s.buildFullResetPrompt(contentAnalysis)
	default:
		prompt = "You are an expert at summarizing conversation history. Create a concise summary of the following conversation that captures all important context, decisions, and information. The summary will be used as context for future messages in this conversation."
	}

	// Add description if provided
	if description != "" {
		prompt = fmt.Sprintf("Task: %s\n\n%s", description, prompt)
	}

	// Create summary using existing compression logic
	summaryMsg, err := s.createSummaryFromMessages(ctx, messagesToCompress, prompt)
	if err != nil {
		return fmt.Errorf("failed to create summary: %v", err)
	}

	// Replace the range with the summary
	newMessages := make([]entities.Message, 0, len(messages)-len(messagesToCompress)+1)
	newMessages = append(newMessages, messages[:start]...)
	newMessages = append(newMessages, *summaryMsg)
	newMessages = append(newMessages, messages[end+1:]...)

	chat.Messages = newMessages

	// Save the updated chat
	if err := s.chatRepo.UpdateChat(ctx, chat); err != nil {
		return fmt.Errorf("failed to save compressed chat: %v", err)
	}

	s.logger.Info("Successfully compressed messages",
		zap.Int("original_count", len(messages)),
		zap.Int("compressed_count", len(newMessages)),
		zap.Int("removed_count", len(messagesToCompress)-1))

	// EMIT TOOL CALL EVENT FOR COMPRESSION COMPLETE
	event := entities.NewToolCallEvent(
		uuid.New().String(), // toolCallID
		"compression",       // toolName
		fmt.Sprintf("Compressed messages %d-%d (%s)", start, end, summaryType),                             // arguments summary
		fmt.Sprintf("Successfully compressed %d messages into 1 summary message", len(messagesToCompress)), // result
		"", // error
		fmt.Sprintf("-%d messages", len(messagesToCompress)-1), // diff
		map[string]string{
			"chat_id":     chat.ID,
			"action":      "complete_compression",
			"summaryType": summaryType,
		},
	)
	events.PublishToolCallEvent(event)

	return nil
}

// analyzeMessageContent performs content analysis to identify architectural vs implementation content
func (s *chatService) analyzeMessageContent(content string) map[string]bool {
	analysis := map[string]bool{
		"has_interface":       false,
		"has_contract":        false,
		"has_design_decision": false,
		"has_error_handling":  false,
		"has_configuration":   false,
		"has_debugging":       false,
		"has_implementation":  false,
	}

	contentLower := strings.ToLower(content)

	// Architectural indicators
	if strings.Contains(contentLower, "interface") ||
		strings.Contains(contentLower, "type ") ||
		strings.Contains(contentLower, "struct") {
		analysis["has_interface"] = true
	}

	if strings.Contains(contentLower, "contract") ||
		strings.Contains(contentLower, "agreement") ||
		strings.Contains(contentLower, "promise") {
		analysis["has_contract"] = true
	}

	if strings.Contains(contentLower, "decision") ||
		strings.Contains(contentLower, "design") ||
		strings.Contains(contentLower, "architecture") {
		analysis["has_design_decision"] = true
	}

	if strings.Contains(contentLower, "error") ||
		strings.Contains(contentLower, "handle") ||
		strings.Contains(contentLower, "catch") ||
		strings.Contains(contentLower, "panic") {
		analysis["has_error_handling"] = true
	}

	if strings.Contains(contentLower, "config") ||
		strings.Contains(contentLower, "setup") ||
		strings.Contains(contentLower, "initialize") {
		analysis["has_configuration"] = true
	}

	// Implementation indicators
	if strings.Contains(contentLower, "debug") ||
		strings.Contains(contentLower, "print") ||
		strings.Contains(contentLower, "log") ||
		strings.Contains(contentLower, "fmt.printf") {
		analysis["has_debugging"] = true
	}

	if strings.Contains(contentLower, "func ") ||
		strings.Contains(contentLower, "method") ||
		strings.Contains(contentLower, "implementation") {
		analysis["has_implementation"] = true
	}

	return analysis
}

// createSummaryFromMessages creates a summary message from a slice of messages
func (s *chatService) createSummaryFromMessages(ctx context.Context, messages []entities.Message, prompt string) (*entities.Message, error) {
	// Get AI model for summarization (reuse existing logic)
	model, err := s.modelRepo.GetModel(ctx, "gpt-4") // Use GPT-4 for summarization
	if err != nil {
		// Fallback to first available model
		models, err := s.modelRepo.ListModels(ctx)
		if err != nil || len(models) == 0 {
			return nil, fmt.Errorf("no AI model available for summarization")
		}
		model = models[0]
	}

	provider, err := s.providerRepo.GetProvider(ctx, model.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	apiKeyReference := "#{" + provider.APIKeyName + "}#"
	apiKey, err := s.config.ResolveEnvironmentVariable(apiKeyReference)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve API key: %v", err)
	}

	aiModelFactory := integrations.NewAIModelFactory(s.toolRepo, s.logger)
	aiModel, err := aiModelFactory.CreateModelIntegration(model, provider, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI model: %v", err)
	}

	// Create summary prompt message
	summaryPrompt := &entities.Message{
		Role:    "system",
		Content: prompt,
	}

	// Messages for summarization
	var historyMsgs []*entities.Message
	historyMsgs = append(historyMsgs, summaryPrompt)

	// Add messages to summarize
	for i := range messages {
		msg := messages[i]
		historyMsgs = append(historyMsgs, &msg)
	}

	// Generate summary
	options := map[string]any{
		"temperature": 0.0,
		"max_tokens":  1000,
	}

	summaryResponse, err := aiModel.GenerateResponse(ctx, historyMsgs, nil, options, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %v", err)
	}

	if len(summaryResponse) == 0 {
		return nil, fmt.Errorf("no summary generated")
	}

	// Create summary message
	summaryMsg := &entities.Message{
		ID:        uuid.New().String(),
		Role:      "assistant",
		Content:   summaryResponse[0].Content,
		Timestamp: time.Now(),
	}

	return summaryMsg, nil
}

// analyzeMessages performs content analysis on a slice of messages
func (s *chatService) analyzeMessages(messages []entities.Message) map[string]int {
	analysis := map[string]int{
		"interfaces":       0,
		"contracts":        0,
		"design_decisions": 0,
		"error_handling":   0,
		"configuration":    0,
		"debugging":        0,
		"implementation":   0,
	}

	for _, msg := range messages {
		msgAnalysis := s.analyzeMessageContent(msg.Content)
		if msgAnalysis["has_interface"] {
			analysis["interfaces"]++
		}
		if msgAnalysis["has_contract"] {
			analysis["contracts"]++
		}
		if msgAnalysis["has_design_decision"] {
			analysis["design_decisions"]++
		}
		if msgAnalysis["has_error_handling"] {
			analysis["error_handling"]++
		}
		if msgAnalysis["has_configuration"] {
			analysis["configuration"]++
		}
		if msgAnalysis["has_debugging"] {
			analysis["debugging"]++
		}
		if msgAnalysis["has_implementation"] {
			analysis["implementation"]++
		}
	}

	return analysis
}

// buildTaskCleanupPrompt creates a targeted prompt for task completion cleanup
func (s *chatService) buildTaskCleanupPrompt(analysis map[string]int) string {
	prompt := "Summarize this completed task implementation. "

	// Emphasize preservation based on content found
	if analysis["interfaces"] > 0 {
		prompt += "PRESERVE: All interface definitions and type contracts created. "
	}
	if analysis["design_decisions"] > 0 {
		prompt += "PRESERVE: Design decisions that affect future development. "
	}
	if analysis["error_handling"] > 0 {
		prompt += "PRESERVE: Error handling patterns and exception strategies established. "
	}
	if analysis["configuration"] > 0 {
		prompt += "PRESERVE: Configuration changes and setup requirements. "
	}

	// Emphasize removal of implementation details
	prompt += "\n\nREMOVE/CONDENSE: "
	if analysis["debugging"] > 0 {
		prompt += "Debugging output, print statements, and troubleshooting steps. "
	}
	prompt += "Step-by-step implementation details, intermediate code iterations, and tool execution logs. "

	prompt += "\n\nFocus on the final result and architectural impact, not the journey."
	return prompt
}

// buildPlanUpdatePrompt creates a prompt for updating project plans
func (s *chatService) buildPlanUpdatePrompt(analysis map[string]int) string {
	prompt := "Update the project plan and overview. "

	if analysis["interfaces"] > 0 {
		prompt += "PRESERVE: Current active interfaces and contracts. "
	}
	if analysis["design_decisions"] > 0 {
		prompt += "PRESERVE: Current architectural decisions and patterns. "
	}

	prompt += "\n\nREMOVE: Previous/outdated plans, abandoned approaches, and historical context that no longer applies. "

	prompt += "\n\nCreate a clean, current overview that reflects the latest project state."
	return prompt
}

// buildContextPreservationPrompt creates a prompt for preserving important context
func (s *chatService) buildContextPreservationPrompt(analysis map[string]int) string {
	prompt := "Preserve essential context while removing conversational noise. "

	if analysis["interfaces"] > 0 || analysis["contracts"] > 0 {
		prompt += "KEEP: All architectural definitions, interfaces, and contracts. "
	}
	if analysis["design_decisions"] > 0 {
		prompt += "KEEP: Key design decisions and architectural choices. "
	}
	if analysis["error_handling"] > 0 {
		prompt += "KEEP: Established error handling patterns. "
	}

	prompt += "\n\nCONDENSE/REMOVE: Debugging sessions, repetitive tool calls, intermediate steps, and chatty conversation. "

	prompt += "\n\nMaintain continuity for future development work."
	return prompt
}

// buildFullResetPrompt creates a prompt for major project resets
func (s *chatService) buildFullResetPrompt(analysis map[string]int) string {
	prompt := "Create a minimal foundation summary for major project changes. "

	if analysis["interfaces"] > 0 {
		prompt += "KEEP ONLY: Core interface definitions still relevant. "
	}
	if analysis["design_decisions"] > 0 {
		prompt += "KEEP ONLY: Fundamental architectural decisions. "
	}

	prompt += "\n\nREMOVE: Implementation details, historical context, previous plans, and everything not essential for starting fresh. "

	prompt += "\n\nProvide just enough context for new development to begin."
	return prompt
}

// compressMessages summarizes older messages to reduce token count while preserving context
// Returns the compressed messages, a flag indicating if the chat messages were replaced, and any error
func (s *chatService) compressMessages(
	ctx context.Context,
	chat *entities.Chat,
	model *entities.Model,
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

	// Create AI model for summarization, using model for context window
	aiModelFactory := integrations.NewAIModelFactory(s.toolRepo, s.logger)
	aiModel, err := aiModelFactory.CreateModelIntegration(model, provider, apiKey)
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

	summaryResponse, err := aiModel.GenerateResponse(ctx, historyMsgs, nil, options, nil)
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

// ensureToolCallResponses validates that every tool call has a corresponding response
// and creates error responses for any orphaned tool calls
func (s *chatService) ensureToolCallResponses(messages []*entities.Message) []*entities.Message {
	// Collect all tool call IDs from assistant messages
	toolCallIDs := make(map[string]bool)
	for _, msg := range messages {
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				toolCallIDs[toolCall.ID] = false // false = no response found yet
			}
		}
	}

	// Mark tool calls that have responses
	for _, msg := range messages {
		if msg.Role == "tool" && msg.ToolCallID != "" {
			if _, exists := toolCallIDs[msg.ToolCallID]; exists {
				toolCallIDs[msg.ToolCallID] = true // response found
			}
		}
	}

	// Create error responses for orphaned tool calls
	for toolCallID, hasResponse := range toolCallIDs {
		if !hasResponse {
			s.logger.Warn("Found orphaned tool call without response", zap.String("tool_call_id", toolCallID))
			errorMessage := &entities.Message{
				ID:         uuid.New().String(),
				Role:       "tool",
				Content:    "Tool execution failed: No response generated",
				ToolCallID: toolCallID,
				Timestamp:  time.Now(),
			}
			messages = append(messages, errorMessage)
		}
	}

	return messages
}

// CalculateTotalChatCost calculates the total cost of all messages in a chat
func (s *chatService) CalculateTotalChatCost(ctx context.Context, chatID string) (float64, error) {
	if chatID == "" {
		return 0, errors.ValidationErrorf("chat ID is required")
	}

	chat, err := s.chatRepo.GetChat(ctx, chatID)
	if err != nil {
		return 0, err
	}

	if chat.Usage == nil {
		return 0, nil
	}

	return chat.Usage.TotalCost, nil
}
