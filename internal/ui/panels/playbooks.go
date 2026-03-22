package panels

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/core"
)

// RunRequestMsg is sent when the user presses Enter on a playbook.
type RunRequestMsg struct {
	Playbook *core.Playbook
	Limit    string
	Check    bool
	Diff     bool
	Tags     string
}

// PlaybooksPanel lists discovered playbooks and tracks run options.
type PlaybooksPanel struct {
	playbooks []*core.Playbook
	cursor    int
	focused   bool
	width     int
	height    int

	checkMode    bool
	diffMode     bool
	activeTags   string // set via tags overlay
	limit        string // set from inventory panel
	extraVarsRaw string // set via extra-vars overlay
}

func NewPlaybooksPanel(playbooks []*core.Playbook, width, height int) *PlaybooksPanel {
	return &PlaybooksPanel{playbooks: playbooks, width: width, height: height}
}

func (p *PlaybooksPanel) SetSize(w, h int)                  { p.width = w; p.height = h }
func (p *PlaybooksPanel) SetFocused(f bool)                 { p.focused = f }
func (p *PlaybooksPanel) SetPlaybooks(pbs []*core.Playbook) { p.playbooks = pbs }
func (p *PlaybooksPanel) SetLimit(limit string)             { p.limit = limit }
func (p *PlaybooksPanel) SetActiveTags(tags string)         { p.activeTags = tags }
func (p *PlaybooksPanel) SetExtraVars(raw string)           { p.extraVarsRaw = raw }
func (p *PlaybooksPanel) SetCheckMode(v bool)               { p.checkMode = v }
func (p *PlaybooksPanel) SetDiffMode(v bool)                { p.diffMode = v }
func (p *PlaybooksPanel) CurrentLimit() string              { return p.limit }
func (p *PlaybooksPanel) CheckMode() bool                   { return p.checkMode }
func (p *PlaybooksPanel) DiffMode() bool                    { return p.diffMode }

// SelectedTags returns the active tags as a slice (split by comma).
func (p *PlaybooksPanel) SelectedTags() []string {
	if p.activeTags == "" {
		return nil
	}
	var tags []string
	for _, t := range strings.Split(p.activeTags, ",") {
		if s := strings.TrimSpace(t); s != "" {
			tags = append(tags, s)
		}
	}
	return tags
}

// SelectByName moves the cursor to the playbook whose name matches.
func (p *PlaybooksPanel) SelectByName(name string) {
	for i, pb := range p.playbooks {
		if pb.Name == name {
			p.cursor = i
			return
		}
	}
}

func (p *PlaybooksPanel) SelectedPlaybook() *core.Playbook {
	if p.cursor < len(p.playbooks) {
		return p.playbooks[p.cursor]
	}
	return nil
}

func (p *PlaybooksPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.focused {
		return nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "j", "down":
		if p.cursor < len(p.playbooks)-1 {
			p.cursor++
		}
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
	case "g":
		p.cursor = 0
	case "G":
		if len(p.playbooks) > 0 {
			p.cursor = len(p.playbooks) - 1
		}
	case "c":
		p.checkMode = !p.checkMode
	case "d":
		p.diffMode = !p.diffMode
	case "enter", "r":
		if pb := p.SelectedPlaybook(); pb != nil {
			return func() tea.Msg {
				return RunRequestMsg{
					Playbook: pb,
					Limit:    p.limit,
					Check:    p.checkMode,
					Diff:     p.diffMode,
					Tags:     p.activeTags,
				}
			}
		}
	}
	return nil
}

func (p *PlaybooksPanel) View() string {
	if len(p.playbooks) == 0 {
		return mutedText("No playbooks found.\nPlace *.yml files in your project directory.")
	}

	var sb strings.Builder

	// ── Active option badges ───────────────────────────────────────────────
	var badges []string
	if p.checkMode {
		badges = append(badges, flagStyle.Render("✓check"))
	}
	if p.diffMode {
		badges = append(badges, flagStyle.Render("±diff"))
	}
	if p.limit != "" {
		badges = append(badges, limitStyle.Render("⊢ "+truncateBadge(p.limit, 14)))
	}
	if p.activeTags != "" {
		badges = append(badges, tagsStyle.Render("# "+truncateBadge(p.activeTags, 14)))
	}
	if p.extraVarsRaw != "" {
		badges = append(badges, extraVarsStyle.Render("-e "+truncateBadge(p.extraVarsRaw, 12)))
	}
	if len(badges) > 0 {
		sb.WriteString(strings.Join(badges, " ") + "\n")
	}

	// ── Playbook list ──────────────────────────────────────────────────────
	// Each non-selected entry = 1 line. Selected entry = 3 lines (name + hosts + tags/path).
	// Reserve space for the detail block of the selected item.
	detailLines := 2 // hosts row + tags/path row for selected item
	contentH := p.height - 4 - len(badges) - detailLines
	if contentH < 1 {
		contentH = 1
	}
	start := 0
	if p.cursor >= contentH {
		start = p.cursor - contentH + 1
	}
	end := start + contentH
	if end > len(p.playbooks) {
		end = len(p.playbooks)
	}

	for i := start; i < end; i++ {
		pb := p.playbooks[i]
		selected := i == p.cursor && p.focused

		if selected {
			// ── Selected: name row ────────────────────────────────────────
			sb.WriteString(pbSelectedStyle.Render("▶ "+truncateBadge(pb.Name, p.width-4)) + "\n")

			// ── Hosts row ────────────────────────────────────────────────
			if len(pb.Hosts) > 0 {
				hostLabels := make([]string, 0, len(pb.Hosts))
				for _, h := range pb.Hosts {
					hostLabels = append(hostLabels, cleanHost(h))
				}
				hostsStr := strings.Join(hostLabels, ", ")
				hostsStr = truncateBadge(hostsStr, p.width-8)
				sb.WriteString(pbHostsStyle.Render("  hosts: "+hostsStr) + "\n")
			} else {
				sb.WriteString(pbHostsStyle.Render("  hosts: (not set)") + "\n")
			}

			// ── Tags / path row ───────────────────────────────────────────
			if len(pb.Tags) > 0 {
				tagStr := strings.Join(pb.Tags, ", ")
				tagStr = truncateBadge(tagStr, p.width-8)
				sb.WriteString(pbTagLineStyle.Render("  tags: "+tagStr) + "\n")
			} else {
				// Show short path hint when no tags.
				shortPath := pb.Path
				if len(shortPath) > p.width-10 {
					shortPath = "…" + shortPath[len(shortPath)-(p.width-11):]
				}
				sb.WriteString(pbPathStyle.Render("  "+shortPath) + "\n")
			}
		} else {
			// ── Normal row: name + hosts summary on one line ──────────────
			hostsHint := ""
			if len(pb.Hosts) > 0 {
				labels := make([]string, 0, len(pb.Hosts))
				for _, h := range pb.Hosts {
					labels = append(labels, cleanHost(h))
				}
				hostsHint = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#374151")).
					Render("  [" + truncateBadge(strings.Join(labels, ","), 16) + "]")
			}
			name := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF")).
				Render("  " + truncateBadge(pb.Name, p.width-20))
			sb.WriteString(name + hostsHint + "\n")
		}
	}

	return sb.String()
}

// cleanHost converts a raw Ansible hosts value into a human-friendly label.
// Jinja2 expressions like {{ target | default('all') }} become $target.
func cleanHost(h string) string {
	trimmed := strings.TrimSpace(h)
	if !strings.Contains(trimmed, "{{") {
		return trimmed
	}
	// Extract the variable name from {{ varname | ... }}
	inner := trimmed
	inner = strings.TrimPrefix(inner, "{{")
	inner = strings.TrimSuffix(inner, "}}")
	inner = strings.TrimSpace(inner)
	// Drop any filters (pipe onwards)
	if idx := strings.Index(inner, "|"); idx >= 0 {
		inner = inner[:idx]
	}
	varName := strings.TrimSpace(inner)
	if varName == "" {
		return "(dynamic)"
	}
	return "$" + varName
}

func truncateBadge(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

var (
	pbSelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#06B6D4")).
			Bold(true).
			Background(lipgloss.Color("#1F2937"))

	pbTagLineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)

	flagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)

	limitStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)

	tagsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#06B6D4")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)

	extraVarsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)

	pbHostsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")).
			Italic(true)

	pbPathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151")).
			Italic(true)
)
