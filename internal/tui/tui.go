package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type (
	errMsg         error
	updatedChatMsg *entities.Chat
)

type TUI struct {
	chatService  services.ChatService
	agentService services.AgentService
	activeChat   *entities.Chat
	chatView     ChatView
	chatForm     ChatForm
	state        string
	err          error
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
		chatView:     NewChatView(chatService, activeChat),
		state:        "chat/view",
		err:          nil,
	}
}

func (t TUI) Init() tea.Cmd {
	return nil
}

func (t TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return t, tea.Quit
		case tea.KeyCtrlN:
			t.state = "chat/create"
			return t, nil
		default:
			if t.state == "chat/view" {
				var cmd tea.Cmd
				t.chatView, cmd = t.chatView.Update(msg)
				if cmd != nil {
					return t, cmd
				}
			}
			if t.state == "chat/create" {
				var cmd tea.Cmd
				t.chatForm, cmd = t.chatForm.Update(msg)
				if cmd != nil {
					return t, cmd
				}
			}
		}
	case errMsg:
		t.err = msg
		return t, nil
	default:
		if t.state == "chat/view" {
			var cmd tea.Cmd
			t.chatView, cmd = t.chatView.Update(msg)
			if cmd != nil {
				return t, cmd
			}
		} else if t.state == "chat/create" {
			var cmd tea.Cmd
			t.chatForm, cmd = t.chatForm.Update(msg)
			if cmd != nil {
				return t, cmd
			}
		}
	}

	return t, nil
}

func (t TUI) View() string {
	switch t.state {
	case "chat/view":
		return t.chatView.View()
	case "chat/create":
		return t.chatForm.View()
	}
	return "Error: Invalid state"
}
