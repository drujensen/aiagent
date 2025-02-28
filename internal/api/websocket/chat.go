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
	ChatID string
	conn   *websocket.Conn
}

type unregistration struct {
	ChatID string
	conn   *websocket.Conn
}

type broadcastMessage struct {
	ChatID  string
	message entities.Message
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
			h.connections[reg.ChatID] = append(h.connections[reg.ChatID], reg.conn)
		case unreg := <-h.unregister:
			if conns, ok := h.connections[unreg.ChatID]; ok {
				for i, conn := range conns {
					if conn == unreg.conn {
						h.connections[unreg.ChatID] = append(conns[:i], conns[i+1:]...)
						break
					}
				}
				if len(h.connections[unreg.ChatID]) == 0 {
					delete(h.connections, unreg.ChatID)
				}
			}
		case msg := <-h.broadcast:
			if conns, ok := h.connections[msg.ChatID]; ok {
				for _, conn := range conns {
					err := conn.WriteJSON(msg.message)
					if err != nil {
						log.Println("Write error:", err)
						h.unregister <- unregistration{msg.ChatID, conn}
					}
				}
			}
		}
	}
}

func (h *ChatHub) MessageListener(chatID string, message entities.Message) {
	h.broadcast <- broadcastMessage{chatID, message}
}

func ChatHandler(hub *ChatHub, chatService services.ChatService, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chatID := r.URL.Query().Get("chat_id") // Changed from "Chat_id" to "chat_id"
		if chatID == "" {
			http.Error(w, "Missing chat_id", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}
		defer conn.Close()

		hub.register <- registration{chatID, conn}

		defer func() {
			hub.unregister <- unregistration{chatID, conn}
		}()

		for {
			var msg struct {
				ChatID  string           `json:"Chat_id"`
				Message entities.Message `json:"message"`
			}
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("Read error:", err)
				break
			}

			if msg.ChatID != chatID {
				log.Println("Mismatched chat ID")
				continue
			}

			err = chatService.SendMessage(r.Context(), chatID, msg.Message)
			if err != nil {
				log.Println("SendMessage error:", err)
				conn.WriteJSON(map[string]string{"error": err.Error()})
			}
		}
	}
}
