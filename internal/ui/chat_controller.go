package ui

import (
	"html/template"
	"net/http"
	"strings"

	"aiagent/internal/domain/repositories"
	"aiagent/internal/domain/services"

	"go.uber.org/zap"
)

// ChatController manages chat-related UI requests.
type ChatController struct {
	logger       *zap.Logger
	tmpl         *template.Template
	chatService  services.ChatService
	agentService services.AgentService
}

// NewChatController creates a new ChatController instance.
func NewChatController(logger *zap.Logger, tmpl *template.Template, chatService services.ChatService, agentService services.AgentService) *ChatController {
	return &ChatController{
		logger:       logger,
		tmpl:         tmpl,
		chatService:  chatService,
		agentService: agentService,
	}
}

// ChatHandler handles GET requests to /chat, rendering the chat page.
func (c *ChatController) ChatHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch active conversations
	conversations, err := c.chatService.ListActiveConversations(r.Context())
	if err != nil {
		c.logger.Error("Failed to list active conversations", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Fetch agents for sidebar hierarchy
	agents, err := c.agentService.ListAgents(r.Context())
	if err != nil {
		c.logger.Error("Failed to list agents", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	rootAgents := buildHierarchy(agents)

	// Prepare template data
	data := map[string]interface{}{
		"Title":           "Chat",
		"ContentTemplate": "chat_content",
		"Conversations":   conversations,
		"RootAgents":      rootAgents,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		c.logger.Error("Failed to render chat", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ChatConversationHandler handles GET requests to /chat/{id}, rendering the message history partial.
func (c *ChatController) ChatConversationHandler(w http.ResponseWriter, r *http.Request) {
	// Extract conversation ID from URL path
	id := strings.TrimPrefix(r.URL.Path, "/chat/")
	if id == "" {
		http.Error(w, "Conversation ID is required", http.StatusBadRequest)
		return
	}

	// Fetch conversation
	conversation, err := c.chatService.GetConversation(r.Context(), id)
	if err != nil {
		if err == repositories.ErrNotFound {
			http.Error(w, "Conversation not found", http.StatusNotFound)
		} else {
			c.logger.Error("Failed to get conversation", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Prepare data for template
	data := map[string]interface{}{
		"ConversationID": id,
		"Messages":       conversation.Messages,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := c.tmpl.ExecuteTemplate(w, "message_history", data); err != nil {
		c.logger.Error("Failed to render message_history", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
