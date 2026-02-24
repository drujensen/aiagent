package events

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/kelindar/event"
)

// Event types
const (
	ToolCallEventType        uint32 = 1
	ProcessFinishedEventType uint32 = 2
	ProcessFailedEventType   uint32 = 3
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

// ProcessFinishedEventData wraps the ProcessFinishedEvent for publishing
type ProcessFinishedEventData struct {
	Event *entities.ProcessFinishedEvent
}

// Type implements the Event interface
func (p ProcessFinishedEventData) Type() uint32 {
	return ProcessFinishedEventType
}

// PublishProcessFinishedEvent publishes a process finished event
func PublishProcessFinishedEvent(processEvent *entities.ProcessFinishedEvent) {
	event.Emit(ProcessFinishedEventData{Event: processEvent})
}

// SubscribeToProcessFinishedEvents subscribes to process finished events
func SubscribeToProcessFinishedEvents(handler func(data ProcessFinishedEventData)) func() {
	return event.On(handler)
}

// ProcessFailedEventData wraps the ProcessFailedEvent for publishing
type ProcessFailedEventData struct {
	Event *entities.ProcessFailedEvent
}

// Type implements the Event interface
func (p ProcessFailedEventData) Type() uint32 {
	return ProcessFailedEventType
}

// PublishProcessFailedEvent publishes a process failed event
func PublishProcessFailedEvent(processEvent *entities.ProcessFailedEvent) {
	event.Emit(ProcessFailedEventData{Event: processEvent})
}

// SubscribeToProcessFailedEvents subscribes to process failed events
func SubscribeToProcessFailedEvents(handler func(data ProcessFailedEventData)) func() {
	return event.On(handler)
}
