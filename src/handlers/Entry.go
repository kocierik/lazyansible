package handlers

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		case 0:
			h.ListView, cmd = h.ListView.Update(msg)
			cmds = append(cmds, cmd)
		case 1:
			h.PagerView, cmd = h.PagerView.Update(msg)
			cmds = append(cmds, cmd)
		default:
			h.ListView, cmd = h.ListView.Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	return h, tea.Batch(cmds...)
}

func (h MainModel) View() string {
	// if h.State == "" {
	// 	return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", h.Choice))
	// }
	// if h.Quitting {
	// 	return quitTextStyle.Render("Not hungry? That’s cool.")
	// }
	// return "\n" + h.List.View()
	// s := h.ListView.View()
	// s += h.PagerView.View()
	s := lipgloss.JoinHorizontal(lipgloss.Top, h.ListView.View(), h.PagerView.Content)
	return s
}

func (h MainModel) InitializeListModel() ListModel {

	items := []list.Item{
		item("Ramen"),
		item("Tomato Soup"),
		item("Hamburgers"),
		item("Cheeseburgers"),
		item("Currywurst"),
		item("Okonomiyaki"),
		item("Pasta"),
		item("Fillet Mignon"),
		item("Caviar"),
		item("Just Wine"),
	}

	const defaultWidth = 20
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "What do you want for dinner?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	listModel := ListModel{List: l, Choice: "", Quitting: false}
	return listModel
}
