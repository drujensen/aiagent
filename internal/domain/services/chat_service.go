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
	DeleteChat(ctx context.Context, id string) error
}

type chatService struct {
	chatRepo  interfaces.ChatRepository
	agentRepo interfaces.AgentRepository
	toolRepo  interfaces.ToolRepository
	config    *config.Config
}

func NewChatService(chatRepo interfaces.ChatRepository, agentRepo interfaces.AgentRepository, toolRepo interfaces.ToolRepository, cfg *config.Config) *chatService {
	return &chatService{
		chatRepo:  chatRepo,
		agentRepo: agentRepo,
		toolRepo:  toolRepo,
		config:    cfg,
	}
}

// estimateTokens approximates the token count for a message.
// Uses len(content) / 4 + 4 as a simple estimation (1 token ≈ 4 characters + overhead).
func estimateTokens(message map[string]string) int {
	content, ok := message["content"]
	if !ok {
		return 0
	}
	// Ceiling division for content length and add 4 tokens for role/overhead
	contentTokens := (len(content) + 3) / 4
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

	aiModel, err := integrations.NewAIModelIntegration(agent.Endpoint, resolvedAPIKey, agent.Model, s.toolRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI model: %v", err)
	}

	// Define constants
	const contextLength = 131072 // Model's maximum context length
	const toolBuffer = 500       // Estimated tokens for tool schemas

	// Set maxTokens with a reasonable default
	maxTokens := 4096 // Default completion tokens
	if agent.MaxTokens != nil {
		maxTokens = *agent.MaxTokens
	}

	// Calculate token limit for input messages (system + chat messages)
	tokenLimit := contextLength - maxTokens - toolBuffer

	// Prepare system message
	systemMessage := map[string]string{
		"role":    "system",
		"content": agent.SystemPrompt,
	}
	systemTokens := estimateTokens(systemMessage)
	if systemTokens > tokenLimit {
		return nil, fmt.Errorf("system prompt too large for the context window")
	}

	// Collect messages from newest to oldest to prioritize recent messages
	var tempMessages []map[string]string
	currentTokens := systemTokens
	for i := len(chat.Messages) - 1; i >= 0; i-- {
		msg := chat.Messages[i]
		msgMap := map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.Role == "tool" {
			msgMap["tool_call_id"] = msg.ID
		}
		msgTokens := estimateTokens(msgMap)
		if currentTokens+msgTokens > tokenLimit {
			break
		}
		tempMessages = append(tempMessages, msgMap)
		currentTokens += msgTokens
	}

	// Reverse tempMessages to chronological order (oldest to newest)
	for i, j := 0, len(tempMessages)-1; i < j; i, j = i+1, j-1 {
		tempMessages[i], tempMessages[j] = tempMessages[j], tempMessages[i]
	}

	// Prepare messagesToSend: [system, oldest, ..., newest]
	messagesToSend := append([]map[string]string{systemMessage}, tempMessages...)

	// Prepare tools
	tools := []*interfaces.ToolIntegration{}
	for _, toolName := range agent.Tools {
		tool, err := s.toolRepo.GetToolByName(toolName)
		if err != nil {
			return nil, fmt.Errorf("failed to get tool %s: %v", toolName, err)
		}
		tools = append(tools, tool)
	}

	// Prepare options
	options := map[string]interface{}{
		"temperature": 0.0, // Default temperature
		"max_tokens":  maxTokens,
	}
	if agent.Temperature != nil {
		options["temperature"] = *agent.Temperature
	}

	// Generate response with trimmed messages
	response, err := aiModel.GenerateResponse(messagesToSend, tools, options)
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
