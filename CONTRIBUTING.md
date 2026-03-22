# Contributing to lazyansible

Thank you for your interest in contributing. Please take a moment to read these guidelines before opening an issue or pull request.

---

## Reporting Issues

- Search existing issues before opening a new one.
- Include your OS, Go version, Ansible version, and terminal emulator.
- For panics or unexpected output, attach the relevant log lines or a minimal reproduction.

## Development Setup

```bash
git clone https://github.com/kocierik/lazyansible
cd lazyansible
go mod download

# Build
go build -o lazyansible ./cmd/lazyansible

# Run against the bundled sample project
./lazyansible -i ansible/inventories/local.ini -d ansible/playbooks/
```

**Requirements:** Go 1.21+, `ansible-playbook` in `$PATH`.

## Code Style

- Standard `gofmt` formatting — run `go fmt ./...` before committing.
- Keep packages small and focused. The `core` package must have no UI or I/O dependencies.
- New overlays go in `internal/ui/` and follow the existing pattern: a struct with `Update(tea.Msg) tea.Cmd` and `View() string`.
- Avoid `evalCmd` for long-running or timer-based commands — return the command to Bubble Tea for async execution.

## Pull Request Guidelines

1. Fork the repository and create a branch from `main`.
2. Keep each PR focused on a single feature or fix.
3. Add or update entries in `CHANGELOG.md` under `## [Unreleased]`.
4. Ensure `go build ./...` and `go vet ./...` pass with no errors.
5. Describe the motivation and approach in the PR description.

## Commit Style

Use short, descriptive commit messages in the imperative mood:

```
add log level filter to logs panel
fix border misalignment when terminal width < 80
remove vars browser overlay
```
