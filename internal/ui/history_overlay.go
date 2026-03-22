package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/history"
)

// HistoryRunMsg is sent when the user re-runs a history record.
type HistoryRunMsg struct{ Record *history.Record }

// HistoryOverlay shows the run history with the option to re-run.
type HistoryOverlay struct {
	records []*history.Record
	cursor  int
	width   int
	height  int
	err     error
}

func newHistoryOverlay(width, height int) *HistoryOverlay {
	return &HistoryOverlay{width: width, height: height}
}

// Reload fetches history from disk. Call before showing the overlay.
func (h *HistoryOverlay) Reload() {
	records, err := history.Load()
	h.records = history.Limit(records, 100)
	h.err = err
	h.cursor = 0
}

func (h *HistoryOverlay) selected() *history.Record {
	if h.cursor < len(h.records) {
		return h.records[h.cursor]
	}
	return nil
}

func (h *HistoryOverlay) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "j", "down":
		if h.cursor < len(h.records)-1 {
			h.cursor++
		}
	case "k", "up":
		if h.cursor > 0 {
			h.cursor--
		}
	case "g":
		h.cursor = 0
	case "G":
		if len(h.records) > 0 {
			h.cursor = len(h.records) - 1
		}
	case "enter", "r":
		if r := h.selected(); r != nil {
			return func() tea.Msg { return HistoryRunMsg{Record: r} }
		}
	}
	return nil
}

func (h *HistoryOverlay) View() string {
	boxW := min(h.width-6, 80)
	boxH := min(h.height-4, 34)

	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Run History") + "\n\n")

	if h.err != nil {
		sb.WriteString(overlayMutedStyle.Render("Could not load history: " + h.err.Error()))
		sb.WriteString("\n\n" + overlayHintStyle.Render("[esc] close"))
		return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
	}

	if len(h.records) == 0 {
		sb.WriteString(overlayMutedStyle.Render("No run history yet.\nRun a playbook to create history entries."))
		sb.WriteString("\n\n" + overlayHintStyle.Render("[esc] close"))
		return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
	}

	// ── Column header ──
	header := fmt.Sprintf("%-19s  %-18s  %-6s  %-5s  %s",
		"Time", "Playbook", "Dur", "Code", "Limit/Tags")
	sb.WriteString(overlayLabelStyle.Render(header) + "\n")
	sb.WriteString(overlayMutedStyle.Render(strings.Repeat("─", min(boxW-6, 72))) + "\n")

	contentH := boxH - 8
	if contentH < 1 {
		contentH = 1
	}
	start := 0
	if h.cursor >= contentH {
		start = h.cursor - contentH + 1
	}
	end := start + contentH
	if end > len(h.records) {
		end = len(h.records)
	}

	for i := start; i < end; i++ {
		r := h.records[i]
		selected := i == h.cursor

		result := r.Result()
		resultStyle := overlayMutedStyle
		if r.ExitCode == 0 {
			resultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))
		} else {
			resultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
		}

		extra := r.Limit
		if r.Tags != "" {
			extra += " #" + r.Tags
		}
		extra = truncateStr(extra, 18)

		timeStr := r.StartTime.Format("01/02 15:04:05")
		pbName := truncateStr(r.PlaybookName, 18)
		dur := truncateStr(r.Duration(), 6)

		row := fmt.Sprintf("%-19s  %-18s  %-6s  ", timeStr, pbName, dur)

		if selected {
			sb.WriteString(overlaySelectedStyle.Render(row) +
				resultStyle.Render(fmt.Sprintf("%-5s", result)) +
				overlaySelectedStyle.Render("  "+extra) + "\n")
		} else {
			sb.WriteString(overlayItemStyle.Render(row) +
				resultStyle.Render(fmt.Sprintf("%-5s", result)) +
				overlayItemStyle.Render("  "+extra) + "\n")
		}
	}

	// ── Selected record detail ──
	if r := h.selected(); r != nil {
		sb.WriteString("\n" + overlayLabelStyle.Render("Path: ") +
			overlayMutedStyle.Render(truncateStr(r.PlaybookPath, boxW-12)) + "\n")
		if r.ExtraVars != "" {
			sb.WriteString(overlayLabelStyle.Render("Vars: ") +
				overlayMutedStyle.Render(truncateStr(r.ExtraVars, boxW-12)) + "\n")
		}
	}

	sb.WriteString("\n" + overlayHintStyle.Render("[j/k] navigate  [enter/r] re-run  [esc] close"))

	return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
}
