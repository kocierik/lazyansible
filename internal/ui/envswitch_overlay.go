package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/inventory"
)

// EnvSwitchMsg is sent when the user picks a new inventory.
type EnvSwitchMsg struct{ Path string }

// EnvEntry describes a discovered inventory file.
type EnvEntry struct {
	Path       string
	Name       string
	HostCount  int
	GroupCount int
	Active     bool
}

// EnvSwitchOverlay lets the user pick an inventory file at runtime.
type EnvSwitchOverlay struct {
	entries    []EnvEntry
	cursor     int
	activePath string
	workDir    string
	width      int
	height     int
}

func newEnvSwitchOverlay(width, height int) *EnvSwitchOverlay {
	return &EnvSwitchOverlay{width: width, height: height}
}

// Scan discovers inventory files in workDir and parses them for host/group counts.
func (o *EnvSwitchOverlay) Scan(workDir, activePath string) {
	o.workDir = workDir
	o.activePath = activePath
	o.cursor = 0
	o.entries = nil

	paths := discoverAllInventories(workDir)
	for _, p := range paths {
		e := EnvEntry{
			Path:   p,
			Name:   inventoryDisplayName(p, workDir),
			Active: p == activePath,
		}
		// Quick parse for counts.
		if inv, err := inventory.Parse(p); err == nil {
			e.HostCount = len(inv.Hosts)
			e.GroupCount = len(inv.Groups)
		}
		o.entries = append(o.entries, e)
		if e.Active {
			o.cursor = len(o.entries) - 1
		}
	}
}

func (o *EnvSwitchOverlay) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "j", "down":
		if o.cursor < len(o.entries)-1 {
			o.cursor++
		}
	case "k", "up":
		if o.cursor > 0 {
			o.cursor--
		}
	case "enter":
		if o.cursor < len(o.entries) {
			return func() tea.Msg {
				return EnvSwitchMsg{Path: o.entries[o.cursor].Path}
			}
		}
	}
	return nil
}

func (o *EnvSwitchOverlay) View() string {
	boxW := min(o.width-8, 70)
	boxH := min(o.height-6, 20)

	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Switch Environment") + "\n\n")

	if len(o.entries) == 0 {
		sb.WriteString(overlayMutedStyle.Render("No inventory files found in:\n  "+o.workDir) + "\n")
		sb.WriteString("\n" + overlayHintStyle.Render("[esc] close"))
		return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
	}

	contentH := boxH - 7
	start := 0
	if o.cursor >= contentH {
		start = o.cursor - contentH + 1
	}
	end := start + contentH
	if end > len(o.entries) {
		end = len(o.entries)
	}

	for i := start; i < end; i++ {
		e := o.entries[i]
		selected := i == o.cursor

		// Active indicator.
		dot := "○ "
		dotStyle := overlayMutedStyle
		if e.Active {
			dot = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render("● ")
			dotStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E"))
			_ = dotStyle
		}

		counts := fmt.Sprintf("%2d hosts  %2d groups", e.HostCount, e.GroupCount)
		nameW := boxW - len(counts) - 8
		name := truncateStr(e.Name, nameW)
		line := dot + fmt.Sprintf("%-*s  %s", nameW, name, counts)

		if selected {
			sb.WriteString(overlaySelectedStyle.Render(line) + "\n")
		} else {
			sb.WriteString(overlayItemStyle.Render(line) + "\n")
		}
	}

	// Current active.
	active := "none"
	if o.activePath != "" {
		active = inventoryDisplayName(o.activePath, o.workDir)
	}
	sb.WriteString("\n" + overlayLabelStyle.Render("Current: ") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render(active))
	sb.WriteString("\n\n" + overlayHintStyle.Render("[j/k] navigate  [enter] switch  [esc] cancel"))

	return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// discoverAllInventories walks the directory tree for inventory files.
func discoverAllInventories(dir string) []string {
	known := map[string]bool{}
	var found []string

	add := func(p string) {
		abs, err := filepath.Abs(p)
		if err != nil || known[abs] {
			return
		}
		if _, err := os.Stat(abs); err == nil {
			known[abs] = true
			found = append(found, abs)
		}
	}

	// Standard names at root.
	for _, name := range []string{"inventory", "hosts", "inventory.ini", "inventory.yaml", "inventory.yml"} {
		add(filepath.Join(dir, name))
	}

	// Scan inventories/ subdirectory.
	invDir := filepath.Join(dir, "inventories")
	if entries, err := os.ReadDir(invDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				add(filepath.Join(invDir, e.Name()))
			}
		}
	}

	// Also walk one level deep for any .ini/.yaml inventory files.
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext == ".ini" || ext == ".yaml" || ext == ".yml" {
				add(filepath.Join(dir, e.Name()))
			}
		}
	}

	return found
}

func inventoryDisplayName(path, workDir string) string {
	rel, err := filepath.Rel(workDir, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}
