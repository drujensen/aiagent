package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type HistoryView struct {
	chatService services.ChatService
	list        list.Model
	width       int
	height      int
}

func NewHistoryView(chatService services.ChatService) HistoryView {
	ctx := context.Background()
	chats, err := chatService.ListChats(ctx)
	if err != nil {
		fmt.Printf("Error listing chats: %v\n", err)
		chats = []*entities.Chat{}
	}

	items := make([]list.Item, len(chats))
	for i, chat := range chats {
		items[i] = chat
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("6")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("7"))
	delegate.SetHeight(2)

	l := list.New(items, delegate, 100, 10)
	l.Title = "Chat History"
	l.SetShowStatusBar(false)
	l.SetShowFilter(false)
	l.SetShowPagination(len(chats) > 10)

	return HistoryView{
		chatService: chatService,
		list:        l,
	}
}

func (h HistoryView) Init() tea.Cmd {
	return nil
}

func (h HistoryView) Update(msg tea.Msg) (HistoryView, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = m.Width
		h.height = m.Height
		// Adjust for borders/padding
		listHeight := m.Height - 4
		h.list.SetSize(m.Width-4, listHeight)
		return h, nil

	case tea.KeyMsg:
		switch m.String() {
		case "esc":
			return h, func() tea.Msg { return historyCancelledMsg{} }
		case "enter":
			if selected := h.list.SelectedItem(); selected != nil {
				chat := selected.(*entities.Chat)
				return h, func() tea.Msg { return historySelectedMsg{chatID: chat.ID} }
			}
		}
	}

	var cmd tea.Cmd
	h.list, cmd = h.list.Update(msg)
	return h, cmd
}

func (h HistoryView) View() string {
	// Outer container style (Vim-like overall border)
	outerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("4")). // Blue for outer border
		Width(h.width - 2).
		Height(h.height - 2).
		Padding(1)

	// Inner border for list (always "focused" since single component)
	innerBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")). // Bright cyan
		Width(h.list.Width()).
		Height(h.list.Height())

	instructions := "Use arrows or j/k to navigate, Enter to select, Esc to cancel"
	view := innerBorder.Render(h.list.View()) + "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions)

	// Wrap in outer border
	return outerStyle.Render(view)
}
