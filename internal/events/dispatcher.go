package events

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	kelindarEvent "github.com/kelindar/event"
)

// Global event dispatcher instance
var Dispatcher = kelindarEvent.NewDispatcher()

// PublishToolCallStarted publishes a tool call started event
func PublishToolCallStarted(chatID, toolCallID, toolName, arguments string) {
	eventData := entities.NewToolCallStartedEvent(chatID, toolCallID, toolName, arguments)
	kelindarEvent.Publish(Dispatcher, eventData)
}

// PublishToolCallProgress publishes a tool call progress event
func PublishToolCallProgress(chatID, toolCallID, toolName, progress string) {
	eventData := entities.NewToolCallProgressEvent(chatID, toolCallID, toolName, progress)
	kelindarEvent.Publish(Dispatcher, eventData)
}

// PublishToolCallCompleted publishes a tool call completed event
func PublishToolCallCompleted(chatID, toolCallID, toolName, result string) {
	eventData := entities.NewToolCallCompletedEvent(chatID, toolCallID, toolName, result)
	kelindarEvent.Publish(Dispatcher, eventData)
}

// PublishToolCallError publishes a tool call error event
func PublishToolCallError(chatID, toolCallID, toolName, errorMsg string) {
	eventData := entities.NewToolCallErrorEvent(chatID, toolCallID, toolName, errorMsg)
	kelindarEvent.Publish(Dispatcher, eventData)
}

// SubscribeToToolCallEvents subscribes to all tool call events for a specific chat
func SubscribeToToolCallEvents(chatID string, handler func(entities.ToolCallEventData)) func() {
	return kelindarEvent.Subscribe(Dispatcher, func(e entities.ToolCallEventData) {
		// Only handle events for the specified chat
		if e.ChatID == chatID {
			handler(e)
		}
	})
}
