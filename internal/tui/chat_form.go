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
	agentsList.SetShowPagination(len(agents) > 10)

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
		listHeight := m.Height - 3
		c.agentsList.SetSize(m.Width-4, listHeight)
		return c, nil

	case tea.KeyMsg:
		switch m.String() {
		case "esc":
			c.err = errors.New("chat creation cancelled")
			return c, func() tea.Msg { return startCreateChatMsg("") }
		case "ctrl+c", "q":
			c.err = errors.New("chat creation cancelled")
			return c, tea.Batch(
				tea.Quit,
				func() tea.Msg { return startCreateChatMsg("") },
			)
		case "tab":
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
			return c, func() tea.Msg {
				return chatFormSubmittedMsg{
					name:    c.nameField.Value(),
					agentID: selectedAgent.ID,
				}
			}
		case "j", "k":
			if c.focused == "list" {
				var cmd tea.Cmd
				c.agentsList, cmd = c.agentsList.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
				return c, tea.Batch(cmds...)
			}
			// If name field is focused, pass to textinput
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
	var sb strings.Builder

	// Render the name input field
	nameFieldStyle := lipgloss.NewStyle().Width(c.width - 4)
	sb.WriteString("Chat Name: ")
	if c.focused == "name" {
		sb.WriteString(nameFieldStyle.Render(c.nameField.View()))
	} else {
		sb.WriteString(nameFieldStyle.Render(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(c.nameField.Value())))
	}
	sb.WriteString("\n\n")

	// Render the agents list
	listStyle := lipgloss.NewStyle().Width(c.width - 4)
	sb.WriteString(listStyle.Render(c.agentsList.View()))
	sb.WriteString("\n")

	// Render instructions
	instructions := "Press Enter to create chat, Tab to switch focus, Esc to cancel, Ctrl+C or q to quit"
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions + "\n"))

	// Render error if any
	if c.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("Error: %s\n", c.err.Error())))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(sb.String())
}
