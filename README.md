# lazyansible

A terminal-based UI for Ansible, inspired by [lazydocker](https://github.com/jesseduffield/lazydocker).

Manage inventories, run playbooks, and monitor execution — all from a fast, keyboard-driven TUI.

---

## Why lazyansible?

Running Ansible from the CLI is powerful but low-visibility: you get a wall of text, no easy host status overview, and no way to navigate your inventory interactively. lazyansible fixes this by wrapping Ansible in a panel-based UI that shows inventory, playbooks, per-host status, and streaming logs — all in one terminal window.

---

## Features

| Feature | Status |
|---|---|
| Inventory Explorer (INI + YAML) | ✅ v0.1 |
| Playbook Discovery | ✅ v0.1 |
| Panel-based TUI (inventory / playbooks / status / logs) | ✅ v0.1 |
| Vim-style keyboard navigation | ✅ v0.1 |
| Playbook Runner with live log streaming | ✅ v0.1 |
| Per-host status tracking | ✅ v0.1 |
| `--check` / `--diff` mode toggles | ✅ v0.1 |
| Host/group limit selection from inventory | ✅ v0.1 |
| Log colorisation (ok / changed / failed) | ✅ v0.1 |
| Help overlay (`?`) | ✅ v0.1 |
| Variable browser for hosts and groups (`v`) | ✅ v0.2 |
| Tags browser with multi-select and filter (`t`) | ✅ v0.2 |
| `--extra-vars` input prompt (`e`) | ✅ v0.2 |
| Ad-hoc command runner with `--become` toggle (`!`) | ✅ v0.2/v0.3 |
| Context-aware status bar hints | ✅ v0.2 |
| Ansible Vault password overlay (`V`) | ✅ v0.3 |
| Run history browser with re-run (`H`) | ✅ v0.3 |
| Retry failed hosts in one keystroke (`R`) | ✅ v0.3 |
| Logs panel: TASK/PLAY header formatting | ✅ v0.3 |
| Logs panel: correct height (62% of terminal) | ✅ v0.3 |
| Logs panel: scroll position indicator + timestamp toggle | ✅ v0.3 |
| Role browser — tasks, defaults, handlers, deps (`O`) | ✅ v0.4 |
| Run any role directly from the browser | ✅ v0.4 |
| Multi-environment switcher — hot-swap inventory at runtime (`N`) | ✅ v0.4 |
| SSH profile manager — save & apply connection params (`P`) | ✅ v0.4 |
| Diff visualisation — `+`/`-` lines colour-coded in logs panel | ✅ v0.4 |

---

## Installation

### From source

```bash
git clone https://github.com/kocierik/lazyansible
cd lazyansible
go build -o lazyansible ./cmd/lazyansible
sudo mv lazyansible /usr/local/bin/
```

**Requirements:**
- Go 1.21+
- `ansible-playbook` in your `$PATH`

---

## Usage

```bash
# Auto-detect inventory and playbooks in the current directory
lazyansible

# Specify an inventory file
lazyansible -i inventory.ini

# Specify a playbook directory
lazyansible -d ./playbooks

# Both
lazyansible -i inventories/staging.yaml -d playbooks/
```

---

## Keyboard Reference

| Key | Context | Action |
|---|---|---|
| `tab` / `shift+tab` | Global | Cycle focus between panels |
| `1` `2` `3` `4` | Global | Jump directly to panel |
| `j` / `k` | Any panel | Move cursor down / up |
| `g` / `G` | Any panel | Jump to top / bottom |
| `enter` / `space` | Inventory | Expand / collapse group |
| `enter` on host/group | Inventory | Set as playbook run limit |
| `v` | Inventory | Variable browser for host/group |
| `!` | Inventory / Playbooks | Ad-hoc command runner (with `--become` toggle) |
| `r` / `enter` | Playbooks | Run selected playbook |
| `c` | Playbooks | Toggle `--check` mode |
| `d` | Playbooks | Toggle `--diff` mode |
| `t` | Playbooks | Tags browser (multi-select + filter) |
| `e` | Playbooks | Set `--extra-vars` |
| `j` / `k` | Logs | Scroll down / up |
| `ctrl+d` / `ctrl+u` | Logs | Half-page scroll |
| `G` | Logs | Jump to bottom (auto-scroll) |
| `T` | Logs | Toggle timestamps |
| `ctrl+l` | Logs | Clear logs |
| `V` | Global | Set Ansible Vault password |
| `H` | Global | Run history browser (re-run any past run) |
| `R` | Global | Retry failed hosts from last run |
| `O` | Global | Role browser (inspect tasks/defaults, run role) |
| `N` | Global | Switch environment / inventory file at runtime |
| `P` | Global | SSH profile manager (save & apply connection params) |
| `?` | Global | Toggle help overlay |
| `q` / `ctrl+c` | Global | Quit (cancels active run) |

---

## Architecture

```
cmd/lazyansible/          # Entry point & CLI flags
internal/
  core/                   # Domain types (Inventory, Playbook, AdHocOptions, …)
  inventory/
    parser.go             # INI and YAML inventory parsers
    playbooks.go          # Playbook discovery + tag extraction
  runner/
    runner.go             # ansible-playbook / ansible execution + streaming
  ui/
    app.go                # Root Bubble Tea model, mode/overlay state machine
    styles.go             # Lip Gloss colour palette
    vars_overlay.go       # Variable browser modal
    adhoc_overlay.go      # Ad-hoc command form modal (with become toggle)
    extravars_overlay.go  # --extra-vars text-input modal
    tags_overlay.go       # Tags multi-select + filter modal
    vault_overlay.go      # Vault password input modal
    history_overlay.go    # Run history browser + re-run modal
    roles_overlay.go      # Two-pane role browser + temp playbook runner
    envswitch_overlay.go  # Runtime inventory switcher
    sshprofile_overlay.go # SSH profile form + apply
internal/
  history/
    history.go            # JSON run records in ~/.lazyansible/history/
  vault/
    vault.go              # Vault file detection + temp password file helper
  roles/
    scanner.go            # Scan roles/ dir, parse tasks/defaults/handlers/meta
  ssh/
    profiles.go           # SSH profiles stored in ~/.lazyansible/ssh-profiles.json
    panels/
      inventory.go        # Inventory tree panel
      playbooks.go        # Playbook list panel (badges, inline tags)
      status.go           # Per-host status panel
      logs.go             # Streaming log panel
configs/                  # Example inventory files
ansible/                  # Sample Ansible project for testing
```

**Design principles:**

- **Event-driven** – ansible output is streamed as `tea.Msg` values via goroutines, keeping the UI non-blocking.
- **Clean layers** – the `core` package holds domain types with no UI or I/O dependencies. Panels depend on `core`, never on each other.
- **Single source of truth** – the root `App` model owns all state; panels are stateless renderers with local cursor tracking.

---

## Roadmap

### v0.1 ✅
- [x] INI and YAML inventory parsing
- [x] Playbook auto-discovery
- [x] Four-panel TUI layout (inventory / playbooks / status / logs)
- [x] Live log streaming with colour coding
- [x] Per-host status display
- [x] `--check` / `--diff` toggles
- [x] Host / group limit from inventory panel

### v0.2 ✅ (current)
- [x] Variable browser — press `v` on any host or group to inspect vars
- [x] Tags browser — press `t` to browse, filter and multi-select playbook tags
- [x] `--extra-vars` prompt — press `e` to set free-form extra variables
- [x] Ad-hoc command runner — press `!` with a target selected to run any module
- [x] Tag extraction from playbook YAML (recursive, handles `block/rescue/always`)
- [x] Inline tag preview under selected playbook
- [x] Context-aware status bar hints (different per focused panel)
- [x] `AdHocStreamCmd` for `ansible` binary alongside `ansible-playbook`

### v0.3 ✅
- [x] **Logs panel redesign** – correct 62% height, TASK/PLAY headers, scroll indicator, timestamp toggle (`T`)
- [x] **Ansible Vault** – auto-detects encrypted files on startup, `V` to set password (temp file, auto-cleaned)
- [x] **Run history** – every run saved to `~/.lazyansible/history/`, `H` to browse and re-run
- [x] **Retry failed hosts** – `R` after a failed run to set the limit to failed hosts
- [x] **`--become` toggle** in ad-hoc overlay (tab to reach, space/enter to toggle)
- [x] Layout fix: `resizePanels` now uses correct border math (no more empty space)

### v0.4 ✅ (current)
- [x] **Role browser** (`O`) — two-pane overlay listing roles with tasks, defaults, handlers, deps; press `enter` to run the role directly via a generated temp playbook
- [x] **Multi-environment switcher** (`N`) — discover all inventory files in the project and hot-swap the active one at runtime without restarting
- [x] **SSH profile manager** (`P`) — create, delete and apply named SSH connection profiles (`ansible_user`, `ansible_ssh_private_key_file`, etc.) as extra-vars; stored in `~/.lazyansible/ssh-profiles.json`; 🔑 badge shown when a profile is active
- [x] **Diff visualisation** — `--diff` output lines now rendered with distinct colours: `+` lines bright green, `-` lines bright red, `@@` hunks cyan, `--- / +++` headers bold white

### v0.5 (planned)
- [ ] **Auto-discovery improvement** — check both `.` and `..` for common inventory and playbook names:
  - Inventory: `inventory.yml`, `inventory.yaml`, `hosts.yml`, `hosts.yaml`, `hosts`
  - Playbook: `playbook.yml`, `playbook.yaml`, `site.yml`, `site.yaml`
- [ ] Plugin system for custom panels
- [ ] Mouse support
- [ ] Export run summary as Markdown
- [ ] Integration with AWX / Ansible Tower API
- [ ] Role Galaxy integration (browse & install roles)

### Backlog / ideas
- [ ] Multi-pane diff viewer (side-by-side before/after)
- [ ] Ansible lint integration
- [ ] Export run summary as Markdown
- [ ] Integration with AWX / Ansible Tower API

---

## Contributing

PRs welcome. Please keep changes focused and idiomatic Go.

---

## License

MIT
