package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/drujensen/aiagent/internal/domain/entities"
)

var (
	ErrEmptyChatName   = errors.New("chat name cannot be empty")
	ErrNoAgentSelected = errors.New("no agent selected")
)

type ChatForm struct {
	nameField  textinput.Model
	agentsList list.Model
	agents     []*entities.Agent
	err        error
}

func NewChatForm(agents []*entities.Agent) ChatForm {
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
		nameField:  nameField,
		agentsList: agentsList,
		agents:     agents,
	}
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
			return c, tea.Quit
		case "enter":
			if c.nameField.Value() == "" {
				c.err = ErrEmptyChatName
				return c, nil
			}
			if c.agentsList.SelectedItem() == nil {
				c.err = ErrNoAgentSelected
				return c, nil
			}
			return c, tea.Quit // Proceed with chat creation
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

	// Render error if any
	if c.err != nil {
		sb.WriteString(fmt.Sprintf("Error: %s\n", c.err.Error()))
	}

	return sb.String()
}
