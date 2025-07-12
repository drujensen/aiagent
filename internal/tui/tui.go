package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type (
	errMsg            error
	updatedChatMsg    *entities.Chat
	startCreateChatMsg string
)

type TUI struct {
	chatService   services.ChatService
	agentService  services.AgentService
	activeChat    *entities.Chat
	chatView      ChatView
	agentSelector AgentSelector
	state         string
	err           error
}

func NewTUI(chatService services.ChatService, agentService services.AgentService) TUI {
	ctx := context.Background()

	activeChat, err := chatService.GetActiveChat(ctx)
	if err != nil {
		fmt.Println("Error getting active chat:", err)
		activeChat = nil
	}

	return TUI{
		chatService:   chatService,
		agentService:  agentService,
		activeChat:    activeChat,
		chatView:      NewChatView(chatService, agentService, activeChat),
		agentSelector: NewAgentSelector(chatService, agentService),
		state:         "chat/view",
		err:           nil,
	}
}

func (t TUI) Init() tea.Cmd {
	return nil
}

func (t TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case startCreateChatMsg:
		t.state = "chat/create"
		t.agentSelector.SetChatName(string(msg))
		return t, t.agentSelector.Init()
	case updatedChatMsg:
		t.activeChat = msg
		t.chatView.SetActiveChat(msg)
		t.state = "chat/view"
		return t, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return t, tea.Quit
		default:
			if t.state == "chat/view" {
				var cmd tea.Cmd
				t.chatView, cmd = t.chatView.Update(msg)
				return t, cmd
			} else if t.state == "chat/create" {
				var cmd tea.Cmd
				t.agentSelector, cmd = t.agentSelector.Update(msg)
				return t, cmd
			}
		}
	case errMsg:
		t.err = msg
		return t, nil
	default:
		if t.state == "chat/view" {
			var cmd tea.Cmd
				t.chatView, cmd = t.chatView.Update(msg)
				return t, cmd
			} else if t.state == "chat/create" {
				var cmd tea.Cmd
				t.agentSelector, cmd = t.agentSelector.Update(msg)
				return t, cmd
			}
	}

	return t, nil
}

func (t TUI) View() string {
	switch t.state {
	case "chat/view":
		return t.chatView.View()
	case "chat/create":
		return t.agentSelector.View()
	}
	return "Error: Invalid state"
}
