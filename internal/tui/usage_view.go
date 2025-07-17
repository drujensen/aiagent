package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type UsageView struct {
	chatService  services.ChatService
	agentService services.AgentService
	width        int
	height       int
	usageInfo    string // Pre-formatted usage string
	err          error
}

func NewUsageView(chatService services.ChatService, agentService services.AgentService) UsageView {
	return UsageView{
		chatService:  chatService,
		agentService: agentService,
	}
}

func (u UsageView) Init() tea.Cmd {
	return u.fetchUsageCmd()
}

func (u UsageView) Update(msg tea.Msg) (UsageView, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		u.width = m.Width
		u.height = m.Height
		return u, nil

	case tea.KeyMsg:
		switch m.String() {
		case "esc":
			return u, func() tea.Msg { return usageCancelledMsg{} }
		}

	case updatedUsageMsg:
		u.usageInfo = m.info
		u.err = nil
		return u, nil

	case errMsg:
		u.err = m
		return u, nil
	}

	return u, nil
}

func (u UsageView) View() string {

	if u.width == 0 || u.height == 0 {
		return ""
	}

	// Outer container style (Vim-like overall border)
	outerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("4")). // Blue for outer border
		Width(u.width - 2).
		Height(u.height - 2)

	// Inner style for content (focused since single component)
	innerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")). // Bright cyan
		Width(u.width - 4)

	var sb strings.Builder

	if u.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("Error: %s\n", u.err.Error())))
	} else if u.usageInfo != "" {
		sb.WriteString(innerStyle.Render(u.usageInfo))
	} else {
		sb.WriteString(innerStyle.Render("Loading usage information..."))
	}

	// Instructions
	instructions := "\nPress Esc to close"
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions))

	// Wrap in outer border
	return outerStyle.Render(sb.String())
}

// fetchUsageCmd fetches and formats usage info asynchronously
func (u UsageView) fetchUsageCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		activeChat, err := u.chatService.GetActiveChat(ctx)
		if err != nil {
			return errMsg(err)
		}
		agent, err := u.agentService.GetAgent(ctx, activeChat.AgentID)
		if err != nil {
			return errMsg(err)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Provider: %s\n", agent.ProviderType))
		sb.WriteString(fmt.Sprintf("Model: %s\n", agent.Model))
		sb.WriteString(fmt.Sprintf("Prompt Tokens: %d\n", activeChat.Usage.TotalPromptTokens))
		sb.WriteString(fmt.Sprintf("Completion Tokens: %d\n", activeChat.Usage.TotalCompletionTokens))
		sb.WriteString(fmt.Sprintf("Total Tokens: %d\n", activeChat.Usage.TotalTokens))
		sb.WriteString(fmt.Sprintf("Total Cost: $%.2f\n", activeChat.Usage.TotalCost))

		return updatedUsageMsg{info: sb.String()}
	}
}
