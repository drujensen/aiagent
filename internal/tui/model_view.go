package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	key "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type ModelView struct {
	modelService    services.ModelService
	providerService services.ProviderService
	filterService   *services.ModelFilterService
	list            list.Model
	width           int
	height          int
	err             error
	mode            string // "view" or "switch"
}

// ModelListItem represents either a provider header or a model in the list
type ModelListItem struct {
	IsHeader     bool
	ProviderName string
	Model        *entities.Model
	Pricing      *entities.ModelPricing
}

func (m ModelListItem) FilterValue() string {
	if m.IsHeader {
		return m.ProviderName
	}
	return fmt.Sprintf("%s %s", m.Model.Name, m.ProviderName)
}

func (m ModelListItem) Title() string {
	if m.IsHeader {
		return "▶ " + m.ProviderName
	}
	return "  " + m.Model.Name
}

func (m ModelListItem) Description() string {
	if m.IsHeader {
		return "" // Headers don't need descriptions
	}
	if m.Pricing != nil {
		return fmt.Sprintf("%s - input: $%.2f output: $%.2f",
			m.ProviderName,
			m.Pricing.InputPricePerMille,
			m.Pricing.OutputPricePerMille)
	}
	// When no pricing available, show context window if available
	if m.Model.ContextWindow != nil {
		return fmt.Sprintf("Context: %d tokens", *m.Model.ContextWindow)
	}
	return ""
}

func NewModelView(modelService services.ModelService, providerService services.ProviderService) ModelView {
	return NewModelViewWithMode(modelService, providerService, "view")
}

func NewModelViewWithMode(modelService services.ModelService, providerService services.ProviderService, mode string) ModelView {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("6")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("7"))

	// Make headers visually distinct when selected
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(lipgloss.Color("15")) // White for normal items
	delegate.SetHeight(2)                                                                      // Standard height for title + description

	l := list.New([]list.Item{}, delegate, 100, 10)
	l.Title = "Available Models"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(true)

	// Disable the default quit key to prevent quitting the app from the model view
	l.KeyMap.Quit = key.NewBinding(key.WithDisabled())
	l.SetShowHelp(true)

	return ModelView{
		modelService:    modelService,
		providerService: providerService,
		filterService:   services.NewModelFilterService(),
		list:            l,
		mode:            mode,
	}
}

func (v *ModelView) SetMode(mode string) {
	v.mode = mode
}

func (v ModelView) Init() tea.Cmd {
	return v.fetchModelsCmd()
}

func (v ModelView) Update(msg tea.Msg) (ModelView, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = m.Width
		v.height = m.Height
		// Reserve space for borders and instructions
		listHeight := v.height - 8
		if listHeight < 10 {
			listHeight = 10 // Minimum height
		}
		v.list.SetSize(v.width-6, listHeight)
		return v, nil

	case tea.KeyMsg:
		// Let the list handle filtering keys first
		var cmd tea.Cmd
		v.list, cmd = v.list.Update(msg)

		// Then handle our custom keys
		switch m.String() {
		case "esc":
			// Only close if not actively filtering
			if v.list.FilterState() != list.Filtering {
				v.list.SetFilterText("")
				return v, func() tea.Msg { return modelsCancelledMsg{} }
			}
		case "q":
			v.list.SetFilterText("")
			return v, func() tea.Msg { return modelsCancelledMsg{} }
		case "enter":
			if v.mode != "switch" {
				return v, nil
			}
			if selected, ok := v.list.SelectedItem().(ModelListItem); ok && !selected.IsHeader {
				v.list.SetFilterText("")
				return v, func() tea.Msg { return modelSelectedMsg{modelID: selected.Model.ID} }
			}
		}
		return v, cmd

	case modelsFetchedMsg:
		items := m.models
		if len(items) == 0 {
			items = append(items, ModelListItem{
				IsHeader: false,
				Model:    &entities.Model{Name: "No models available", ModelName: ""},
				Pricing:  nil,
			})
		}
		v.list.SetFilteringEnabled(false)
		v.list.SetItems(items)
		v.list.SetFilteringEnabled(true)
		v.err = nil
		return v, nil

	case errMsg:
		v.err = m
		return v, nil
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v ModelView) View() string {

	if v.width == 0 || v.height == 0 {
		return ""
	}

	// Outer container style (Vim-like overall border)
	outerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("4")). // Blue for outer border
		Width(v.width - 2).
		Height(v.height - 2)

	// Inner border for list (always "focused" since single component)
	innerBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")). // Bright cyan
		Width(v.list.Width()).
		Height(v.list.Height())

	var sb strings.Builder
	view := innerBorder.Render(v.list.View())
	sb.WriteString(view)

	if v.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("\nError: "+v.err.Error()) + "\n")
	}

	// Wrap in outer border
	return outerStyle.Render(sb.String())
}

// fetchModelsCmd fetches models asynchronously
func (v ModelView) fetchModelsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		allModels, err := v.modelService.ListModels(ctx)
		if err != nil {
			return errMsg(err)
		}

		// Filter to only show chat-compatible models
		filteredModels := v.filterService.FilterChatCompatibleModels(allModels)

		// Group models by provider
		modelsByProvider := make(map[string][]*entities.Model)
		providerNames := make(map[string]string)

		for _, model := range filteredModels {
			if provider, err := v.providerService.GetProvider(ctx, model.ProviderID); err == nil {
				providerNames[model.ProviderID] = provider.Name
				modelsByProvider[model.ProviderID] = append(modelsByProvider[model.ProviderID], model)
			}
		}

		// Sort providers for consistent ordering
		var providerIDs []string
		for providerID := range modelsByProvider {
			providerIDs = append(providerIDs, providerID)
		}
		sort.Strings(providerIDs)

		// Create grouped items
		var groupedItems []list.Item
		for _, providerID := range providerIDs {
			providerName := providerNames[providerID]
			models := modelsByProvider[providerID]

			// Add provider header
			groupedItems = append(groupedItems, ModelListItem{
				IsHeader:     true,
				ProviderName: providerName,
			})

			// Add models for this provider
			for _, model := range models {
				var pricing *entities.ModelPricing
				if provider, err := v.providerService.GetProvider(ctx, model.ProviderID); err == nil {
					pricing = provider.GetModelPricing(model.ModelName)
					// Debug: if no pricing found, show provider info instead
					if pricing == nil {
						// For models without pricing, show some info
					}
				}
				groupedItems = append(groupedItems, ModelListItem{
					IsHeader:     false,
					ProviderName: providerName,
					Model:        model,
					Pricing:      pricing,
				})
			}
		}

		return modelsFetchedMsg{models: groupedItems}
	}
}
