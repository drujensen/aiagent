package tui

import (
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

	if h.width == 0 || h.height == 0 {
		return ""
	}

	// Help content (concise)
	helpText := `Commands:
	/new     - New chat
	/history - Chat history
	/agents   - Available agents
	/tools    - Available tools
	/usage    - Usage stats
	/help     - This help
	/exit     - Exit app

	Shortcuts:
	Ctrl+A - Switch agent
	Ctrl+G - Switch model
	Ctrl+N - New chat
	Ctrl+L - Toggle line numbers

	Tips:
	• Type / to filter lists
	• Use arrows/j/k to navigate
	• Ctrl+C to quit`

	// Create the content first
	instructions := "\nPress Esc to close"
	content := helpText + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions)

	// Apply inner border
	innerBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")).
		Align(lipgloss.Center)

	borderedContent := innerBorder.Render(content)

	// Apply outer border with full available space
	outerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("4")).
		Align(lipgloss.Center)

	return outerStyle.Render(borderedContent)
}
