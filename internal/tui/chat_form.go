package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

var (
	ErrEmptyChatName   = errors.New("chat name cannot be empty")
	ErrNoAgentSelected = errors.New("no agent selected")
)

type ChatForm struct {
	chatService services.ChatService
	nameField   textinput.Model
	agentsList  list.Model
	agents      []*entities.Agent
	chatName    string
	focused     string // "name" or "list"
	err         error
	width       int
	height      int
}

func NewChatForm(chatService services.ChatService, agentService services.AgentService) ChatForm {
	ctx := context.Background()
	agents, err := agentService.ListAgents(ctx)
	if err != nil {
		fmt.Printf("Error listing agents: %v\n", err)
		agents = []*entities.Agent{}
	}

	// Initialize name field with wider width and explicit placeholder styling
	nameField := textinput.New()
	nameField.Placeholder = "Enter chat name"
	nameField.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	nameField.Focus()
	nameField.CharLimit = 50
	nameField.Width = 50 // Wide enough for placeholder and input

	// Initialize agent list with custom delegate and reasonable default size
	agentItems := make([]list.Item, len(agents))
	for i, agent := range agents {
		agentItems[i] = agent
	}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("6")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("7"))
	delegate.SetHeight(2)

	agentsList := list.New(agentItems, delegate, 100, 10)
	agentsList.Title = "Select an Agent"
	agentsList.SetShowStatusBar(false)
	agentsList.SetShowFilter(false)
	agentsList.SetShowPagination(true)

	return ChatForm{
		chatService: chatService,
		nameField:   nameField,
		agentsList:  agentsList,
		agents:      agents,
		focused:     "name",
	}
}

func (c *ChatForm) SetChatName(name string) {
	c.chatName = name
	c.nameField.SetValue(name)
}

func (c ChatForm) Init() tea.Cmd {
	c.nameField.Focus()
	return textinput.Blink
}

func (c ChatForm) Update(msg tea.Msg) (ChatForm, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = m.Width
		c.height = m.Height
		nameFieldWidth := m.Width - 4
		c.nameField.Width = nameFieldWidth
		c.agentsList.SetSize(m.Width-4, m.Height-4)
		return c, nil

	case tea.KeyMsg:
		switch m.String() {
		case "esc":
			return c, func() tea.Msg { return canceledCreateChatMsg{} }
		case "tab", "shift+tab":
			if c.focused == "name" {
				c.focused = "list"
				c.nameField.Blur()
				c.agentsList.SetShowStatusBar(true)
			} else {
				c.focused = "name"
				c.nameField.Focus()
				c.agentsList.SetShowStatusBar(false)
			}
			return c, nil
		case "enter":
			if c.nameField.Value() == "" {
				c.err = ErrEmptyChatName
				return c, nil
			}
			if c.agentsList.SelectedItem() == nil {
				c.err = ErrNoAgentSelected
				return c, nil
			}
			selectedAgent := c.agentsList.SelectedItem().(*entities.Agent)
			return c, createChatCmd(c.chatService, c.nameField.Value(), selectedAgent.ID)
		}
	}

	if c.focused == "name" {
		var cmd tea.Cmd
		c.nameField, cmd = c.nameField.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	} else {
		var cmd tea.Cmd
		c.agentsList, cmd = c.agentsList.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return c, tea.Batch(cmds...)
}

func (c ChatForm) View() string {

	if c.width == 0 || c.height == 0 {
		return ""
	}

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
		Height(c.height - 2).
		Padding(1)

	var sb strings.Builder

	// Render the name input field with border
	nameFieldStyle := unfocusedBorder.Copy().Width(c.width - 4)
	if c.focused == "name" {
		nameFieldStyle = focusedBorder.Copy().Width(c.width - 4)
	}
	sb.WriteString("Chat Name:\n")
	if c.focused == "name" {
		sb.WriteString(nameFieldStyle.Render(c.nameField.View()))
	} else {
		sb.WriteString(nameFieldStyle.Render(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(c.nameField.Value())))
	}
	sb.WriteString("\n\n")

	// Render the agents list with border
	listStyle := unfocusedBorder.Copy().Width(c.width - 4).Height(c.agentsList.Height())
	if c.focused == "list" {
		listStyle = focusedBorder.Copy().Width(c.width - 4).Height(c.agentsList.Height())
	}
	sb.WriteString(listStyle.Render(c.agentsList.View()))
	sb.WriteString("\n")

	// Render instructions
	instructions := "Press Enter to create chat, Tab to switch focus, Esc to cancel"
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions + "\n"))

	// Render error if any
	if c.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("\nError: %s\n", c.err.Error())))
	}

	// Wrap in outer border
	return outerStyle.Render(sb.String())
}

func createChatCmd(cs services.ChatService, name, agentID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		newChat, err := cs.CreateChat(ctx, agentID, name)
		if err != nil {
			return errMsg(err)
		}
		err = cs.SetActiveChat(ctx, newChat.ID)
		if err != nil {
			return errMsg(err)
		}
		return updatedChatMsg(newChat)
	}
}
