package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type ModelView struct {
	modelService services.ModelService
	list         list.Model
	width        int
	height       int
	err          error
	mode         string // "view" or "switch"
}

func NewModelView(modelService services.ModelService) ModelView {
	ctx := context.Background()
	models, err := modelService.ListModels(ctx)
	if err != nil {
		models = []*entities.Model{}
	}

	modelItems := make([]list.Item, len(models))
	for i, model := range models {
		modelItems[i] = model
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("6")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("7"))
	delegate.SetHeight(2)

	l := list.New(modelItems, delegate, 100, 10)
	l.Title = "Available Models"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(true)

	return ModelView{
		modelService: modelService,
		list:         l,
		mode:         "view",
	}
}

func (v ModelView) Init() tea.Cmd {
	return nil
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
			if selected, ok := v.list.SelectedItem().(*entities.Model); ok {
				return v, func() tea.Msg { return modelSelectedMsg{modelID: selected.ID} }
			}
		}

	case errMsg:
		v.err = m
		return v, nil
	}

	var cmd tea.Cmd
	v.list, cmd = v.list.Update(msg)
	return v, cmd
}

func (v ModelView) View() string {
	if v.err != nil {
		return v.err.Error()
	}
	return v.list.View()
}
