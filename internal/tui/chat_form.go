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
	err         error
}

func NewChatForm(chatService services.ChatService, agentService services.AgentService) ChatForm {
	ctx := context.Background()
	agents, err := agentService.ListAgents(ctx)
	if err != nil {
		agents = []*entities.Agent{}
	}

	nameField := textinput.New()
	nameField.Placeholder = "Enter chat name"
	nameField.Focus()
	nameField.CharLimit = 50
	nameField.Width = 30

	agentItems := make([]list.Item, len(agents))
	for i, agent := range agents {
		agentItems[i] = agent
	}
	agentsList := list.New(agentItems, list.NewDefaultDelegate(), 30, 10)
	agentsList.Title = "Select an Agent"
	agentsList.SetShowStatusBar(false)

	return ChatForm{
		chatService: chatService,
		nameField:   nameField,
		agentsList:  agentsList,
		agents:      agents,
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
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "ctrl+c", "q":
			c.err = errors.New("chat creation cancelled")
			return c, tea.Batch(
				tea.Quit,
				func() tea.Msg { return startCreateChatMsg("") }, // Signal to return to chat view
			)
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
		}
	case tea.WindowSizeMsg:
		c.nameField.Width = m.Width - 2
		c.agentsList.SetSize(m.Width-2, m.Height-5)
	}

	var cmd tea.Cmd
	c.nameField, cmd = c.nameField.Update(msg)
	if cmd != nil {
		return c, cmd
	}

	c.agentsList, cmd = c.agentsList.Update(msg)
	if cmd != nil {
		return c, cmd
	}

	return c, nil
}

func (c ChatForm) View() string {
	var sb strings.Builder

	// Render the name input field
	sb.WriteString("Chat Name: ")
	sb.WriteString(c.nameField.View())
	sb.WriteString("\n\n")

	// Render the agents list
	sb.WriteString(c.agentsList.View())
	sb.WriteString("\n")

	// Render instructions
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Press Enter to create chat, Ctrl+C or q to cancel\n"))

	// Render error if any
	if c.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("Error: %s\n", c.err.Error())))
	}

	return sb.String()
}
