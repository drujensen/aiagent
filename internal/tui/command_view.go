package tui

import (
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
		CommandItem{name: "new", desc: "Create a new chat (Ctrl+N)"},
		CommandItem{name: "history", desc: "View chat history (Ctrl+H)"},
		CommandItem{name: "agents", desc: "View available agents (Ctrl+A)"},
		CommandItem{name: "tools", desc: "View available tools (Ctrl+T)"},
		CommandItem{name: "models", desc: "View available models (Ctrl+M)"},
		CommandItem{name: "usage", desc: "Show usage statistics (Ctrl+U)"},
		CommandItem{name: "exit", desc: "Exit the application (Ctrl+X)"},
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
		list:   l,
		width:  80, // Default width
		height: 20, // Default height
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
		// Make list fill available height minus borders and instructions
		// Content: list + \n + instructions = listHeight + 1
		// Inner border: +2 (top/bottom)
		// Outer border: +2 (top/bottom)
		// Total: listHeight + 5 = m.height
		listHeight := m.height - 5
		if listHeight < 6 {
			listHeight = 6 // Minimum height
		}
		m.list.SetSize(m.width-6, listHeight)
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
		return "Loading..."
	}

	// Create the content first
	instructions := "Type to filter, arrows/j/k to navigate, Enter to execute, Esc to cancel"
	instructionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Width(m.width - 6) // Fit within inner border
	content := m.list.View() + "\n" + instructionStyle.Render(instructions)

	// Apply inner border
	innerBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")).
		Width(m.width - 4) // Account for outer border width

	borderedContent := innerBorder.Render(content)

	// Apply outer border with full available space
	outerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("4"))

	return outerStyle.Render(borderedContent)
}
