package uicontrollers

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/services"
	"github.com/drujensen/aiagent/internal/impl/config"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type ChatController struct {
	logger          *zap.Logger
	tmpl            *template.Template
	chatService     services.ChatService
	agentService    services.AgentService
	modelService    services.ModelService
	providerService services.ProviderService
	filterService   *services.ModelFilterService
	globalConfig    *config.GlobalConfig
	activeCancelers sync.Map // Maps chatID to context cancelFunc
}

func NewChatController(logger *zap.Logger, tmpl *template.Template, chatService services.ChatService, agentService services.AgentService, modelService services.ModelService, providerService services.ProviderService, filterService *services.ModelFilterService, globalConfig *config.GlobalConfig) *ChatController {
	return &ChatController{
		logger:          logger,
		tmpl:            tmpl,
		chatService:     chatService,
		agentService:    agentService,
		modelService:    modelService,
		providerService: providerService,
		filterService:   filterService,
		globalConfig:    globalConfig,
		activeCancelers: sync.Map{},
	}
}

func (c *ChatController) RegisterRoutes(e *echo.Echo) {
	e.GET("/chats/new", c.ChatFormHandler)
	e.POST("/chats", c.CreateChatHandler)
	e.GET("/chats/:id", c.ChatHandler)
	e.GET("/chats/:id/edit", c.ChatFormHandler)
	e.PUT("/chats/:id", c.UpdateChatHandler)
	e.PUT("/chats/:id/agent", c.SwitchAgentHandler)
	e.PUT("/chats/:id/model", c.SwitchModelHandler)
	e.DELETE("/chats/:id", c.DeleteChatHandler)

	e.POST("/chats/:id/messages", c.SendMessageHandler)
	e.POST("/chats/:id/cancel", c.CancelMessageHandler)
	e.GET("/chat-cost", c.ChatCostHandler)
	e.GET("/chats/:id/messages", c.GetMessagesHandler)

	// Title management endpoints
	e.GET("/chats/:id/title", c.GetChatTitleHandler)
	e.POST("/chats/:id/generate-title", c.GenerateTitleHandler)
}

func (c *ChatController) ChatHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.Redirect(http.StatusFound, "/")
	}

	chat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.Redirect(http.StatusFound, "/")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
		}
	}

	agent, err := c.agentService.GetAgent(eCtx.Request().Context(), chat.AgentID)
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to get agent")
	}

	model, err := c.modelService.GetModel(eCtx.Request().Context(), chat.ModelID)
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to get model")
	}

	provider, err := c.providerService.GetProvider(eCtx.Request().Context(), model.ProviderID)
	providerName := string(model.ProviderType)
	if err == nil {
		providerName = provider.Name
	}

	var inputPrice, outputPrice float64
	if err == nil {
		for _, pricing := range provider.Models {
			if pricing.Name == model.ModelName {
				inputPrice = pricing.InputPricePerMille
				outputPrice = pricing.OutputPricePerMille
				break
			}
		}
	}

	availableAgents, err := c.agentService.ListAgents(eCtx.Request().Context())
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to list agents")
	}

	availableModels, err := c.modelService.ListModels(eCtx.Request().Context())
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to list models")
	}

	// Filter out tool execution notification messages from the chat messages
	filteredMessages := make([]entities.Message, 0, len(chat.Messages))
	for _, msg := range chat.Messages {
		if msg.Role == "assistant" {
			content := msg.Content
			// Skip messages that are just tool execution notifications
			if strings.Contains(content, " tool with parameters:") && strings.HasPrefix(content, "Executing ") {
				continue // Skip this notification message
			}
			if strings.HasPrefix(content, "Executing tool call.") {
				continue // Skip generic tool execution messages
			}
		}
		filteredMessages = append(filteredMessages, msg)
	}

	title := "Chat"
	if chat.Name != "" {
		title = chat.Name
	}

	data := map[string]any{
		"Title":           title,
		"ContentTemplate": "chat_content",
		"ChatID":          chatID,
		"ChatName":        chat.Name,
		"AgentName":       agent.Name,
		"ModelName":       model.Name,
		"ProviderName":    providerName,
		"InputPrice":      inputPrice,
		"OutputPrice":     outputPrice,
		"CurrentAgentID":  agent.ID,
		"CurrentModelID":  model.ID,
		"AvailableAgents": availableAgents,
		"AvailableModels": availableModels,
		"ChatCost":        chat.Usage.TotalCost,
		"TotalTokens":     chat.Usage.TotalTokens,
		"Messages":        filteredMessages,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

// EnrichedModelData represents a model with provider and pricing information for display
type EnrichedModelData struct {
	ID                  string
	Name                string
	ProviderName        string
	DisplayName         string
	InputPricePerMille  float64
	OutputPricePerMille float64
}

func (c *ChatController) ChatFormHandler(eCtx echo.Context) error {
	agents, err := c.agentService.ListAgents(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load agents")
	}

	models, err := c.modelService.ListModels(eCtx.Request().Context())
	if err != nil {
		c.logger.Error("Failed to list models", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to load models")
	}

	// Filter models to only include chat-compatible ones
	filteredModels := c.filterService.FilterChatCompatibleModels(models)

	// Enrich model data with provider and pricing information
	enrichedModels := make([]EnrichedModelData, 0, len(filteredModels))
	for _, model := range filteredModels {
		enriched := EnrichedModelData{
			ID:   model.ID,
			Name: model.Name,
		}

		// Get provider information
		if provider, err := c.providerService.GetProvider(eCtx.Request().Context(), model.ProviderID); err == nil {
			enriched.ProviderName = provider.Name

			// Find pricing for this specific model
			for _, pricing := range provider.Models {
				if pricing.Name == model.ModelName {
					enriched.InputPricePerMille = pricing.InputPricePerMille
					enriched.OutputPricePerMille = pricing.OutputPricePerMille

					// Format display name: {provider} - {name} in: ${cost} out: ${cost}
					inputCost := fmt.Sprintf("$%.2f", pricing.InputPricePerMille)
					outputCost := fmt.Sprintf("$%.2f", pricing.OutputPricePerMille)
					enriched.DisplayName = fmt.Sprintf("%s - %s in: %s out: %s", provider.Name, model.Name, inputCost, outputCost)
					break
				}
			}
		}

		// Fallback if pricing not found
		if enriched.DisplayName == "" {
			enriched.DisplayName = model.Name
		}

		enrichedModels = append(enrichedModels, enriched)
	}

	var chat *entities.Chat
	path := eCtx.Request().URL.Path
	isEdit := strings.HasSuffix(path, "/edit")
	if isEdit {
		id := eCtx.Param("id")
		if id == "" {
			return eCtx.String(http.StatusBadRequest, "Chat ID is required for editing")
		}
		chat, err = c.chatService.GetChat(eCtx.Request().Context(), id)
		if err != nil {
			switch err.(type) {
			case *errors.NotFoundError:
				return eCtx.Redirect(http.StatusFound, "/")
			default:
				return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
			}
		}
	}

	chatData := struct {
		ID      string
		Name    string
		AgentID string
		ModelID string
	}{}

	if chat != nil {
		chatData.ID = chat.ID
		chatData.Name = chat.Name
		chatData.AgentID = chat.AgentID
		chatData.ModelID = chat.ModelID
	} else {
		chatData.ID = uuid.New().String()

		// Set defaults from last used agent/model
		if c.globalConfig.LastUsedAgent != "" {
			// Verify the agent still exists
			for _, agent := range agents {
				if agent.ID == c.globalConfig.LastUsedAgent {
					chatData.AgentID = c.globalConfig.LastUsedAgent
					break
				}
			}
		}
		if c.globalConfig.LastUsedModel != "" {
			// Verify the model still exists
			for _, model := range models {
				if model.ID == c.globalConfig.LastUsedModel {
					chatData.ModelID = c.globalConfig.LastUsedModel
					break
				}
			}
		}
	}

	data := map[string]any{
		"Title":           "AI Agents - New Chat",
		"ContentTemplate": "chat_form_content",
		"Chat":            chatData,
		"Agents":          agents,
		"Models":          enrichedModels,
		"IsEdit":          isEdit,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "layout", data)
}

func (c *ChatController) CreateChatHandler(eCtx echo.Context) error {
	// Check if this is JSON request (quick start) or form data (advanced)
	contentType := eCtx.Request().Header.Get("Content-Type")

	var agentID, modelID, name string
	var message string

	if strings.Contains(contentType, "application/json") {
		// Quick start - use defaults
		var input struct {
			Name    string `json:"name"`
			AgentID string `json:"agent_id"`
			ModelID string `json:"model_id"`
		}
		if err := eCtx.Bind(&input); err != nil {
			return eCtx.String(http.StatusBadRequest, "Invalid request")
		}

		name = input.Name

		// Use defaults if not specified
		if input.AgentID != "" {
			agentID = input.AgentID
		} else {
			// Use first available agent as default
			agents, err := c.agentService.ListAgents(eCtx.Request().Context())
			if err != nil || len(agents) == 0 {
				return eCtx.String(http.StatusInternalServerError, "No agents available")
			}
			agentID = agents[0].ID
		}

		if input.ModelID != "" {
			modelID = input.ModelID
		} else {
			// Use first available model as default
			models, err := c.modelService.ListModels(eCtx.Request().Context())
			if err != nil || len(models) == 0 {
				return eCtx.String(http.StatusInternalServerError, "No models available")
			}
			modelID = models[0].ID
		}
	} else {
		// Form data - could be quick start (with message) or advanced options
		agentID = eCtx.FormValue("agent-select")
		modelID = eCtx.FormValue("model-select")
		name = eCtx.FormValue("chat-name")
		message = eCtx.FormValue("message")

		// If message is provided, this is a quick start
		if message != "" {
			// Use defaults if not specified (last used or first available)
			if agentID == "" {
				if c.globalConfig.LastUsedAgent != "" {
					agentID = c.globalConfig.LastUsedAgent
				} else {
					// Use first available agent as default
					agents, err := c.agentService.ListAgents(eCtx.Request().Context())
					if err != nil || len(agents) == 0 {
						return eCtx.String(http.StatusInternalServerError, "No agents available")
					}
					agentID = agents[0].ID
				}
			}
			if modelID == "" {
				if c.globalConfig.LastUsedModel != "" {
					modelID = c.globalConfig.LastUsedModel
				} else {
					// Use first available model as default
					models, err := c.modelService.ListModels(eCtx.Request().Context())
					if err != nil || len(models) == 0 {
						return eCtx.String(http.StatusInternalServerError, "No models available")
					}
					modelID = models[0].ID
				}
			}

			// Generate temporary title for quick start
			if name == "" {
				name = fmt.Sprintf("New Chat - %s", time.Now().Format("2006-01-02 15:04"))
			}
		} else {
			// Traditional form - require selections
			if agentID == "" {
				return eCtx.String(http.StatusBadRequest, "Agent selection is required")
			}
			if modelID == "" {
				return eCtx.String(http.StatusBadRequest, "Model selection is required")
			}
		}
	}

	chat, err := c.chatService.CreateChat(eCtx.Request().Context(), agentID, modelID, name)
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to create chat")
	}

	// Update global config with last used agent and model
	if agentID != "" && agentID != c.globalConfig.LastUsedAgent {
		c.globalConfig.LastUsedAgent = agentID
		if err := config.SaveGlobalConfig(c.globalConfig, c.logger); err != nil {
			c.logger.Warn("Failed to save global config with last used agent", zap.Error(err))
		}
	}
	if modelID != "" && modelID != c.globalConfig.LastUsedModel {
		c.globalConfig.LastUsedModel = modelID
		if err := config.SaveGlobalConfig(c.globalConfig, c.logger); err != nil {
			c.logger.Warn("Failed to save global config with last used model", zap.Error(err))
		}
	}

	// If this is a quick start with message, send the first message
	if message != "" {
		userMessage := entities.NewMessage("user", message)

		// Create a cancellable context
		ctx, cancel := context.WithCancel(eCtx.Request().Context())
		defer cancel()

		// Store the cancellation function
		c.activeCancelers.Store(chat.ID, cancel)

		// Send the message and get the AI responses
		_, err := c.chatService.SendMessage(ctx, chat.ID, userMessage)
		if err != nil {
			// If sending fails, still redirect to chat so user can retry
			c.logger.Warn("Failed to send initial message, redirecting to chat anyway", zap.Error(err), zap.String("chatID", chat.ID))
		}
		// Note: SendMessage already generates title synchronously for first exchange

		// Clean up cancellation
		defer func() {
			cancel()
			c.activeCancelers.Delete(chat.ID)
		}()
	}

	// For JSON requests, return JSON response
	if strings.Contains(contentType, "application/json") {
		return eCtx.JSON(http.StatusCreated, map[string]string{
			"id":   chat.ID,
			"name": chat.Name,
		})
	}

	// For form requests, redirect
	eCtx.Response().Header().Set("HX-Redirect", "/chats/"+chat.ID)
	return eCtx.String(http.StatusOK, "Chat created successfully")
}

func (c *ChatController) UpdateChatHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	agentID := eCtx.FormValue("agent-select")
	name := eCtx.FormValue("chat-name")
	if chatID == "" || name == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID and name are required")
	}

	// Get existing chat to preserve model ID
	existingChat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to get chat")
	}

	_, err = c.chatService.UpdateChat(eCtx.Request().Context(), chatID, agentID, existingChat.ModelID, name)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Chat not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
		}
	}

	eCtx.Response().Header().Set("HX-Redirect", "/chats/"+chatID)
	return eCtx.String(http.StatusOK, "Chat updated successfully")
}

func (c *ChatController) SwitchModelHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	var input struct {
		ModelID string `json:"model_id"`
	}
	if err := eCtx.Bind(&input); err != nil {
		return eCtx.String(http.StatusBadRequest, "Invalid request body")
	}

	if input.ModelID == "" {
		return eCtx.String(http.StatusBadRequest, "Model ID is required")
	}

	// Get existing chat to preserve agent
	existingChat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to get chat")
	}

	// Update chat with new model
	updatedChat, err := c.chatService.UpdateChat(eCtx.Request().Context(), chatID, existingChat.AgentID, input.ModelID, existingChat.Name)
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to switch model")
	}

	return eCtx.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"chat":    updatedChat,
	})
}

func (c *ChatController) SwitchAgentHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	var input struct {
		AgentID string `json:"agent_id"`
	}
	if err := eCtx.Bind(&input); err != nil {
		return eCtx.String(http.StatusBadRequest, "Invalid request body")
	}

	if input.AgentID == "" {
		return eCtx.String(http.StatusBadRequest, "Agent ID is required")
	}

	// Get existing chat to preserve model
	existingChat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to get chat")
	}

	// Update chat with new agent
	updatedChat, err := c.chatService.UpdateChat(eCtx.Request().Context(), chatID, input.AgentID, existingChat.ModelID, existingChat.Name)
	if err != nil {
		return eCtx.String(http.StatusInternalServerError, "Failed to switch agent")
	}

	return eCtx.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"chat":    updatedChat,
	})
}

func (c *ChatController) DeleteChatHandler(eCtx echo.Context) error {
	id := eCtx.Param("id")
	if id == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	err := c.chatService.DeleteChat(eCtx.Request().Context(), id)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Chat not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
		}
	}

	// After successful deletion, return the updated chats list
	eCtx.Response().Header().Set("HX-Trigger", `{"refreshChats": true}`)
	return eCtx.String(http.StatusOK, "Chat deleted successfully")
}

func (c *ChatController) SendMessageHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	messageContent := eCtx.FormValue("message")
	if messageContent == "" {
		return eCtx.String(http.StatusBadRequest, "Message content is required")
	}

	userMessage := entities.NewMessage("user", messageContent)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(eCtx.Request().Context())

	// Store the cancellation function
	c.activeCancelers.Store(chatID, cancel)

	// Ensure cancellation when the operation completes (success or failure)
	defer func() {
		cancel()
		c.activeCancelers.Delete(chatID)
	}()

	// Send the message and get the AI responses
	aiMessage, err := c.chatService.SendMessage(ctx, chatID, userMessage)
	if err != nil {
		switch err.(type) {
		case *errors.CanceledError:
			c.logger.Info("Message processing was canceled", zap.String("chatID", chatID))
			return eCtx.String(http.StatusRequestTimeout, "Request was canceled")
		case *errors.NotFoundError:
			return eCtx.String(http.StatusNotFound, "Chat not found")
		default:
			return eCtx.String(http.StatusInternalServerError, "Failed to load chat")
		}
	}

	// Get the chat to find all messages since the user's message
	chat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		c.logger.Error("Failed to get chat after sending message", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to retrieve updated messages: "+err.Error())
	}

	// Find the index of the user's message
	userMessageIndex := -1
	for i, msg := range chat.Messages {
		if msg.ID == userMessage.ID {
			userMessageIndex = i
			break
		}
	}

	// Extract all responses after the user message (AI assistant + tool messages)
	var aiMessages []entities.Message
	if userMessageIndex >= 0 && userMessageIndex < len(chat.Messages)-1 {
		rawMessages := chat.Messages[userMessageIndex+1:]
		// Filter out tool execution notification messages
		for _, msg := range rawMessages {
			if msg.Role == "assistant" {
				content := msg.Content
				// Skip messages that are just tool execution notifications
				if strings.Contains(content, " tool with parameters:") && strings.HasPrefix(content, "Executing ") {
					continue // Skip this notification message
				}
				if strings.HasPrefix(content, "Executing tool call.") {
					continue // Skip generic tool execution messages
				}
			}
			aiMessages = append(aiMessages, msg)
		}
	} else if aiMessage != nil {
		// Fallback to just the direct AI response if we can't find all messages
		aiMessages = []entities.Message{*aiMessage}
	}

	// Create data for the template
	messageSessionID := fmt.Sprintf("message-session-%d", len(chat.Messages)/2) // Rough estimate of session count

	data := map[string]any{
		"UserMessage": userMessage,
		"AIMessages":  aiMessages,
		"SessionID":   messageSessionID,
	}

	// Render complete message session using the template
	var buf bytes.Buffer
	if err := c.tmpl.ExecuteTemplate(&buf, "message_session_partial", data); err != nil {
		c.logger.Error("Failed to render message session", zap.Error(err))
		return eCtx.String(http.StatusInternalServerError, "Failed to render messages: "+err.Error())
	}

	// Create a placeholder for the next message session
	responseHTML := fmt.Sprintf(`<div id="message-session-%s" class="message-session">%s</div><div id="next-message-session"></div>`,
		messageSessionID, buf.String())

	// Set the header to trigger scrolling to the response and refresh chat cost
	eCtx.Response().Header().Set("HX-Trigger", "messageReceived, refreshChatCost")

	return eCtx.HTML(http.StatusOK, responseHTML)
}

// CancelMessageHandler cancels an ongoing message processing operation
// ChatCostHandler handles the request to update the token and cost display
func (c *ChatController) ChatCostHandler(eCtx echo.Context) error {
	chatID := eCtx.QueryParam("chat_id")
	if chatID == "" {
		return eCtx.HTML(http.StatusBadRequest, "<div class=\"chat-cost\">Tokens: 0 Cost: $0.00</div>")
	}

	chat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		c.logger.Error("Failed to get chat for cost update", zap.Error(err))
		return eCtx.HTML(http.StatusInternalServerError, "<div class=\"chat-cost\">Tokens: 0 Cost: $0.00</div>")
	}

	// Recalculate usage to ensure it's up to date
	chat.UpdateUsage()

	c.logger.Info("Chat cost update", zap.String("chatID", chatID), zap.Int("totalTokens", chat.Usage.TotalTokens), zap.Float64("totalCost", chat.Usage.TotalCost))

	data := map[string]any{
		"TotalTokens": chat.Usage.TotalTokens,
		"ChatCost":    chat.Usage.TotalCost,
	}

	return c.tmpl.ExecuteTemplate(eCtx.Response().Writer, "chat_cost_partial", data)
}

func (c *ChatController) CancelMessageHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.String(http.StatusBadRequest, "Chat ID is required")
	}

	// Check if there's an active cancellation function for this chat
	cancelValue, exists := c.activeCancelers.Load(chatID)
	if !exists {
		c.logger.Warn("No active request to cancel for this chat", zap.String("chatID", chatID))
		return eCtx.String(http.StatusOK, "No active request to cancel")
	}

	// Execute the cancellation
	if cancelFunc, ok := cancelValue.(context.CancelFunc); ok {
		cancelFunc()
		c.logger.Info("Request canceled successfully", zap.String("chatID", chatID))
		return eCtx.String(http.StatusOK, "Request canceled")
	}

	c.logger.Error("Invalid cancellation function", zap.String("chatID", chatID))
	return eCtx.String(http.StatusInternalServerError, "Failed to cancel request")
}

// GetMessagesHandler returns the latest messages for a chat
func (c *ChatController) GetMessagesHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Chat ID is required"})
	}

	chat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.JSON(http.StatusNotFound, map[string]string{"error": "Chat not found"})
		default:
			c.logger.Error("Failed to get chat", zap.Error(err))
			return eCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load chat"})
		}
	}

	// Filter out tool execution notification messages
	filteredMessages := make([]entities.Message, 0, len(chat.Messages))
	for _, msg := range chat.Messages {
		if msg.Role == "assistant" {
			content := msg.Content
			// Skip messages that are just tool execution notifications
			if strings.Contains(content, " tool with parameters:") && strings.HasPrefix(content, "Executing ") {
				continue // Skip this notification message
			}
			if strings.HasPrefix(content, "Executing tool call.") {
				continue // Skip generic tool execution messages
			}
		}
		filteredMessages = append(filteredMessages, msg)
	}

	return eCtx.JSON(http.StatusOK, map[string]interface{}{
		"chat_id":  chatID,
		"messages": filteredMessages,
	})
}

// GetChatTitleHandler returns the current title of a chat
func (c *ChatController) GetChatTitleHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Chat ID is required"})
	}

	chat, err := c.chatService.GetChat(eCtx.Request().Context(), chatID)
	if err != nil {
		switch err.(type) {
		case *errors.NotFoundError:
			return eCtx.JSON(http.StatusNotFound, map[string]string{"error": "Chat not found"})
		default:
			c.logger.Error("Failed to get chat title", zap.Error(err))
			return eCtx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load chat"})
		}
	}

	return eCtx.JSON(http.StatusOK, map[string]string{
		"title": chat.Name,
	})
}

// GenerateTitleHandler triggers asynchronous title generation for a chat
func (c *ChatController) GenerateTitleHandler(eCtx echo.Context) error {
	chatID := eCtx.Param("id")
	if chatID == "" {
		return eCtx.JSON(http.StatusBadRequest, map[string]string{"error": "Chat ID is required"})
	}

	// Trigger title generation asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if _, err := c.chatService.GenerateAndUpdateTitle(ctx, chatID); err != nil {
			c.logger.Warn("Failed to generate title", zap.Error(err), zap.String("chatID", chatID))
		}
	}()

	return eCtx.JSON(http.StatusOK, map[string]string{
		"status": "Title generation started",
	})
}
