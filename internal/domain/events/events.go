package events

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/kelindar/event"
)

// Event types
const (
	ToolCallEventType       uint32 = 1
	MessageHistoryEventType uint32 = 2
)

// ToolCallEventData wraps the ToolCallEvent for publishing
type ToolCallEventData struct {
	Event *entities.ToolCallEvent
}

// MessageHistoryEventData wraps message history change events
type MessageHistoryEventData struct {
	ChatID   string
	Messages []*entities.Message
}

// Type implements the Event interface
func (t ToolCallEventData) Type() uint32 {
	return ToolCallEventType
}

// Type implements the Event interface
func (m MessageHistoryEventData) Type() uint32 {
	return MessageHistoryEventType
}

// PublishToolCallEvent publishes a tool call event
func PublishToolCallEvent(toolEvent *entities.ToolCallEvent) {
	event.Emit(ToolCallEventData{Event: toolEvent})
}

// SubscribeToToolCallEvents subscribes to tool call events
func SubscribeToToolCallEvents(handler func(data ToolCallEventData)) func() {
	return event.On(handler)
}

// PublishMessageHistoryEvent publishes a message history change event
func PublishMessageHistoryEvent(chatID string, messages []*entities.Message) {
	event.Emit(MessageHistoryEventData{ChatID: chatID, Messages: messages})
}

// SubscribeToMessageHistoryEvents subscribes to message history change events
func SubscribeToMessageHistoryEvents(handler func(data MessageHistoryEventData)) func() {
	return event.On(handler)
}
