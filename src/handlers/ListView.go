package handlers

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ListModel struct {
	List     list.Model
	Choice   string
	Quitting bool
}

const listHeight = 25

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func (h ListModel) Init() tea.Cmd {
	return nil
}

func (h ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.List.SetWidth(msg.Width)
		return h, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			h.Quitting = true
			return h, tea.Quit

		case "enter":
			i, ok := h.List.SelectedItem().(item)
			if ok {
				h.Choice = string(i)
			}
			return h, tea.Quit
		}
	}

	var cmd tea.Cmd
	h.List, cmd = h.List.Update(msg)
	return h, cmd
}

func (h ListModel) View() string {
	if h.Choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", h.Choice))
	}
	if h.Quitting {
		return quitTextStyle.Render("Not hungry? That’s cool.")
	}
	return "\n" + h.List.View()
}

func (h ListModel) InitializeListModel() ListModel {

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
