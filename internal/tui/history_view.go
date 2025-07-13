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
		// Handle error gracefully (print to console for now; could log in production)
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
		listHeight := m.Height - 3
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
	instructions := "Use arrows or j/k to navigate, Enter to select, Esc to cancel"
	view := h.list.View() + "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions)
	return lipgloss.NewStyle().Padding(1, 2).Render(view)
}
