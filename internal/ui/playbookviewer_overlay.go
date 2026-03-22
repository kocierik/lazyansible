package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/editor"
)

// PlaybookViewerOverlay shows the raw YAML of a playbook with syntax colouring.
type PlaybookViewerOverlay struct {
	width  int
	height int

	title  string
	path   string   // filesystem path (used by the editor)
	lines  []string // raw file lines
	offset int      // scroll position
	err    string
}

func newPlaybookViewerOverlay(width, height int) *PlaybookViewerOverlay {
	return &PlaybookViewerOverlay{width: width, height: height}
}

// Load reads the given file path into the viewer.
func (v *PlaybookViewerOverlay) Load(title, path string) {
	v.title = title
	v.path = path
	v.offset = 0
	v.err = ""
	data, err := os.ReadFile(path)
	if err != nil {
		v.lines = nil
		v.err = err.Error()
		return
	}
	v.lines = strings.Split(string(data), "\n")
}

// Reload re-reads the file from disk (called after editor returns).
func (v *PlaybookViewerOverlay) Reload() {
	if v.path == "" {
		return
	}
	data, err := os.ReadFile(v.path)
	if err != nil {
		v.err = err.Error()
		return
	}
	v.err = ""
	v.lines = strings.Split(string(data), "\n")
}

func (v *PlaybookViewerOverlay) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	contentH := v.contentHeight()
	maxOff := len(v.lines) - contentH
	if maxOff < 0 {
		maxOff = 0
	}
	switch key.String() {
	case "j", "down":
		if v.offset < maxOff {
			v.offset++
		}
	case "k", "up":
		if v.offset > 0 {
			v.offset--
		}
	case "ctrl+d":
		v.offset += contentH / 2
		if v.offset > maxOff {
			v.offset = maxOff
		}
	case "ctrl+u":
		v.offset -= contentH / 2
		if v.offset < 0 {
			v.offset = 0
		}
	case "g":
		v.offset = 0
	case "G":
		v.offset = maxOff
	case "e":
		// Open in external editor; TUI suspends and resumes after.
		if v.path != "" {
			return editor.Open(v.path)
		}
	case "esc", "q":
		return func() tea.Msg { return pbViewerCloseMsg{} }
	}
	return nil
}

// pbViewerCloseMsg is sent when the viewer should be closed.
type pbViewerCloseMsg struct{}

func (v *PlaybookViewerOverlay) contentHeight() int {
	h := v.height - 8
	if h < 4 {
		h = 4
	}
	return h
}

func (v *PlaybookViewerOverlay) View() string {
	boxW := min(v.width-4, 100)
	boxH := min(v.height-4, 40)

	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Playbook: "+v.title) + "\n\n")

	if v.err != "" {
		sb.WriteString(overlayMutedStyle.Render("Could not read file: " + v.err))
		sb.WriteString("\n\n" + overlayHintStyle.Render("[esc] close"))
		return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
	}

	contentH := v.contentHeight()
	total := len(v.lines)
	end := v.offset + contentH
	if end > total {
		end = total
	}

	// Gutter width based on total line count digits.
	gutterW := len(fmt.Sprintf("%d", total)) + 1

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))
	codeW := boxW - gutterW - 6
	if codeW < 10 {
		codeW = 10
	}
	for i := v.offset; i < end; i++ {
		numStr := fmt.Sprintf("%*d ", gutterW, i+1)
		num := lineNumStyle.Render(numStr)
		sb.WriteString(num + yamlHighlight(v.lines[i], codeW) + "\n")
	}

	// Scroll indicator row.
	if total > contentH {
		pct := min((v.offset+contentH)*100/total, 100)
		indicator := fmt.Sprintf("── %d/%d lines (%d%%) ──", min(v.offset+contentH, total), total, pct)
		sb.WriteString("\n" + overlayMutedStyle.Render(indicator) + "\n")
	}

	sb.WriteString("\n" + overlayHintStyle.Render("[j/k] scroll  [ctrl+d/u] half-page  [g/G] top/bottom  [e] edit  [esc] close"))
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
}

// yamlHighlight applies simple YAML syntax colouring to a single line.
func yamlHighlight(line string, maxW int) string {
	runes := []rune(line)
	if len(runes) > maxW && maxW > 3 {
		line = string(runes[:maxW-1]) + "…"
	}

	trimmed := strings.TrimSpace(line)

	// Blank line.
	if trimmed == "" {
		return ""
	}

	// Comment.
	if strings.HasPrefix(trimmed, "#") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).Italic(true).Render(line)
	}

	// List item indicator "- ".
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	// Key: value pair.
	if idx := strings.Index(trimmed, ":"); idx > 0 {
		key := trimmed[:idx]
		rest := trimmed[idx+1:]

		// Ansible task keywords get a special colour.
		ansibleKeywords := map[string]bool{
			"name": true, "hosts": true, "tasks": true, "become": true,
			"vars": true, "roles": true, "handlers": true, "notify": true,
			"when": true, "register": true, "loop": true, "block": true,
			"rescue": true, "always": true, "tags": true, "include_tasks": true,
			"import_tasks": true, "import_playbook": true, "gather_facts": true,
		}

		keyColor := lipgloss.Color("#06B6D4") // cyan — normal key
		if ansibleKeywords[strings.TrimPrefix(key, "- ")] {
			keyColor = lipgloss.Color("#7C3AED") // purple — ansible keyword
		}

		keyRendered := lipgloss.NewStyle().Foreground(keyColor).Bold(true).Render(key + ":")
		valRendered := ""
		if rest != "" {
			val := strings.TrimSpace(rest)
			// Jinja2 template.
			if strings.Contains(val, "{{") {
				valRendered = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render(val)
			} else if val == "true" || val == "false" || val == "yes" || val == "no" {
				valRendered = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render(val)
			} else {
				valRendered = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(val)
			}
		}
		return indent + keyRendered + valRendered
	}

	// Plain list item or continuation.
	if strings.HasPrefix(trimmed, "- ") {
		bullet := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("- ")
		rest := lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(trimmed[2:])
		return indent + bullet + rest
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(line)
}
