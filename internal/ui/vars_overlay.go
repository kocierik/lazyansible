package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/core"
)

// VarsOverlay shows the variables for a host or group in a centered modal.
type VarsOverlay struct {
	title  string
	keys   []string
	vars   map[string]string
	cursor int
	width  int
	height int
}

func newVarsOverlay(width, height int) *VarsOverlay {
	return &VarsOverlay{width: width, height: height}
}

func (v *VarsOverlay) SetHost(host *core.Host) {
	v.title = "Variables – " + host.Name
	v.vars = host.Vars
	v.buildKeys()
	v.cursor = 0
}

func (v *VarsOverlay) SetGroup(group *core.Group) {
	v.title = "Variables – [" + group.Name + "]"
	v.vars = group.Vars
	v.buildKeys()
	v.cursor = 0
}

func (v *VarsOverlay) buildKeys() {
	v.keys = make([]string, 0, len(v.vars))
	for k := range v.vars {
		v.keys = append(v.keys, k)
	}
	sort.Strings(v.keys)
}

func (v *VarsOverlay) Update(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "j", "down":
			if v.cursor < len(v.keys)-1 {
				v.cursor++
			}
		case "k", "up":
			if v.cursor > 0 {
				v.cursor--
			}
		case "g":
			v.cursor = 0
		case "G":
			if len(v.keys) > 0 {
				v.cursor = len(v.keys) - 1
			}
		}
	}
	return nil
}

func (v *VarsOverlay) View() string {
	boxW := min(v.width-8, 72)
	boxH := min(v.height-6, 30)

	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render(v.title) + "\n\n")

	if len(v.keys) == 0 {
		sb.WriteString(overlayMutedStyle.Render("  (no variables defined)") + "\n")
	} else {
		// Find longest key for alignment.
		maxKeyLen := 0
		for _, k := range v.keys {
			if len(k) > maxKeyLen {
				maxKeyLen = len(k)
			}
		}
		if maxKeyLen > 32 {
			maxKeyLen = 32
		}

		contentH := boxH - 6
		start := 0
		if v.cursor >= contentH {
			start = v.cursor - contentH + 1
		}
		end := start + contentH
		if end > len(v.keys) {
			end = len(v.keys)
		}

		for i := start; i < end; i++ {
			k := v.keys[i]
			val := v.vars[k]
			keyPad := fmt.Sprintf("%-*s", maxKeyLen, k)
			line := fmt.Sprintf("  %s  %s", keyPad, val)
			if i == v.cursor {
				sb.WriteString(overlaySelectedStyle.Render(line) + "\n")
			} else {
				sb.WriteString(overlayItemStyle.Render(line) + "\n")
			}
		}

		if len(v.keys) > contentH {
			sb.WriteString("\n" + overlayMutedStyle.Render(
				fmt.Sprintf("  %d/%d  j/k to scroll", v.cursor+1, len(v.keys)),
			) + "\n")
		}
	}

	sb.WriteString("\n" + overlayHintStyle.Render("  [esc] close   [j/k] scroll   [g/G] top/bottom"))

	return overlayBoxStyle.
		Width(boxW).
		Height(boxH).
		Render(sb.String())
}

// ─── Shared overlay styles ────────────────────────────────────────────────────

var (
	overlayBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Background(lipgloss.Color("#111827")).
			Padding(1, 2)

	overlayTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")).
				Bold(true)

	overlaySelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F9FAFB")).
				Background(lipgloss.Color("#374151")).
				Bold(true)

	overlayItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D1D5DB"))

	overlayMutedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4B5563")).
				Italic(true)

	overlayHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4B5563"))

	overlayLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF"))

	overlayActiveInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")).
				Bold(true)
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
