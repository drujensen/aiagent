package tui

import "github.com/drujensen/aiagent/internal/domain/entities"

// messages for chat view
type (
	updatedChatMsg        *entities.Chat
	startCreateChatMsg    string
	canceledCreateChatMsg struct{}
)

// messages for history view
type (
	startHistoryMsg    struct{}
	historySelectedMsg struct {
		chatID string
	}
	historyCancelledMsg struct{}
)

// messages for usage view
type (
	startUsageMsg   struct{}
	updatedUsageMsg struct {
		info string
	}
	usageCancelledMsg struct{}
)

// messages for help view
type (
	startHelpMsg     struct{}
	helpCancelledMsg struct{}
)

// messages for error handling
type errMsg error
