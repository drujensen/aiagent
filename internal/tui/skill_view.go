package tui

import (
	"context"
	"strings"

	key "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"
)

type SkillView struct {
	skillService services.SkillService
	list         list.Model
	width        int
	height       int
	err          error
}

func NewSkillView(skillService services.SkillService) SkillView {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("6")).Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("7"))
	delegate.SetHeight(2)

	l := list.New([]list.Item{}, delegate, 100, 10)
	l.Title = "Available Skills"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(true)
	l.KeyMap.Quit = key.NewBinding(key.WithDisabled())
	l.SetShowHelp(true)

	return SkillView{
		skillService: skillService,
		list:         l,
	}
}

func (v SkillView) Init() tea.Cmd {
	return v.fetchSkillsCmd()
}

func (v SkillView) Update(msg tea.Msg) (SkillView, tea.Cmd) {
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
		var cmd tea.Cmd
		v.list, cmd = v.list.Update(msg)

		switch m.String() {
		case "esc":
			if v.list.FilterState() != list.Filtering {
				v.list.SetFilterText("")
				return v, func() tea.Msg { return skillsCancelledMsg{} }
			}
		case "q":
			v.list.SetFilterText("")
			return v, func() tea.Msg { return skillsCancelledMsg{} }
		case "enter":
			if selected, ok := v.list.SelectedItem().(*entities.Skill); ok && selected.Name != "No skills found" {
				v.list.SetFilterText("")
				return v, func() tea.Msg { return skillSelectedMsg{skillName: selected.Name} }
			}
		}
		return v, cmd

	case skillsFetchedMsg:
		items := make([]list.Item, len(m.skills))
		for i, skill := range m.skills {
			items[i] = skill
		}
		if len(items) == 0 {
			items = append(items, &entities.Skill{Name: "No skills found"})
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

func (v SkillView) View() string {

	if v.width == 0 {
		v.width = 80
	}
	if v.height == 0 {
		v.height = 24
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
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("Error: "+v.err.Error()))
	}

	// Wrap in outer border
	return outerStyle.Render(sb.String())
}

// fetchSkillsCmd fetches skills asynchronously
func (v SkillView) fetchSkillsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		skills, err := v.skillService.ListSkills(ctx)
		if err != nil {
			return errMsg(err)
		}
		return skillsFetchedMsg{skills: skills}
	}
}
