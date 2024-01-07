package handlers

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kocierik/lazyansible/src/models"
	"github.com/kocierik/lazyansible/src/utils"
	"os"
	"strings"
)

const useHighPerformanceRenderer = false

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()
)

func (h handler) HeaderView() string {
	title := titleStyle.Render("Mr. Pager")
	line := strings.Repeat("─", max(0, h.Model.Viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (h handler) FooterView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", h.Model.Viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, h.Model.Viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (h handler) Init() tea.Cmd {
	return nil
}

func (h handler) InitialModel() models.Model {
	files, _ := utils.GetFiles()
	content, err := os.ReadFile(files[0])

	if err != nil {
		fmt.Println("could not load file:", err)
		os.Exit(1)
	}
	return models.Model{
		Choices: files,
		Content: string(content),

		Selected: make(map[int]struct{}),
	}
}

func (h handler) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(h.HeaderView())
		footerHeight := lipgloss.Height(h.FooterView())
		verticalMarginHeight := headerHeight + footerHeight

		if !h.Model.Ready {
			h.Model.Viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			h.Model.Viewport.YPosition = headerHeight
			h.Model.Viewport.HighPerformanceRendering = useHighPerformanceRenderer
			h.Model.Viewport.SetContent(h.Model.Content)
			h.Model.Ready = true

			h.Model.Viewport.YPosition = headerHeight + 1
		} else {
			h.Model.Viewport.Width = msg.Width
			h.Model.Viewport.Height = msg.Height - verticalMarginHeight
		}

		if useHighPerformanceRenderer {
			cmds = append(cmds, viewport.Sync(h.Model.Viewport))
		}
	case tea.KeyMsg:

		switch msg.String() {

		case "ctrl+c", "q":
			return h, tea.Quit

		case "up", "k":
			if h.Model.Cursor > 0 {
				h.Model.Cursor--
			}

		case "down", "j":
			if h.Model.Cursor < len(h.Model.Choices)-1 {
				h.Model.Cursor++
			}

		case "enter", " ":
			_, ok := h.Model.Selected[h.Model.Cursor]
			if ok {
				delete(h.Model.Selected, h.Model.Cursor)
			} else {
				h.Model.Selected[h.Model.Cursor] = struct{}{}
			}

		}
	}

	h.Model.Viewport, cmd = h.Model.Viewport.Update(msg)
	cmds = append(cmds, cmd)
	return h, tea.Batch(cmds...)
}

func (h handler) View() string {
	var style = lipgloss.NewStyle().
		Width(60).
		Height(30).
		MaxHeight(30).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))

	s := "LazyAnsible\n\n"

	for i, choice := range h.Model.Choices {

		cursor := " " // no cursor
		if h.Model.Cursor == i {
			cursor = ">" // cursor!
			h.Model.Content = string(utils.ReadFile(choice))
		}

		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	box1 := style.Render(s)
	box2 := style.Render(h.Model.Content)

	boxInventory := style.Render("host")
	return lipgloss.JoinHorizontal(lipgloss.Center, box1, box2, boxInventory)
}
