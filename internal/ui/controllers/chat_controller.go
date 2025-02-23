package uicontrollers

import (
	"html/template"
	"net/http"
	"strings"

	"aiagent/internal/domain/services"

	"go.mongodb.org/mongo-driver/mongo"

	"go.uber.org/zap"
)

type ChatController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	chatService  services.ChatService
	agentService services.AgentService
}

func NewChatController(logger *zap.Logger, tmpl *template.Template, chatService services.ChatService, agentService services.AgentService) *ChatController {
	return &ChatController{
		logger:       logger,
		tmpl:         tmpl,
		chatService:  chatService,
		agentService: agentService,
	}
}

func (c *ChatController) ChatHandler(w http.ResponseWriter, r *http.Request) {
	conversations, err := c.chatService.ListActiveConversations(r.Context())
	if err != nil {
		c.logger.Error("Failed to list active conversations", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":           "Chat",
		"ContentTemplate": "chat_content",
		"Conversations":   conversations,
		"RootAgents":      agents,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render chat", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (c *ChatController) ChatConversationHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/chat/")
	if id == "" {
		http.Error(w, "Conversation ID is required", http.StatusBadRequest)
		return
	}

	conversation, err := c.chatService.GetConversation(r.Context(), id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Conversation not found", http.StatusNotFound)
		} else {
			c.logger.Error("Failed to get conversation", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	data := map[string]interface{}{
		"ConversationID": id,
		"Messages":       conversation.Messages,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "messages", data); err != nil {
		c.logger.Error("Failed to render message_history", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
