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
| Auto-discovery checks `.` **and** `..` for standard inventory/playbook names | ✅ v0.5 |
| `ansible-lint` integration — lint selected playbook before running (`L`) | ✅ v0.5 |
| Export run summary as Markdown with host status table (`X`) | ✅ v0.5 |
| Mouse support — click to focus any panel | ✅ v0.5 |
| Log search — `/` to filter, `n`/`N` to jump between highlighted matches | ✅ v0.6 |
| Run profiles — save/load named run configurations (`F`) | ✅ v0.6 |
| Ansible Galaxy browser — list & install roles/collections (`A`) | ✅ v0.6 |
| Inventory live reload without restarting (`I`) | ✅ v0.7 |
| Playbook YAML viewer with syntax highlighting (`space` on playbook) | ✅ v0.7 |
| Log level filter — cycle all/failed/changed/ok (`f` in logs) | ✅ v0.7 |
| Desktop notifications on run finish (via `notify-send` / `osascript`) | ✅ v0.7 |
| Config file `~/.lazyansible/config.yml` for persistent defaults | ✅ v0.7 |

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

# Disable mouse (allows native terminal text selection)
lazyansible --no-mouse

# Create a config file with documented defaults
lazyansible --init-config
```

### Config file

Run `lazyansible --init-config` to create `~/.lazyansible/config.yml`:

```yaml
# Default inventory file (same as -i flag).
# inventory: ./inventories/hosts.yml

# Default playbook search directory (same as -d flag).
# playbook_dir: ./playbooks

# Disable mouse capture so you can select text normally.
# no_mouse: false

# Send a desktop notification when a run completes.
notify_on_finish: true

# Start with --check / --diff pre-enabled.
# default_check_mode: false
# default_diff_mode: false
```

CLI flags always take precedence over config file values.

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
| `L` | Playbooks | Run `ansible-lint` on selected playbook |
| `X` | Global | Export logs + run metadata as Markdown file |
| `/` | Logs (focused) | Open search bar; highlights matching lines |
| `n` / `N` | Logs (focused) | Jump to next / previous search match |
| `A` | Global | Ansible Galaxy browser (list & install roles/collections) |
| `F` | Global | Run profiles — save current config or load a saved one |
| `I` | Global | Live reload inventory + playbooks |
| `space` | Playbooks | View playbook YAML source with syntax highlighting |
| `E` | Playbooks | Open selected playbook directly in `$EDITOR` |
| `E` | Inventory | Open (or create) `host_vars`/`group_vars` file for selected host/group in `$EDITOR` |
| `e` | Vars overlay | Open (or create) vars file for the current host/group in `$EDITOR` |
| `e` | Playbook viewer | Edit the viewed playbook in `$EDITOR`; reloads on return |
| `f` | Logs (focused) | Cycle log level filter (all → failed → changed → ok → warning) |
| click | Global | Mouse click focuses the panel under the cursor |
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
    export.go             # Markdown run-report exporter
    galaxy_overlay.go     # Ansible Galaxy browser (list/install roles & collections)
    runprofiles_overlay.go # Run profiles — save/load named configurations
    playbookviewer_overlay.go # YAML source viewer with syntax highlighting
internal/
  history/
    history.go            # JSON run records in ~/.lazyansible/history/
  vault/
    vault.go              # Vault file detection + temp password file helper
  roles/
    scanner.go            # Scan roles/ dir, parse tasks/defaults/handlers/meta
  ssh/
    profiles.go           # SSH profiles stored in ~/.lazyansible/ssh-profiles.json
  galaxy/
    galaxy.go             # ansible-galaxy CLI wrapper (list/install roles & collections)
  runprofiles/
    profiles.go           # Named run configurations stored in ~/.lazyansible/run-profiles.json
  notify/
    notify.go             # Desktop notification helper (notify-send / osascript)
  config/
    config.go             # User config loader (~/.lazyansible/config.yml)
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

### v0.4 ✅
- [x] **Role browser** (`O`) — two-pane overlay listing roles with tasks, defaults, handlers, deps; press `enter` to run the role directly via a generated temp playbook
- [x] **Multi-environment switcher** (`N`) — discover all inventory files in the project and hot-swap the active one at runtime without restarting
- [x] **SSH profile manager** (`P`) — create, delete and apply named SSH connection profiles (`ansible_user`, `ansible_ssh_private_key_file`, etc.) as extra-vars; stored in `~/.lazyansible/ssh-profiles.json`; 🔑 badge shown when a profile is active
- [x] **Diff visualisation** — `--diff` output lines now rendered with distinct colours: `+` lines bright green, `-` lines bright red, `@@` hunks cyan, `--- / +++` headers bold white

### v0.5 ✅
- [x] **Auto-discovery improvement** — checks both `.` and `..` for standard names:
  - Inventory: `inventory.yml`, `inventory.yaml`, `hosts.yml`, `hosts.yaml`, `hosts`
  - Playbook: `playbook.yml`, `playbook.yaml`, `site.yml`, `site.yaml`
- [x] **`ansible-lint` integration** (`L`) — run the linter on the selected playbook; output streams live in the log panel; header shows `⚑ LINTING` badge
- [x] **Export as Markdown** (`X`) — saves `lazyansible-run-TIMESTAMP.md` in the working directory with run metadata, host status table, and full log output
- [x] **Mouse support** — click to focus any panel (inventory, playbooks, status, logs)

### v0.6 ✅
- [x] **Log search** (`/`) — opens an interactive search bar in the logs panel; matching lines are highlighted (current match in purple, others in dark indigo); press `n`/`N` to jump between matches, `Esc` to close
- [x] **Run profiles** (`F`) — save the current run configuration (playbook, limit, tags, extra-vars, --check/--diff, inventory) as a named profile stored in `~/.lazyansible/run-profiles.json`; load it back with a single keystroke
- [x] **Ansible Galaxy browser** (`A`) — two-tab overlay listing all installed roles and collections; press `i` to install a new one via `ansible-galaxy role/collection install`; list auto-refreshes after each install

### v0.7 ✅ (current)
- [x] **Inventory live reload** (`I`) — reloads inventory file and rediscovers playbooks on the fly, no restart needed
- [x] **Playbook YAML viewer** (`space` on selected playbook) — scrollable overlay showing the raw YAML with syntax highlighting (keywords, Jinja2 templates, booleans, comments in distinct colours)
- [x] **Log level filter** (`f` in logs panel) — cycles through all → failed → changed → ok → warning; match count shown in title bar
- [x] **Desktop notifications** — sends a `notify-send` (Linux) or `osascript` (macOS) notification when a run completes; configurable via config file
- [x] **Config file** (`~/.lazyansible/config.yml`) — persistent defaults for inventory, playbook dir, mouse mode, notifications, check/diff mode; `--init-config` flag writes an annotated example
- [x] **Inline editor** (`E` on playbook / `E` on host-group / `e` in vars overlay / `e` in playbook viewer) — opens `$VISUAL` / `$EDITOR` / `nano` / `vi`; creates `host_vars` or `group_vars` files automatically if they don't exist; playbook viewer reloads file after editor exits

### v0.8 (planned)
- [ ] Inventory graph view (group hierarchy visualisation)
- [ ] Multi-pane diff viewer (side-by-side before/after)
- [ ] Export run summary as HTML
- [ ] Integration with AWX / Ansible Tower API

### Backlog / ideas
- [ ] Plugin system for custom panels
- [ ] Inventory graph view (group hierarchy visualisation)
- [ ] Export run summary as HTML
- [ ] AWX / Ansible Tower API integration

---

## Contributing

PRs welcome. Please keep changes focused and idiomatic Go.

---

## License

MIT
