package tui

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/events"
)

type (
	updatedChatMsg         *entities.Chat
	startCreateChatMsg     string
	canceledCreateChatMsg  struct{}
	cancelSpinnerMsg       struct{}
	toolCallEventMsg       *entities.ToolCallEvent
	messageHistoryEventMsg events.MessageHistoryEventData
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

type (
	startCommandsMsg     struct{}
	executeCommandMsg    struct{ command string }
	commandsCancelledMsg struct{}
)

type errMsg error

type (
	startAgentSwitchMsg struct{}
	agentSelectedMsg    struct{ agentID string }
)

type (
	startModelSwitchMsg struct{}
	modelSelectedMsg    struct{ modelID string }
	modelsFetchedMsg    struct {
		models []*entities.Model
	}
	modelsCancelledMsg struct{}
)
