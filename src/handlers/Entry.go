package handlers

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kocierik/lazyansible/src/utils"
)

func (h MainModel) Init() tea.Cmd {
	return h.FilePickerView.Init()
}

func (h MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.Height = msg.Height
		h.Width = msg.Width
		h.PagerView, cmd = h.PagerView.Update(msg)
		h.FilePickerView, cmd = h.FilePickerView.Update(msg)
		return h, cmd
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "tab":
			h.State++
		case "shift+tab":
			h.State--
		}

		switch h.State {
		case utils.PagerModelState:
			h.PagerView, cmd = h.PagerView.Update(msg)
			cmds = append(cmds, cmd)
		case utils.FilePickerState:
			h.FilePickerView, cmd = h.FilePickerView.Update(msg)
			cmds = append(cmds, cmd)
		case utils.ListHostState:
			h.ListHostView, cmd = h.ListHostView.Update(msg)
			cmds = append(cmds, cmd)
		default:
			h.State = 0
		}
	}

	return h, tea.Batch(cmds...)
}

func (h MainModel) View() string {
	var view string
	selectedStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("170")).Height(25)
	paddingStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Height(25)
	widthStylePager := lipgloss.NewStyle().Width(h.Width / 2)
	widthStyleComponent := lipgloss.NewStyle().Width(h.Width / 4)

	switch h.State {
	case utils.FilePickerState:
		view = lipgloss.JoinHorizontal(lipgloss.Top,
			widthStyleComponent.Render(selectedStyle.Render(h.FilePickerView.View())),
			widthStylePager.Render(paddingStyle.Render(h.PagerView.View())),
			widthStyleComponent.Render(paddingStyle.Render(h.ListHostView.View())))
	case utils.PagerModelState:
		view = lipgloss.JoinHorizontal(lipgloss.Top,
			widthStyleComponent.Render(paddingStyle.Render(h.FilePickerView.View())),
			widthStylePager.Render(selectedStyle.Render(h.PagerView.View())),
			widthStyleComponent.Render(paddingStyle.Render(h.ListHostView.View())))
	case utils.ListHostState:
		view = lipgloss.JoinHorizontal(lipgloss.Top,
			widthStyleComponent.Render(paddingStyle.Render(h.FilePickerView.View())),
			widthStylePager.Render(paddingStyle.Render(h.PagerView.View())),
			widthStyleComponent.Render(selectedStyle.Render(h.ListHostView.View())))
	}

	s := lipgloss.JoinHorizontal(lipgloss.Top, view)
	return s
}
