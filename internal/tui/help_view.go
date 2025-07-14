package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type HelpView struct {
	width  int
	height int
}

func NewHelpView() HelpView {
	return HelpView{}
}

func (h HelpView) Init() tea.Cmd {
	return nil
}

func (h HelpView) Update(msg tea.Msg) (HelpView, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = m.Width
		h.height = m.Height
		return h, nil

	case tea.KeyMsg:
		switch m.String() {
		case "esc":
			return h, func() tea.Msg { return helpCancelledMsg{} }
		}
	}

	return h, nil
}

func (h HelpView) View() string {
	// Help content (can be expanded as needed)
	helpText := `AI Agent TUI Help

Commands:
/new <name>   - Start a new chat
/history      - View and select from history
/agents       - View available agents
/tools        - View available tools
/usage        - Show usage statistics
/help         - Show this help screen
/exit         - Exit the application

Usage Tips:
- Type messages and press Enter to send.
- Use arrows or mouse wheel to scroll history.
- Press Ctrl+C to force quit.

Press Esc to close.`

	// Outer container style (Vim-like overall border)
	outerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("4")). // Blue for outer border
		Width(h.width-2).
		Height(h.height-2).
		Padding(1, 2).
		Align(lipgloss.Center)

	// Inner style for text (focused since single component)
	innerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")). // Bright cyan
		Width(h.width - 4).
		Height(h.height - 4).
		Padding(1)

	var sb strings.Builder
	sb.WriteString(innerStyle.Render(helpText))

	// Wrap in outer border
	return outerStyle.Render(sb.String())
}
