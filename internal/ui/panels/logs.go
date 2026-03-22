package panels

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/core"
)

const maxLogLines = 10000

// LogsPanel displays streamed ansible-playbook output.
type LogsPanel struct {
	lines      []core.LogLine
	offset     int // lines from the bottom (0 = at bottom)
	focused    bool
	width      int
	height     int
	autoScroll bool
	showTime   bool // toggle with T
}

func NewLogsPanel(width, height int) *LogsPanel {
	return &LogsPanel{width: width, height: height, autoScroll: true}
}

func (p *LogsPanel) SetSize(w, h int)  { p.width = w; p.height = h }
func (p *LogsPanel) SetFocused(f bool) { p.focused = f }

func (p *LogsPanel) AddLine(line core.LogLine) {
	p.lines = append(p.lines, line)
	if len(p.lines) > maxLogLines {
		p.lines = p.lines[len(p.lines)-maxLogLines:]
	}
	if p.autoScroll {
		p.offset = 0
	}
}

func (p *LogsPanel) Clear() { p.lines = nil; p.offset = 0 }

func (p *LogsPanel) visibleLines() int {
	h := p.height - 2 // title row + separator
	if h < 1 {
		h = 1
	}
	return h
}

func (p *LogsPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.focused {
		return nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	contentH := p.visibleLines()
	switch key.String() {
	case "j", "down":
		if p.offset > 0 {
			p.offset--
			p.autoScroll = (p.offset == 0)
		}
	case "k", "up":
		maxOff := len(p.lines) - contentH
		if maxOff < 0 {
			maxOff = 0
		}
		if p.offset < maxOff {
			p.offset++
			p.autoScroll = false
		}
	case "G", "end":
		p.offset = 0
		p.autoScroll = true
	case "g", "home":
		maxOff := len(p.lines) - contentH
		if maxOff < 0 {
			maxOff = 0
		}
		p.offset = maxOff
		p.autoScroll = false
	case "ctrl+d":
		p.offset -= contentH / 2
		if p.offset < 0 {
			p.offset = 0
			p.autoScroll = true
		}
	case "ctrl+u":
		maxOff := len(p.lines) - contentH
		if maxOff < 0 {
			maxOff = 0
		}
		p.offset += contentH / 2
		if p.offset > maxOff {
			p.offset = maxOff
		}
		p.autoScroll = false
	case "T":
		p.showTime = !p.showTime
	}
	return nil
}

func (p *LogsPanel) View() string {
	contentH := p.visibleLines()
	total := len(p.lines)

	// ── Title bar with live scroll position ──────────────────────────────
	title := p.renderTitle(total, contentH)

	if total == 0 {
		empty := mutedText("No output yet.  Select a playbook → [r] to run,  or [!] for ad-hoc.")
		// Return exactly 2 lines: title + placeholder. No trailing newline so
		// the panel height is controlled solely by wrapPanel.
		return title + "\n" + empty
	}

	// ── Visible window into lines ─────────────────────────────────────────
	end := total - p.offset
	if end < 0 {
		end = 0
	}
	start := end - contentH
	if start < 0 {
		start = 0
	}

	// Emit at most contentH rendered lines after the title.
	// Each rendered line must not itself contain a newline (ansible output
	// is already split by the scanner, but strip any stray \r just in case).
	var sb strings.Builder
	sb.WriteString(title)

	written := 0
	for i := start; i < end && written < contentH; i++ {
		line := p.lines[i]
		// Strip stray carriage-returns that could corrupt the layout.
		line.Text = strings.ReplaceAll(line.Text, "\r", "")
		rendered := renderLogLine(line, p.width, p.showTime)
		// renderLogLine should never contain a bare \n, but guard anyway.
		rendered = strings.SplitN(rendered, "\n", 2)[0]
		sb.WriteByte('\n')
		sb.WriteString(rendered)
		written++
	}

	return sb.String()
}

func (p *LogsPanel) renderTitle(total, contentH int) string {
	base := lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4")).Bold(true).Render("Logs")

	var right string
	if total > 0 {
		pos := total - p.offset
		pct := pos * 100 / total
		scrollIcon := "↓"
		if !p.autoScroll {
			scrollIcon = "↑"
		}
		right = lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).
			Render(fmt.Sprintf("%s %d/%d (%d%%)", scrollIcon, pos, total, pct))
		if p.showTime {
			right += lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Render(" [T]ime")
		}
	}

	sepLen := p.width - lipgloss.Width(base) - lipgloss.Width(right) - 2
	if sepLen < 0 {
		sepLen = 0
	}
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#1F2937")).Render(strings.Repeat("─", sepLen))
	return base + sep + right
}

// ─── Line rendering ───────────────────────────────────────────────────────────

func renderLogLine(line core.LogLine, maxW int, showTime bool) string {
	text := line.Text
	trimmed := strings.TrimSpace(text)

	// ── Ansible separator lines: "TASK [foo] ****..." ─────────────────────
	if isAnsibleHeader(trimmed) {
		return renderHeaderLine(trimmed, maxW)
	}

	// ── Timestamp prefix ──────────────────────────────────────────────────
	prefix := ""
	if showTime {
		ts := line.Timestamp.Format("15:04:05")
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("#374151")).Render(ts + " ")
	}

	// ── Truncate to visible width ─────────────────────────────────────────
	avail := maxW - lipgloss.Width(prefix)
	if avail < 4 {
		avail = 4
	}
	runes := []rune(text)
	if len(runes) > avail {
		text = string(runes[:avail-1]) + "…"
	}

	styled := applyLogStyle(text, line.Level)
	return prefix + styled
}

// isAnsibleHeader detects "TASK [...]  ****" / "PLAY [...]  ****" / "PLAY RECAP" lines.
func isAnsibleHeader(s string) bool {
	if (strings.HasPrefix(s, "TASK [") ||
		strings.HasPrefix(s, "PLAY [") ||
		strings.HasPrefix(s, "PLAY RECAP") ||
		strings.HasPrefix(s, "TASKS RECAP")) &&
		strings.HasSuffix(strings.TrimSpace(s), "*") {
		return true
	}
	// Pure star lines (e.g. after errors)
	stripped := strings.TrimLeft(s, "*")
	return stripped == "" && len(s) > 4
}

// renderHeaderLine turns "TASK [foo] ***..." into a styled ── TASK [foo] ──── line.
func renderHeaderLine(s string, maxW int) string {
	// Strip trailing asterisks and whitespace.
	label := strings.TrimRight(s, "* ")
	label = strings.TrimSpace(label)

	if label == "" {
		// Pure star divider.
		return dimSepStyle.Render(strings.Repeat("─", maxW))
	}

	prefix := "── "
	fillLen := maxW - len([]rune(prefix)) - len([]rune(label)) - 2
	if fillLen < 2 {
		fillLen = 2
	}
	fill := " " + strings.Repeat("─", fillLen)

	switch {
	case strings.HasPrefix(label, "PLAY RECAP"), strings.HasPrefix(label, "TASKS RECAP"):
		return recapHeaderStyle.Render(prefix + label + fill)
	case strings.HasPrefix(label, "PLAY"):
		return playHeaderStyle.Render(prefix + label + fill)
	case strings.HasPrefix(label, "TASK"):
		return taskHeaderStyle.Render(prefix + label + fill)
	default:
		return dimSepStyle.Render(prefix + label + fill)
	}
}

func applyLogStyle(text string, level core.LogLevel) string {
	switch level {
	case core.LogLevelOK:
		return okLogStyle.Render(text)
	case core.LogLevelChanged:
		return changedLogStyle.Render(text)
	case core.LogLevelFailed:
		return failedLogStyle.Render(text)
	case core.LogLevelWarning:
		return warnLogStyle.Render(text)
	case core.LogLevelDebug:
		return debugLogStyle.Render(text)
	case core.LogLevelDiffAdd:
		return diffAddStyle.Render(text)
	case core.LogLevelDiffRemove:
		return diffRemoveStyle.Render(text)
	case core.LogLevelDiffHunk:
		return diffHunkStyle.Render(text)
	case core.LogLevelDiffHeader:
		return diffHeaderStyle.Render(text)
	default:
		return infoLogStyle.Render(text)
	}
}

// ─── Log-specific styles ──────────────────────────────────────────────────────

var (
	taskHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Bold(true)

	playHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

	recapHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")).
				Bold(true)

	dimSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151"))

	okLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E"))

	changedLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B"))

	failedLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)

	warnLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F97316"))

	debugLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	infoLogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	diffAddStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ADE80")) // bright green

	diffRemoveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F87171")) // bright red

	diffHunkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22D3EE")).
			Bold(true) // cyan

	diffHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			Bold(true) // bold white
)
