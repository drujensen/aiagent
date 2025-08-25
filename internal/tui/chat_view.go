package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type ChatView struct {
	chatService  services.ChatService
	agentService services.AgentService
	activeChat   *entities.Chat
	viewport     viewport.Model
	textarea     textarea.Model
	spinner      spinner.Model
	userStyle    lipgloss.Style
	asstStyle    lipgloss.Style
	systemStyle  lipgloss.Style
	err          error
	cancel       context.CancelFunc
	isProcessing bool
	startTime    time.Time
	focused      string // "textarea" or "viewport"
	width        int
	height       int
}

func NewChatView(chatService services.ChatService, agentService services.AgentService, activeChat *entities.Chat) ChatView {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.SetWidth(30)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(30, 5)

	us := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	as := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	ss := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	cv := ChatView{
		chatService:  chatService,
		agentService: agentService,
		activeChat:   activeChat,
		textarea:     ta,
		viewport:     vp,
		spinner:      s,
		userStyle:    us,
		asstStyle:    as,
		systemStyle:  ss,
		err:          nil,
		focused:      "textarea",
		width:        30,
		height:       5,
	}

	if activeChat != nil {
		cv.SetActiveChat(activeChat)
	}

	return cv
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
			sb.WriteString(c.systemStyle.Render("System: Tool Called") + "\n")
		}
	}
	c.viewport.SetContent(lipgloss.NewStyle().Width(c.viewport.Width).Render(sb.String()))
	c.viewport.GotoBottom()
}

func (c ChatView) Init() tea.Cmd {
	c.textarea.Focus()
	c.focused = "textarea"
	return textarea.Blink
}

func (c ChatView) Update(msg tea.Msg) (ChatView, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case tea.KeyMsg:
		if c.isProcessing {
			if m.Type == tea.KeyEsc {
				if c.cancel != nil {
					c.cancel()
					c.isProcessing = false
					c.err = fmt.Errorf("request cancelled")
					c.viewport.GotoBottom()
				}
				return c, nil
			}
			return c, nil
		}

		switch m.String() {
		case "ctrl+c":
			return c, tea.Quit
		case "esc":
			return c, nil
		case "ctrl+p":
			if c.focused == "textarea" {
				return c, func() tea.Msg { return startCommandsMsg{} }
			}
		case "enter":
			if c.focused == "textarea" {
				input := c.textarea.Value()
				if input == "" {
					c.err = fmt.Errorf("message cannot be empty")
					return c, nil
				}
				if c.activeChat == nil {
					c.err = fmt.Errorf("no active chat")
					return c, nil
				}
				message := &entities.Message{
					Content: input,
					Role:    "user",
				}
				c.textarea.Reset()
				c.err = nil
				ctx, cancel := context.WithCancel(context.Background())
				c.cancel = cancel
				c.isProcessing = true
				c.startTime = time.Now()
				return c, tea.Batch(sendMessageCmd(c.chatService, c.activeChat.ID, message, ctx), c.spinner.Tick)
			}
		case "tab", "shift+tab":
			if c.focused == "textarea" {
				c.focused = "viewport"
				c.textarea.Blur()
			} else {
				c.focused = "textarea"
				c.textarea.Focus()
				cmd := textarea.Blink
				cmds = append(cmds, cmd)
			}
			return c, tea.Batch(cmds...)
		case "j", "down":
			if c.focused == "viewport" {
				c.viewport.ScrollDown(1)
			} else {
				var cmd tea.Cmd
				c.textarea, cmd = c.textarea.Update(m)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		case "k", "up":
			if c.focused == "viewport" {
				c.viewport.ScrollUp(1)
			} else {
				var cmd tea.Cmd
				c.textarea, cmd = c.textarea.Update(m)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		default:
			if c.focused == "textarea" {
				var cmd tea.Cmd
				c.textarea, cmd = c.textarea.Update(m)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}

	case spinner.TickMsg:
		if c.isProcessing {
			var cmd tea.Cmd
			c.spinner, cmd = c.spinner.Update(m)
			return c, cmd
		}

	case updatedChatMsg:
		c.textarea.Reset()
		c.SetActiveChat(m)
		c.isProcessing = false
		c.cancel = nil
		return c, nil

	case errMsg:
		c.isProcessing = false
		c.cancel = nil
		c.err = m
		return c, nil

	case tea.WindowSizeMsg:
		c.width = m.Width
		c.height = m.Height
		innerWidth := c.width - 4
		innerHeight := c.height - 4

		c.viewport.Width = innerWidth
		// Subtract textarea height (3), instructions (1), possible error (1), and adjust for borders
		c.viewport.Height = innerHeight - 3 - 1 - 1 - 2

		c.textarea.SetWidth(innerWidth)

		if c.activeChat != nil {
			var sb strings.Builder
			for _, message := range c.activeChat.Messages {
				if message.Role == "user" {
					sb.WriteString(c.userStyle.Render("User: ") + message.Content + "\n")
				} else if message.Role == "assistant" {
					sb.WriteString(c.asstStyle.Render("Assistant: ") + message.Content + "\n")
				} else {
					sb.WriteString(c.systemStyle.Render("System: Tool Called") + "\n")
				}
			}
			c.viewport.SetContent(lipgloss.NewStyle().Width(c.viewport.Width).Render(sb.String()))
		}
		c.viewport.GotoBottom()
		return c, nil
	case tea.MouseMsg:
		viewportYStart := 1
		viewportBlockHeight := c.viewport.Height + 2
		viewportYEnd := viewportYStart + viewportBlockHeight
		if m.Y >= viewportYStart && m.Y < viewportYEnd {
			switch m.Type {
			case tea.MouseWheelUp:
				c.viewport.LineUp(3)
			case tea.MouseWheelDown:
				c.viewport.LineDown(3)
			}
		}
		return c, nil
	}

	return c, tea.Batch(cmds...)
}

func (c ChatView) View() string {
	// Define border styles
	focusedBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")) // Bright cyan for focused

	unfocusedBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")) // Dim gray for unfocused

	// Outer container style (Vim-like overall border)
	outerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("4")). // Blue for outer border
		Width(c.width - 2).
		Height(c.height - 2)

	var sb strings.Builder

	// Style viewport
	vpStyle := unfocusedBorder.Width(c.width - 4).Height(c.viewport.Height)
	if c.focused == "viewport" {
		vpStyle = focusedBorder.Width(c.width - 4).Height(c.viewport.Height)
	}

	// Check if activeChat is nil OR has no messages
	if c.activeChat == nil || len(c.activeChat.Messages) == 0 {
		c.viewport.SetContent(lipgloss.NewStyle().Width(c.viewport.Width).Render("How can I help you today?"))

		sb.WriteString(vpStyle.Render(c.viewport.View()))

	} else {
		// Display chat messages
		var chatContent strings.Builder
		for _, message := range c.activeChat.Messages {
			if message.Role == "user" {
				chatContent.WriteString(c.userStyle.Render("User: ") + message.Content + "\n")
			} else if message.Role == "assistant" {
				chatContent.WriteString(c.asstStyle.Render("Assistant: ") + message.Content + "\n")
			} else {
				chatContent.WriteString(c.systemStyle.Render("System: Tool Called") + "\n")
			}
		}

		//Set the viewport to contain the chat messages
		c.viewport.SetContent(lipgloss.NewStyle().Width(c.viewport.Width).Render(chatContent.String()))

		sb.WriteString(vpStyle.Render(c.viewport.View()))
	}

	// Style textarea
	taStyle := unfocusedBorder.Width(c.width - 4).Height(c.textarea.Height())
	if c.focused == "textarea" {
		taStyle = focusedBorder.Width(c.width - 4).Height(c.textarea.Height())
	}
	sb.WriteString(taStyle.Render(c.textarea.View()))

	if c.isProcessing {
		elapsed := time.Since(c.startTime).Round(time.Second)
		sb.WriteString("\n" + c.spinner.View() + fmt.Sprintf(" Thinking... (%ds)", int(elapsed.Seconds())))
	} else {
		instructions := "Press Ctrl+P for menu, Tab to switch focus, j/k to navigate, Ctrl+C to exit."
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions))
	}

	// Render error if any
	if c.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("\n%s", c.err.Error())))
	}

	// Wrap everything in the outer border
	return outerStyle.Render(sb.String())
}

func sendMessageCmd(cs services.ChatService, chatID string, msg *entities.Message, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
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
