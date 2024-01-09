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

type PagerModel struct {
	Content  string
	Ready    bool
	Viewport viewport.Model
}

// You generally won't need this unless you're processing stuff with
// complicated ANSI escape sequences. Turn it on if you notice flickering.
//
// Also keep in mind that high performance rendering only works for programs
// that use the full size of the terminal. We're enabling that below with
// tea.EnterAltScreen().
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

func (h PagerModel) Init() tea.Cmd {
	return nil
}

func (h PagerModel) Update(msg tea.Msg) (PagerModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return h, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(h.headerView())
		footerHeight := lipgloss.Height(h.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !h.Ready {
			// Since this program is using the full size of the Viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the Viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			h.Viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			h.Viewport.YPosition = headerHeight
			h.Viewport.HighPerformanceRendering = useHighPerformanceRenderer
			h.Viewport.SetContent(h.Content)
			h.Ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the Viewport one line below the header.
			h.Viewport.YPosition = headerHeight + 1
		} else {
			h.Viewport.Width = msg.Width
			h.Viewport.Height = msg.Height - verticalMarginHeight
		}

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole Viewport. Necessary both to
			// initialize the Viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(h.Viewport))
		}
	}

	// Handle keyboard and mouse events in the Viewport
	h.Viewport, cmd = h.Viewport.Update(msg)
	cmds = append(cmds, cmd)

	return h, tea.Batch(cmds...)
}

func (h PagerModel) View() string {
	if !h.Ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", h.headerView(), h.Viewport.View(), h.footerView())
}

func (h PagerModel) headerView() string {
	title := titleStylePager.Render("Mr. Pager")
	line := strings.Repeat("─", max(0, h.Viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (h PagerModel) footerView() string {
	info := infoStylePager.Render(fmt.Sprintf("%3.f%%", h.Viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, h.Viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (h PagerModel) InitializePagerModel() PagerModel {
	// Load some text for our Viewport
	content, err := os.ReadFile("main.go")
	if err != nil {
		fmt.Println("could not load file:", err)
		os.Exit(1)
	}
	return PagerModel{Content: string(content)}
}
