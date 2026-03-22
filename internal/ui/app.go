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
	"github.com/kocierik/lazyansible/internal/editor"
	"github.com/kocierik/lazyansible/internal/galaxy"
	"github.com/kocierik/lazyansible/internal/history"
	"github.com/kocierik/lazyansible/internal/inventory"
	"github.com/kocierik/lazyansible/internal/notify"
	"github.com/kocierik/lazyansible/internal/runner"
	"github.com/kocierik/lazyansible/internal/runprofiles"
	"github.com/kocierik/lazyansible/internal/ui/panels"
	"github.com/kocierik/lazyansible/internal/vault"
)

// version is set at build time via -ldflags "-X github.com/kocierik/lazyansible/internal/ui.version=x.y.z"
var version = "1.0.0"

// AppMode tracks which overlay (if any) is currently shown.
type AppMode int

const (
	AppModeNormal AppMode = iota
	AppModeAdHoc
	AppModeExtraVars
	AppModeTagsBrowser
	AppModeVault
	AppModeHistory
	AppModeRoles
	AppModeEnvSwitch
	AppModeSSHProfile
	AppModeHelp
	AppModeGalaxy
	AppModeRunProfiles
	AppModePlaybookViewer
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

	// v0.6 overlays.
	galaxyOverlay      *GalaxyOverlay
	runProfilesOverlay *RunProfilesOverlay

	// v0.7 overlays.
	pbViewerOverlay *PlaybookViewerOverlay

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

	// v0.5 state.
	linting    bool   // ansible-lint run in progress
	lastExport string // path of last Markdown export

	// v0.6 state (nothing extra — overlays are self-contained)

	// v0.7 state.
	notifyOnFinish bool // send desktop notification when run completes
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

	a.adhocOverlay = newAdHocOverlay(0, 0)
	a.extraVarsOverlay = newExtraVarsOverlay(0, 0)
	a.tagsOverlay = newTagsOverlay(0, 0)
	a.vaultOverlay = newVaultOverlay(0, 0)
	a.historyOverlay = newHistoryOverlay(0, 0)
	a.rolesOverlay = newRolesOverlay(0, 0)
	a.envSwitchOverlay = newEnvSwitchOverlay(0, 0)
	a.sshProfileOverlay = newSSHProfileOverlay(0, 0)
	a.galaxyOverlay = newGalaxyOverlay(0, 0)
	a.runProfilesOverlay = newRunProfilesOverlay(0, 0)
	a.pbViewerOverlay = newPlaybookViewerOverlay(0, 0)

	a.updateFocus()
	return a
}

func (a *App) SetProgram(p *tea.Program) { a.program = p }

// SetNotifyOnFinish enables desktop notifications at run completion.
func (a *App) SetNotifyOnFinish(v bool) { a.notifyOnFinish = v }

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
type lintFinishedMsg struct{ exitCode int }
type exportDoneMsg struct {
	path string
	err  error
}
type errMsg struct{ err error }

// ─── Update ──────────────────────────────────────────────────────────────────

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		a.width = sz.Width
		a.height = sz.Height
		a.resizePanels()
		return a, nil
	}

	// Mouse events: only handle left-click for panel focus; discard everything
	// else (motion, scroll, hover) so mouse movement never triggers a full
	// rerender and never reaches text-input overlays (which would cause lag).
	if mouse, ok := msg.(tea.MouseMsg); ok {
		if mouse.Action == tea.MouseActionPress && mouse.Button == tea.MouseButtonLeft {
			if a.mode == AppModeNormal {
				a.handleMouseClick(mouse.X, mouse.Y)
			}
		}
		return a, nil
	}

	// Editor done: reload content and show status.
	if ed, ok := msg.(editor.DoneMsg); ok {
		if ed.Err != nil {
			a.statusMsg = "Editor error: " + ed.Err.Error()
		} else {
			// Reload the viewer if it's open and still shows the same file.
			a.pbViewerOverlay.Reload()
			a.statusMsg = fmt.Sprintf("Saved: %s", ed.Path)
		}
		return a, nil
	}

	// Two-step open: resolved path → launch editor.
	if ep, ok := msg.(editorOpenPathMsg); ok {
		return a, editor.Open(ep.path)
	}

	if a.mode != AppModeNormal {
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

	case lintFinishedMsg:
		a.linting = false
		if msg.exitCode == 0 {
			a.statusMsg = "ansible-lint: no issues found ✓"
		} else {
			a.statusMsg = fmt.Sprintf("ansible-lint: issues found (exit %d) — see logs", msg.exitCode)
		}

	case exportDoneMsg:
		if msg.err != nil {
			a.statusMsg = "Export failed: " + msg.err.Error()
		} else {
			a.lastExport = msg.path
			a.statusMsg = "Exported → " + msg.path
		}

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

	case galaxyLoadedMsg:
		return a, a.galaxyOverlay.Update(msg)

	case galaxyInstallDoneMsg:
		return a, a.galaxyOverlay.Update(msg)

	case RunProfileLoadMsg:
		a.applyRunProfile(msg.Profile)
		a.mode = AppModeNormal
		a.statusMsg = fmt.Sprintf("Profile loaded: %s", msg.Profile.Name)
		return a, nil

	case errMsg:
		a.statusMsg = "Error: " + msg.err.Error()

	case tea.KeyMsg:
		return a.updateNormalKeys(msg)
	}

	return a, a.delegateToPanel(msg)
}

func (a *App) updateNormalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When the log panel's search bar is active, forward ALL key events to it
	// so global shortcuts (q, V, etc.) don't fire during text input.
	if a.focused == core.PanelLogs && a.logsPanel.SearchActive() {
		return a, a.delegateToPanel(msg)
	}

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

	case "L":
		// Ansible-lint on the selected playbook.
		if a.running || a.linting {
			a.statusMsg = "Cannot lint while a run is in progress"
			return a, nil
		}
		if pb := a.pbPanel.SelectedPlaybook(); pb != nil {
			if err := runner.CheckLintBinary(); err != nil {
				a.statusMsg = err.Error()
				return a, nil
			}
			a.linting = true
			a.logsPanel.Clear()
			a.statusMsg = fmt.Sprintf("Linting %s…", pb.Name)
			ctx, cancel := context.WithCancel(a.ctx)
			a.cancelRun = cancel
			sendFn := func(m tea.Msg) {
				if a.program != nil {
					a.program.Send(m)
				}
			}
			return a, func() tea.Msg {
				msg := runner.LintCmd(ctx, pb.Path, sendFn)()
				if rf, ok := msg.(runner.RunFinishedMsg); ok {
					return lintFinishedMsg{exitCode: rf.ExitCode}
				}
				return lintFinishedMsg{exitCode: -1}
			}
		}
		a.statusMsg = "No playbook selected"

	case "X":
		// Export run summary as Markdown.
		lines := a.logsPanel.Lines()
		if len(lines) == 0 {
			a.statusMsg = "Nothing to export yet"
			return a, nil
		}
		rec := a.runRecord // may be nil if run already finished
		workDir := a.config.WorkDir
		return a, func() tea.Msg {
			path, err := exportRunMarkdown(workDir, rec, lines)
			return exportDoneMsg{path: path, err: err}
		}

	// ── v0.7 features ─────────────────────────────────────────────────────

	case "I":
		// Live reload inventory + playbooks without restarting.
		a.statusMsg = "Reloading inventory and playbooks…"
		return a, tea.Batch(loadInventoryCmd(a.config), loadPlaybooksCmd(a.config))

	case " ":
		// Playbook viewer: show YAML source of selected playbook.
		if a.focused == core.PanelPlaybooks {
			if pb := a.pbPanel.SelectedPlaybook(); pb != nil {
				a.pbViewerOverlay.Load(pb.Name, pb.Path)
				a.mode = AppModePlaybookViewer
				return a, nil
			}
		}

	case "E":
		// Open selected playbook directly in $EDITOR (skip viewer overlay).
		if a.focused == core.PanelPlaybooks {
			if pb := a.pbPanel.SelectedPlaybook(); pb != nil {
				return a, editor.Open(pb.Path)
			}
		}
		// Also allow editing from inventory: open host_vars / group_vars file.
		if a.focused == core.PanelInventory {
			if host := a.invPanel.SelectedHost(); host != "" {
				return a, editorOpenVarsFile(inventoryBaseDir(a.config), "host_vars", host)
			} else if group := a.invPanel.SelectedGroup(); group != "" {
				return a, editorOpenVarsFile(inventoryBaseDir(a.config), "group_vars", group)
			}
		}

	// ── v0.6 overlays ─────────────────────────────────────────────────────

	case "A":
		// Ansible Galaxy browser.
		if err := checkGalaxyBinary(); err != nil {
			a.statusMsg = err.Error()
			return a, nil
		}
		a.mode = AppModeGalaxy
		return a, a.galaxyOverlay.Load()

	case "F":
		// Run profiles.
		a.runProfilesOverlay.reload()
		// Snapshot current state for save.
		pb := a.pbPanel.SelectedPlaybook()
		pbName := ""
		if pb != nil {
			pbName = pb.Name
		}
		a.runProfilesOverlay.SetSnapshot(
			pbName,
			a.pbPanel.CurrentLimit(),
			a.pbPanel.SelectedTags(),
			a.extraVarsRaw,
			a.pbPanel.CheckMode(),
			a.pbPanel.DiffMode(),
			a.config.InventoryPath,
		)
		a.mode = AppModeRunProfiles
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

	// isEnter is true when the message is the Enter key — the only key that
	// overlays use to emit confirmation commands. evalCmd is called ONLY in
	// this case so that the blink-cursor command returned by textinput on
	// every regular keypress is never executed synchronously (it would block
	// ~530 ms and make typing feel laggy).
	isEnter := isEnterKey(msg)

	var cmd tea.Cmd
	switch a.mode {

	case AppModeAdHoc:
		cmd = a.adhocOverlay.Update(msg)
		if isEnter && cmd != nil {
			if result, ok := evalCmd(cmd).(AdHocRunMsg); ok {
				a.mode = AppModeNormal
				return a, a.startAdHoc(result.Opts)
			}
		}

	case AppModeExtraVars:
		cmd = a.extraVarsOverlay.Update(msg)
		if isEnter && cmd != nil {
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
		if isEnter && cmd != nil {
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
		if isEnter && cmd != nil {
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
		// History is a pure list — no textinput, evalCmd is safe on all keys.
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
		// SSHProfile has a form with textinputs; only eval on Enter.
		if isEnter && cmd != nil {
			if sp, ok := evalCmd(cmd).(SSHProfileAppliedMsg); ok {
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

	case AppModeGalaxy:
		// Galaxy overlay emits async commands (load/install) — returned as-is.
		cmd = a.galaxyOverlay.Update(msg)

	case AppModeRunProfiles:
		cmd = a.runProfilesOverlay.Update(msg)
		if isEnter && cmd != nil {
			if rp, ok := evalCmd(cmd).(RunProfileLoadMsg); ok {
				a.applyRunProfile(rp.Profile)
				a.mode = AppModeNormal
				a.statusMsg = fmt.Sprintf("Profile loaded: %s", rp.Profile.Name)
				return a, nil
			}
		}

	case AppModePlaybookViewer:
		// Viewer has no textinput; evalCmd is safe on all keys.
		cmd = a.pbViewerOverlay.Update(msg)
		if cmd != nil {
			if _, ok := evalCmd(cmd).(pbViewerCloseMsg); ok {
				a.mode = AppModeNormal
				return a, nil
			}
		}
	}

	return a, cmd
}

// isEnterKey reports whether msg is a key-press of Enter.
func isEnterKey(msg tea.Msg) bool {
	k, ok := msg.(tea.KeyMsg)
	return ok && k.String() == "enter"
}

// evalCmd executes a Cmd synchronously and returns the Msg.
// Only call this when you are certain the command returns immediately
// (e.g. overlay confirmation commands triggered by Enter).
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
	case AppModeGalaxy:
		return a.renderOverlay(a.galaxyOverlay.View())
	case AppModeRunProfiles:
		return a.renderOverlay(a.runProfilesOverlay.View())
	case AppModePlaybookViewer:
		return a.renderOverlay(a.pbViewerOverlay.View())
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
			a.wrapPanel(a.logsPanel.View(), a.width, available, true, "Logs"), "\n")
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
		a.wrapPanel(a.invPanel.View(), invW, topH, a.focused == core.PanelInventory, "Inventory"), "\n")
	pbView := strings.TrimRight(
		a.wrapPanel(a.pbPanel.View(), pbW, topH, a.focused == core.PanelPlaybooks, "Playbooks"), "\n")
	statusView := strings.TrimRight(
		a.wrapPanel(a.statusPanel.View(), statusW, topH, a.focused == core.PanelStatus, "Status"), "\n")
	topRow := strings.TrimRight(
		lipgloss.JoinHorizontal(lipgloss.Top, invView, pbView, statusView), "\n")
	logsView = strings.TrimRight(
		a.wrapPanel(a.logsPanel.View(), a.width, botH, a.focused == core.PanelLogs, "Logs"), "\n")

	// Assemble and guarantee exactly a.height rows so alt-screen never scrolls.
	return forceHeight(header+"\n"+topRow+"\n"+logsView+"\n"+statusBar, a.height)
}

func (a *App) renderOverlay(content string) string {
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, content)
}

// wrapPanel wraps content in a bordered panel box of exactly w×h terminal cells.
// title is injected into the top border line (e.g. "╭─ Inventory ───╮").
func (a *App) wrapPanel(content string, w, h int, focused bool, title string) string {
	// Inner dimensions: width-4 (border 2 + padding 2), height-2 (border top+bottom).
	innerW := w - 4
	innerH := h - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}
	content = clipLines(content, innerH)

	style := panelStyle.Width(innerW).Height(innerH)
	if focused {
		style = panelFocusedStyle.Width(innerW).Height(innerH)
	}
	rendered := style.Render(content)

	// Inject the panel title into the top border line.
	if title != "" {
		rendered = injectBorderTitle(rendered, title, focused)
	}
	return rendered
}

// injectBorderTitle replaces the start of the top border dash-run with the title text.
// Input:  ╭──────────────────────────────╮
// Output: ╭─ Inventory ─────────────────╮
func injectBorderTitle(box, title string, focused bool) string {
	lines := strings.SplitN(box, "\n", 2)
	if len(lines) == 0 {
		return box
	}
	topLine := lines[0]

	// Strip ANSI codes to measure and find the dash run.
	plain := stripANSI(topLine)
	// The rounded top border starts with ╭ (3 bytes) followed by ─ runes.
	// We want to replace "╭─" with "╭─ Title ─".
	titleText := " " + title + " "

	// Find position of first ─ in the plain string.
	dashStart := strings.Index(plain, "─")
	if dashStart < 0 {
		return box // not a bordered box we recognise
	}

	// Build the replacement top line by working on the visual plain text,
	// then re-applying the border colour.
	var titleStyled string
	if focused {
		titleStyled = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).Bold(true).Render(titleText)
	} else {
		titleStyled = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4B5563")).Render(titleText)
	}

	// The corner ╭ in the plain string is 3 bytes wide (UTF-8).
	// Replace "╭─" with "╭─<titleStyled>".
	// We need the total visual width to stay the same, so we count how many
	// dashes the title consumes and remove them from the plain run.
	titleVisualW := lipgloss.Width(titleStyled)
	// Count how many ─ runes are in the original plain top border.
	totalDashes := strings.Count(plain, "─")
	// We emit: ╭ + ─ (1 explicit) + titleStyled + remainingDashes + ╮
	// To match original width (corners + totalDashes) we need:
	//   1 + 1 + titleVisualW + remainingDashes + 1 == 2 + totalDashes
	//   remainingDashes = totalDashes - titleVisualW - 1
	remainingDashes := totalDashes - titleVisualW - 1
	if remainingDashes < 1 {
		// Title too wide to fit; skip injection.
		return box
	}

	// Rebuild the border colour.
	borderColor := colorBorder
	if focused {
		borderColor = colorBorderFocus
	}
	cornerStyle := lipgloss.NewStyle().Foreground(borderColor)
	dashStyle := lipgloss.NewStyle().Foreground(borderColor)

	newTop := cornerStyle.Render("╭") +
		dashStyle.Render("─") +
		titleStyled +
		dashStyle.Render(strings.Repeat("─", remainingDashes)) +
		cornerStyle.Render("╮")

	if len(lines) == 1 {
		return newTop
	}
	return newTop + "\n" + lines[1]
}

// stripANSI removes ANSI escape sequences from s.
func stripANSI(s string) string {
	var out strings.Builder
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
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
	// ── Style atoms ───────────────────────────────────────────────────────
	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#06B6D4")).Bold(true)
	verStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#334155"))
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1E293B"))
	badgeMuted := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#64748B"))

	// ── Logo section ──────────────────────────────────────────────────────
	logo := logoStyle.Render("⚡ lazyansible") +
		verStyle.Render(" v"+version)

	// ── State badges ──────────────────────────────────────────────────────
	var badges []string
	if a.vaultPassword != "" {
		badges = append(badges, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).Bold(true).
			Background(lipgloss.Color("#1C1A0E")).
			Padding(0, 1).Render("🔐 vault"))
	}
	if a.sshExtraVars != "" {
		badges = append(badges, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22C55E")).Bold(true).
			Background(lipgloss.Color("#0A1A0E")).
			Padding(0, 1).Render("🔑 ssh"))
	}
	if len(a.retryHosts) > 0 {
		badges = append(badges, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).Bold(true).
			Background(lipgloss.Color("#1A0A0A")).
			Padding(0, 1).Render(fmt.Sprintf("↺ %d failed", len(a.retryHosts))))
	}
	if a.running {
		badges = append(badges, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111827")).Bold(true).
			Background(lipgloss.Color("#06B6D4")).
			Padding(0, 1).Render("▶ RUNNING"))
	} else if a.linting {
		badges = append(badges, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111827")).Bold(true).
			Background(lipgloss.Color("#F59E0B")).
			Padding(0, 1).Render("⚑ LINTING"))
	}
	badgeStr := ""
	if len(badges) > 0 {
		badgeStr = "  " + strings.Join(badges, " ")
	}

	// ── Inventory name (right-aligned) ────────────────────────────────────
	invName := ""
	if a.config.InventoryPath != "" {
		invName = badgeMuted.Render("  " + filepath.Base(a.config.InventoryPath))
	}

	left := logo + badgeStr

	// ── Tab bar ───────────────────────────────────────────────────────────
	type tab struct {
		num   string
		label string
		panel core.Panel
	}
	tabs := []tab{
		{"1", "Inventory", core.PanelInventory},
		{"2", "Playbooks", core.PanelPlaybooks},
		{"3", "Status", core.PanelStatus},
		{"4", "Logs", core.PanelLogs},
	}

	var tabParts []string
	for _, t := range tabs {
		numPart := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#475569")).
			Render(t.num + " ")
		if t.panel == a.focused {
			// Active tab: bright, underlined appearance
			active := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#06B6D4")).Bold(true).
				Background(lipgloss.Color("#0F2233")).
				Padding(0, 1).
				Render(t.num + " " + t.label)
			tabParts = append(tabParts, active)
		} else {
			inactive := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#64748B")).
				Padding(0, 1).
				Render(numPart + t.label)
			tabParts = append(tabParts, inactive)
		}
	}
	tabBar := strings.Join(tabParts, sepStyle.Render(" "))

	right := tabBar + invName

	gap := a.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	return headerStyle.
		Width(a.width).
		Padding(0, 1).
		Render(left + strings.Repeat(" ", gap) + right)
}

func (a *App) renderStatusBar() string {
	// ── Styles ────────────────────────────────────────────────────────────
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#111827")).
		Background(lipgloss.Color("#6B7280")).
		Bold(true).
		Padding(0, 1)
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF"))
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151"))
	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D1D5DB"))

	pill := func(k, d string) string {
		return keyStyle.Render(k) + descStyle.Render(" "+d)
	}
	sep := sepStyle.Render("  │  ")

	// ── Context-specific groups ───────────────────────────────────────────
	var contextGroup []string
	switch a.focused {
	case core.PanelInventory:
		contextGroup = []string{
			pill("enter", "set limit"),
			pill("v", "vars"),
			pill("E", "edit vars"),
			pill("!", "ad-hoc"),
			pill("I", "reload"),
			pill("N", "env"),
		}
	case core.PanelPlaybooks:
		checkMark := ""
		if a.pbPanel.CheckMode() {
			checkMark = "✓"
		}
		diffMark := ""
		if a.pbPanel.DiffMode() {
			diffMark = "✓"
		}
		contextGroup = []string{
			pill("r", "run"),
			pill("space", "view"),
			pill("E", "edit"),
			pill("t", "tags"),
			pill("c", "check"+checkMark),
			pill("d", "diff"+diffMark),
			pill("L", "lint"),
			pill("F", "profiles"),
		}
	case core.PanelLogs:
		if a.logsFullscreen {
			contextGroup = []string{
				pill("/", "search"),
				pill("f", "filter"),
				pill("j/k", "scroll"),
				pill("G", "end"),
				pill("T", "time"),
				pill("ctrl+l", "clear"),
				pill("X", "export"),
				pill("Z", "normal"),
			}
		} else {
			contextGroup = []string{
				pill("/", "search"),
				pill("f", "filter"),
				pill("j/k", "scroll"),
				pill("T", "time"),
				pill("ctrl+l", "clear"),
				pill("Z", "fullscreen"),
			}
		}
	case core.PanelStatus:
		contextGroup = []string{
			pill("r", "run"),
		}
		if len(a.retryHosts) > 0 {
			contextGroup = append(contextGroup, pill("R", "retry"))
		}
	}

	// ── Global shortcuts (always visible) ─────────────────────────────────
	globalGroup := []string{
		pill("A", "galaxy"),
		pill("H", "history"),
		pill("V", "vault"),
		pill("?", "help"),
		pill("q", "quit"),
	}

	// ── Assemble right-side hint string ───────────────────────────────────
	allPills := append(contextGroup, sep)
	allPills = append(allPills, globalGroup...)

	// Hard-cap width so it never wraps.
	maxHintW := a.width * 3 / 4
	if maxHintW < 30 {
		maxHintW = 30
	}
	hintRaw := strings.Join(allPills, "  ")
	if lipgloss.Width(hintRaw) > maxHintW {
		// Fall back to minimal set.
		minimal := append(contextGroup[:min(len(contextGroup), 4)], sep)
		minimal = append(minimal, pill("?", "help"), pill("q", "quit"))
		hintRaw = strings.Join(minimal, "  ")
	}

	// ── Left: status message ───────────────────────────────────────────────
	hintW := lipgloss.Width(hintRaw)
	msgMaxW := a.width - hintW - 4
	if msgMaxW < 0 {
		msgMaxW = 0
	}
	msg := msgStyle.Render(truncateStr(a.statusMsg, msgMaxW))

	gap := a.width - lipgloss.Width(msg) - hintW - 2
	if gap < 1 {
		gap = 1
	}

	bar := msg + strings.Repeat(" ", gap) + hintRaw
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#111827")).
		Width(a.width).
		Render(bar)
}

func (a *App) helpContent() string {
	// ── Styles ────────────────────────────────────────────────────────────────
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).Bold(true)
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#06B6D4")).Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF"))
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151"))
	colDivStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151"))

	// row: fixed-width key badge + description, left-padded.
	row := func(k, desc string) string {
		return " " + keyStyle.Render(fmt.Sprintf("%-14s", k)) + descStyle.Render(desc)
	}
	blank := ""

	// ── Column 1: Navigation + Inventory ──────────────────────────────────────
	col1 := strings.Join([]string{
		sectionStyle.Render(" Navigation"),
		row("tab/shift+tab", "cycle panels"),
		row("1 2 3 4", "jump to panel"),
		row("j / k", "up / down"),
		row("g / G", "top / bottom"),
		blank,
		sectionStyle.Render(" Inventory"),
		row("enter", "set run limit"),
		row("space", "expand / collapse"),
		row("v", "variable browser"),
		row("E", "edit vars in $EDITOR"),
		row("!", "ad-hoc command"),
		row("I", "reload inventory"),
		row("N", "switch environment"),
	}, "\n")

	// ── Column 2: Playbooks + Logs ────────────────────────────────────────────
	col2 := strings.Join([]string{
		sectionStyle.Render(" Playbooks"),
		row("r / enter", "run playbook"),
		row("space", "view YAML source"),
		row("E", "edit in $EDITOR"),
		row("c", "--check mode"),
		row("d", "--diff mode"),
		row("t", "tags browser"),
		row("e", "--extra-vars"),
		row("L", "ansible-lint"),
		blank,
		sectionStyle.Render(" Logs"),
		row("/", "search"),
		row("n / N", "next / prev match"),
		row("f", "filter by level"),
		row("ctrl+d / ctrl+u", "half-page scroll"),
		row("T", "toggle timestamps"),
		row("ctrl+l", "clear"),
		row("Z", "fullscreen toggle"),
		row("X", "export → Markdown"),
	}, "\n")

	// ── Column 3: Run Control + Tools + Global ────────────────────────────────
	col3 := strings.Join([]string{
		sectionStyle.Render(" Run control"),
		row("V", "vault password"),
		row("H", "history browser"),
		row("R", "retry failed hosts"),
		blank,
		sectionStyle.Render(" Tools"),
		row("O", "role browser"),
		row("P", "SSH profiles"),
		row("A", "Ansible Galaxy"),
		row("F", "run profiles"),
		blank,
		sectionStyle.Render(" Vars overlay"),
		row("e", "edit vars file"),
		blank,
		sectionStyle.Render(" Global"),
		row("?", "toggle help"),
		row("q / ctrl+c", "quit"),
		row("click", "focus panel"),
	}, "\n")

	// ── Sizing ────────────────────────────────────────────────────────────────
	// Use up to 95% of terminal width, capped at 132 cols.
	boxW := min(a.width-2, 132)
	if boxW < 60 {
		boxW = 60
	}
	// Each column gets a third of inner box width; dividers add 1 col each.
	innerW := boxW - 6 // overlayBoxStyle has Padding(1,2) = 4 + border 2
	colW := (innerW - 4) / 3

	_ = colDivStyle // used via BorderForeground below
	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(colW).Render(col1),
		lipgloss.NewStyle().
			Width(1).PaddingLeft(1).PaddingRight(1).
			BorderLeft(true).BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Render(""),
		lipgloss.NewStyle().Width(colW).Render(col2),
		lipgloss.NewStyle().
			Width(1).PaddingLeft(1).PaddingRight(1).
			BorderLeft(true).BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Render(""),
		lipgloss.NewStyle().Width(colW).Render(col3),
	)

	// ── Header ────────────────────────────────────────────────────────────────
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#06B6D4")).Bold(true)
	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED"))
	hintStyle2 := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563"))

	title := titleStyle.Render("lazyansible ") +
		versionStyle.Render("v"+version) +
		titleStyle.Render(" — keyboard shortcuts")
	subhint := hintStyle2.Render("press any key to close")
	headerGap := boxW - 6 - lipgloss.Width(title) - lipgloss.Width(subhint)
	if headerGap < 1 {
		headerGap = 1
	}
	header := title + strings.Repeat(" ", headerGap) + subhint
	divider := dimStyle.Render(strings.Repeat("─", boxW-6))

	content := header + "\n" + divider + "\n" + cols

	// ── Clip to terminal height so the title is never off-screen ─────────────
	maxBoxH := a.height - 4
	if maxBoxH < 10 {
		maxBoxH = 10
	}
	content = clipLines(content, maxBoxH-2) // -2 for box border

	return overlayBoxStyle.Width(boxW).Render(content)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// inventoryBaseDir returns the directory containing the inventory file,
// or the working directory if no inventory is configured.
func inventoryBaseDir(cfg Config) string {
	if cfg.InventoryPath != "" {
		return filepath.Dir(cfg.InventoryPath)
	}
	return cfg.WorkDir
}

// editorOpenVarsFile finds or creates a vars file then opens it in $EDITOR.
func editorOpenVarsFile(baseDir, subdir, entityName string) tea.Cmd {
	return func() tea.Msg {
		path, err := editor.FindOrCreate(baseDir, subdir, entityName)
		if err != nil {
			return editor.DoneMsg{Err: err}
		}
		// We must run tea.ExecProcess synchronously via a Cmd.
		// Return an openEditorCmd so app can dispatch it as a tea.Cmd.
		return editorOpenPathMsg{path: path}
	}
}

// editorOpenPathMsg carries a path that should be opened in the editor.
type editorOpenPathMsg struct{ path string }

// handleMouseClick focuses the panel that contains the clicked cell.
func (a *App) handleMouseClick(x, y int) {
	if a.mode != AppModeNormal || a.logsFullscreen {
		return
	}
	// Row 0 = header, rows 1..topH = top panels, rows topH+1.. = logs.
	topH := topPanelHeight
	if topH > a.height-6 {
		topH = a.height - 6
	}

	if y == 0 || y >= a.height-1 {
		return // header or statusbar
	}

	if y >= 1 && y <= topH {
		// Top row: determine which column.
		invW := a.width / 4
		pbW := a.width / 4
		switch {
		case x < invW:
			a.focused = core.PanelInventory
		case x < invW+pbW:
			a.focused = core.PanelPlaybooks
		default:
			a.focused = core.PanelStatus
		}
	} else {
		a.focused = core.PanelLogs
	}
	a.updateFocus()
}

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
	a.galaxyOverlay.width = a.width
	a.galaxyOverlay.height = a.height
	a.runProfilesOverlay.width = a.width
	a.runProfilesOverlay.height = a.height
	a.pbViewerOverlay.width = a.width
	a.pbViewerOverlay.height = a.height
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

	// Echo the full command as the first log line so the user always knows
	// exactly what is being executed.
	a.logsPanel.AddLine(core.LogLine{
		Text:      "$ " + runner.BuildPlaybookCommand(opts),
		Level:     core.LogLevelCommand,
		Timestamp: time.Now(),
	})

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

	// Echo the full command as the first log line.
	a.logsPanel.AddLine(core.LogLine{
		Text:      "$ " + runner.BuildAdHocCommand(opts),
		Level:     core.LogLevelCommand,
		Timestamp: time.Now(),
	})

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

	// Desktop notification.
	if a.notifyOnFinish {
		pbName := a.pbPanel.SelectedPlaybook()
		name := "playbook"
		if pbName != nil {
			name = pbName.Name
		}
		dur := ""
		if msg.Duration > 0 {
			dur = msg.Duration.Round(time.Second).String()
		}
		exitCode := msg.ExitCode
		go notify.Send(notify.RunResult{
			PlaybookName: name,
			ExitCode:     exitCode,
			Duration:     dur,
		})
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

// playbookStdNames are the well-known root-level playbook file names.
var playbookStdNames = []string{
	"playbook.yml", "playbook.yaml",
	"site.yml", "site.yaml",
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

		// If nothing found, also search the parent directory for standard names.
		if len(pbs) == 0 {
			parent := filepath.Dir(dir)
			if parent != dir {
				pbsParent, _ := inventory.DiscoverPlaybooks(parent)
				pbs = append(pbs, pbsParent...)
			}
		}

		// Additionally surface any standard-named playbooks in . and .. that
		// the walker might have skipped (e.g. site.yml at repo root above dir).
		seen := map[string]bool{}
		for _, p := range pbs {
			seen[p.Path] = true
		}
		for _, searchDir := range []string{dir, filepath.Dir(dir)} {
			for _, name := range playbookStdNames {
				p := filepath.Join(searchDir, name)
				abs, _ := filepath.Abs(p)
				if seen[abs] {
					continue
				}
				if extra, ok := inventory.ParseSinglePlaybook(p); ok {
					seen[abs] = true
					pbs = append(pbs, extra)
				}
			}
		}

		return playbooksLoadedMsg{pbs: pbs}
	}
}

// ─── v0.6 helpers ─────────────────────────────────────────────────────────────

// checkGalaxyBinary returns an error if ansible-galaxy is not available.
func checkGalaxyBinary() error {
	return galaxy.CheckBinary()
}

// applyRunProfile loads a saved profile into the active run state.
func (a *App) applyRunProfile(p runprofiles.Profile) {
	// Switch inventory if specified and different.
	if p.Inventory != "" && p.Inventory != a.config.InventoryPath {
		_ = a.switchInventory(p.Inventory)
	}
	// Apply playbook selection by name.
	if p.Playbook != "" {
		a.pbPanel.SelectByName(p.Playbook)
	}
	// Apply limit.
	if p.Limit != "" {
		a.pbPanel.SetLimit(p.Limit)
	}
	// Apply tags.
	if len(p.Tags) > 0 {
		a.pbPanel.SetActiveTags(strings.Join(p.Tags, ","))
	} else {
		a.pbPanel.SetActiveTags("")
	}
	// Apply extra-vars.
	a.extraVarsRaw = p.ExtraVars
	a.pbPanel.SetExtraVars(p.ExtraVars)
	// Apply modes.
	a.pbPanel.SetCheckMode(p.CheckMode)
	a.pbPanel.SetDiffMode(p.DiffMode)
}
