package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
)

/**
 * @description
 * This file implements the WebSocket handler for real-time chat functionality in the AI Workflow Automation Platform.
 * It manages bidirectional communication between clients and the server, broadcasting messages to connected clients
 * for specific conversations, enabling human-agent interaction.
 *
 * Key features:
 * - WebSocket Connection Management: Upgrades HTTP connections to WebSocket, registering clients with a hub.
 * - Message Broadcasting: Broadcasts new messages to all connected clients for a conversation.
 * - Authentication: Validates clients using the X-API-Key query parameter.
 * - Real-time Messaging: Handles sending and receiving messages, integrating with ChatService.
 *
 * @dependencies
 * - github.com/gorilla/websocket: For WebSocket protocol handling.
 * - aiagent/internal/domain/entities: Provides the Message entity definition.
 * - aiagent/internal/domain/services: Provides ChatService for message persistence.
 * - aiagent/internal/infrastructure/config: Provides configuration for API key validation.
 * - net/http: For HTTP handling and WebSocket upgrades.
 * - log: For logging errors and connection events.
 *
 * @notes
 * - Authentication uses api_key query parameter; consider tightening origin checks in production.
 * - Messages require conversation_id for routing; mismatches are logged and ignored.
 * - Errors during WebSocket writes trigger connection unregistration to prevent stale connections.
 * - Edge cases include connection failures, invalid JSON, and unauthorized access.
 * - Assumption: ChatService is properly initialized and injected; MessageListener is registered elsewhere.
 * - Limitation: Currently supports one connection per conversation; extendable for multiple clients if needed.
 */

// upgrader configures the WebSocket upgrader with permissive origin checking for development.
// It upgrades HTTP connections to WebSocket protocol, ensuring compatibility with clients.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now; tighten in production
	},
}

// ChatHub manages WebSocket connections and broadcasts messages to clients subscribed to specific conversations.
// It runs in a goroutine, handling registration, unregistration, and broadcasting events.
type ChatHub struct {
	connections map[string][]*websocket.Conn // conversationID to list of connections
	register    chan registration            // Channel for registering new connections
	unregister  chan unregistration          // Channel for unregistering connections
	broadcast   chan broadcastMessage        // Channel for broadcasting messages
}

// registration represents a new WebSocket connection to be registered.
// It specifies the conversation and connection details.
type registration struct {
	conversationID string
	conn           *websocket.Conn
}

// unregistration represents a WebSocket connection to be unregistered.
// It specifies the conversation and connection details.
type unregistration struct {
	conversationID string
	conn           *websocket.Conn
}

// broadcastMessage represents a message to be broadcasted to all clients in a conversation.
// It includes the conversation ID and the message content.
type broadcastMessage struct {
	conversationID string
	message        entities.Message
}

// NewChatHub initializes a new ChatHub with channels for registration, unregistration, and broadcasting.
// It prepares the hub for managing WebSocket connections and message broadcasts.
//
// Returns:
// - *ChatHub: A new instance ready to run.
func NewChatHub() *ChatHub {
	return &ChatHub{
		connections: make(map[string][]*websocket.Conn),
		register:    make(chan registration),
		unregister:  make(chan unregistration),
		broadcast:   make(chan broadcastMessage),
	}
}

// Run starts the hub's event loop to handle registrations, unregistrations, and message broadcasts.
// It runs in a separate goroutine, continuously processing events from channels.
func (h *ChatHub) Run() {
	for {
		select {
		case reg := <-h.register:
			// Append new connection to the conversation's list
			h.connections[reg.conversationID] = append(h.connections[reg.conversationID], reg.conn)
		case unreg := <-h.unregister:
			// Remove connection from the conversation's list
			if conns, ok := h.connections[unreg.conversationID]; ok {
				for i, conn := range conns {
					if conn == unreg.conn {
						h.connections[unreg.conversationID] = append(conns[:i], conns[i+1:]...)
						break
					}
				}
				// Clean up empty conversation lists
				if len(h.connections[unreg.conversationID]) == 0 {
					delete(h.connections, unreg.conversationID)
				}
			}
		case msg := <-h.broadcast:
			// Broadcast message to all connections in the conversation
			if conns, ok := h.connections[msg.conversationID]; ok {
				for _, conn := range conns {
					err := conn.WriteJSON(msg.message)
					if err != nil {
						log.Println("Write error:", err)
						// Unregister connection on write failure
						h.unregister <- unregistration{msg.conversationID, conn}
					}
				}
			}
		}
	}
}

// MessageListener is a callback function for the ChatService to notify the hub of new messages.
// It sends the message to the broadcast channel for distribution to clients.
//
// Parameters:
// - conversationID: The ID of the conversation the message belongs to.
// - message: The Message entity that was added.
func (h *ChatHub) MessageListener(conversationID string, message entities.Message) {
	h.broadcast <- broadcastMessage{conversationID, message}
}

// ChatHandler returns an HTTP handler function for WebSocket connections to /ws/chat.
// It authenticates clients, upgrades connections, registers with the hub, and handles incoming messages.
//
// Parameters:
// - hub: The ChatHub instance for managing connections.
// - chatService: The ChatService for sending messages.
// - cfg: The configuration for API key validation.
//
// Returns:
// - http.HandlerFunc: The handler function for WebSocket connections.
func ChatHandler(hub *ChatHub, chatService services.ChatService, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate API key from query parameter
		apiKey := r.URL.Query().Get("api_key")
		if apiKey != cfg.LocalAPIKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate conversation ID from query parameter
		conversationID := r.URL.Query().Get("conversation_id")
		if conversationID == "" {
			http.Error(w, "Missing conversation_id", http.StatusBadRequest)
			return
		}

		// Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		defer conn.Close()

		// Register connection with hub
		hub.register <- registration{conversationID, conn}

		// Unregister on connection close
		defer func() {
			hub.unregister <- unregistration{conversationID, conn}
		}()

		// Read loop to handle incoming messages from the client
		for {
			// Define struct for incoming message parsing
			var msg struct {
				ConversationID string           `json:"conversation_id"`
				Message        entities.Message `json:"message"`
			}
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("Read error:", err)
				break
			}

			// Verify conversation ID matches query parameter
			if msg.ConversationID != conversationID {
				log.Println("Mismatched conversation ID")
				continue
			}

			// Send the message via ChatService
			err = chatService.SendMessage(r.Context(), conversationID, msg.Message)
			if err != nil {
				log.Println("SendMessage error:", err)
				// Send error back to client
				conn.WriteJSON(map[string]string{"error": err.Error()})
			}
		}
	}
}
