package panels

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/core"
)

// StatusPanel shows per-host execution status.
type StatusPanel struct {
	results  map[string]*core.HostResult
	order    []string // insertion order
	focused  bool
	width    int
	height   int
	cursor   int
	running  bool
	runStart time.Time
}

func NewStatusPanel(width, height int) *StatusPanel {
	return &StatusPanel{
		results: make(map[string]*core.HostResult),
		width:   width,
		height:  height,
	}
}

func (p *StatusPanel) SetSize(w, h int)  { p.width = w; p.height = h }
func (p *StatusPanel) SetFocused(f bool) { p.focused = f }

func (p *StatusPanel) SetRunning(running bool) {
	p.running = running
	if running {
		p.runStart = time.Now()
	}
}

func (p *StatusPanel) UpdateHost(host string, status core.TaskStatus, task string) {
	if _, ok := p.results[host]; !ok {
		p.order = append(p.order, host)
		p.results[host] = &core.HostResult{Host: host}
	}
	r := p.results[host]
	r.Status = status
	r.TaskName = task
	r.ChangedAt = time.Now()
}

func (p *StatusPanel) Reset() {
	p.results = make(map[string]*core.HostResult)
	p.order = nil
	p.running = false
}

// FailedHosts returns the names of hosts with failed or unreachable status.
func (p *StatusPanel) FailedHosts() []string {
	var failed []string
	for _, host := range p.order {
		r := p.results[host]
		if r.Status == core.TaskStatusFailed || r.Status == core.TaskStatusUnreachable {
			failed = append(failed, host)
		}
	}
	return failed
}

// HostStatsMap returns a map of host → status string for history storage.
func (p *StatusPanel) HostStatsMap() map[string]string {
	m := make(map[string]string, len(p.results))
	for host, r := range p.results {
		m[host] = r.Status.String()
	}
	return m
}

func (p *StatusPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.focused {
		return nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if p.cursor < len(p.results)-1 {
				p.cursor++
			}
		case "k", "up":
			if p.cursor > 0 {
				p.cursor--
			}
		}
	}
	return nil
}

func (p *StatusPanel) View() string {
	var sb strings.Builder
	// title is shown in the panel border; no need to repeat it here

	if p.running {
		elapsed := time.Since(p.runStart).Round(time.Second)
		sb.WriteString(runningStyle.Render(fmt.Sprintf("▶ Running… %s", elapsed)) + "\n")
	} else if len(p.results) == 0 {
		sb.WriteString(mutedText("No run data yet."))
		return sb.String()
	}

	// Summary counts.
	counts := map[core.TaskStatus]int{}
	for _, r := range p.results {
		counts[r.Status]++
	}
	var summary []string
	if n := counts[core.TaskStatusOK]; n > 0 {
		summary = append(summary, okStatusStyle.Render(fmt.Sprintf("ok=%d", n)))
	}
	if n := counts[core.TaskStatusChanged]; n > 0 {
		summary = append(summary, changedStatusStyle.Render(fmt.Sprintf("changed=%d", n)))
	}
	if n := counts[core.TaskStatusFailed]; n > 0 {
		summary = append(summary, failedStatusStyle.Render(fmt.Sprintf("failed=%d", n)))
	}
	if len(summary) > 0 {
		sb.WriteString(strings.Join(summary, " ") + "\n")
	}

	// Sort hosts by status priority (failed first, then changed, then ok).
	hosts := make([]string, len(p.order))
	copy(hosts, p.order)
	sort.SliceStable(hosts, func(i, j int) bool {
		si := statusPriority(p.results[hosts[i]].Status)
		sj := statusPriority(p.results[hosts[j]].Status)
		return si < sj
	})

	contentH := p.height - 6
	if contentH < 1 {
		contentH = 1
	}
	start := 0
	if p.cursor >= contentH {
		start = p.cursor - contentH + 1
	}
	end := start + contentH
	if end > len(hosts) {
		end = len(hosts)
	}

	for i := start; i < end; i++ {
		host := hosts[i]
		r := p.results[host]
		selected := i == p.cursor

		statusBadge := renderStatusBadge(r.Status)
		hostText := fmt.Sprintf("%-20s %s", truncate(host, 20), statusBadge)
		if r.TaskName != "" {
			hostText += "\n" + mutedText("  └ "+truncate(r.TaskName, p.width-8))
		}

		if selected && p.focused {
			sb.WriteString(lipgloss.NewStyle().
				Background(lipgloss.Color("#1F2937")).
				Render(hostText) + "\n")
		} else {
			sb.WriteString(hostText + "\n")
		}
	}

	return sb.String()
}

func renderStatusBadge(s core.TaskStatus) string {
	switch s {
	case core.TaskStatusOK:
		return okStatusStyle.Render("OK")
	case core.TaskStatusChanged:
		return changedStatusStyle.Render("CHANGED")
	case core.TaskStatusFailed:
		return failedStatusStyle.Render("FAILED")
	case core.TaskStatusSkipped:
		return skippedStatusStyle.Render("SKIPPED")
	case core.TaskStatusUnreachable:
		return unreachableStatusStyle.Render("UNREACHABLE")
	default:
		return mutedText("PENDING")
	}
}

func statusPriority(s core.TaskStatus) int {
	switch s {
	case core.TaskStatusFailed:
		return 0
	case core.TaskStatusUnreachable:
		return 1
	case core.TaskStatusChanged:
		return 2
	case core.TaskStatusOK:
		return 3
	case core.TaskStatusSkipped:
		return 4
	default:
		return 5
	}
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

var (
	okStatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")).
			Bold(true)

	changedStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B")).
				Bold(true)

	failedStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EF4444")).
				Bold(true)

	skippedStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	unreachableStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F97316")).
				Bold(true)

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#06B6D4")).
			Bold(true)
)
