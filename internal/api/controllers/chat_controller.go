package apicontrollers

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"
	"encoding/json"
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type ChatController struct {
	logger      *zap.Logger
	chatService services.ChatService
}

func NewChatController(logger *zap.Logger, chatService services.ChatService) *ChatController {
	return &ChatController{
		logger:      logger,
		chatService: chatService,
	}
}

// ListChats handles GET requests to list all chats
func (cc *ChatController) ListChats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chats, err := cc.chatService.ListChats(ctx)
	if err != nil {
		cc.logger.Error("Failed to list chats", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chats)
}

// GetChat handles GET requests to retrieve a specific chat
func (cc *ChatController) GetChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.URL.Query().Get("id")

	chat, err := cc.chatService.GetChat(ctx, id)
	if err != nil {
		if err == entities.ErrChatNotFound {
			http.Error(w, "Chat not found", http.StatusNotFound)
			return
		}
		cc.logger.Error("Failed to get chat", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chat)
}

// CreateChat handles POST requests to create a new chat
func (cc *ChatController) CreateChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var input struct {
		AgentID primitive.ObjectID `json:"agent_id"`
		Name    string             `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	chat, err := cc.chatService.CreateChat(ctx, input.AgentID.Hex(), input.Name)
	if err != nil {
		cc.logger.Error("Failed to create chat", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chat)
}

// UpdateChat handles PUT requests to update an existing chat
func (cc *ChatController) UpdateChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.URL.Query().Get("id")

	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	chat, err := cc.chatService.UpdateChat(ctx, id, input.Name)
	if err != nil {
		if err == entities.ErrChatNotFound {
			http.Error(w, "Chat not found", http.StatusNotFound)
			return
		}
		cc.logger.Error("Failed to update chat", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chat)
}

// DeleteChat handles DELETE requests to delete a specific chat
func (cc *ChatController) DeleteChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.URL.Query().Get("id")

	if err := cc.chatService.DeleteChat(ctx, id); err != nil {
		if err == entities.ErrChatNotFound {
			http.Error(w, "Chat not found", http.StatusNotFound)
			return
		}
		cc.logger.Error("Failed to delete chat", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SendMessage handles POST requests to send a new message to a chat
func (cc *ChatController) SendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.URL.Query().Get("id")

	var message entities.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	newMessage, err := cc.chatService.SendMessage(ctx, id, message)
	if err != nil {
		if err == entities.ErrChatNotFound {
			http.Error(w, "Chat not found", http.StatusNotFound)
			return
		}
		cc.logger.Error("Failed to send message", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newMessage)
}
