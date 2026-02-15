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

// ModelWithPricing wraps a model with its pricing info for display
type ModelWithPricing struct {
	*entities.Model
	Pricing *entities.ModelPricing
}

func (m ModelWithPricing) FilterValue() string {
	return m.Model.FilterValue()
}

func (m ModelWithPricing) Title() string {
	return m.Model.Title()
}

func (m ModelWithPricing) Description() string {
	baseDesc := m.Model.Description()
	if m.Pricing != nil {
		priceDesc := fmt.Sprintf("$%.2f/$%.2f per 1M tokens", m.Pricing.InputPricePerMille, m.Pricing.OutputPricePerMille)
		return fmt.Sprintf("%s | %s", baseDesc, priceDesc)
	}
	return baseDesc
}

type ChatForm struct {
	chatService     services.ChatService
	agentService    services.AgentService
	modelService    services.ModelService
	filterService   *services.ModelFilterService
	providerService services.ProviderService
	nameField       textinput.Model
	agentsList      list.Model
	modelsList      list.Model
	agents          []*entities.Agent
	models          []ModelWithPricing
	chatName        string
	focused         string // "name", "agents", or "models"
	err             error
	width           int
	height          int
}

func NewChatForm(chatService services.ChatService, agentService services.AgentService, modelService services.ModelService, providerService services.ProviderService) ChatForm {
	ctx := context.Background()
	agents, err := agentService.ListAgents(ctx)
	if err != nil {
		fmt.Printf("Error listing agents: %v\n", err)
		agents = []*entities.Agent{}
	}
	rawModels, err := modelService.ListModels(ctx)
	if err != nil {
		fmt.Printf("Error listing models: %v\n", err)
		rawModels = []*entities.Model{}
	}

	// Filter models to only show chat-compatible ones
	filterService := services.NewModelFilterService()
	filteredModels := filterService.FilterChatCompatibleModels(rawModels)

	// Create models with pricing info
	models := make([]ModelWithPricing, len(filteredModels))
	for i, model := range filteredModels {
		// Look up pricing for this model
		var pricing *entities.ModelPricing
		if provider, err := providerService.GetProvider(ctx, model.ProviderID); err == nil {
			pricing = provider.GetModelPricing(model.ModelName)
		}
		models[i] = ModelWithPricing{
			Model:   model,
			Pricing: pricing,
		}
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
	agentsList.SetFilteringEnabled(true)
	agentsList.SetShowPagination(false)

	modelItems := make([]list.Item, len(models))
	for i, model := range models {
		modelItems[i] = model
	}
	modelsList := list.New(modelItems, delegate, 100, 10)
	modelsList.Title = "Select a Model"
	modelsList.SetShowStatusBar(true)
	modelsList.SetFilteringEnabled(true)
	modelsList.SetShowPagination(false)

	return ChatForm{
		chatService:     chatService,
		agentService:    agentService,
		modelService:    modelService,
		filterService:   filterService,
		providerService: providerService,
		nameField:       nameField,
		agentsList:      agentsList,
		modelsList:      modelsList,
		agents:          agents,
		models:          models,
		focused:         "name",
	}
}

func (c *ChatForm) SetChatName(name string) {
	c.chatName = name
	c.nameField.SetValue(name)
	// Reset form state for new chat session
	c.focused = "name"
	c.err = nil
	// Ensure name field has focus
	c.nameField.Focus()
	// Reset selections to first items
	if len(c.agents) > 0 {
		c.agentsList.Select(0)
	}
	if len(c.models) > 0 {
		c.modelsList.Select(0) // Default to first model (Grok 4.1 Fast)
	}
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
		c.nameField.Width = c.width - 6

		// Calculate available height for lists (accounting for fixed elements: name field + labels + instructions + borders)
		listHeight := c.height - 10
		if listHeight < 5 {
			listHeight = 5
		}

		listWidth := (c.width - 10) / 2 // Split width, accounting for borders and spacing
		if listWidth < 25 {
			listWidth = 25 // Minimum width for readable content
		}

		c.agentsList.SetSize(listWidth, listHeight)
		c.modelsList.SetSize(listWidth, listHeight)
		return c, nil

	case tea.KeyMsg:
		switch m.String() {
		case "esc":
			return c, func() tea.Msg { return canceledCreateChatMsg{} }
		case "tab", "shift+tab":
			switch c.focused {
			case "name":
				c.focused = "agents"
				c.nameField.Blur()
				c.agentsList.SetShowStatusBar(true)
				c.modelsList.SetShowStatusBar(false)
			case "agents":
				c.focused = "models"
				c.agentsList.SetShowStatusBar(false)
				c.modelsList.SetShowStatusBar(true)
			case "models":
				c.focused = "name"
				c.modelsList.SetShowStatusBar(false)
				c.nameField.Focus()
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
			if c.modelsList.SelectedItem() == nil {
				c.err = errors.New("no model selected")
				return c, nil
			}
			selectedAgent := c.agentsList.SelectedItem().(*entities.Agent)
			selectedModelWithPricing := c.modelsList.SelectedItem().(ModelWithPricing)
			return c, createChatCmd(c.chatService, c.nameField.Value(), selectedAgent.ID, selectedModelWithPricing.ID)
		}
	}

	switch c.focused {
	case "name":
		var cmd tea.Cmd
		c.nameField, cmd = c.nameField.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case "agents":
		var cmd tea.Cmd
		c.agentsList, cmd = c.agentsList.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case "models":
		var cmd tea.Cmd
		c.modelsList, cmd = c.modelsList.Update(msg)
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
		Height(c.height)

	var sb strings.Builder

	// Render the name input field with border
	nameFieldStyle := unfocusedBorder.Width(c.width - 4).Height(1)
	if c.focused == "name" {
		nameFieldStyle = focusedBorder.Width(c.width - 4).Height(1)
	}
	sb.WriteString("Chat Name:\n")
	if c.focused == "name" {
		sb.WriteString(nameFieldStyle.Render(c.nameField.View()))
	} else {
		sb.WriteString(nameFieldStyle.Render(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(c.nameField.Value())))
	}

	// Render the agents and models lists side by side
	listWidth := (c.width - 10) / 2
	if listWidth < 25 {
		listWidth = 25
	}

	// Agents list
	agentsStyle := unfocusedBorder.Width(listWidth).Height(c.agentsList.Height())
	if c.focused == "agents" {
		agentsStyle = focusedBorder.Width(listWidth).Height(c.agentsList.Height())
	}

	// Models list
	modelsStyle := unfocusedBorder.Width(listWidth).Height(c.modelsList.Height())
	if c.focused == "models" {
		modelsStyle = focusedBorder.Width(listWidth).Height(c.modelsList.Height())
	}

	// Create two-column layout
	agentsContent := "Agents:\n" + agentsStyle.Render(c.agentsList.View())
	modelsContent := "Models:\n" + modelsStyle.Render(c.modelsList.View())

	// Use lipgloss.JoinHorizontal to properly align the columns
	layout := lipgloss.JoinHorizontal(lipgloss.Top, agentsContent, modelsContent)
	sb.WriteString(layout)
	sb.WriteString("\n")

	// Render instructions
	instructions := "Press Enter to create chat, Tab to switch between name/agents/models, Esc to cancel"
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions + "\n"))

	// Render error if any
	if c.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("\nError: %s\n", c.err.Error())))
	}

	// Wrap in outer border
	return outerStyle.Render(sb.String())
}

func createChatCmd(cs services.ChatService, name, agentID, modelID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		newChat, err := cs.CreateChat(ctx, agentID, modelID, name)
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
