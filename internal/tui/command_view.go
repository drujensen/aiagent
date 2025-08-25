package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CommandItem struct {
	name string
	desc string
}

func (i CommandItem) Title() string       { return "/" + i.name }
func (i CommandItem) Description() string { return i.desc }
func (i CommandItem) FilterValue() string { return i.name }

type CommandMenu struct {
	list   list.Model
	width  int
	height int
}

func NewCommandMenu() CommandMenu {
	items := []list.Item{
		CommandItem{name: "new", desc: "Create a new chat"},
		CommandItem{name: "history", desc: "View chat history"},
		CommandItem{name: "agents", desc: "View available agents"},
		CommandItem{name: "tools", desc: "View available tools"},
		CommandItem{name: "usage", desc: "Show usage statistics"},
		CommandItem{name: "help", desc: "Show help screen"},
		CommandItem{name: "exit", desc: "Exit the application"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("6")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("7"))
	delegate.SetHeight(2)

	l := list.New(items, delegate, 100, 10)
	l.Title = "Commands"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(false)

	return CommandMenu{
		list: l,
	}
}

func (m CommandMenu) Init() tea.Cmd {
	return nil
}

func (m CommandMenu) Update(msg tea.Msg) (CommandMenu, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(m.width-6, m.height-6)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return commandsCancelledMsg{} }
		case "enter":
			if selected, ok := m.list.SelectedItem().(CommandItem); ok {
				return m, func() tea.Msg { return executeCommandMsg{command: selected.name} }
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m CommandMenu) View() string {

	if m.width == 0 || m.height == 0 {
		return ""
	}

	outerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("4")).
		Width(m.width - 2).
		Height(m.height - 2)

	innerBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")).
		Width(m.list.Width()).
		Height(m.list.Height())

	var sb strings.Builder
	instructions := "Type to filter, arrows/j/k to navigate, Enter to execute, Esc to cancel"
	view := innerBorder.Render(m.list.View()) + "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions)
	sb.WriteString(view)

	return outerStyle.Render(sb.String())
}
