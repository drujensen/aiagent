package events

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/kelindar/event"
)

// Event types
const (
	ToolCallEventType uint32 = 1
)

// ToolCallEventData wraps the ToolCallEvent for publishing
type ToolCallEventData struct {
	Event *entities.ToolCallEvent
}

// Type implements the Event interface
func (t ToolCallEventData) Type() uint32 {
	return ToolCallEventType
}

// PublishToolCallEvent publishes a tool call event
func PublishToolCallEvent(toolEvent *entities.ToolCallEvent) {
	event.Emit(ToolCallEventData{Event: toolEvent})
}

// SubscribeToToolCallEvents subscribes to tool call events
func SubscribeToToolCallEvents(handler func(data ToolCallEventData)) func() {
	return event.On(handler)
}
