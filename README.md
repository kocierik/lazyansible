# lazyansible

<p align="center">
  <img src="https://img.shields.io/badge/version-1.0.0-7C3AED?style=flat-square" alt="version">
  <img src="https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat-square&logo=go" alt="go version">
  <img src="https://img.shields.io/badge/license-MIT-22C55E?style=flat-square" alt="license">
  <img src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS-06B6D4?style=flat-square" alt="platform">
</p>

<p align="center">
  <strong>A terminal UI for Ansible — manage inventories, run playbooks, stream logs.</strong><br>
  Inspired by <a href="https://github.com/jesseduffield/lazydocker">lazydocker</a>.
</p>

---

<img width="1331" height="707" alt="lazyansible TUI" src="https://github.com/user-attachments/assets/fb7a46d6-4417-47d7-97c2-7fd8f28c4c2f" />

---

## Why lazyansible?

Running Ansible from the CLI is powerful but low-visibility: you get a wall of text, no easy host status overview, and no way to navigate your inventory interactively.

**lazyansible** wraps Ansible in a panel-based TUI that shows inventory, playbooks, per-host status, and streaming logs — all in one terminal window, all keyboard-driven.

---

## Features

- **Inventory Explorer** — browse INI and YAML inventories as a collapsible tree; set host/group limits with a single keypress
- **Playbook Runner** — discover and run playbooks with `--check`, `--diff`, tags, extra-vars and limit; see the exact command echoed in the log panel
- **Live Log Streaming** — colorised output with TASK/PLAY section headers, scroll, search, and level filter (failed / changed / ok)
- **Per-host Status** — real-time ok / changed / failed / unreachable counters for every host
- **Ansible Vault** — auto-detects encrypted files; set the password once and it's used for every run
- **Ad-hoc Commands** — run any Ansible module against any host or group, with optional `--become`
- **Run History** — every run is persisted; browse and re-run any past execution
- **Role Browser** — inspect tasks, defaults, handlers, and dependencies; run a role directly
- **Ansible Galaxy** — browse installed roles and collections, install new ones with live output
- **SSH Profiles** — save named connection configurations and apply them as extra-vars
- **Run Profiles** — save the full run configuration (playbook + limit + tags + extra-vars + flags) as a named profile
- **Multi-environment** — hot-swap inventory files at runtime without restarting
- **Inline Editor** — open any playbook or `group_vars`/`host_vars` file in `$EDITOR` without leaving the TUI
- **Desktop Notifications** — get notified when a long run completes (Linux: `notify-send`, macOS: `osascript`)
- **Config File** — persistent defaults via `~/.lazyansible/config.yml`

---

## Installation

### go install

```bash
go install github.com/kocierik/lazyansible/cmd/lazyansible@latest
```

### From source

```bash
git clone https://github.com/kocierik/lazyansible
cd lazyansible
go build -o lazyansible ./cmd/lazyansible
sudo mv lazyansible /usr/local/bin/
```

### Requirements

- Go 1.21 or later
- `ansible-playbook` in `$PATH`
- `ansible` (for ad-hoc commands)
- `ansible-lint` *(optional — for lint integration)*
- `ansible-galaxy` *(optional — for Galaxy browser)*
- `notify-send` on Linux or `terminal-notifier` on macOS *(optional — for desktop notifications)*

---

## Quick Start

```bash
# Auto-discover inventory and playbooks in the current directory
lazyansible

# Specify an inventory file
lazyansible -i inventories/production.yml

# Specify a playbook directory
lazyansible -d playbooks/

# Both
lazyansible -i inventories/staging.yaml -d playbooks/

# Disable mouse capture (allows native terminal text selection)
lazyansible --no-mouse

# Create an annotated config file at ~/.lazyansible/config.yml
lazyansible --init-config
```

### Auto-discovery

When no flags are given, lazyansible searches for inventory and playbook files in the current directory and its parent, checking the following names in order:

| Type | Candidates |
|---|---|
| Inventory | `inventory.yml`, `inventory.yaml`, `hosts.yml`, `hosts.yaml`, `hosts`, `inventory`, `inventory.ini` |
| Playbooks | all `.yml` / `.yaml` files in `playbooks/`, `.`, and `..` |

### Config file

```yaml
# ~/.lazyansible/config.yml

# Default inventory (same as -i flag)
# inventory: ./inventories/hosts.yml

# Default playbook directory (same as -d flag)
# playbook_dir: ./playbooks

# Disable mouse so you can select terminal text normally
# no_mouse: false

# Send a desktop notification when a run completes
notify_on_finish: true

# Start with --check / --diff pre-enabled
# default_check_mode: false
# default_diff_mode: false
```

CLI flags always override config file values.

---

## Keyboard Reference

### Navigation

| Key | Action |
|---|---|
| `tab` / `shift+tab` | Cycle focus between panels |
| `1` `2` `3` `4` | Jump directly to Inventory / Playbooks / Status / Logs |
| `j` / `k` | Move cursor down / up |
| `g` / `G` | Jump to top / bottom |
| `?` | Toggle help overlay |
| `q` / `ctrl+c` | Quit (cancels active run) |

### Inventory panel

| Key | Action |
|---|---|
| `enter` / `space` | Expand / collapse group |
| `enter` on host or group | Set as run limit |
| `E` | Open `host_vars` / `group_vars` file in `$EDITOR` (creates if missing) |
| `!` | Ad-hoc command runner for selected host / group |

### Playbooks panel

| Key | Action |
|---|---|
| `r` / `enter` | Run selected playbook |
| `c` | Toggle `--check` mode |
| `d` | Toggle `--diff` mode |
| `t` | Tags browser (multi-select with filter) |
| `V` | Set `--extra-vars` |
| `L` | Run `ansible-lint` on selected playbook |
| `space` | View playbook YAML source with syntax highlighting |
| `E` | Open selected playbook in `$EDITOR` |
| `!` | Ad-hoc command runner |

### Logs panel

| Key | Action |
|---|---|
| `j` / `k` | Scroll down / up one line |
| `ctrl+d` / `ctrl+u` | Half-page scroll |
| `G` | Jump to bottom (resume auto-scroll) |
| `Z` | Toggle fullscreen logs |
| `/` | Open inline search bar |
| `n` / `N` | Jump to next / previous search match |
| `f` | Cycle log level filter: all → failed → changed → ok → warning |
| `T` | Toggle timestamps |
| `ctrl+l` | Clear logs |

### Global overlays

| Key | Action |
|---|---|
| `ctrl+V` | Ansible Vault password prompt |
| `H` | Run history browser (browse and re-run past executions) |
| `R` | Retry failed hosts from last run |
| `O` | Role browser (inspect and run roles) |
| `N` | Switch environment / inventory file at runtime |
| `P` | SSH profile manager |
| `A` | Ansible Galaxy browser (list and install roles / collections) |
| `F` | Run profiles — save or load named run configurations |
| `I` | Live-reload inventory and playbooks |
| `X` | Export run summary as a Markdown file |

---

## Project Layout

```
cmd/lazyansible/          Entry point, CLI flag parsing
internal/
  core/                   Domain types — Inventory, Host, Group, Playbook, LogLine
  inventory/
    parser.go             INI + YAML inventory parser; loads group_vars / host_vars
    playbooks.go          Playbook discovery and tag extraction
  runner/
    runner.go             ansible-playbook / ansible execution with live streaming
  history/                Run records persisted in ~/.lazyansible/history/
  vault/                  Vault-file detection and temp password-file helper
  roles/                  Role scanner (tasks, defaults, handlers, meta)
  ssh/                    SSH profile persistence (~/.lazyansible/ssh-profiles.json)
  galaxy/                 ansible-galaxy CLI wrapper
  runprofiles/            Named run config persistence
  notify/                 Desktop notification helper (notify-send / osascript)
  config/                 User config loader (~/.lazyansible/config.yml)
  editor/                 $EDITOR launcher via tea.ExecProcess
  ui/
    app.go                Root Bubble Tea model — layout, keybindings, state machine
    styles.go             Lip Gloss colour palette and shared overlay styles
    adhoc_overlay.go      Ad-hoc command form
    extravars_overlay.go  --extra-vars text input
    tags_overlay.go       Tags multi-select browser
    vault_overlay.go      Vault password input
    history_overlay.go    Run history browser
    roles_overlay.go      Two-pane role browser
    envswitch_overlay.go  Runtime inventory switcher
    sshprofile_overlay.go SSH profile manager
    galaxy_overlay.go     Ansible Galaxy browser
    runprofiles_overlay.go Run profile save/load
    playbookviewer_overlay.go YAML source viewer
    export.go             Markdown run-report exporter
    panels/
      inventory.go        Inventory tree panel
      playbooks.go        Playbook list panel
      status.go           Per-host status panel
      logs.go             Streaming log panel with search and filter
```

---

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

---

## License

[MIT](LICENSE)
