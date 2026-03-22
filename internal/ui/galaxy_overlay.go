package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/galaxy"
)

// galaxyLoadedMsg is sent when the list of roles/collections has been fetched.
type galaxyLoadedMsg struct {
	roles       []galaxy.Item
	collections []galaxy.Item
	rolesErr    error
	colsErr     error
}

// galaxyInstallDoneMsg is sent when an install command completes.
type galaxyInstallDoneMsg struct {
	output string
	err    error
}

type galaxyTab int

const (
	galaxyTabRoles galaxyTab = iota
	galaxyTabCollections
)

type galaxyViewMode int

const (
	galaxyModeList    galaxyViewMode = iota
	galaxyModeInstall                // install name input
	galaxyModeResult                 // shows full install output
)

// GalaxyOverlay browses installed roles/collections and allows installing new ones.
type GalaxyOverlay struct {
	width  int
	height int

	tab  galaxyTab
	mode galaxyViewMode

	roles    []galaxy.Item
	cols     []galaxy.Item
	rolesErr string
	colsErr  string

	cursor  int
	loading bool

	input textinput.Model

	// Inline filter (/ to open, Esc to clear).
	filterActive bool
	filterQuery  string

	// Install result state (persists after load refresh).
	lastInstallName   string
	lastInstallOutput []string // lines of output
	lastInstallOK     bool
	resultScroll      int // scroll offset into lastInstallOutput
}

func newGalaxyOverlay(width, height int) *GalaxyOverlay {
	ti := textinput.New()
	ti.Placeholder = "namespace.role_name  or  namespace.collection"
	ti.CharLimit = 128
	ti.Width = 46
	return &GalaxyOverlay{width: width, height: height, input: ti}
}

// Load triggers fetching the list of roles and collections.
// It does NOT clear the install result so the user can still see it.
func (g *GalaxyOverlay) Load() tea.Cmd {
	g.loading = true
	g.cursor = 0
	return func() tea.Msg {
		roles, rolesErr := galaxy.ListRoles()
		cols, colsErr := galaxy.ListCollections()
		return galaxyLoadedMsg{roles: roles, collections: cols, rolesErr: rolesErr, colsErr: colsErr}
	}
}

func (g *GalaxyOverlay) currentList() []galaxy.Item {
	if g.tab == galaxyTabRoles {
		return g.roles
	}
	return g.cols
}

// filteredList returns items from the current tab that match filterQuery.
func (g *GalaxyOverlay) filteredList() []galaxy.Item {
	all := g.currentList()
	if g.filterQuery == "" {
		return all
	}
	q := strings.ToLower(g.filterQuery)
	var out []galaxy.Item
	for _, item := range all {
		if strings.Contains(strings.ToLower(item.Name), q) ||
			strings.Contains(strings.ToLower(item.Version), q) {
			out = append(out, item)
		}
	}
	return out
}

func (g *GalaxyOverlay) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case galaxyLoadedMsg:
		g.loading = false
		g.roles = m.roles
		g.cols = m.collections
		g.rolesErr = ""
		g.colsErr = ""
		if m.rolesErr != nil {
			g.rolesErr = m.rolesErr.Error()
		}
		if m.colsErr != nil {
			g.colsErr = m.colsErr.Error()
		}
		// After loading, if we just finished an install switch to result view.
		if g.mode != galaxyModeResult {
			g.mode = galaxyModeList
		}
		return nil

	case galaxyInstallDoneMsg:
		g.loading = false
		lines := strings.Split(strings.TrimRight(m.output, "\n"), "\n")
		g.lastInstallOutput = lines
		g.lastInstallOK = m.err == nil
		g.resultScroll = 0
		g.mode = galaxyModeResult
		// Also refresh the list in background.
		return g.Load()
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		if g.mode == galaxyModeInstall {
			var cmd tea.Cmd
			g.input, cmd = g.input.Update(msg)
			return cmd
		}
		return nil
	}

	switch g.mode {
	case galaxyModeInstall:
		return g.updateInstallForm(key)
	case galaxyModeResult:
		return g.updateResult(key)
	default:
		return g.updateList(key)
	}
}

func (g *GalaxyOverlay) updateList(key tea.KeyMsg) tea.Cmd {
	// ── Filter input mode ─────────────────────────────────────────────────
	if g.filterActive {
		switch key.String() {
		case "esc":
			g.filterActive = false
			g.filterQuery = ""
			g.cursor = 0
		case "enter":
			g.filterActive = false
			g.cursor = 0
		case "backspace", "ctrl+h":
			if len(g.filterQuery) > 0 {
				runes := []rune(g.filterQuery)
				g.filterQuery = string(runes[:len(runes)-1])
				g.cursor = 0
			}
		default:
			if key.Type == tea.KeyRunes {
				g.filterQuery += key.String()
				g.cursor = 0
			}
		}
		return nil
	}

	// ── Normal list navigation ────────────────────────────────────────────
	list := g.filteredList()
	switch key.String() {
	case "/":
		g.filterActive = true
		g.filterQuery = ""
		g.cursor = 0
	case "tab":
		if g.tab == galaxyTabRoles {
			g.tab = galaxyTabCollections
		} else {
			g.tab = galaxyTabRoles
		}
		g.cursor = 0
		g.filterQuery = ""
		g.filterActive = false
	case "j", "down":
		if g.cursor < len(list)-1 {
			g.cursor++
		}
	case "k", "up":
		if g.cursor > 0 {
			g.cursor--
		}
	case "g":
		g.cursor = 0
	case "G":
		if len(list) > 0 {
			g.cursor = len(list) - 1
		}
	case "i":
		g.mode = galaxyModeInstall
		g.input.SetValue("")
		g.input.Focus()
	case "r":
		g.filterQuery = ""
		g.filterActive = false
		return g.Load()
	}
	return nil
}

func (g *GalaxyOverlay) updateInstallForm(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "esc":
		g.mode = galaxyModeList
	case "enter":
		name := strings.TrimSpace(g.input.Value())
		if name == "" {
			g.mode = galaxyModeList
			return nil
		}
		g.loading = true
		g.lastInstallName = name
		g.mode = galaxyModeList
		isCol := g.tab == galaxyTabCollections
		return func() tea.Msg {
			var out string
			var err error
			if isCol {
				out, err = galaxy.InstallCollection(name)
			} else {
				out, err = galaxy.InstallRole(name)
			}
			return galaxyInstallDoneMsg{output: out, err: err}
		}
	default:
		var cmd tea.Cmd
		g.input, cmd = g.input.Update(key)
		return cmd
	}
	return nil
}

func (g *GalaxyOverlay) updateResult(key tea.KeyMsg) tea.Cmd {
	maxScroll := len(g.lastInstallOutput) - 1
	if maxScroll < 0 {
		maxScroll = 0
	}
	switch key.String() {
	case "j", "down":
		if g.resultScroll < maxScroll {
			g.resultScroll++
		}
	case "k", "up":
		if g.resultScroll > 0 {
			g.resultScroll--
		}
	case "g":
		g.resultScroll = 0
	case "G":
		g.resultScroll = maxScroll
	case "q", "enter", "esc", " ":
		g.mode = galaxyModeList
	}
	return nil
}

func (g *GalaxyOverlay) View() string {
	boxW := min(g.width-6, 82)
	boxH := min(g.height-4, 34)

	var sb strings.Builder

	// ── Title + tab bar ───────────────────────────────────────────────────
	rolesTab := "  Roles  "
	colsTab := "  Collections  "
	if g.tab == galaxyTabRoles {
		rolesTab = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true).Underline(true).Render(rolesTab)
		colsTab = overlayMutedStyle.Render(colsTab)
	} else {
		rolesTab = overlayMutedStyle.Render(rolesTab)
		colsTab = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true).Underline(true).Render(colsTab)
	}
	sb.WriteString(overlayTitleStyle.Render("Ansible Galaxy") + "   " + rolesTab + colsTab + "\n\n")

	// ── Install input form ────────────────────────────────────────────────
	if g.mode == galaxyModeInstall {
		var what, cmd, example, note string
		if g.tab == galaxyTabCollections {
			what = "collection"
			cmd = "ansible-galaxy collection install <name>"
			example = "e.g.  amazon.aws   community.general   ansible.posix"
			note = "Collections use namespace.collection format"
		} else {
			what = "role"
			cmd = "ansible-galaxy role install <name>"
			example = "e.g.  geerlingguy.docker   geerlingguy.nodejs"
			note = "⚠  For collections (amazon.aws, community.*) switch to the Collections tab first"
		}
		sb.WriteString(overlayLabelStyle.Render(fmt.Sprintf("Install %s:", what)) + "\n")
		sb.WriteString(g.input.View() + "\n\n")
		sb.WriteString(overlayMutedStyle.Render(example) + "\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).Render("runs: "+cmd) + "\n\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render(note) + "\n\n")
		sb.WriteString(overlayHintStyle.Render("[enter] install  [tab] switch to Collections tab  [esc] cancel"))
		return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
	}

	// ── Install result view ───────────────────────────────────────────────
	if g.mode == galaxyModeResult {
		return g.viewResult(boxW, boxH, &sb)
	}

	// ── Loading spinner ───────────────────────────────────────────────────
	if g.loading {
		sb.WriteString(overlayMutedStyle.Render(fmt.Sprintf("Installing %s…", g.lastInstallName)) + "\n")
		sb.WriteString("\n" + overlayHintStyle.Render("[esc] close"))
		return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
	}

	// ── Filter bar (shown in list mode) ──────────────────────────────────
	allList := g.currentList()
	list := g.filteredList()

	errMsg := g.rolesErr
	if g.tab == galaxyTabCollections {
		errMsg = g.colsErr
	}

	// Render the filter row above the table.
	if g.filterActive {
		prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true).Render("/")
		cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true).Render("█")
		queryText := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Bold(true).Render(g.filterQuery)
		count := lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).
			Render(fmt.Sprintf("  %d/%d", len(list), len(allList)))
		hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).Render("  enter·esc")
		sb.WriteString(prompt + queryText + cursor + count + hint + "\n")
	} else if g.filterQuery != "" {
		prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true).Render("/")
		queryText := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Render(g.filterQuery)
		count := lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).
			Render(fmt.Sprintf("  %d/%d match", len(list), len(allList)))
		hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).Render("  esc:clear")
		sb.WriteString(prompt + queryText + count + hint + "\n")
	}

	// ── List view ─────────────────────────────────────────────────────────
	if errMsg != "" {
		sb.WriteString(overlayMutedStyle.Render("Could not load: "+errMsg) + "\n")
	} else if len(allList) == 0 {
		sb.WriteString(overlayMutedStyle.Render("None installed. Press [i] to install one.") + "\n")
	} else if len(list) == 0 {
		sb.WriteString(overlayMutedStyle.Render(fmt.Sprintf("No match for %q — try a different query.", g.filterQuery)) + "\n")
	} else {
		// Reserve extra row when filter bar is shown.
		filterRows := 0
		if g.filterActive || g.filterQuery != "" {
			filterRows = 1
		}
		contentH := boxH - 9 - filterRows
		if contentH < 1 {
			contentH = 1
		}
		start := 0
		if g.cursor >= contentH {
			start = g.cursor - contentH + 1
		}
		end := start + contentH
		if end > len(list) {
			end = len(list)
		}

		header := fmt.Sprintf("  %-38s  %s", "Name", "Version")
		sb.WriteString(overlayLabelStyle.Render(header) + "\n")
		sb.WriteString(overlayMutedStyle.Render(strings.Repeat("─", min(boxW-6, 62))) + "\n")

		for i := start; i < end; i++ {
			item := list[i]
			// Highlight the matching portion of the name when filtering.
			name := item.Name
			if g.filterQuery != "" {
				name = highlightMatch(name, g.filterQuery)
			}
			line := fmt.Sprintf("  %-38s  %s",
				truncateStr(item.Name, 38),
				truncateStr(item.Version, 12),
			)
			if i == g.cursor {
				if g.filterQuery != "" {
					// For the selected+filtered row render name with highlight.
					highlighted := "  " + highlightMatch(truncateStr(item.Name, 38), g.filterQuery)
					ver := overlaySelectedStyle.Render("  " + truncateStr(item.Version, 12))
					sb.WriteString(overlaySelectedStyle.Render(highlighted) + ver + "\n")
				} else {
					sb.WriteString(overlaySelectedStyle.Render(line) + "\n")
				}
			} else {
				if g.filterQuery != "" {
					highlighted := "  " + highlightMatch(truncateStr(item.Name, 38), g.filterQuery)
					ver := overlayItemStyle.Render("  " + truncateStr(item.Version, 12))
					sb.WriteString(overlayItemStyle.Render(highlighted) + ver + "\n")
				} else {
					sb.WriteString(overlayItemStyle.Render(line) + "\n")
				}
			}
			_ = name
		}
		total := len(allList)
		shown := len(list)
		if g.filterQuery != "" {
			sb.WriteString("\n" + overlayMutedStyle.Render(
				fmt.Sprintf("%d of %d match", shown, total)) + "\n")
		} else {
			sb.WriteString("\n" + overlayMutedStyle.Render(
				fmt.Sprintf("%d installed", total)) + "\n")
		}
	}

	hints := "[/] filter  [tab] switch tab  [i] install  [r] refresh  [esc] close"
	if g.filterActive {
		hints = "[type] filter  [enter] confirm  [esc] clear"
	} else if g.filterQuery != "" {
		hints = "[/] edit filter  [esc] clear  [i] install  [r] refresh  [esc] close"
	}
	sb.WriteString("\n" + overlayHintStyle.Render(hints))
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
}

// highlightMatch wraps the first occurrence of query inside text with a
// bright colour so the matching part stands out in the filtered list.
func highlightMatch(text, query string) string {
	lower := strings.ToLower(text)
	q := strings.ToLower(query)
	idx := strings.Index(lower, q)
	if idx < 0 {
		return overlayItemStyle.Render(text)
	}
	before := text[:idx]
	match := text[idx : idx+len(query)]
	after := text[idx+len(query):]
	hl := lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true).Render(match)
	return overlayItemStyle.Render(before) + hl + overlayItemStyle.Render(after)
}

func (g *GalaxyOverlay) viewResult(boxW, boxH int, sb *strings.Builder) string {
	// Header with success/error badge.
	if g.lastInstallOK {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")).Bold(true).
			Render(fmt.Sprintf("✓  %s installed successfully", g.lastInstallName)) + "\n\n")
	} else {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).Bold(true).
			Render(fmt.Sprintf("✗  installation failed: %s", g.lastInstallName)) + "\n\n")
	}

	// Scrollable output area.
	outputH := boxH - 9
	if outputH < 3 {
		outputH = 3
	}

	lines := g.lastInstallOutput
	total := len(lines)
	end := g.resultScroll + outputH
	if end > total {
		end = total
	}
	start := g.resultScroll
	if start > total {
		start = total
	}

	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB"))
	for _, l := range lines[start:end] {
		// Colour key output lines for readability.
		var rendered string
		switch {
		case strings.HasPrefix(l, "- downloading") || strings.HasPrefix(l, "- extracting"):
			rendered = lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4")).Render(l)
		case strings.HasPrefix(l, "- ") && strings.Contains(l, "was installed"):
			rendered = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render(l)
		case strings.HasPrefix(l, "[WARNING]") || strings.HasPrefix(l, "WARNING"):
			rendered = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render(l)
		case strings.HasPrefix(l, "[ERROR]") || strings.HasPrefix(l, "ERROR") ||
			strings.HasPrefix(l, "error") || strings.Contains(l, "fatal"):
			rendered = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render(l)
		default:
			rendered = lineStyle.Render(l)
		}
		sb.WriteString(rendered + "\n")
	}

	// Scroll indicator.
	if total > outputH {
		pct := (g.resultScroll + outputH) * 100 / total
		sb.WriteString(overlayMutedStyle.Render(
			fmt.Sprintf("── %d/%d lines (%d%%) ──", min(end, total), total, pct)) + "\n")
	}

	sb.WriteString("\n" + overlayHintStyle.Render("[j/k] scroll  [enter/esc] back to list"))
	return overlayBoxStyle.Width(boxW).Height(boxH).Render(sb.String())
}
