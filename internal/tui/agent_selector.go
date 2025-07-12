package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type AgentSelector struct {
	chatService  services.ChatService
	agentService services.AgentService
	list         list.Model
	chatName     string
	err          error
}

func NewAgentSelector(chatService services.ChatService, agentService services.AgentService) AgentSelector {
	return AgentSelector{
		chatService:  chatService,
		agentService: agentService,
	}
}

func (a *AgentSelector) SetChatName(name string) {
	a.chatName = name
}

func (a AgentSelector) Init() tea.Cmd {
	return a.loadAgentsCmd()
}

func (a AgentSelector) Update(msg tea.Msg) (AgentSelector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if a.list.SelectedItem() == nil {
				a.err = fmt.Errorf("no agent selected")
				return a, nil
			}
			selectedAgent := a.list.SelectedItem().(*entities.Agent)
			return a, a.createChatCmd(selectedAgent.ID)
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "ctrl+c"))):
			return a, tea.Cmd(func() tea.Msg { return errMsg(fmt.Errorf("chat creation cancelled")) })
		}
	case tea.WindowSizeMsg:
		a.list.SetSize(msg.Width, msg.Height)
	case agentsLoadedMsg:
		items := make([]list.Item, len(msg))
		for i, agent := range msg {
			items[i] = agent
		}
		a.list = list.New(items, list.NewDefaultDelegate(), 0, 0)
		a.list.Title = "Select an Agent for Chat: " + a.chatName
		a.list.SetShowStatusBar(false)
		a.list.KeyMap.NextPage = key.NewBinding(key.WithKeys("pgdown", "J"))
		a.list.KeyMap.PrevPage = key.NewBinding(key.WithKeys("pgup", "K"))
		a.list.KeyMap.CursorUp = key.NewBinding(key.WithKeys("up", "k"))
		a.list.KeyMap.CursorDown = key.NewBinding(key.WithKeys("down", "j"))
		return a, nil
	}

	var cmd tea.Cmd
	a.list, cmd = a.list.Update(msg)
	return a, cmd
}

func (a AgentSelector) View() string {
	var sb strings.Builder
	sb.WriteString(a.list.View())
	if a.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("Error: %s\n", a.err)))
	}
	sb.WriteString("\nPress Enter to select, Esc to cancel")
	return sb.String()
}

type agentsLoadedMsg []*entities.Agent

func (a AgentSelector) loadAgentsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		agents, err := a.agentService.ListAgents(ctx)
		if err != nil {
			return errMsg(err)
		}
		return agentsLoadedMsg(agents)
	}
}

func (a AgentSelector) createChatCmd(agentID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		newChat, err := a.chatService.CreateChat(ctx, a.chatName, agentID)
		if err != nil {
			return errMsg(err)
		}
		err = a.chatService.SetActiveChat(ctx, newChat.ID)
		if err != nil {
			return errMsg(err)
		}
		return updatedChatMsg(newChat)
	}
}
