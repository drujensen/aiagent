package tui

import (
	"context"
	"strings"

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

func NewModelView(modelService services.ModelService, providerService services.ProviderService) ModelView {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("6")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("7"))
	delegate.SetHeight(2)

	l := list.New([]list.Item{}, delegate, 100, 10)
	l.Title = "Available Models"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(true)

	return ModelView{
		modelService:    modelService,
		providerService: providerService,
		filterService:   services.NewModelFilterService(),
		list:            l,
		mode:            "view",
	}
}

func (v ModelView) Init() tea.Cmd {
	return v.fetchModelsCmd()
}

func (v ModelView) Update(msg tea.Msg) (ModelView, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = m.Width
		v.height = m.Height
		v.list.SetSize(v.width-6, v.height-6)
		return v, nil

	case tea.KeyMsg:
		switch m.String() {
		case "esc":
			return v, func() tea.Msg { return modelsCancelledMsg{} }
		case "enter":
			if v.mode != "switch" {
				return v, nil
			}
			if selected, ok := v.list.SelectedItem().(ModelWithPricing); ok {
				return v, func() tea.Msg { return modelSelectedMsg{modelID: selected.ID} }
			}
		}

	case modelsFetchedMsg:
		items := m.models
		if len(items) == 0 {
			items = append(items, ModelWithPricing{
				Model:   &entities.Model{Name: "No models available", ModelName: ""},
				Pricing: nil,
			})
		}
		v.list.SetItems(items)
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
	instructions := "Use arrows or j/k to navigate, Esc to return to chat"
	if v.mode == "switch" {
		instructions += ", Enter to switch model"
	}
	view := innerBorder.Render(v.list.View()) + "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions)
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

		// Create models with pricing info
		modelsWithPricing := make([]ModelWithPricing, len(filteredModels))
		for i, model := range filteredModels {
			// Look up pricing for this model
			var pricing *entities.ModelPricing
			if provider, err := v.providerService.GetProvider(ctx, model.ProviderID); err == nil {
				pricing = provider.GetModelPricing(model.ModelName)
			}
			modelsWithPricing[i] = ModelWithPricing{
				Model:   model,
				Pricing: pricing,
			}
		}

		return modelsFetchedMsg{models: func() []list.Item {
			items := make([]list.Item, len(modelsWithPricing))
			for i, model := range modelsWithPricing {
				items[i] = model
			}
			return items
		}()}
	}
}
