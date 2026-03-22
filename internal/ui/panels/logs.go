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

	// Search state (/ to open, Esc to close)
	searchActive  bool
	searchQuery   string
	searchMatches []int // indices into lines that match
	matchCursor   int   // current match index within searchMatches
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

func (p *LogsPanel) Clear() {
	p.lines = nil
	p.offset = 0
	p.searchMatches = nil
	p.searchQuery = ""
	p.searchActive = false
}

// Lines returns a copy of the raw log lines for export.
func (p *LogsPanel) Lines() []core.LogLine { return append([]core.LogLine(nil), p.lines...) }

// SearchActive reports whether the search bar is open.
func (p *LogsPanel) SearchActive() bool { return p.searchActive }

// SearchQuery returns the current search string.
func (p *LogsPanel) SearchQuery() string { return p.searchQuery }

// rebuildMatches recomputes searchMatches for the current query.
func (p *LogsPanel) rebuildMatches() {
	p.searchMatches = p.searchMatches[:0]
	if p.searchQuery == "" {
		return
	}
	q := strings.ToLower(p.searchQuery)
	for i, l := range p.lines {
		if strings.Contains(strings.ToLower(l.Text), q) {
			p.searchMatches = append(p.searchMatches, i)
		}
	}
}

// jumpToMatch scrolls so that searchMatches[matchCursor] is visible.
func (p *LogsPanel) jumpToMatch() {
	if len(p.searchMatches) == 0 {
		return
	}
	idx := p.searchMatches[p.matchCursor]
	contentH := p.visibleLines()
	total := len(p.lines)
	// offset is "lines from bottom". idx 0 = first line, idx total-1 = last.
	// We want idx to be the last visible line → offset = total-1-idx
	want := total - 1 - idx
	if want < 0 {
		want = 0
	}
	maxOff := total - contentH
	if maxOff < 0 {
		maxOff = 0
	}
	if want > maxOff {
		want = maxOff
	}
	p.offset = want
	p.autoScroll = false
}

func (p *LogsPanel) visibleLines() int {
	h := p.height - 2 // title row + content
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

	// ── Search mode input ─────────────────────────────────────────────────
	if p.searchActive {
		switch key.String() {
		case "esc":
			// First Esc: close the input bar but keep the current matches.
			// If there is no query at all, fully clear everything.
			p.searchActive = false
			if p.searchQuery == "" {
				p.searchMatches = nil
			}
		case "ctrl+c":
			// Hard cancel: close and wipe query.
			p.searchActive = false
			p.searchQuery = ""
			p.searchMatches = nil
		case "enter":
			// Confirm: close the input bar, keep query + matches visible.
			p.searchActive = false
		case "backspace", "ctrl+h":
			if len(p.searchQuery) > 0 {
				runes := []rune(p.searchQuery)
				p.searchQuery = string(runes[:len(runes)-1])
				p.rebuildMatches()
				// Live scroll to best match while typing.
				if len(p.searchMatches) > 0 {
					p.matchCursor = len(p.searchMatches) - 1
					p.jumpToMatch()
				}
			}
		default:
			if key.Type == tea.KeyRunes {
				p.searchQuery += key.String()
				p.rebuildMatches()
				// Live scroll to best (newest) match as each character is typed.
				if len(p.searchMatches) > 0 {
					p.matchCursor = len(p.searchMatches) - 1
					p.jumpToMatch()
				}
			}
		}
		return nil
	}

	contentH := p.visibleLines()
	switch key.String() {
	case "/":
		p.searchActive = true
		// Re-open search: move cursor to end if a query already exists.
	case "esc":
		// When bar is already closed, Esc clears the query and highlights.
		if p.searchQuery != "" {
			p.searchQuery = ""
			p.searchMatches = nil
		}
	case "n":
		// Next match (towards newer / bottom)
		if len(p.searchMatches) > 0 {
			if p.matchCursor > 0 {
				p.matchCursor--
			} else {
				p.matchCursor = len(p.searchMatches) - 1
			}
			p.jumpToMatch()
		}
	case "N":
		// Previous match (towards older / top)
		if len(p.searchMatches) > 0 {
			if p.matchCursor < len(p.searchMatches)-1 {
				p.matchCursor++
			} else {
				p.matchCursor = 0
			}
			p.jumpToMatch()
		}
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

	// ── Title bar — doubles as the search input when active ───────────────
	title := p.renderTitle(total, contentH)

	if total == 0 {
		empty := mutedText("No output yet.  Select a playbook → [r] to run,  or [!] for ad-hoc.")
		return title + "\n" + empty
	}

	// ── Build match lookup for highlight rendering ────────────────────────
	matchSet := make(map[int]bool, len(p.searchMatches))
	for _, idx := range p.searchMatches {
		matchSet[idx] = true
	}
	currentMatchIdx := -1
	if len(p.searchMatches) > 0 {
		currentMatchIdx = p.searchMatches[p.matchCursor]
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

	var sb strings.Builder
	sb.WriteString(title)

	written := 0
	for i := start; i < end && written < contentH; i++ {
		line := p.lines[i]
		line.Text = strings.ReplaceAll(line.Text, "\r", "")
		rendered := renderLogLine(line, p.width, p.showTime)
		rendered = strings.SplitN(rendered, "\n", 2)[0]

		// Highlight matching lines: current match in bright purple, others in dark bg.
		if p.searchQuery != "" && matchSet[i] {
			if i == currentMatchIdx {
				rendered = lipgloss.NewStyle().
					Background(lipgloss.Color("#7C3AED")).
					Foreground(lipgloss.Color("#FFFFFF")).
					Bold(true).
					Render(rendered)
			} else {
				rendered = lipgloss.NewStyle().
					Background(lipgloss.Color("#1E1B4B")).
					Render(rendered)
			}
		}

		sb.WriteByte('\n')
		sb.WriteString(rendered)
		written++
	}

	return sb.String()
}

func (p *LogsPanel) renderTitle(total, contentH int) string {
	// ── Right side: scroll position + flags ──────────────────────────────
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
			right += lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Render(" [T]")
		}
	}

	// ── Left side: either the label or the live search input ─────────────
	var left string
	if p.searchActive {
		// Show the search input inline in the title bar.
		prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true).Render("/")
		cursor := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).Bold(true).Render("█")
		queryText := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Bold(true).Render(p.searchQuery)

		// Match count badge shown live while typing.
		matchBadge := ""
		if p.searchQuery != "" {
			if len(p.searchMatches) > 0 {
				cur := len(p.searchMatches) - p.matchCursor
				matchBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).
					Render(fmt.Sprintf(" [%d/%d]", cur, len(p.searchMatches)))
			} else {
				matchBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render(" [0]")
			}
		}
		hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).Render("  enter·esc")
		left = prompt + queryText + cursor + matchBadge + hint
	} else if p.searchQuery != "" {
		// Search bar is closed but a query is active — keep it visible in the title.
		prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true).Render("/")
		queryText := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Render(p.searchQuery)
		matchCount := len(p.searchMatches)
		var badge string
		if matchCount > 0 {
			cur := len(p.searchMatches) - p.matchCursor
			badge = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).
				Render(fmt.Sprintf(" [%d/%d] n/N", cur, matchCount))
		} else {
			badge = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render(" [0]")
		}
		hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).Render("  esc:clear")
		left = prompt + queryText + badge + hint
	} else {
		// Normal state.
		left = lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4")).Bold(true).Render("Logs")
	}

	// ── Separator fills the gap between left and right ────────────────────
	sepLen := p.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if sepLen < 0 {
		sepLen = 0
	}
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#1F2937")).Render(strings.Repeat("─", sepLen))
	return left + sep + right
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
	case core.LogLevelCommand:
		return commandStyle.Render(text)
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

	commandStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true) // purple — command echo at run start
)
