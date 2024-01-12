package handlers

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kocierik/lazyansible/src/utils"
)

func (h MainModel) Init() tea.Cmd {
	return nil
}

func (h MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return h, nil
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "tab":
			h.State++
		case "shift+tab":
			h.State--
		}

		switch h.State {
		case utils.ListModelState:
			h.ListView, cmd = h.ListView.Update(msg)
			cmds = append(cmds, cmd)
		case utils.PagerModelState:
			h.PagerView, cmd = h.PagerView.Update(msg)
			cmds = append(cmds, cmd)
		default:
			h.State = 0
		}
	}

	return h, tea.Batch(cmds...)
}

func (h MainModel) View() string {
	var view string
	SelectedStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("170"))
	paddingStyle := lipgloss.NewStyle().Padding(2)
	h.PagerView = h.PagerView.InitializePagerModel()

	switch h.State {
	case utils.ListModelState:
		view = lipgloss.JoinHorizontal(lipgloss.Top, SelectedStyle.Render(h.ListView.View()), paddingStyle.Render(h.PagerView.View()))
	case utils.PagerModelState:
		view = lipgloss.JoinHorizontal(lipgloss.Top, paddingStyle.Render(h.ListView.View()), SelectedStyle.Render(h.PagerView.View()))
	}

	s := lipgloss.JoinHorizontal(lipgloss.Top,
		view,
	)
	return s
}
