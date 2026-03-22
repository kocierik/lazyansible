// Package ui contains the top-level Bubble Tea application model.
package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kocierik/lazyansible/internal/core"
	"github.com/kocierik/lazyansible/internal/history"
	"github.com/kocierik/lazyansible/internal/inventory"
	"github.com/kocierik/lazyansible/internal/runner"
	"github.com/kocierik/lazyansible/internal/ui/panels"
	"github.com/kocierik/lazyansible/internal/vault"
)

const version = "0.4.0"

// AppMode tracks which overlay (if any) is currently shown.
type AppMode int

const (
	AppModeNormal AppMode = iota
	AppModeVarsBrowser
	AppModeAdHoc
	AppModeExtraVars
	AppModeTagsBrowser
	AppModeVault
	AppModeHistory
	AppModeRoles
	AppModeEnvSwitch
	AppModeSSHProfile
	AppModeHelp
)

// Config holds the launch-time configuration.
type Config struct {
	InventoryPath string
	PlaybookDir   string
	WorkDir       string
}

// App is the root Bubble Tea model.
type App struct {
	config    Config
	program   *tea.Program
	ctx       context.Context
	cancelRun context.CancelFunc

	width  int
	height int

	// Panel models.
	invPanel    *panels.InventoryPanel
	pbPanel     *panels.PlaybooksPanel
	statusPanel *panels.StatusPanel
	logsPanel   *panels.LogsPanel

	focused core.Panel
	mode    AppMode

	// v0.2 overlays.
	varsOverlay      *VarsOverlay
	adhocOverlay     *AdHocOverlay
	extraVarsOverlay *ExtraVarsOverlay
	tagsOverlay      *TagsOverlay

	// v0.3 overlays.
	vaultOverlay   *VaultOverlay
	historyOverlay *HistoryOverlay

	// v0.4 overlays.
	rolesOverlay      *RolesOverlay
	envSwitchOverlay  *EnvSwitchOverlay
	sshProfileOverlay *SSHProfileOverlay

	// Run state.
	inventory    *core.Inventory
	playbooks    []*core.Playbook
	running      bool
	statusMsg    string
	extraVarsRaw string

	// v0.3 state.
	vaultPassword     string          // current vault password (cleared after run)
	vaultPasswordFile string          // temp file path (cleaned up after run)
	retryHosts        []string        // failed hosts from last run, for retry
	runRecord         *history.Record // in-progress record, saved on finish

	// v0.4 state.
	sshExtraVars   string // applied SSH profile extra-vars
	tempPlaybook   string // temp role-runner playbook (cleaned up after run)
	logsFullscreen bool   // Z toggles logs to full height

	err error
}

// New creates a new App with the given configuration.
func New(cfg Config) *App {
	a := &App{
		config:  cfg,
		focused: core.PanelInventory,
		mode:    AppModeNormal,
		ctx:     context.Background(),
	}

	a.invPanel = panels.NewInventoryPanel(nil, 0, 0)
	a.pbPanel = panels.NewPlaybooksPanel(nil, 0, 0)
	a.statusPanel = panels.NewStatusPanel(0, 0)
	a.logsPanel = panels.NewLogsPanel(0, 0)

	a.varsOverlay = newVarsOverlay(0, 0)
	a.adhocOverlay = newAdHocOverlay(0, 0)
	a.extraVarsOverlay = newExtraVarsOverlay(0, 0)
	a.tagsOverlay = newTagsOverlay(0, 0)
	a.vaultOverlay = newVaultOverlay(0, 0)
	a.historyOverlay = newHistoryOverlay(0, 0)
	a.rolesOverlay = newRolesOverlay(0, 0)
	a.envSwitchOverlay = newEnvSwitchOverlay(0, 0)
	a.sshProfileOverlay = newSSHProfileOverlay(0, 0)

	a.updateFocus()
	return a
}

func (a *App) SetProgram(p *tea.Program) { a.program = p }

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		loadInventoryCmd(a.config),
		loadPlaybooksCmd(a.config),
		a.scanVaultCmd(),
	)
}

// ─── Messages ────────────────────────────────────────────────────────────────

type inventoryLoadedMsg struct{ inv *core.Inventory }
type playbooksLoadedMsg struct{ pbs []*core.Playbook }
type vaultScanDoneMsg struct{ hasVault bool }
type errMsg struct{ err error }

// ─── Update ──────────────────────────────────────────────────────────────────

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		a.width = sz.Width
		a.height = sz.Height
		a.resizePanels()
		return a, nil
	}

	if a.mode != AppModeNormal && a.mode != AppModeHelp {
		return a.updateOverlay(msg)
	}

	switch msg := msg.(type) {

	case inventoryLoadedMsg:
		a.inventory = msg.inv
		a.invPanel.SetInventory(msg.inv)
		a.statusMsg = fmt.Sprintf("Inventory: %d hosts, %d groups",
			len(msg.inv.Hosts), len(msg.inv.Groups))

	case playbooksLoadedMsg:
		a.playbooks = msg.pbs
		a.pbPanel.SetPlaybooks(msg.pbs)
		a.statusMsg = fmt.Sprintf("Found %d playbooks", len(msg.pbs))

	case vaultScanDoneMsg:
		if msg.hasVault {
			a.statusMsg = "⚠ Vault-encrypted files detected — press V to set password"
		}

	case runner.LogMsg:
		a.logsPanel.AddLine(msg.Line)

	case runner.HostStatusMsg:
		a.statusPanel.UpdateHost(msg.Host, msg.Status, msg.Task)

	case runner.RunFinishedMsg:
		return a, a.handleRunFinished(msg)

	case panels.RunRequestMsg:
		return a, a.startRun(msg)

	case EnvSwitchMsg:
		return a, a.switchInventory(msg.Path)

	case SSHProfileAppliedMsg:
		a.sshExtraVars = msg.ExtraVars
		if msg.ExtraVars != "" {
			a.statusMsg = "SSH profile applied (will be used on next run)"
		} else {
			a.statusMsg = "SSH profile cleared"
		}
		a.mode = AppModeNormal
		return a, nil

	case errMsg:
		a.statusMsg = "Error: " + msg.err.Error()

	case tea.KeyMsg:
		return a.updateNormalKeys(msg)
	}

	return a, a.delegateToPanel(msg)
}

func (a *App) updateNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if a.cancelRun != nil {
			a.cancelRun()
		}
		a.cleanupVaultFile()
		a.cleanupTempPlaybook()
		return a, tea.Quit

	case "?":
		a.mode = AppModeHelp
		return a, nil

	case "tab":
		a.cycleFocus(1)
		return a, nil
	case "shift+tab":
		a.cycleFocus(-1)
		return a, nil

	case "1":
		a.focused = core.PanelInventory
		a.updateFocus()
	case "2":
		a.focused = core.PanelPlaybooks
		a.updateFocus()
	case "3":
		a.focused = core.PanelStatus
		a.updateFocus()
	case "4":
		a.focused = core.PanelLogs
		a.updateFocus()

	case "ctrl+l":
		a.logsPanel.Clear()

	case "Z":
		a.logsFullscreen = !a.logsFullscreen
		a.resizePanels()

	// ── v0.2 overlays ─────────────────────────────────────────────────────

	case "v":
		if a.focused == core.PanelInventory && a.inventory != nil {
			if host := a.invPanel.SelectedHost(); host != "" {
				if h, ok := a.inventory.Hosts[host]; ok {
					a.varsOverlay.SetHost(h)
					a.mode = AppModeVarsBrowser
					return a, nil
				}
			} else if group := a.invPanel.SelectedGroup(); group != "" {
				if g, ok := a.inventory.Groups[group]; ok {
					a.varsOverlay.SetGroup(g)
					a.mode = AppModeVarsBrowser
					return a, nil
				}
			}
		}

	case "!":
		target := ""
		if a.focused == core.PanelInventory {
			if h := a.invPanel.SelectedHost(); h != "" {
				target = h
			} else if g := a.invPanel.SelectedGroup(); g != "" {
				target = g
			}
		}
		if target == "" {
			target = a.pbPanel.CurrentLimit()
		}
		a.adhocOverlay.SetTarget(target, a.config.InventoryPath)
		a.mode = AppModeAdHoc
		return a, nil

	case "e":
		if a.focused == core.PanelPlaybooks {
			a.extraVarsOverlay.SetCurrent(a.extraVarsRaw)
			a.mode = AppModeExtraVars
			return a, nil
		}

	case "t":
		if a.focused == core.PanelPlaybooks {
			if pb := a.pbPanel.SelectedPlaybook(); pb != nil {
				a.tagsOverlay.SetTags(pb.Tags)
				a.mode = AppModeTagsBrowser
				return a, nil
			}
		}

	// ── v0.3 overlays ─────────────────────────────────────────────────────

	case "V":
		a.vaultOverlay.Reset()
		a.mode = AppModeVault
		return a, nil

	case "H":
		a.historyOverlay.Reload()
		a.mode = AppModeHistory
		return a, nil

	case "R":
		if len(a.retryHosts) > 0 && !a.running {
			limit := strings.Join(a.retryHosts, ",")
			a.pbPanel.SetLimit(limit)
			a.statusMsg = fmt.Sprintf("Retry limit set: %s", limit)
		} else if a.running {
			a.statusMsg = "Cannot retry while a run is in progress"
		} else {
			a.statusMsg = "No failed hosts to retry"
		}

	// ── v0.4 overlays ─────────────────────────────────────────────────────

	case "O":
		// Role browser.
		rolesDir := filepath.Join(a.config.WorkDir, "roles")
		limit := a.pbPanel.CurrentLimit()
		a.rolesOverlay.Load(rolesDir, a.config.InventoryPath, limit)
		a.mode = AppModeRoles
		return a, nil

	case "N":
		// Environment / inventory switcher.
		a.envSwitchOverlay.Scan(a.config.WorkDir, a.config.InventoryPath)
		a.mode = AppModeEnvSwitch
		return a, nil

	case "P":
		// SSH profile manager.
		a.sshProfileOverlay.loadProfiles()
		a.mode = AppModeSSHProfile
		return a, nil

	case "enter":
		if a.focused == core.PanelInventory {
			if host := a.invPanel.SelectedHost(); host != "" {
				a.pbPanel.SetLimit(host)
				a.statusMsg = "Limit → " + host
			} else if group := a.invPanel.SelectedGroup(); group != "" {
				a.pbPanel.SetLimit(group)
				a.statusMsg = "Limit → " + group + " (group)"
			}
			return a, nil
		}
	}

	return a, a.delegateToPanel(msg)
}

func (a *App) updateOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		a.mode = AppModeNormal
		return a, nil
	}

	if a.mode == AppModeHelp {
		if _, ok := msg.(tea.KeyMsg); ok {
			a.mode = AppModeNormal
		}
		return a, nil
	}

	var cmd tea.Cmd
	switch a.mode {

	case AppModeVarsBrowser:
		cmd = a.varsOverlay.Update(msg)

	case AppModeAdHoc:
		cmd = a.adhocOverlay.Update(msg)
		if cmd != nil {
			if result, ok := evalCmd(cmd).(AdHocRunMsg); ok {
				a.mode = AppModeNormal
				return a, a.startAdHoc(result.Opts)
			}
		}

	case AppModeExtraVars:
		cmd = a.extraVarsOverlay.Update(msg)
		if cmd != nil {
			if ev, ok := evalCmd(cmd).(ExtraVarsConfirmedMsg); ok {
				a.extraVarsRaw = ev.Raw
				a.pbPanel.SetExtraVars(ev.Raw)
				if ev.Raw != "" {
					a.statusMsg = "Extra vars: " + ev.Raw
				} else {
					a.statusMsg = "Extra vars cleared"
				}
				a.mode = AppModeNormal
				return a, nil
			}
		}

	case AppModeTagsBrowser:
		cmd = a.tagsOverlay.Update(msg)
		if cmd != nil {
			if tc, ok := evalCmd(cmd).(TagsConfirmedMsg); ok {
				a.pbPanel.SetActiveTags(tc.Tags)
				if tc.Tags != "" {
					a.statusMsg = "Tags → " + tc.Tags
				} else {
					a.statusMsg = "Tags cleared"
				}
				a.mode = AppModeNormal
				return a, nil
			}
		}

	case AppModeVault:
		cmd = a.vaultOverlay.Update(msg)
		if cmd != nil {
			if vp, ok := evalCmd(cmd).(VaultPasswordMsg); ok {
				a.vaultPassword = vp.Password
				if vp.Password != "" {
					a.statusMsg = "Vault password set (will be used on next run)"
				} else {
					a.statusMsg = "Vault password cleared"
				}
				a.mode = AppModeNormal
				return a, nil
			}
		}

	case AppModeHistory:
		cmd = a.historyOverlay.Update(msg)
		if cmd != nil {
			if hr, ok := evalCmd(cmd).(HistoryRunMsg); ok {
				a.mode = AppModeNormal
				return a, a.startRunFromHistory(hr.Record)
			}
		}

	case AppModeRoles:
		cmd = a.rolesOverlay.Update(msg)
		if cmd != nil {
			if rr, ok := evalCmd(cmd).(RoleRunMsg); ok {
				a.mode = AppModeNormal
				return a, a.startRoleRun(rr)
			}
		}

	case AppModeEnvSwitch:
		cmd = a.envSwitchOverlay.Update(msg)
		if cmd != nil {
			if es, ok := evalCmd(cmd).(EnvSwitchMsg); ok {
				a.mode = AppModeNormal
				return a, a.switchInventory(es.Path)
			}
		}

	case AppModeSSHProfile:
		cmd = a.sshProfileOverlay.Update(msg)
		if cmd != nil {
			result := evalCmd(cmd)
			if sp, ok := result.(SSHProfileAppliedMsg); ok {
				a.sshExtraVars = sp.ExtraVars
				if sp.ExtraVars != "" {
					a.statusMsg = "SSH profile applied"
				} else {
					a.statusMsg = "SSH profile cleared"
				}
				a.mode = AppModeNormal
				return a, nil
			}
		}
	}

	return a, cmd
}

// evalCmd executes a Cmd synchronously and returns the Msg. Used only for
// overlay result inspection (not for long-running cmds).
func evalCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func (a *App) delegateToPanel(msg tea.Msg) tea.Cmd {
	switch a.focused {
	case core.PanelInventory:
		return a.invPanel.Update(msg)
	case core.PanelPlaybooks:
		return a.pbPanel.Update(msg)
	case core.PanelStatus:
		return a.statusPanel.Update(msg)
	case core.PanelLogs:
		return a.logsPanel.Update(msg)
	}
	return nil
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (a *App) View() string {
	if a.width == 0 {
		return "Initializing…"
	}

	switch a.mode {
	case AppModeHelp:
		return a.renderOverlay(a.helpContent())
	case AppModeVarsBrowser:
		return a.renderOverlay(a.varsOverlay.View())
	case AppModeAdHoc:
		return a.renderOverlay(a.adhocOverlay.View())
	case AppModeExtraVars:
		return a.renderOverlay(a.extraVarsOverlay.View())
	case AppModeTagsBrowser:
		return a.renderOverlay(a.tagsOverlay.View())
	case AppModeVault:
		return a.renderOverlay(a.vaultOverlay.View())
	case AppModeHistory:
		return a.renderOverlay(a.historyOverlay.View())
	case AppModeRoles:
		return a.renderOverlay(a.rolesOverlay.View())
	case AppModeEnvSwitch:
		return a.renderOverlay(a.envSwitchOverlay.View())
	case AppModeSSHProfile:
		return a.renderOverlay(a.sshProfileOverlay.View())
	}

	return a.baseView()
}

// topPanelHeight is the fixed row count reserved for the inventory/playbooks/status row.
// Keeping this constant avoids any dynamic arithmetic that could introduce off-by-one
// errors when log lines stream in. Adjust if you want more/less space at the top.
const topPanelHeight = 14

func (a *App) baseView() string {
	// Layout: header(1) + topRow(topPanelHeight) + logsBox(rest) + statusBar(1) = a.height
	available := a.height - 2 // subtract header and statusBar
	header := strings.TrimRight(a.renderHeader(), "\n")
	statusBar := strings.TrimRight(a.renderStatusBar(), "\n")

	var logsView string
	if a.logsFullscreen {
		logsView = strings.TrimRight(
			a.wrapPanel(a.logsPanel.View(), a.width, available, true), "\n")
		return forceHeight(header+"\n"+logsView+"\n"+statusBar, a.height)
	}

	// Top row gets a fixed height; logs get everything else.
	topH := topPanelHeight
	if topH > available-4 {
		topH = available - 4 // guarantee at least 4 rows for logs
	}
	botH := available - topH

	invW := a.width / 4
	pbW := a.width / 4
	statusW := a.width - invW - pbW

	invView := strings.TrimRight(
		a.wrapPanel(a.invPanel.View(), invW, topH, a.focused == core.PanelInventory), "\n")
	pbView := strings.TrimRight(
		a.wrapPanel(a.pbPanel.View(), pbW, topH, a.focused == core.PanelPlaybooks), "\n")
	statusView := strings.TrimRight(
		a.wrapPanel(a.statusPanel.View(), statusW, topH, a.focused == core.PanelStatus), "\n")
	topRow := strings.TrimRight(
		lipgloss.JoinHorizontal(lipgloss.Top, invView, pbView, statusView), "\n")
	logsView = strings.TrimRight(
		a.wrapPanel(a.logsPanel.View(), a.width, botH, a.focused == core.PanelLogs), "\n")

	// Assemble and guarantee exactly a.height rows so alt-screen never scrolls.
	return forceHeight(header+"\n"+topRow+"\n"+logsView+"\n"+statusBar, a.height)
}

func (a *App) renderOverlay(content string) string {
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, content)
}

// wrapPanel wraps content in a bordered panel box of exactly w×h terminal cells.
// lipgloss Width/Height set the inner content size; border adds 2h + (border+pad) 4w.
func (a *App) wrapPanel(content string, w, h int, focused bool) string {
	// Inner dimensions: width-4 (border 2 + padding 2), height-2 (border top+bottom).
	innerW := w - 4
	innerH := h - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}
	// Hard-clip the content to innerH lines BEFORE handing it to lipgloss.
	// lipgloss.Height() is a MINIMUM, not a maximum: if content is taller than
	// innerH the panel overflows, pushing the header off screen.
	content = clipLines(content, innerH)

	style := panelStyle.Width(innerW).Height(innerH)
	if focused {
		style = panelFocusedStyle.Width(innerW).Height(innerH)
	}
	return style.Render(content)
}

// clipLines truncates s to at most maxLines newline-separated lines.
func clipLines(s string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	count := 0
	for i, ch := range s {
		if ch == '\n' {
			count++
			if count >= maxLines {
				return s[:i]
			}
		}
	}
	return s
}

// forceHeight ensures the view string is EXACTLY h terminal rows.
//
// It always preserves the first line (header) and the last line (status bar).
// Any surplus lines are trimmed from the end of the middle section (logs).
// Any shortage is padded with empty lines before the status bar.
// This is the only reliable guard against Bubble Tea alt-screen scroll-off.
func forceHeight(view string, h int) string {
	if h <= 0 {
		return view
	}
	lines := strings.Split(view, "\n")
	// Strip a single trailing empty element produced by a final "\n".
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	switch {
	case len(lines) == h:
		return strings.Join(lines, "\n")

	case len(lines) > h:
		// Too tall: remove excess lines from the bottom of the middle section
		// so that the header (lines[0]) and status bar (lines[last]) are kept.
		excess := len(lines) - h
		header := lines[0]
		statusBar := lines[len(lines)-1]
		middle := lines[1 : len(lines)-1]
		if excess >= len(middle) {
			middle = nil
		} else {
			middle = middle[:len(middle)-excess]
		}
		out := make([]string, 0, h)
		out = append(out, header)
		out = append(out, middle...)
		out = append(out, statusBar)
		return strings.Join(out, "\n")

	default:
		// Too short: pad with blank lines before the status bar.
		diff := h - len(lines)
		statusBar := lines[len(lines)-1]
		lines = lines[:len(lines)-1]
		for i := 0; i < diff; i++ {
			lines = append(lines, "")
		}
		lines = append(lines, statusBar)
		return strings.Join(lines, "\n")
	}
}

func (a *App) renderHeader() string {
	vaultBadge := ""
	if a.vaultPassword != "" {
		vaultBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).Bold(true).Render("  🔐")
	}
	retryBadge := ""
	if len(a.retryHosts) > 0 {
		retryBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).Bold(true).
			Render(fmt.Sprintf("  ↺ %d failed", len(a.retryHosts)))
	}
	sshBadge := ""
	if a.sshExtraVars != "" {
		sshBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")).Bold(true).Render("  🔑")
	}

	left := titleStyle.Render("lazyansible") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563")).Render(" v"+version) +
		vaultBadge + retryBadge + sshBadge

	var runIndicator string
	if a.running {
		runIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#06B6D4")).Bold(true).Render("  ▶ RUNNING")
	}

	tabs := strings.Join([]string{
		tabHint("1", "Inventory"),
		tabHint("2", "Playbooks"),
		tabHint("3", "Status"),
		tabHint("4", "Logs"),
	}, "  ")

	right := tabs + runIndicator
	gap := a.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gap < 1 {
		gap = 1
	}

	return headerStyle.Width(a.width).Render(left + strings.Repeat(" ", gap) + right)
}

func tabHint(key, label string) string {
	return keyStyle.Render("["+key+"]") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(label)
}

func (a *App) renderStatusBar() string {
	// Panel-specific shortcuts.
	var hints []string
	switch a.focused {
	case core.PanelInventory:
		hints = []string{"[v]vars", "[!]adhoc", "[enter]limit"}
	case core.PanelPlaybooks:
		hints = []string{"[r]run", "[t]tags", "[e]vars", "[c]check", "[d]diff"}
	case core.PanelLogs:
		hints = []string{"[k/j]scroll", "[G]end", "[T]time", "[ctrl+l]clear"}
	default:
		hints = []string{"[r]run"}
	}
	// Compact global hints – never more than a handful to prevent line wrap.
	if a.logsFullscreen {
		hints = append(hints, "[Z]normal")
	} else {
		hints = append(hints, "[Z]zoom")
	}
	if len(a.retryHosts) > 0 {
		hints = append(hints, "[R]retry")
	}
	hints = append(hints, "[?]help", "[q]quit")

	// Build the hint string and hard-cap it so it NEVER wraps to a second line.
	hintStr := strings.Join(hints, "  ")
	maxHintW := a.width * 2 / 3
	if maxHintW < 20 {
		maxHintW = 20
	}
	if lipgloss.Width(hintStr) > maxHintW {
		hintStr = truncateStr(hintStr, maxHintW)
	}
	hint := helpStyle.Render(hintStr)

	msgW := a.width - lipgloss.Width(hint) - 4
	if msgW < 0 {
		msgW = 0
	}
	msg := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D1D5DB")).
		Render(truncateStr(a.statusMsg, msgW))

	gap := a.width - lipgloss.Width(msg) - lipgloss.Width(hint) - 4
	if gap < 1 {
		gap = 1
	}

	return lipgloss.NewStyle().
		Background(lipgloss.Color("#111827")).
		Width(a.width).
		Render(msg + strings.Repeat(" ", gap) + hint)
}

func (a *App) helpContent() string {
	help := `
 lazyansible v` + version + ` – keyboard shortcuts
 ──────────────────────────────────────────────────────

 Navigation
   tab / shift+tab     cycle panels
   1 2 3 4             jump to panel
   j / k               move down / up
   g / G               jump to top / bottom

 Inventory panel
   enter / space       expand / collapse group
   enter on host       set as playbook run limit
   v                   variable browser for host/group
   !                   ad-hoc command runner

 Playbooks panel
   r / enter           run selected playbook
   c                   toggle --check mode
   d                   toggle --diff mode
   t                   tags browser (multi-select + filter)
   e                   set --extra-vars

 Logs panel
   j / k               scroll down / up
   ctrl+d / ctrl+u     half-page scroll
   G                   jump to bottom
   T                   toggle timestamps
   ctrl+l              clear logs
   Z                   toggle fullscreen logs (hides top panels)

 Global (v0.3)
   V                   set Ansible Vault password
   H                   browse run history (re-run from history)
   R                   retry failed hosts from last run

 Global (v0.4)
   O                   role browser (view tasks, defaults, run role)
   N                   switch environment / inventory at runtime
   P                   SSH profile manager (apply connection params)
   --diff output       +/- lines auto-highlighted in logs panel

 Global
   ?                   toggle this help
   q / ctrl+c          quit (cancels active run)
`
	return overlayBoxStyle.
		Width(min(a.width-8, 64)).
		Render(lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(help))
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (a *App) cycleFocus(dir int) {
	order := []core.Panel{
		core.PanelInventory,
		core.PanelPlaybooks,
		core.PanelStatus,
		core.PanelLogs,
	}
	idx := 0
	for i, p := range order {
		if p == a.focused {
			idx = i
			break
		}
	}
	a.focused = order[(idx+dir+len(order))%len(order)]
	a.updateFocus()
}

func (a *App) updateFocus() {
	a.invPanel.SetFocused(a.focused == core.PanelInventory)
	a.pbPanel.SetFocused(a.focused == core.PanelPlaybooks)
	a.statusPanel.SetFocused(a.focused == core.PanelStatus)
	a.logsPanel.SetFocused(a.focused == core.PanelLogs)
}

func (a *App) resizePanels() {
	available := a.height - 2
	invW := a.width / 4
	pbW := a.width / 4
	statusW := a.width - invW - pbW

	if a.logsFullscreen {
		a.invPanel.SetSize(invW-4, 1)
		a.pbPanel.SetSize(pbW-4, 1)
		a.statusPanel.SetSize(statusW-4, 1)
		a.logsPanel.SetSize(a.width-4, available-2)
	} else {
		topH := topPanelHeight
		if topH > available-4 {
			topH = available - 4
		}
		botH := available - topH
		a.invPanel.SetSize(invW-4, topH-2)
		a.pbPanel.SetSize(pbW-4, topH-2)
		a.statusPanel.SetSize(statusW-4, topH-2)
		a.logsPanel.SetSize(a.width-4, botH-2)
	}

	a.varsOverlay.width = a.width
	a.varsOverlay.height = a.height
	a.adhocOverlay.width = a.width
	a.adhocOverlay.height = a.height
	a.extraVarsOverlay.width = a.width
	a.extraVarsOverlay.height = a.height
	a.tagsOverlay.width = a.width
	a.tagsOverlay.height = a.height
	a.vaultOverlay.width = a.width
	a.vaultOverlay.height = a.height
	a.historyOverlay.width = a.width
	a.historyOverlay.height = a.height
	a.rolesOverlay.width = a.width
	a.rolesOverlay.height = a.height
	a.envSwitchOverlay.width = a.width
	a.envSwitchOverlay.height = a.height
	a.sshProfileOverlay.width = a.width
	a.sshProfileOverlay.height = a.height
}

// ─── Run lifecycle ────────────────────────────────────────────────────────────

func (a *App) startRun(req panels.RunRequestMsg) tea.Cmd {
	if a.running {
		a.statusMsg = "A run is already in progress"
		return nil
	}
	if err := runner.CheckBinary(); err != nil {
		a.statusMsg = err.Error()
		return nil
	}

	a.running = true
	a.retryHosts = nil
	a.statusPanel.Reset()
	a.statusPanel.SetRunning(true)
	a.logsPanel.Clear()
	a.statusMsg = fmt.Sprintf("Running %s…", req.Playbook.Name)

	// Write vault password to temp file if set.
	vaultFile := ""
	if a.vaultPassword != "" {
		f, err := vault.WriteTempPassword(a.vaultPassword)
		if err == nil {
			vaultFile = f
			a.vaultPasswordFile = f
		}
	}

	// Build run record.
	a.runRecord = &history.Record{
		ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
		Kind:         "playbook",
		PlaybookName: req.Playbook.Name,
		PlaybookPath: req.Playbook.Path,
		Inventory:    a.config.InventoryPath,
		Limit:        req.Limit,
		Tags:         req.Tags,
		ExtraVars:    a.extraVarsRaw,
		CheckMode:    req.Check,
		DiffMode:     req.Diff,
		StartTime:    time.Now(),
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelRun = cancel

	// Merge SSH profile vars with explicit extra-vars (explicit takes precedence).
	mergedExtra := a.extraVarsRaw
	if a.sshExtraVars != "" {
		if mergedExtra != "" {
			mergedExtra = a.sshExtraVars + " " + mergedExtra
		} else {
			mergedExtra = a.sshExtraVars
		}
	}

	opts := core.RunOptions{
		Playbook:          req.Playbook.Path,
		Inventory:         a.config.InventoryPath,
		Limit:             req.Limit,
		Tags:              req.Tags,
		CheckMode:         req.Check,
		DiffMode:          req.Diff,
		ExtraVarsRaw:      mergedExtra,
		VaultPasswordFile: vaultFile,
	}

	sendFn := func(m tea.Msg) {
		if a.program != nil {
			a.program.Send(m)
		}
	}
	return runner.StreamCmd(ctx, opts, sendFn)
}

func (a *App) startRunFromHistory(r *history.Record) tea.Cmd {
	if a.running {
		a.statusMsg = "A run is already in progress"
		return nil
	}
	if err := runner.CheckBinary(); err != nil {
		a.statusMsg = err.Error()
		return nil
	}

	a.running = true
	a.retryHosts = nil
	a.statusPanel.Reset()
	a.statusPanel.SetRunning(true)
	a.logsPanel.Clear()
	a.statusMsg = fmt.Sprintf("Re-running %s from history…", r.PlaybookName)

	a.runRecord = &history.Record{
		ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
		Kind:         "playbook",
		PlaybookName: r.PlaybookName,
		PlaybookPath: r.PlaybookPath,
		Inventory:    r.Inventory,
		Limit:        r.Limit,
		Tags:         r.Tags,
		ExtraVars:    r.ExtraVars,
		CheckMode:    r.CheckMode,
		DiffMode:     r.DiffMode,
		StartTime:    time.Now(),
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelRun = cancel

	opts := core.RunOptions{
		Playbook:     r.PlaybookPath,
		Inventory:    r.Inventory,
		Limit:        r.Limit,
		Tags:         r.Tags,
		CheckMode:    r.CheckMode,
		DiffMode:     r.DiffMode,
		ExtraVarsRaw: r.ExtraVars,
	}

	sendFn := func(m tea.Msg) {
		if a.program != nil {
			a.program.Send(m)
		}
	}
	return runner.StreamCmd(ctx, opts, sendFn)
}

func (a *App) startAdHoc(opts core.AdHocOptions) tea.Cmd {
	if a.running {
		a.statusMsg = "A run is already in progress"
		return nil
	}
	if err := runner.CheckAdHocBinary(); err != nil {
		a.statusMsg = err.Error()
		return nil
	}

	a.running = true
	a.statusPanel.Reset()
	a.statusPanel.SetRunning(true)
	a.logsPanel.Clear()
	a.statusMsg = fmt.Sprintf("Ad-hoc: ansible %s -m %s", opts.Hosts, opts.Module)

	a.runRecord = &history.Record{
		ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
		Kind:         "adhoc",
		PlaybookName: opts.Module,
		Inventory:    opts.Inventory,
		Limit:        opts.Hosts,
		Module:       opts.Module,
		Args:         opts.Args,
		StartTime:    time.Now(),
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelRun = cancel

	sendFn := func(m tea.Msg) {
		if a.program != nil {
			a.program.Send(m)
		}
	}
	return runner.AdHocStreamCmd(ctx, opts, sendFn)
}

func (a *App) handleRunFinished(msg runner.RunFinishedMsg) tea.Cmd {
	a.running = false
	a.statusPanel.SetRunning(false)
	if a.cancelRun != nil {
		a.cancelRun()
	}
	a.cleanupVaultFile()
	a.cleanupTempPlaybook()

	// Collect failed hosts for retry.
	a.retryHosts = a.statusPanel.FailedHosts()

	// Persist history.
	if a.runRecord != nil {
		a.runRecord.EndTime = time.Now()
		a.runRecord.ExitCode = msg.ExitCode
		a.runRecord.HostStats = a.statusPanel.HostStatsMap()
		go func(r *history.Record) { _ = history.Save(r) }(a.runRecord)
		a.runRecord = nil
	}

	if msg.Err != nil {
		a.statusMsg = "Run error: " + msg.Err.Error()
	} else if msg.ExitCode == 0 {
		a.statusMsg = "Completed successfully ✓"
	} else {
		failStr := ""
		if len(a.retryHosts) > 0 {
			failStr = fmt.Sprintf("  [R] retry %d failed host(s)", len(a.retryHosts))
		}
		a.statusMsg = fmt.Sprintf("Exit code %d%s", msg.ExitCode, failStr)
	}

	return nil
}

// startRoleRun generates a temp playbook that applies the given role and runs it.
func (a *App) startRoleRun(req RoleRunMsg) tea.Cmd {
	if a.running {
		a.statusMsg = "A run is already in progress"
		return nil
	}
	if err := runner.CheckBinary(); err != nil {
		a.statusMsg = err.Error()
		return nil
	}

	hosts := req.Limit
	tmpPB, err := GenerateTempPlaybook(req.RoleName, req.RolePath, hosts)
	if err != nil {
		a.statusMsg = "Failed to create temp playbook: " + err.Error()
		return nil
	}
	a.tempPlaybook = tmpPB

	a.running = true
	a.retryHosts = nil
	a.statusPanel.Reset()
	a.statusPanel.SetRunning(true)
	a.logsPanel.Clear()
	a.statusMsg = fmt.Sprintf("Running role %s…", req.RoleName)

	rolesDir := filepath.Dir(req.RolePath)
	envVars := []string{"ANSIBLE_ROLES_PATH=" + rolesDir}

	a.runRecord = &history.Record{
		ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
		Kind:         "role",
		PlaybookName: "role:" + req.RoleName,
		PlaybookPath: tmpPB,
		Inventory:    req.Inventory,
		Limit:        hosts,
		StartTime:    time.Now(),
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelRun = cancel

	opts := core.RunOptions{
		Playbook:  tmpPB,
		Inventory: req.Inventory,
		Limit:     hosts,
		Env:       envVars,
	}

	sendFn := func(m tea.Msg) {
		if a.program != nil {
			a.program.Send(m)
		}
	}
	return runner.StreamCmd(ctx, opts, sendFn)
}

// switchInventory reloads the inventory from a new path.
func (a *App) switchInventory(path string) tea.Cmd {
	a.config.InventoryPath = path
	a.statusMsg = "Switching to " + filepath.Base(path) + "…"
	return func() tea.Msg {
		inv, err := inventory.Parse(path)
		if err != nil {
			return errMsg{err: fmt.Errorf("parse inventory %s: %w", filepath.Base(path), err)}
		}
		return inventoryLoadedMsg{inv: inv}
	}
}

func (a *App) cleanupVaultFile() {
	if a.vaultPasswordFile != "" {
		_ = os.Remove(a.vaultPasswordFile)
		a.vaultPasswordFile = ""
	}
}

func (a *App) cleanupTempPlaybook() {
	if a.tempPlaybook != "" {
		_ = os.Remove(a.tempPlaybook)
		a.tempPlaybook = ""
	}
}

func (a *App) scanVaultCmd() tea.Cmd {
	workDir := a.config.WorkDir
	return func() tea.Msg {
		files := vault.FindEncryptedFiles(workDir)
		return vaultScanDoneMsg{hasVault: len(files) > 0}
	}
}

func truncateStr(s string, max int) string {
	if max < 1 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// ─── Startup commands ─────────────────────────────────────────────────────────

func loadInventoryCmd(cfg Config) tea.Cmd {
	return func() tea.Msg {
		path := cfg.InventoryPath
		if path == "" {
			paths := inventory.Discover(cfg.WorkDir)
			if len(paths) == 0 {
				return inventoryLoadedMsg{inv: &core.Inventory{
					Hosts:  make(map[string]*core.Host),
					Groups: make(map[string]*core.Group),
				}}
			}
			path = paths[0]
		}
		inv, err := inventory.Parse(path)
		if err != nil {
			return errMsg{err: fmt.Errorf("parse inventory %s: %w", filepath.Base(path), err)}
		}
		return inventoryLoadedMsg{inv: inv}
	}
}

func loadPlaybooksCmd(cfg Config) tea.Cmd {
	return func() tea.Msg {
		dir := cfg.PlaybookDir
		if dir == "" {
			dir = cfg.WorkDir
		}
		pbs, err := inventory.DiscoverPlaybooks(dir)
		if err != nil {
			return errMsg{err: fmt.Errorf("discover playbooks: %w", err)}
		}
		return playbooksLoadedMsg{pbs: pbs}
	}
}
