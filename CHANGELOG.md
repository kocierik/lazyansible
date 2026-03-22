# Changelog

All notable changes to lazyansible are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [1.0.0] — 2026-03-21

First stable release.

### Added
- Four-panel TUI layout: Inventory, Playbooks, Status, Logs
- INI and YAML inventory parser with `group_vars` / `host_vars` directory support
- Playbook auto-discovery (checks current dir, parent dir, and `inventories/` subdirectory)
- Live log streaming with TASK/PLAY section headers and colour coding
- Per-host status tracking (ok / changed / failed / unreachable counters)
- `--check` and `--diff` mode toggles with badges
- Host and group limit selection from the inventory tree
- Tags browser with multi-select and live filter
- `--extra-vars` text input prompt
- Ad-hoc command runner with `--become` toggle
- Ansible Vault password overlay — auto-detects encrypted files, stores password in a temp file
- Run history — every run persisted to `~/.lazyansible/history/`; re-run with `H`
- Retry failed hosts in one keystroke (`R`)
- Role browser — two-pane overlay with tasks, defaults, handlers, dependencies; run directly
- Multi-environment switcher — hot-swap inventory at runtime (`N`)
- SSH profile manager — save and apply named connection profiles (`P`)
- Diff visualisation — `+`/`-` lines colour-coded in logs
- `ansible-lint` integration — lint selected playbook before running (`L`)
- Export run summary as Markdown (`X`)
- Mouse support — click to focus any panel; `--no-mouse` to disable
- Log search — `/` to open inline search bar, `n`/`N` to navigate matches
- Run profiles — save and load named run configurations (`F`)
- Ansible Galaxy browser — list installed roles/collections, install new ones (`A`)
- Inventory live reload without restarting (`I`)
- Playbook YAML viewer with syntax highlighting (`space`)
- Log level filter — cycle all / failed / changed / ok / warning (`f`)
- Desktop notifications on run completion (Linux: `notify-send`, macOS: `osascript`)
- Config file `~/.lazyansible/config.yml` with `--init-config` flag
- Inline editor — open playbooks and `group_vars`/`host_vars` files in `$EDITOR` (`E`)
- Executed command echoed as the first log line on every run
- Word-wrapped command display in the logs panel
- Fullscreen log toggle (`Z`)

### Fixed
- Keyboard input lag in text-input overlays (blink-cursor command no longer evaluated synchronously)
- Mouse motion events no longer trigger full UI rerenders
- Log panel bottom border no longer disappears during playbook execution
- `group_vars` / `host_vars` files created in the correct directory (parent of `inventories/`) when editing

---

## [Unreleased]

_Nothing yet._
