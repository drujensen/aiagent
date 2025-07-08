package tui

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

type ChatView struct {
	chatService services.ChatService
	activeChat  *entities.Chat
	viewport    viewport.Model
	textarea    textarea.Model
	userStyle   lipgloss.Style
	asstStyle   lipgloss.Style
	systemStyle lipgloss.Style
	err         error
}

func NewChatView(chatService services.ChatService) ChatView {
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

	us := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	as := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	ss := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	return ChatView{
		chatService: chatService,
		activeChat:  nil,
		textarea:    ta,
		viewport:    vp,
		userStyle:   us,
		asstStyle:   as,
		systemStyle: ss,
		err:         nil,
	}
}

func (c *ChatView) SetActiveChat(chat *entities.Chat) {
	c.activeChat = chat
	var sb strings.Builder
	for _, message := range chat.Messages {
		if message.Role == "user" {
			sb.WriteString(c.userStyle.Render("User: ") + message.Content + "\n")
		} else if message.Role == "assistant" {
			sb.WriteString(c.asstStyle.Render("Assistant: ") + message.Content + "\n")
		} else {
			sb.WriteString(c.systemStyle.Render("System: ") + message.Content + "\n")
		}
	}
	c.viewport.SetContent(lipgloss.NewStyle().Width(c.viewport.Width).Render(sb.String()))
	c.viewport.GotoBottom()
}

func (c ChatView) Init() tea.Cmd {
	c.textarea.Focus()
	return textarea.Blink
}

func (c ChatView) Update(msg tea.Msg) (ChatView, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "enter":
			if c.textarea.Value() == "" {
				c.err = fmt.Errorf("message cannot be empty")
				return c, nil
			}
			if c.activeChat == nil {
				c.err = fmt.Errorf("no active chat")
				return c, nil
			}
			// Create a new Message
			msg := &entities.Message{
				Content: c.textarea.Value(),
				Role:    "user",
			}
			// Send the message to the chat services
			return c, sendMessageCmd(c.chatService, c.activeChat.ID, msg)
		case "up", "down":
			c.textarea, _ = c.textarea.Update(msg)
		case "left", "right":
			c.viewport, _ = c.viewport.Update(msg)
		}

	case updatedChatMsg:
		c.SetActiveChat(m)

	case tea.WindowSizeMsg:
		c.viewport.Width = m.Width
		c.textarea.SetWidth(m.Width)
		c.viewport.Height = m.Height - c.textarea.Height() - lipgloss.Height("┃ ")
		if c.activeChat != nil {
			var sb strings.Builder
			for _, message := range c.activeChat.Messages {
				if message.Role == "user" {
					sb.WriteString(c.userStyle.Render("User: ") + message.Content + "\n")
				} else if message.Role == "assistant" {
					sb.WriteString(c.asstStyle.Render("Assistant: ") + message.Content + "\n")
				} else {
					sb.WriteString(c.systemStyle.Render("System: ") + message.Content + "\n")
				}
			}
			c.viewport.SetContent(lipgloss.NewStyle().Width(c.viewport.Width).Render(sb.String()))
		}
		c.viewport.GotoBottom()

	case tea.MouseMsg:
		// Handle mouse events if needed
		if m.Type == tea.MouseWheelUp || m.Type == tea.MouseWheelDown {
			c.viewport, _ = c.viewport.Update(msg) // Update viewport for mouse wheel events
		}
	default:
		var taCmd tea.Cmd
		var vpCmd tea.Cmd
		c.textarea, taCmd = c.textarea.Update(msg)
		c.viewport, vpCmd = c.viewport.Update(msg)
		if taCmd != nil {
			return c, taCmd
		}
		if vpCmd != nil {
			return c, vpCmd
		}
	}

	return c, nil
}

func (c ChatView) View() string {
	var sb strings.Builder
	sb.WriteString(c.viewport.View())
	sb.WriteString(gap)
	sb.WriteString(c.textarea.View())

	// Render error if any
	if c.err != nil {
		sb.WriteString(fmt.Sprintf("Error: %s\n", c.err.Error()))
	}

	return sb.String()
}

func sendMessageCmd(cs services.ChatService, chatID string, msg *entities.Message) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_, err := cs.SendMessage(ctx, chatID, msg)
		if err != nil {
			return errMsg(err)
		}
		updatedChat, err := cs.GetChat(ctx, chatID)
		if err != nil {
			return errMsg(err)
		}
		return updatedChatMsg(updatedChat)
	}
}
