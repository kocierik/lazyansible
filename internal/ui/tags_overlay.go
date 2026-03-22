package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// TagsConfirmedMsg is sent when the user confirms tag selection.
type TagsConfirmedMsg struct{ Tags string }

// TagsOverlay shows the available tags for a playbook with multi-select.
type TagsOverlay struct {
	allTags  []string // full list
	visible  []string // filtered list
	selected map[string]bool
	cursor   int
	filter   textinput.Model
	width    int
	height   int
}

func newTagsOverlay(width, height int) *TagsOverlay {
	ti := textinput.New()
	ti.Placeholder = "filter tags…"
	ti.Width = 30
	ti.Focus()

	return &TagsOverlay{
		selected: make(map[string]bool),
		filter:   ti,
		width:    width,
		height:   height,
	}
}

func (t *TagsOverlay) SetTags(tags []string) {
	t.allTags = tags
	t.selected = make(map[string]bool)
	t.filter.SetValue("")
	t.cursor = 0
	t.applyFilter()
}

func (t *TagsOverlay) applyFilter() {
	q := strings.ToLower(t.filter.Value())
	t.visible = t.visible[:0]
	for _, tag := range t.allTags {
		if q == "" || strings.Contains(strings.ToLower(tag), q) {
			t.visible = append(t.visible, tag)
		}
	}
	if t.cursor >= len(t.visible) && len(t.visible) > 0 {
		t.cursor = len(t.visible) - 1
	}
}

// SelectedTagsString returns a comma-separated list of selected tags.
func (t *TagsOverlay) SelectedTagsString() string {
	var out []string
	for _, tag := range t.allTags { // preserves order
		if t.selected[tag] {
			out = append(out, tag)
		}
	}
	return strings.Join(out, ",")
}

func (t *TagsOverlay) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		t.filter, cmd = t.filter.Update(msg)
		t.applyFilter()
		return cmd
	}

	switch key.String() {
	case "enter":
		return func() tea.Msg {
			return TagsConfirmedMsg{Tags: t.SelectedTagsString()}
		}
	case "j", "down":
		if t.cursor < len(t.visible)-1 {
			t.cursor++
		}
		return nil
	case "k", "up":
		if t.cursor > 0 {
			t.cursor--
		}
		return nil
	case " ":
		if t.cursor < len(t.visible) {
			tag := t.visible[t.cursor]
			t.selected[tag] = !t.selected[tag]
		}
		return nil
	case "a":
		// Select all visible.
		for _, tag := range t.visible {
			t.selected[tag] = true
		}
		return nil
	case "A":
		// Deselect all.
		t.selected = make(map[string]bool)
		return nil
	default:
		var cmd tea.Cmd
		t.filter, cmd = t.filter.Update(msg)
		t.applyFilter()
		return cmd
	}
}

func (t *TagsOverlay) View() string {
	boxW := min(t.width-8, 60)
	boxH := min(t.height-6, 28)

	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Tags Browser") + "\n")

	// Filter input.
	sb.WriteString(overlayLabelStyle.Render("Filter: ") + t.filter.View() + "\n\n")

	if len(t.allTags) == 0 {
		sb.WriteString(overlayMutedStyle.Render("  No tags found in this playbook.") + "\n")
	} else if len(t.visible) == 0 {
		sb.WriteString(overlayMutedStyle.Render("  No tags match the filter.") + "\n")
	} else {
		contentH := boxH - 9
		if contentH < 1 {
			contentH = 1
		}
		start := 0
		if t.cursor >= contentH {
			start = t.cursor - contentH + 1
		}
		end := start + contentH
		if end > len(t.visible) {
			end = len(t.visible)
		}

		for i := start; i < end; i++ {
			tag := t.visible[i]
			check := "  "
			if t.selected[tag] {
				check = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render("✓ ")
			}
			line := fmt.Sprintf("%s%s", check, tag)
			if i == t.cursor {
				sb.WriteString(overlaySelectedStyle.Render(line) + "\n")
			} else {
				sb.WriteString(overlayItemStyle.Render(line) + "\n")
			}
		}
	}

	// Show current selection.
	sel := t.SelectedTagsString()
	if sel != "" {
		sb.WriteString("\n" + overlayLabelStyle.Render("Selected: ") +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render(sel) + "\n")
	}

	sb.WriteString("\n" + overlayHintStyle.Render("[space] toggle  [a] all  [A] none  [enter] confirm  [esc] cancel"))

	return overlayBoxStyle.
		Width(boxW).
		Height(boxH).
		Render(sb.String())
}
