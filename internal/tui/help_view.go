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

	// Style the help text in a box
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(h.width - 4).
		Height(h.height - 4).
		Align(lipgloss.Center)

	// Instructions at the bottom
	instructions := "\nPress Esc to close"
	instructionsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	var sb strings.Builder
	sb.WriteString(style.Render(helpText))
	sb.WriteString(instructionsStyle.Render(instructions))

	return lipgloss.NewStyle().Padding(1, 2).Render(sb.String())
}
