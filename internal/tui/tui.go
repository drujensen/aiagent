package tui

// A simple program demonstrating the text area component from the Bubbles
// component library.

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

const gap = "\n\n"

type (
	errMsg error
)

type TUI struct {
	chatService services.ChatService
	activeChat  *entities.Chat
	viewport    viewport.Model
	textarea    textarea.Model
	senderStyle lipgloss.Style
	err         error
}

func NewTUI(chatService services.ChatService) TUI {
	ctx := context.Background()

	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 280
	ta.SetWidth(30)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(30, 5)
	vp.SetContent(`How can I help you today?`)

	activeChat, err := chatService.GetActiveChat(ctx)
	if err != nil {
		fmt.Println("Error getting active chat:", err)
		activeChat = nil
	}

	return TUI{
		chatService: chatService,
		activeChat:  activeChat,
		textarea:    ta,
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
	}
}

func (m TUI) Init() tea.Cmd {
	return textarea.Blink
}

func (m TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)

		messages := m.activeChat.Messages
		if len(messages) > 0 {
			var sb strings.Builder
			for _, message := range messages {
				if message.Role == "user" {
					sb.WriteString(m.senderStyle.Render("User: ") + message.Content + "\n")
				} else if message.Role == "assistant" {
					sb.WriteString(m.senderStyle.Render("Assistant: ") + message.Content + "\n")
				} else {
					sb.WriteString(m.senderStyle.Render("System: ") + message.Content + "\n")
				}
			}
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(sb.String()))
		}
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			m.textarea.Reset()
			m.viewport.GotoBottom()
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m TUI) View() string {
	return fmt.Sprintf(
		"%s%s%s",
		m.viewport.View(),
		gap,
		m.textarea.View(),
	)
}
