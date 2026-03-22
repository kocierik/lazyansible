package panels

import (
	"fmt"
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
		return panelTitle("Playbooks") + mutedText("No playbooks found.\nPlace *.yml files in your project directory.")
	}

	var sb strings.Builder
	sb.WriteString(panelTitle("Playbooks"))

	// ── Active option badges ──
	var badges []string
	if p.checkMode {
		badges = append(badges, flagStyle.Render("--check"))
	}
	if p.diffMode {
		badges = append(badges, flagStyle.Render("--diff"))
	}
	if p.limit != "" {
		badges = append(badges, limitStyle.Render("⊢ "+p.limit))
	}
	if p.activeTags != "" {
		badges = append(badges, tagsStyle.Render("# "+truncateBadge(p.activeTags, 18)))
	}
	if p.extraVarsRaw != "" {
		badges = append(badges, extraVarsStyle.Render("-e "+truncateBadge(p.extraVarsRaw, 16)))
	}
	if len(badges) > 0 {
		sb.WriteString(strings.Join(badges, " ") + "\n")
	}

	// ── Playbook list ──
	// Reserve 2 extra lines for the tags detail of the selected entry.
	contentH := p.height - 6 - len(badges)
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
		selected := i == p.cursor

		hostsStr := ""
		if len(pb.Hosts) > 0 {
			hostsStr = fmt.Sprintf(" [%s]", strings.Join(pb.Hosts, ","))
		}
		text := pb.Name + hostsStr

		if selected && p.focused {
			sb.WriteString(pbSelectedStyle.Render("▶ "+text) + "\n")
			// Show tags inline under the selected playbook.
			if len(pb.Tags) > 0 {
				tagStr := strings.Join(pb.Tags, ", ")
				if len([]rune(tagStr)) > p.width-6 {
					tagStr = string([]rune(tagStr)[:p.width-7]) + "…"
				}
				sb.WriteString(pbTagLineStyle.Render("  # "+tagStr) + "\n")
			} else {
				sb.WriteString(mutedText("  (no tags)") + "\n")
			}
		} else {
			sb.WriteString(pbItemStyle.Render("  "+text) + "\n")
		}
	}

	return sb.String()
}

func truncateBadge(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

var (
	pbItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

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
)
