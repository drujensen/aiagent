package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ChatHub struct {
	connections map[string][]*websocket.Conn
	register    chan registration
	unregister  chan unregistration
	broadcast   chan broadcastMessage
}

type registration struct {
	conversationID string
	conn           *websocket.Conn
}

type unregistration struct {
	conversationID string
	conn           *websocket.Conn
}

type broadcastMessage struct {
	conversationID string
	message        entities.Message
}

func NewChatHub() *ChatHub {
	return &ChatHub{
		connections: make(map[string][]*websocket.Conn),
		register:    make(chan registration),
		unregister:  make(chan unregistration),
		broadcast:   make(chan broadcastMessage),
	}
}

func (h *ChatHub) Run() {
	for {
		select {
		case reg := <-h.register:
			h.connections[reg.conversationID] = append(h.connections[reg.conversationID], reg.conn)
		case unreg := <-h.unregister:
			if conns, ok := h.connections[unreg.conversationID]; ok {
				for i, conn := range conns {
					if conn == unreg.conn {
						h.connections[unreg.conversationID] = append(conns[:i], conns[i+1:]...)
						break
					}
				}
				if len(h.connections[unreg.conversationID]) == 0 {
					delete(h.connections, unreg.conversationID)
				}
			}
		case msg := <-h.broadcast:
			if conns, ok := h.connections[msg.conversationID]; ok {
				for _, conn := range conns {
					err := conn.WriteJSON(msg.message)
					if err != nil {
						log.Println("Write error:", err)
						h.unregister <- unregistration{msg.conversationID, conn}
					}
				}
			}
		}
	}
}

func (h *ChatHub) MessageListener(conversationID string, message entities.Message) {
	h.broadcast <- broadcastMessage{conversationID, message}
}

func ChatHandler(hub *ChatHub, chatService services.ChatService, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conversationID := r.URL.Query().Get("conversation_id")
		if conversationID == "" {
			http.Error(w, "Missing conversation_id", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		defer conn.Close()

		hub.register <- registration{conversationID, conn}

		defer func() {
			hub.unregister <- unregistration{conversationID, conn}
		}()

		for {
			var msg struct {
				ConversationID string           `json:"conversation_id"`
				Message        entities.Message `json:"message"`
			}
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("Read error:", err)
				break
			}

			if msg.ConversationID != conversationID {
				log.Println("Mismatched conversation ID")
				continue
			}

			err = chatService.SendMessage(r.Context(), conversationID, msg.Message)
			if err != nil {
				log.Println("SendMessage error:", err)
				conn.WriteJSON(map[string]string{"error": err.Error()})
			}
		}
	}
}
