package tui

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

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

// messages for error handling
type errMsg error

type TUI struct {
	chatService  services.ChatService
	agentService services.AgentService
	activeChat   *entities.Chat

	chatView    ChatView
	chatForm    ChatForm
	historyView HistoryView

	state string
	err   error
}

func NewTUI(chatService services.ChatService, agentService services.AgentService) TUI {
	ctx := context.Background()

	activeChat, err := chatService.GetActiveChat(ctx)
	if err != nil {
		fmt.Println("Error getting active chat:", err)
		activeChat = nil
	}

	return TUI{
		chatService:  chatService,
		agentService: agentService,
		activeChat:   activeChat,
		chatView:     NewChatView(chatService, agentService, activeChat),
		chatForm:     NewChatForm(chatService, agentService),
		historyView:  NewHistoryView(chatService),
		state:        "chat/view",
		err:          nil,
	}
}

func (t TUI) Init() tea.Cmd {
	return nil
}

func (t TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle chat view messages
	case startCreateChatMsg:
		t.state = "chat/create"
		t.chatForm.SetChatName(string(msg))
		return t, t.chatForm.Init()
	case updatedChatMsg:
		t.activeChat = msg
		t.chatView.SetActiveChat(msg)
		t.state = "chat/view"
		return t, nil
	case canceledCreateChatMsg:
		t.state = "chat/view"
		t.chatView.err = errors.New("chat creation cancelled")
		if t.activeChat != nil {
			t.chatView.SetActiveChat(t.activeChat)
		}
		return t, t.chatView.Init()

	// Handle history view messages
	case startHistoryMsg:
		t.state = "chat/history"
		return t, t.historyView.Init()
	case historySelectedMsg:
		ctx := context.Background()
		err := t.chatService.SetActiveChat(ctx, msg.chatID)
		if err != nil {
			return t, func() tea.Msg { return errMsg(err) }
		}
		chat, err := t.chatService.GetChat(ctx, msg.chatID)
		if err != nil {
			return t, func() tea.Msg { return errMsg(err) }
		}
		t.activeChat = chat
		t.chatView.SetActiveChat(chat)
		t.state = "chat/view"
		return t, nil
	case historyCancelledMsg:
		t.state = "chat/view"
		if t.activeChat != nil {
			t.chatView.SetActiveChat(t.activeChat)
		}
		return t, nil

	// Handle global key messages and errors
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return t, tea.Quit
		}

	case errMsg:
		fmt.Println("TUI: Received errMsg:", msg)
		t.err = msg
		return t, nil
	}

	var cmd tea.Cmd
	switch t.state {
	case "chat/view":
		t.chatView, cmd = t.chatView.Update(msg)
	case "chat/create":
		t.chatForm, cmd = t.chatForm.Update(msg)
	case "chat/history":
		t.historyView, cmd = t.historyView.Update(msg)
	}
	return t, cmd
}

func (t TUI) View() string {
	switch t.state {
	case "chat/view":
		return t.chatView.View()
	case "chat/create":
		return t.chatForm.View()
	case "chat/history":
		return t.historyView.View()
	}

	return "Error: Invalid state"
}
