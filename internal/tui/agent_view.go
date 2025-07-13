package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type AgentView struct {
	agentService services.AgentService
	list         list.Model
	width        int
	height       int
	err          error
}

func NewAgentView(agentService services.AgentService) AgentView {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("6")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("7"))
	delegate.SetHeight(2)

	l := list.New([]list.Item{}, delegate, 100, 10)
	l.Title = "Available Agents"
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowPagination(true)

	return AgentView{
		agentService: agentService,
		list:         l,
	}
}

func (v AgentView) Init() tea.Cmd {
	return v.fetchAgentsCmd()
}

func (v AgentView) Update(msg tea.Msg) (AgentView, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = m.Width
		v.height = m.Height
		listHeight := m.Height - 3
		v.list.SetSize(m.Width-4, listHeight)
		return v, nil

	case tea.KeyMsg:
		switch m.String() {
		case "esc":
			return v, func() tea.Msg { return agentsCancelledMsg{} }
		}

	case agentsFetchedMsg:
		items := make([]list.Item, len(m.agents))
		for i, agent := range m.agents {
			items[i] = agent
		}
		if len(items) == 0 {
			items = append(items, list.Item(&entities.Agent{Name: "No agents available", Model: ""}))
		}
		v.list.SetItems(items)
		v.err = nil
		return v, nil

	case errMsg:
		v.err = m
		return v, nil
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v AgentView) View() string {
	var sb strings.Builder

	instructions := "Use arrows or j/k to navigate, Esc to return to chat"
	view := v.list.View() + "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions)
	sb.WriteString(view)

	if v.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("\nError: "+v.err.Error()) + "\n")
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(sb.String())
}

// fetchAgentsCmd fetches agents asynchronously
func (v AgentView) fetchAgentsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		agents, err := v.agentService.ListAgents(ctx)
		if err != nil {
			return errMsg(err)
		}
		return agentsFetchedMsg{agents: agents}
	}
}
