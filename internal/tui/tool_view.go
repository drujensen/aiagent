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

type ToolView struct {
	toolService services.ToolService
	list        list.Model
	width       int
	height      int
	err         error
}

func NewToolView(toolService services.ToolService) ToolView {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("6")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("7"))
	delegate.SetHeight(2)

	l := list.New([]list.Item{}, delegate, 100, 10)
	l.Title = "Available Tools"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(true)

	return ToolView{
		toolService: toolService,
		list:        l,
	}
}

func (v ToolView) Init() tea.Cmd {
	return v.fetchToolsCmd()
}

func (v ToolView) Update(msg tea.Msg) (ToolView, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = m.Width
		v.height = m.Height
		v.list.SetSize(m.Width-6, m.Height-6)
		return v, nil

	case tea.KeyMsg:
		switch m.String() {
		case "esc":
			return v, func() tea.Msg { return toolsCancelledMsg{} }
		}

	case toolsFetchedMsg:
		items := make([]list.Item, len(m.tools))
		for i, tool := range m.tools {
			items[i] = entities.ToolItem{Tool: *tool}
		}
		if len(items) == 0 {
			items = append(items, entities.ToolItem{Tool: entities.ToolData{Name: "No tools available", Description: ""}})
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

func (v ToolView) View() string {

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
	view := innerBorder.Render(v.list.View()) + "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(instructions)
	sb.WriteString(view)

	if v.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("\nError: "+v.err.Error()) + "\n")
	}

	// Wrap in outer border
	return outerStyle.Render(sb.String())
}

func (v ToolView) fetchToolsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		toolData, err := v.toolService.ListToolData(ctx)
		if err != nil {
			return errMsg(err)
		}
		return toolsFetchedMsg{tools: toolData}
	}
}
