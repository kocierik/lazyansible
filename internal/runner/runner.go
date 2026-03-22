// Package runner executes Ansible playbooks and streams output.
package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kocierik/lazyansible/internal/core"
)

// LogMsg is sent over the Bubble Tea message bus for each log line.
type LogMsg struct {
	Line core.LogLine
}

// RunFinishedMsg is sent when the ansible-playbook process exits.
type RunFinishedMsg struct {
	ExitCode int
	Err      error
}

// HostStatusMsg is sent when a host status change is detected in output.
type HostStatusMsg struct {
	Host   string
	Status core.TaskStatus
	Task   string
}

// StreamCmd returns a Bubble Tea command that spawns ansible-playbook and streams
// output messages back through the tea.Program's Send channel.
func StreamCmd(ctx context.Context, opts core.RunOptions, sendFn func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		args := buildPlaybookArgs(opts)
		return stream(ctx, "ansible-playbook", args, opts.Env, sendFn)
	}
}

// AdHocStreamCmd runs an ansible ad-hoc command and streams output.
func AdHocStreamCmd(ctx context.Context, opts core.AdHocOptions, sendFn func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		args := buildAdHocArgs(opts)
		return stream(ctx, "ansible", args, nil, sendFn)
	}
}

// CheckBinary returns an error if ansible-playbook is not found in PATH.
func CheckBinary() error {
	_, err := exec.LookPath("ansible-playbook")
	if err != nil {
		return fmt.Errorf("ansible-playbook not found in PATH: %w", err)
	}
	return nil
}

// CheckAdHocBinary returns an error if ansible is not found in PATH.
func CheckAdHocBinary() error {
	_, err := exec.LookPath("ansible")
	if err != nil {
		return fmt.Errorf("ansible not found in PATH: %w", err)
	}
	return nil
}

// CheckLintBinary returns an error if ansible-lint is not found in PATH.
func CheckLintBinary() error {
	_, err := exec.LookPath("ansible-lint")
	if err != nil {
		return fmt.Errorf("ansible-lint not found in PATH: %w", err)
	}
	return nil
}

// LintCmd runs ansible-lint on the given playbook path and streams output.
func LintCmd(ctx context.Context, playbookPath string, sendFn func(tea.Msg)) tea.Cmd {
	return func() tea.Msg {
		return stream(ctx, "ansible-lint", []string{"--nocolor", playbookPath}, nil, sendFn)
	}
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func stream(ctx context.Context, binary string, args []string, extraEnv []string, sendFn func(tea.Msg)) tea.Msg {
	cmd := exec.CommandContext(ctx, binary, args...)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return RunFinishedMsg{ExitCode: -1, Err: fmt.Errorf("stdout pipe: %w", err)}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return RunFinishedMsg{ExitCode: -1, Err: fmt.Errorf("stderr pipe: %w", err)}
	}

	if err := cmd.Start(); err != nil {
		return RunFinishedMsg{ExitCode: -1, Err: fmt.Errorf("start %s: %w", binary, err)}
	}

	done := make(chan struct{}, 2)
	streamPipe := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			text := scanner.Text()
			line := classifyLine(text)
			sendFn(LogMsg{Line: line})

			if status, host, task, ok := parseHostStatus(text); ok {
				sendFn(HostStatusMsg{Host: host, Status: status, Task: task})
			}
		}
		done <- struct{}{}
	}

	go streamPipe(stdout)
	go streamPipe(stderr)

	<-done
	<-done

	exitCode := 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return RunFinishedMsg{ExitCode: -1, Err: err}
		}
	}
	return RunFinishedMsg{ExitCode: exitCode}
}

func buildPlaybookArgs(opts core.RunOptions) []string {
	args := []string{opts.Playbook}
	if opts.Inventory != "" {
		args = append(args, "-i", opts.Inventory)
	}
	if opts.Limit != "" {
		args = append(args, "--limit", opts.Limit)
	}
	if opts.Tags != "" {
		args = append(args, "--tags", opts.Tags)
	}
	if opts.CheckMode {
		args = append(args, "--check")
	}
	if opts.DiffMode {
		args = append(args, "--diff")
	}
	for k, v := range opts.ExtraVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	if opts.ExtraVarsRaw != "" {
		args = append(args, "-e", opts.ExtraVarsRaw)
	}
	if opts.VaultPasswordFile != "" {
		args = append(args, "--vault-password-file", opts.VaultPasswordFile)
	}
	return args
}

func buildAdHocArgs(opts core.AdHocOptions) []string {
	hosts := opts.Hosts
	if hosts == "" {
		hosts = "all"
	}
	args := []string{hosts}
	if opts.Inventory != "" {
		args = append(args, "-i", opts.Inventory)
	}
	args = append(args, "-m", opts.Module)
	if opts.Args != "" {
		args = append(args, "-a", opts.Args)
	}
	if opts.Become {
		args = append(args, "--become")
	}
	for k, v := range opts.ExtraVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	return args
}

// classifyLine assigns a log level to a raw output line.
func classifyLine(text string) core.LogLine {
	lower := strings.ToLower(text)
	level := core.LogLevelInfo

	switch {
	case strings.HasPrefix(lower, "ok:"):
		level = core.LogLevelOK
	case strings.HasPrefix(lower, "changed:"):
		level = core.LogLevelChanged
	case strings.HasPrefix(lower, "failed:"), strings.HasPrefix(lower, "fatal:"):
		level = core.LogLevelFailed
	case strings.HasPrefix(lower, "warning:"), strings.Contains(lower, "[warning]"):
		level = core.LogLevelWarning
	case strings.Contains(lower, "task [") || strings.Contains(lower, "play ["):
		level = core.LogLevelInfo

	// ── Diff visualisation (--diff output) ────────────────────────────────
	case strings.HasPrefix(text, "--- ") || strings.HasPrefix(text, "+++ "):
		level = core.LogLevelDiffHeader
	case strings.HasPrefix(text, "@@"):
		level = core.LogLevelDiffHunk
	case len(text) > 0 && text[0] == '+':
		level = core.LogLevelDiffAdd
	case len(text) > 0 && text[0] == '-':
		level = core.LogLevelDiffRemove
	}

	return core.LogLine{
		Text:      text,
		Level:     level,
		Timestamp: time.Now(),
	}
}

// parseHostStatus extracts host/status from lines like:
//
//	ok: [hostname]
//	changed: [hostname]
//	failed: [hostname]
func parseHostStatus(text string) (status core.TaskStatus, host string, task string, ok bool) {
	lower := strings.ToLower(strings.TrimSpace(text))

	var s core.TaskStatus
	switch {
	case strings.HasPrefix(lower, "ok:"):
		s = core.TaskStatusOK
	case strings.HasPrefix(lower, "changed:"):
		s = core.TaskStatusChanged
	case strings.HasPrefix(lower, "failed:"), strings.HasPrefix(lower, "fatal:"):
		s = core.TaskStatusFailed
	case strings.HasPrefix(lower, "skipping:"):
		s = core.TaskStatusSkipped
	case strings.Contains(lower, "unreachable"):
		s = core.TaskStatusUnreachable
	default:
		return 0, "", "", false
	}

	start := strings.Index(text, "[")
	end := strings.Index(text, "]")
	if start == -1 || end == -1 || end <= start {
		return 0, "", "", false
	}
	hostName := text[start+1 : end]
	return s, hostName, "", true
}
