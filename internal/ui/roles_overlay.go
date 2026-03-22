package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/roles"
)

// RoleRunMsg is sent when the user requests running a role.
type RoleRunMsg struct {
	RolePath  string
	RoleName  string
	Inventory string
	Limit     string
}

// RolesOverlay is a two-pane overlay: role list (left) + details (right).
type RolesOverlay struct {
	roles     []*roles.Role
	cursor    int
	pane      int // 0 = list, 1 = detail
	detailOff int // scroll offset in detail pane
	filter    textinput.Model
	filtering bool

	limit     string
	inventory string
	width     int
	height    int
	err       error
}

func newRolesOverlay(width, height int) *RolesOverlay {
	ti := textinput.New()
	ti.Placeholder = "filter roles…"
	ti.Width = 20

	return &RolesOverlay{width: width, height: height, filter: ti}
}

func (o *RolesOverlay) Load(rolesDir, inventory, limit string) {
	o.inventory = inventory
	o.limit = limit
	o.cursor = 0
	o.detailOff = 0
	o.pane = 0

	r, err := roles.Scan(rolesDir)
	o.roles = r
	o.err = err
}

func (o *RolesOverlay) visible() []*roles.Role {
	if !o.filtering || o.filter.Value() == "" {
		return o.roles
	}
	q := strings.ToLower(o.filter.Value())
	var out []*roles.Role
	for _, r := range o.roles {
		if strings.Contains(strings.ToLower(r.Name), q) {
			out = append(out, r)
		}
	}
	return out
}

func (o *RolesOverlay) selected() *roles.Role {
	vis := o.visible()
	if o.cursor < len(vis) {
		return vis[o.cursor]
	}
	return nil
}

func (o *RolesOverlay) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		if o.filtering {
			var cmd tea.Cmd
			o.filter, cmd = o.filter.Update(msg)
			return cmd
		}
		return nil
	}

	switch key.String() {
	case "/":
		o.filtering = !o.filtering
		if o.filtering {
			o.filter.Focus()
		} else {
			o.filter.Blur()
			o.filter.SetValue("")
		}
		return nil

	case "tab":
		if o.pane == 0 {
			o.pane = 1
		} else {
			o.pane = 0
		}
		return nil

	case "enter":
		if o.pane == 0 {
			if r := o.selected(); r != nil {
				return func() tea.Msg {
					return RoleRunMsg{
						RolePath:  r.Path,
						RoleName:  r.Name,
						Inventory: o.inventory,
						Limit:     o.limit,
					}
				}
			}
		}
		return nil
	}

	// Navigation
	if o.pane == 0 {
		vis := o.visible()
		switch key.String() {
		case "j", "down":
			if o.cursor < len(vis)-1 {
				o.cursor++
				o.detailOff = 0
			}
		case "k", "up":
			if o.cursor > 0 {
				o.cursor--
				o.detailOff = 0
			}
		case "g":
			o.cursor = 0
		case "G":
			if len(vis) > 0 {
				o.cursor = len(vis) - 1
			}
		default:
			if o.filtering {
				var cmd tea.Cmd
				o.filter, cmd = o.filter.Update(msg)
				return cmd
			}
		}
	} else {
		switch key.String() {
		case "j", "down":
			o.detailOff++
		case "k", "up":
			if o.detailOff > 0 {
				o.detailOff--
			}
		case "g":
			o.detailOff = 0
		}
	}
	return nil
}

func (o *RolesOverlay) View() string {
	boxW := min(o.width-4, 90)
	boxH := min(o.height-2, 36)

	if o.err != nil {
		content := overlayTitleStyle.Render("Role Browser") + "\n\n" +
			overlayMutedStyle.Render("Error: "+o.err.Error()) + "\n\n" +
			overlayHintStyle.Render("[esc] close")
		return overlayBoxStyle.Width(boxW).Height(boxH).Render(content)
	}

	if len(o.roles) == 0 {
		content := overlayTitleStyle.Render("Role Browser") + "\n\n" +
			overlayMutedStyle.Render("No roles found.\nCreate roles in a roles/ directory.") + "\n\n" +
			overlayHintStyle.Render("[esc] close")
		return overlayBoxStyle.Width(boxW).Height(boxH).Render(content)
	}

	listW := 26
	detailW := boxW - listW - 5 // 5 = box padding + separator

	listPane := o.renderList(listW, boxH-5)
	detailPane := o.renderDetail(detailW, boxH-5)

	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(listW).Height(boxH-6).Render(listPane),
		lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#374151")).
			PaddingLeft(1).
			Width(detailW).Height(boxH-6).Render(detailPane),
	)

	title := overlayTitleStyle.Render("Role Browser")
	if o.filtering {
		title += "  " + overlayLabelStyle.Render("filter: ") + o.filter.View()
	}

	hint := "[j/k] navigate  [tab] switch pane  [enter] run role  [/] filter  [esc] close"
	if o.selected() != nil && o.limit != "" {
		hint = fmt.Sprintf("[enter] run on: %s  ", o.limit) + hint
	}

	content := title + "\n\n" + panes + "\n\n" + overlayHintStyle.Render(hint)
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(content)
}

func (o *RolesOverlay) renderList(w, h int) string {
	vis := o.visible()
	var sb strings.Builder

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563")).
		Render(fmt.Sprintf("Roles (%d)", len(vis)))
	sb.WriteString(header + "\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#1F2937")).
		Render(strings.Repeat("─", w)) + "\n")

	start := 0
	if o.cursor >= h-2 {
		start = o.cursor - h + 3
	}
	end := start + h - 2
	if end > len(vis) {
		end = len(vis)
	}

	for i := start; i < end; i++ {
		r := vis[i]
		count := fmt.Sprintf("(%d)", len(r.Tasks))
		nameW := w - len(count) - 2
		name := truncateStr(r.Name, nameW)
		line := fmt.Sprintf("%-*s %s", nameW, name, count)

		focused := o.pane == 0
		if i == o.cursor && focused {
			sb.WriteString(overlaySelectedStyle.Render(line) + "\n")
		} else if i == o.cursor {
			sb.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")).Bold(true).Render("▶ "+name+" "+count) + "\n")
		} else {
			sb.WriteString(overlayItemStyle.Render(line) + "\n")
		}
	}
	return sb.String()
}

func (o *RolesOverlay) renderDetail(w, h int) string {
	r := o.selected()
	if r == nil {
		return overlayMutedStyle.Render("Select a role →")
	}

	var lines []string

	// Name + desc.
	lines = append(lines, roleNameStyle.Render(r.Name))
	if r.Desc != "" {
		lines = append(lines, overlayMutedStyle.Render(r.Desc))
	}
	lines = append(lines, "")

	// Tasks.
	lines = append(lines, overlayLabelStyle.Render(fmt.Sprintf("Tasks (%d):", len(r.Tasks))))
	for _, t := range r.Tasks {
		mod := ""
		if t.Module != "" {
			mod = lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).Render("  [" + t.Module + "]")
		}
		name := truncateStr(t.Name, w-len(t.Module)-6)
		lines = append(lines, overlayItemStyle.Render("  "+name)+mod)
	}

	// Defaults.
	if len(r.Defaults) > 0 {
		lines = append(lines, "")
		lines = append(lines, overlayLabelStyle.Render(fmt.Sprintf("Defaults (%d):", len(r.Defaults))))
		keys := make([]string, 0, len(r.Defaults))
		for k := range r.Defaults {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			val := truncateStr(r.Defaults[k], w-len(k)-4)
			lines = append(lines, overlayItemStyle.Render(fmt.Sprintf("  %-20s %s", k, val)))
		}
	}

	// Handlers.
	if len(r.Handlers) > 0 {
		lines = append(lines, "")
		lines = append(lines, overlayLabelStyle.Render(fmt.Sprintf("Handlers (%d):", len(r.Handlers))))
		for _, name := range r.Handlers {
			lines = append(lines, overlayItemStyle.Render("  ↺ "+name))
		}
	}

	// Dependencies.
	if len(r.Deps) > 0 {
		lines = append(lines, "")
		lines = append(lines, overlayLabelStyle.Render("Dependencies:"))
		for _, dep := range r.Deps {
			lines = append(lines, overlayItemStyle.Render("  → "+dep))
		}
	}

	// Apply scroll offset.
	if o.detailOff > len(lines)-h {
		o.detailOff = max(0, len(lines)-h)
	}
	start := o.detailOff
	end := start + h
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]

	return strings.Join(visible, "\n")
}

// GenerateTempPlaybook creates a temporary playbook YAML that applies the role.
// The caller must delete the file after use.
func GenerateTempPlaybook(roleName, rolePath, hosts string) (string, error) {
	if hosts == "" {
		hosts = "all"
	}
	// We need to point to the role by absolute path using roles_path trick.
	rolesDir := filepath.Dir(rolePath)
	content := fmt.Sprintf(`---
- name: "lazyansible: run role %s"
  hosts: %s
  gather_facts: true
  roles:
    - role: %s
`, roleName, hosts, roleName)

	f, err := os.CreateTemp("", "lazyansible-role-*.yml")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return "", err
	}
	_ = rolesDir // used by caller to set ANSIBLE_ROLES_PATH env var
	return f.Name(), nil
}

var roleNameStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#7C3AED")).
	Bold(true)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
