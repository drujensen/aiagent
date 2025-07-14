package tui

import "github.com/drujensen/aiagent/internal/domain/entities"

type (
	updatedChatMsg        *entities.Chat
	startCreateChatMsg    string
	canceledCreateChatMsg struct{}
	cancelSpinnerMsg      struct{}
)

type (
	startHistoryMsg    struct{}
	historySelectedMsg struct {
		chatID string
	}
	historyCancelledMsg struct{}
)

type (
	startUsageMsg   struct{}
	updatedUsageMsg struct {
		info string
	}
	usageCancelledMsg struct{}
)

type (
	startHelpMsg     struct{}
	helpCancelledMsg struct{}
)

type (
	startAgentsMsg   struct{}
	agentsFetchedMsg struct {
		agents []*entities.Agent
	}
	agentsCancelledMsg struct{}
)

type (
	startToolsMsg   struct{}
	toolsFetchedMsg struct {
		tools []*entities.ToolData
	}
	toolsCancelledMsg struct{}
)

// messages for error handling
type errMsg error
