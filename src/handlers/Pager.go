package handlers

// An example program demonstrating the pager component from the Bubbles
// component library.

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const useHighPerformanceRenderer = false

var (
	titleStylePager = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStylePager = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStylePager.Copy().BorderStyle(b)
	}()
)

type PagerModel struct {
	content  string
	ready    bool
	viewport viewport.Model
}

func (m PagerModel) Init() tea.Cmd {
	return nil
}

func (m PagerModel) Update(msg tea.Msg) (PagerModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m PagerModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m PagerModel) headerView() string {
	title := titleStylePager.Render("Filename")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m PagerModel) footerView() string {
	info := infoStylePager.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func ReadFile(filename string) string {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("could not load file:", err)
		os.Exit(1)
	}
	return string(content)
}

func (h PagerModel) InitializePagerModel() PagerModel {
	content, err := os.ReadFile("main.go")
	if err != nil {
		fmt.Println("could not load file:", err)
		os.Exit(1)
	}
	h.viewport.SetContent(string(content))
	h.viewport.Height = 20
	h.viewport.Width = 70
	return PagerModel{content: string(content), ready: true, viewport: h.viewport}
}
